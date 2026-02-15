package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zhou-JK/hls-streamer/internal/config"
	"github.com/Zhou-JK/hls-streamer/internal/queue"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
	"github.com/Zhou-JK/hls-streamer/pkg/ffmpeg"
	"github.com/Zhou-JK/hls-streamer/pkg/hls"
)

type Worker struct {
	id       string
	cfg      *config.Config
	consumer *queue.Consumer
	s3       *storage.S3Client
	ff       *ffmpeg.FFmpeg
	apiBase  string // API server base URL for callbacks
}

func New(id string, cfg *config.Config, consumer *queue.Consumer, s3 *storage.S3Client, apiBase string) *Worker {
	return &Worker{
		id:       id,
		cfg:      cfg,
		consumer: consumer,
		s3:       s3,
		ff:       ffmpeg.New(cfg.FFmpeg.Path, cfg.FFmpeg.FFprobePath, cfg.FFmpeg.Threads),
		apiBase:  apiBase,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	slog.Info("worker started", "id", w.id, "concurrency", w.cfg.Worker.Concurrency)

	// Ensure temp dir exists
	if err := os.MkdirAll(w.cfg.Worker.TempDir, 0755); err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	// Register with API server
	w.registerWithAPI()

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(w.cfg.Worker.HeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.sendHeartbeat()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down", "id", w.id)
			return nil
		default:
		}

		msg, stream, msgID, err := w.consumer.ReadTask(ctx, 5*time.Second)
		if err != nil {
			slog.Error("read task failed", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if msg == nil {
			continue // timeout, no message
		}

		slog.Info("processing task", "task_uuid", msg.TaskUUID, "type", msg.Type)

		if err := w.processTask(ctx, msg); err != nil {
			slog.Error("task failed", "task_uuid", msg.TaskUUID, "error", err)
			w.reportFail(msg.TaskUUID, err.Error())
		}

		// Acknowledge the message
		if err := w.consumer.Ack(ctx, stream, msgID); err != nil {
			slog.Error("ack failed", "error", err)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, msg *queue.TaskMessage) error {
	switch msg.Type {
	case "probe":
		return w.handleProbe(ctx, msg)
	case "transcode":
		return w.handleTranscode(ctx, msg)
	case "thumbnail":
		return w.handleThumbnail(ctx, msg)
	default:
		return fmt.Errorf("unknown task type: %s", msg.Type)
	}
}

func (w *Worker) handleProbe(ctx context.Context, msg *queue.TaskMessage) error {
	var params struct {
		VideoUUID     string `json:"video_uuid"`
		VideoID       uint   `json:"video_id"`
		OriginalS3Key string `json:"original_s3_key"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}

	// Download original file to temp
	tempFile := filepath.Join(w.cfg.Worker.TempDir, fmt.Sprintf("probe_%s", msg.TaskUUID))
	defer os.Remove(tempFile)

	if err := w.downloadFromS3(ctx, params.OriginalS3Key, tempFile); err != nil {
		return fmt.Errorf("download original: %w", err)
	}

	// Run ffprobe
	result, err := w.ff.Probe(ctx, tempFile)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}

	resultJSON, _ := json.Marshal(result)
	w.reportComplete(msg.TaskUUID, resultJSON, params.VideoID, params.VideoUUID)
	return nil
}

func (w *Worker) handleTranscode(ctx context.Context, msg *queue.TaskMessage) error {
	var params struct {
		VideoUUID       string `json:"video_uuid"`
		VideoID         uint   `json:"video_id"`
		OriginalS3Key   string `json:"original_s3_key"`
		Resolution      string `json:"resolution"`
		Width           int    `json:"width"`
		Height          int    `json:"height"`
		BitrateKbps     int    `json:"bitrate_kbps"`
		AudioBitrateKbps int   `json:"audio_bitrate_kbps"`
		Codec           string `json:"codec"`
		DRM             bool   `json:"drm"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}

	// Create work directory
	workDir := filepath.Join(w.cfg.Worker.TempDir, fmt.Sprintf("transcode_%s", msg.TaskUUID))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	// Download original
	inputFile := filepath.Join(workDir, "input")
	if err := w.downloadFromS3(ctx, params.OriginalS3Key, inputFile); err != nil {
		return fmt.Errorf("download original: %w", err)
	}

	w.reportProgress(msg.TaskUUID, 10)

	// Transcode to HLS
	hlsParams := ffmpeg.TranscodeHLSParams{
		Input:           inputFile,
		OutputDir:       workDir,
		PlaylistName:    "playlist.m3u8",
		SegmentPattern:  "segment_%03d.ts",
		Width:           params.Width,
		Height:          params.Height,
		VideoBitrate:    params.BitrateKbps,
		AudioBitrate:    params.AudioBitrateKbps,
		Codec:           params.Codec,
		SegmentDuration: w.cfg.HLS.SegmentDuration,
	}

	if err := w.ff.TranscodeToHLS(ctx, hlsParams); err != nil {
		return fmt.Errorf("transcode: %w", err)
	}

	w.reportProgress(msg.TaskUUID, 60)

	// Rename .ts to .jpeg and rewrite playlist
	if err := w.renameSegments(workDir, w.cfg.HLS.SegmentExtension); err != nil {
		return fmt.Errorf("rename segments: %w", err)
	}

	w.reportProgress(msg.TaskUUID, 70)

	// Upload to S3
	s3Prefix := fmt.Sprintf("videos/%s/variants/%s", params.VideoUUID, params.Resolution)
	if err := w.uploadDirectory(ctx, workDir, s3Prefix); err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	w.reportProgress(msg.TaskUUID, 95)

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"resolution":     params.Resolution,
		"playlist_s3_key": fmt.Sprintf("%s/playlist.m3u8", s3Prefix),
	})
	w.reportComplete(msg.TaskUUID, resultJSON, params.VideoID, params.VideoUUID)
	return nil
}

func (w *Worker) handleThumbnail(ctx context.Context, msg *queue.TaskMessage) error {
	var params struct {
		VideoUUID     string `json:"video_uuid"`
		VideoID       uint   `json:"video_id"`
		OriginalS3Key string `json:"original_s3_key"`
		Count         int    `json:"count"`
		Width         int    `json:"width"`
		Height        int    `json:"height"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}

	workDir := filepath.Join(w.cfg.Worker.TempDir, fmt.Sprintf("thumb_%s", msg.TaskUUID))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	// Download original
	inputFile := filepath.Join(workDir, "input")
	if err := w.downloadFromS3(ctx, params.OriginalS3Key, inputFile); err != nil {
		return fmt.Errorf("download original: %w", err)
	}

	// Get duration for thumbnail spacing
	probe, err := w.ff.Probe(ctx, inputFile)
	if err != nil {
		return fmt.Errorf("probe for thumbnails: %w", err)
	}

	w.reportProgress(msg.TaskUUID, 20)

	// Extract thumbnails
	if err := w.ff.ExtractThumbnails(ctx, inputFile, workDir, params.Count, params.Width, params.Height, probe.Duration); err != nil {
		return fmt.Errorf("extract thumbnails: %w", err)
	}

	w.reportProgress(msg.TaskUUID, 70)

	// Upload thumbnails to S3
	s3Prefix := fmt.Sprintf("videos/%s/thumbnails", params.VideoUUID)
	entries, _ := os.ReadDir(workDir)
	var uploaded []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jpg") {
			continue
		}
		s3Key := fmt.Sprintf("%s/%s", s3Prefix, entry.Name())
		filePath := filepath.Join(workDir, entry.Name())
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}
		if err := w.s3.Upload(ctx, s3Key, f, "image/jpeg"); err != nil {
			f.Close()
			continue
		}
		f.Close()
		uploaded = append(uploaded, s3Key)
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"thumbnails": uploaded,
		"count":      len(uploaded),
	})
	w.reportComplete(msg.TaskUUID, resultJSON, params.VideoID, params.VideoUUID)
	return nil
}

// renameSegments renames .ts files to the configured extension and rewrites the playlist.
func (w *Worker) renameSegments(dir, newExt string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".ts") {
			newName := strings.TrimSuffix(name, ".ts") + newExt
			oldPath := filepath.Join(dir, name)
			newPath := filepath.Join(dir, newName)
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("rename %s: %w", name, err)
			}
		}
	}

	// Rewrite playlist
	playlistPath := filepath.Join(dir, "playlist.m3u8")
	f, err := os.Open(playlistPath)
	if err != nil {
		return fmt.Errorf("open playlist: %w", err)
	}

	rewritten, err := hls.RewritePlaylist(f, newExt)
	f.Close()
	if err != nil {
		return fmt.Errorf("rewrite playlist: %w", err)
	}

	return os.WriteFile(playlistPath, []byte(rewritten), 0644)
}

// uploadDirectory uploads all files in a directory to S3 under the given prefix.
func (w *Worker) uploadDirectory(ctx context.Context, dir, s3Prefix string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "input" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		s3Key := fmt.Sprintf("%s/%s", s3Prefix, entry.Name())

		contentType := "application/octet-stream"
		switch {
		case strings.HasSuffix(entry.Name(), ".m3u8"):
			contentType = "application/vnd.apple.mpegurl"
		case strings.HasSuffix(entry.Name(), ".jpeg"), strings.HasSuffix(entry.Name(), ".jpg"):
			contentType = "image/jpeg"
		case strings.HasSuffix(entry.Name(), ".ts"):
			contentType = "video/mp2t"
		}

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open %s: %w", entry.Name(), err)
		}

		if err := w.s3.Upload(ctx, s3Key, f, contentType); err != nil {
			f.Close()
			return fmt.Errorf("upload %s: %w", entry.Name(), err)
		}
		f.Close()
	}

	return nil
}

func (w *Worker) downloadFromS3(ctx context.Context, s3Key, destPath string) error {
	body, err := w.s3.Download(ctx, s3Key)
	if err != nil {
		return err
	}
	defer body.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, body)
	return err
}

// HTTP callbacks to API server
func (w *Worker) reportProgress(taskUUID string, progress uint8) {
	body, _ := json.Marshal(map[string]interface{}{
		"worker_id": w.id,
		"progress":  progress,
	})
	url := fmt.Sprintf("%s/api/v1/workers/tasks/%s/progress", w.apiBase, taskUUID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("report progress failed", "task_uuid", taskUUID, "error", err)
		return
	}
	resp.Body.Close()
}

func (w *Worker) reportComplete(taskUUID string, result json.RawMessage, videoID uint, videoUUID string) {
	body, _ := json.Marshal(map[string]interface{}{
		"worker_id":  w.id,
		"result":     result,
		"video_id":   videoID,
		"video_uuid": videoUUID,
	})
	url := fmt.Sprintf("%s/api/v1/workers/tasks/%s/complete", w.apiBase, taskUUID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("report complete failed", "task_uuid", taskUUID, "error", err)
		return
	}
	resp.Body.Close()
}

func (w *Worker) reportFail(taskUUID, errMsg string) {
	body, _ := json.Marshal(map[string]interface{}{
		"worker_id": w.id,
		"error":     errMsg,
	})
	url := fmt.Sprintf("%s/api/v1/workers/tasks/%s/fail", w.apiBase, taskUUID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("report fail failed", "task_uuid", taskUUID, "error", err)
		return
	}
	resp.Body.Close()
}

// registerWithAPI registers this worker with the API server.
func (w *Worker) registerWithAPI() {
	body, _ := json.Marshal(map[string]interface{}{
		"id":           w.id,
		"hostname":     w.id,
		"ip_address":   "0.0.0.0",
		"capabilities": map[string]interface{}{"concurrency": w.cfg.Worker.Concurrency},
	})
	url := fmt.Sprintf("%s/api/v1/workers/register", w.apiBase)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Warn("register with API failed", "error", err)
		return
	}
	resp.Body.Close()
	slog.Info("registered with API server", "worker_id", w.id)
}

// sendHeartbeat sends a heartbeat to the API server.
func (w *Worker) sendHeartbeat() {
	body, _ := json.Marshal(map[string]interface{}{
		"worker_id": w.id,
	})
	url := fmt.Sprintf("%s/api/v1/workers/heartbeat", w.apiBase)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Warn("heartbeat failed", "error", err)
		return
	}
	resp.Body.Close()
}
