package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Zhou-JK/hls-streamer/internal/config"
	"github.com/Zhou-JK/hls-streamer/internal/handler"
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/queue"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
	"github.com/Zhou-JK/hls-streamer/internal/router"
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Database
	db, err := gorm.Open(mysql.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	// Auto-migrate
	if err := db.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.RefreshToken{},
		&model.Language{},
		&model.Video{},
		&model.VideoTranslation{},
		&model.VideoVariant{},
		&model.Thumbnail{},
		&model.Subtitle{},
		&model.Person{},
		&model.PersonTranslation{},
		&model.VideoCast{},
		&model.Category{},
		&model.CategoryTranslation{},
		&model.Tag{},
		&model.TagTranslation{},
		&model.TranscodeTask{},
		&model.TaskLog{},
		&model.DRMKey{},
		&model.Worker{},
		&model.ResolutionPreset{},
	); err != nil {
		slog.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}

	// Seed default data
	seedDefaults(db)

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	// S3
	s3Client, err := storage.NewS3Client(cfg.S3)
	if err != nil {
		slog.Error("failed to create S3 client", "error", err)
		os.Exit(1)
	}

	// Init Redis Streams
	producer := queue.NewProducer(rdb)
	if err := producer.InitStreams(context.Background()); err != nil {
		slog.Error("failed to init redis streams", "error", err)
		os.Exit(1)
	}

	// Repositories
	userRepo := repository.NewUserRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	categoryRepo := repository.NewCategoryRepo(db)
	workerRepo := repository.NewWorkerRepo(db)
	drmRepo := repository.NewDRMRepo(db)

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWT)
	videoSvc := service.NewVideoService(videoRepo)
	uploadSvc := service.NewUploadService(s3Client, videoRepo)
	transcodeSvc := service.NewTranscodeService(taskRepo, videoRepo, producer)
	categorySvc := service.NewCategoryService(categoryRepo)
	workerSvc := service.NewWorkerService(workerRepo)
	drmSvc := service.NewDRMService(drmRepo)

	// Handlers
	handlers := router.Handlers{
		Auth:      handler.NewAuthHandler(authSvc),
		Video:     handler.NewVideoHandler(videoSvc),
		Upload:    handler.NewUploadHandler(uploadSvc, transcodeSvc),
		Transcode: handler.NewTranscodeHandler(transcodeSvc),
		Worker:    handler.NewWorkerHandler(workerSvc, transcodeSvc),
		Category:  handler.NewCategoryHandler(categorySvc),
		DRM:       handler.NewDRMHandler(drmSvc, videoSvc),
		Playback:  handler.NewPlaybackHandler(videoRepo, s3Client),
	}

	// Router
	r := router.Setup(cfg, handlers)

	// HTTP Server
	srv := &http.Server{
		Addr:    cfg.Server.Addr(),
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		slog.Info("API server starting", "addr", cfg.Server.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	sqlDB.Close()
	rdb.Close()
	slog.Info("server stopped")
}

func seedDefaults(db *gorm.DB) {
	// Seed roles
	roles := []model.Role{
		{ID: 1, Name: "admin", Permissions: model.JSON(`{"*": true}`)},
		{ID: 2, Name: "editor", Permissions: model.JSON(`{"videos.*": true, "categories.*": true, "tags.*": true}`)},
		{ID: 3, Name: "viewer", Permissions: model.JSON(`{"videos.read": true}`)},
	}
	for _, r := range roles {
		db.FirstOrCreate(&r, "name = ?", r.Name)
	}

	// Seed languages
	langs := []model.Language{
		{Code: "en", Name: "English", NativeName: "English", IsDefault: true, IsActive: true, SortOrder: 1},
		{Code: "zh-CN", Name: "Chinese (Simplified)", NativeName: "简体中文", IsActive: true, SortOrder: 2},
		{Code: "zh-TW", Name: "Chinese (Traditional)", NativeName: "繁體中文", IsActive: true, SortOrder: 3},
		{Code: "ja", Name: "Japanese", NativeName: "日本語", IsActive: true, SortOrder: 4},
		{Code: "ko", Name: "Korean", NativeName: "한국어", IsActive: true, SortOrder: 5},
	}
	for _, l := range langs {
		db.FirstOrCreate(&l, "code = ?", l.Code)
	}

	// Seed resolution presets
	presets := []model.ResolutionPreset{
		{ID: 1, Name: "360p", Width: 640, Height: 360, BitrateKbps: 800, AudioBitrateKbps: 96, IsActive: true, SortOrder: 1},
		{ID: 2, Name: "480p", Width: 854, Height: 480, BitrateKbps: 1200, AudioBitrateKbps: 96, IsActive: true, SortOrder: 2},
		{ID: 3, Name: "720p", Width: 1280, Height: 720, BitrateKbps: 2500, AudioBitrateKbps: 128, IsActive: true, SortOrder: 3},
		{ID: 4, Name: "1080p", Width: 1920, Height: 1080, BitrateKbps: 4500, AudioBitrateKbps: 128, IsActive: true, SortOrder: 4},
		{ID: 5, Name: "1440p", Width: 2560, Height: 1440, BitrateKbps: 8000, AudioBitrateKbps: 192, IsActive: true, SortOrder: 5},
		{ID: 6, Name: "2160p", Width: 3840, Height: 2160, BitrateKbps: 15000, AudioBitrateKbps: 192, IsActive: true, SortOrder: 6},
	}
	for _, p := range presets {
		db.FirstOrCreate(&p, "name = ?", p.Name)
	}

	// Seed default admin user
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err == nil {
			admin := model.User{
				Username:     "admin",
				Email:        "admin@example.com",
				PasswordHash: string(hash),
				RoleID:       1,
				IsActive:     true,
			}
			db.Create(&admin)
			slog.Info("default admin user created", "username", "admin", "password", "admin123")
		}
	}
}
