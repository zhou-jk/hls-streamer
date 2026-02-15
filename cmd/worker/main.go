package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Zhou-JK/hls-streamer/internal/config"
	"github.com/Zhou-JK/hls-streamer/internal/queue"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
	"github.com/Zhou-JK/hls-streamer/internal/worker"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	apiBase := flag.String("api", "http://localhost:8080", "API server base URL")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Worker ID
	workerID := cfg.Worker.ID
	if workerID == "" {
		hostname, _ := os.Hostname()
		workerID = fmt.Sprintf("worker-%s-%s", hostname, uuid.New().String()[:8])
	}

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
	defer rdb.Close()

	// S3
	s3Client, err := storage.NewS3Client(cfg.S3)
	if err != nil {
		slog.Error("failed to create S3 client", "error", err)
		os.Exit(1)
	}

	// Consumer
	consumer := queue.NewConsumer(rdb, workerID)

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Heartbeat
	hb := worker.NewHeartbeat(rdb, workerID, cfg.Worker.HeartbeatInterval)
	go hb.Run(ctx)

	// Worker
	w := worker.New(workerID, cfg, consumer, s3Client, *apiBase)

	slog.Info("worker starting", "id", workerID)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("shutdown signal received")
		cancel()
	}()

	if err := w.Run(ctx); err != nil {
		slog.Error("worker error", "error", err)
		os.Exit(1)
	}

	slog.Info("worker stopped")
}
