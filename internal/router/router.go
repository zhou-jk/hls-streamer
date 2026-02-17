package router

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/config"
	"github.com/Zhou-JK/hls-streamer/internal/handler"
	"github.com/Zhou-JK/hls-streamer/internal/middleware"
)

type Handlers struct {
	Auth      *handler.AuthHandler
	Video     *handler.VideoHandler
	Upload    *handler.UploadHandler
	Transcode *handler.TranscodeHandler
	Worker    *handler.WorkerHandler
	Category  *handler.CategoryHandler
	DRM       *handler.DRMHandler
	Playback  *handler.PlaybackHandler
	Setting   *handler.SettingHandler
}

func Setup(cfg *config.Config, h Handlers) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")

	// Public auth endpoints
	auth := api.Group("/auth")
	{
		auth.POST("/login", h.Auth.Login)
		auth.POST("/register", h.Auth.Register)
		auth.POST("/refresh", h.Auth.Refresh)
		auth.POST("/logout", h.Auth.Logout)
	}

	// Authenticated endpoints
	authed := api.Group("")
	authed.Use(middleware.Auth(cfg.JWT.Secret))
	{
		// Auth - current user
		authed.GET("/auth/me", h.Auth.Me)
		authed.PUT("/auth/me/password", h.Auth.ChangePassword)

		// Users (admin only)
		users := authed.Group("/users")
		users.Use(middleware.RequireRole("admin"))
		{
			users.GET("", h.Auth.ListUsers)
			users.GET("/:id", h.Auth.GetUser)
			users.POST("", h.Auth.CreateUser)
			users.PUT("/:id", h.Auth.UpdateUser)
			users.DELETE("/:id", h.Auth.DeleteUser)
		}

		// Roles
		authed.GET("/roles", h.Auth.ListRoles)

		// Videos
		videos := authed.Group("/videos")
		{
			videos.GET("", h.Video.List)
			videos.GET("/:uuid", h.Video.Get)
			videos.POST("", h.Video.Create)
			videos.PUT("/:uuid", h.Video.Update)
			videos.DELETE("/:uuid", h.Transcode.DeleteVideo)
			videos.POST("/:uuid/restore", h.Video.Restore)

			// Translations
			videos.GET("/:uuid/translations", h.Video.ListTranslations)
			videos.PUT("/:uuid/translations/:lang", h.Video.UpsertTranslation)
			videos.DELETE("/:uuid/translations/:lang", h.Video.DeleteTranslation)

			// Upload
			videos.POST("/:uuid/upload/initiate", h.Upload.Initiate)
			videos.POST("/:uuid/upload/complete", h.Upload.Complete)

			// Transcode
			videos.POST("/:uuid/transcode", h.Transcode.StartTranscode)
			videos.GET("/:uuid/tasks", h.Transcode.ListTasks)
			videos.GET("/:uuid/tasks/:task_uuid", h.Transcode.GetTask)

			// Variants
			videos.GET("/:uuid/variants", h.Video.ListVariants)
			videos.DELETE("/:uuid/variants", h.Transcode.DeleteVariants)
			videos.DELETE("/:uuid/variants/:resolution", h.Transcode.DeleteVariant)

			// Thumbnails
			videos.GET("/:uuid/thumbnails", h.Video.ListThumbnails)
			videos.POST("/:uuid/thumbnails/generate", h.Transcode.GenerateThumbnails)
			videos.PUT("/:uuid/thumbnails/:id/default", h.Video.SetDefaultThumbnail)
			videos.DELETE("/:uuid/thumbnails/:id", h.Video.DeleteThumbnail)

			// Subtitles
			videos.GET("/:uuid/subtitles", h.Video.ListSubtitles)
			videos.POST("/:uuid/subtitles/upload", h.Upload.UploadSubtitle)
			videos.DELETE("/:uuid/subtitles/:id", h.Video.DeleteSubtitle)

			// Cast
			videos.GET("/:uuid/cast", h.Video.ListCast)
			videos.DELETE("/:uuid/cast/:id", h.Video.DeleteCast)

			// DRM
			videos.POST("/:uuid/drm/keys", h.DRM.GenerateKeys)
			videos.GET("/:uuid/drm/keys", h.DRM.GetKeys)
			videos.PUT("/:uuid/drm/toggle", h.Transcode.ToggleDRM)
		}

		// Categories
		categories := authed.Group("/categories")
		{
			categories.GET("", h.Category.ListCategories)
			categories.POST("", h.Category.CreateCategory)
			categories.PUT("/:id", h.Category.UpdateCategory)
			categories.DELETE("/:id", h.Category.DeleteCategory)
			categories.PUT("/:id/translations/:lang", h.Category.UpsertCategoryTranslation)
		}

		// Tags
		tags := authed.Group("/tags")
		{
			tags.GET("", h.Category.ListTags)
			tags.POST("", h.Category.CreateTag)
			tags.DELETE("/:id", h.Category.DeleteTag)
		}

		// Workers (admin)
		authed.GET("/workers", middleware.RequireRole("admin"), h.Worker.ListWorkers)

		// Global tasks
		authed.GET("/tasks", h.Transcode.ListAllTasks)
		authed.POST("/tasks/:task_uuid/cancel", h.Transcode.CancelTask)
		authed.POST("/tasks/:task_uuid/retry", h.Transcode.RetryTask)

		// Settings (admin)
		settings := authed.Group("/settings")
		settings.Use(middleware.RequireRole("admin"))
		{
			settings.GET("", h.Setting.List)
			settings.PUT("", h.Setting.Update)
		}
	}

	// Worker internal API (no JWT, should be protected by network/API key)
	workers := api.Group("/workers")
	{
		workers.POST("/register", h.Worker.Register)
		workers.POST("/heartbeat", h.Worker.Heartbeat)
		workers.POST("/tasks/:task_uuid/progress", h.Worker.ReportProgress)
		workers.POST("/tasks/:task_uuid/complete", h.Worker.CompleteTask)
		workers.POST("/tasks/:task_uuid/fail", h.Worker.FailTask)
	}

	// DRM key endpoint (public, called by hls.js for AES-128 decryption)
	drm := api.Group("/drm")
	{
		drm.GET("/key/:key_id", h.DRM.ServeKey)
	}

	// Public video API (no auth, only public+ready videos)
	pub := api.Group("/public")
	{
		pub.GET("/videos", h.Video.ListPublic)
		pub.GET("/videos/:uuid", h.Video.GetPublic)
		pub.GET("/categories", h.Category.ListCategories)
	}

	// Playback endpoints (public or token-gated)
	play := r.Group("/play")
	{
		play.GET("/:uuid/master.m3u8", h.Playback.MasterPlaylist)
		play.GET("/:uuid/:variant/playlist.m3u8", h.Playback.VariantPlaylist)
		play.GET("/:uuid/:variant/:segment", h.Playback.Segment)
		play.GET("/:uuid/subtitles/:lang.vtt", h.Playback.SubtitleFile)
		play.GET("/:uuid/thumbnails/:filename", h.Playback.ThumbnailImage)
		play.GET("/:uuid/download", h.Playback.DownloadOriginal)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve frontend static files (SPA)
	webDir := "/srv/web"
	if _, err := os.Stat(webDir); err == nil {
		r.Use(func(c *gin.Context) {
			// Skip API and playback routes
			p := c.Request.URL.Path
			if len(p) >= 4 && p[:4] == "/api" || len(p) >= 5 && p[:5] == "/play" || p == "/health" {
				c.Next()
				return
			}
			// Try to serve static file
			filePath := filepath.Join(webDir, p)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				c.Abort()
				return
			}
			// SPA fallback: serve index.html
			c.File(filepath.Join(webDir, "index.html"))
			c.Abort()
		})
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(webDir, "index.html"))
		})
	}

	return r
}
