package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zhou-JK/hls-streamer/internal/config"
	"github.com/Zhou-JK/hls-streamer/internal/queue"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
	"github.com/Zhou-JK/hls-streamer/pkg/ffmpeg"
	"github.com/Zhou-JK/hls-streamer/pkg/hls"
)

var execCommandContext = exec.CommandContext

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
		VideoUUID        string `json:"video_uuid"`
		VideoID          uint   `json:"video_id"`
		OriginalS3Key    string `json:"original_s3_key"`
		Resolution       string `json:"resolution"`
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		BitrateKbps      int    `json:"bitrate_kbps"`
		AudioBitrateKbps int    `json:"audio_bitrate_kbps"`
		Codec            string `json:"codec"`
		HasAudio         bool   `json:"has_audio"`
		DRM              bool   `json:"drm"`
		DRMKeyID         string `json:"drm_key_id"`
		DRMContentKey    string `json:"drm_content_key"`
		DRMIV            string `json:"drm_iv"`
		DRMLicenseURL    string `json:"drm_license_url"`
		SegmentDuration  int    `json:"segment_duration"`
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

	// Resolve segment duration: per-task > config default
	segDur := params.SegmentDuration
	if segDur <= 0 {
		segDur = w.cfg.HLS.SegmentDuration
	}

	if params.DRM && params.DRMKeyID != "" {
		// DRM path: FFmpeg → MP4 → Shaka Packager → encrypted HLS
		return w.transcodeDRM(ctx, msg.TaskUUID, workDir, inputFile, params.VideoUUID, params.VideoID, params.Resolution,
			params.Width, params.Height, params.BitrateKbps, params.AudioBitrateKbps, params.Codec,
			params.HasAudio, params.DRMKeyID, params.DRMContentKey, params.DRMIV, params.DRMLicenseURL, segDur)
	}

	// Non-DRM path: FFmpeg → HLS → rename segments → upload
	return w.transcodeNormal(ctx, msg.TaskUUID, workDir, inputFile, params.VideoUUID, params.VideoID,
		params.Resolution, params.Width, params.Height, params.BitrateKbps, params.AudioBitrateKbps, params.Codec, params.HasAudio, segDur)
}

func (w *Worker) transcodeNormal(ctx context.Context, taskUUID, workDir, inputFile, videoUUID string, videoID uint,
	resolution string, width, height, bitrateKbps, audioBitrateKbps int, codec string, hasAudio bool, segmentDuration int) error {

	hlsParams := ffmpeg.TranscodeHLSParams{
		Input:           inputFile,
		OutputDir:       workDir,
		PlaylistName:    "playlist.m3u8",
		SegmentPattern:  "segment_%03d.ts",
		Width:           width,
		Height:          height,
		VideoBitrate:    bitrateKbps,
		AudioBitrate:    audioBitrateKbps,
		Codec:           codec,
		HasAudio:        hasAudio,
		SegmentDuration: segmentDuration,
	}

	if err := w.ff.TranscodeToHLS(ctx, hlsParams); err != nil {
		return fmt.Errorf("transcode: %w", err)
	}

	w.reportProgress(taskUUID, 60)

	// Rename .ts to .jpeg and rewrite playlist
	if err := w.renameSegments(workDir, w.cfg.HLS.SegmentExtension); err != nil {
		return fmt.Errorf("rename segments: %w", err)
	}

	w.reportProgress(taskUUID, 70)

	// Upload to S3
	s3Prefix := fmt.Sprintf("videos/%s/variants/%s", videoUUID, resolution)
	if err := w.uploadDirectory(ctx, workDir, s3Prefix); err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	w.reportProgress(taskUUID, 95)

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"resolution":      resolution,
		"playlist_s3_key": fmt.Sprintf("%s/playlist.m3u8", s3Prefix),
	})
	w.reportComplete(taskUUID, resultJSON, videoID, videoUUID)
	return nil
}

func (w *Worker) transcodeDRM(ctx context.Context, taskUUID, workDir, inputFile, videoUUID string, videoID uint,
	resolution string, width, height, bitrateKbps, audioBitrateKbps int, codec string,
	hasAudio bool, keyID, contentKey, iv, licenseURL string, segmentDuration int) error {

	// Step 1: FFmpeg transcode to intermediate MP4 (not HLS)
	intermediateMP4 := filepath.Join(workDir, "intermediate.mp4")
	if err := w.ff.TranscodeToMP4(ctx, ffmpeg.TranscodeMP4Params{
		Input:        inputFile,
		Output:       intermediateMP4,
		Width:        width,
		Height:       height,
		VideoBitrate: bitrateKbps,
		AudioBitrate: audioBitrateKbps,
		Codec:        codec,
		HasAudio:     hasAudio,
	}); err != nil {
		return fmt.Errorf("transcode to mp4: %w", err)
	}

	w.reportProgress(taskUUID, 50)

	// Step 2: Shaka Packager — package MP4 into encrypted HLS with CENC
	outputDir := filepath.Join(workDir, "encrypted")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := w.shakaPackage(ctx, intermediateMP4, outputDir, hasAudio, keyID, contentKey, iv, segmentDuration); err != nil {
		return fmt.Errorf("shaka package: %w", err)
	}

	// Inject ClearKey EXT-X-KEY tag into the variant playlist(s) so players know
	// where to fetch the decryption key. Shaka generates its own EXT-X-KEY with
	// data: URIs — we replace them with our ClearKey license URL.
	if licenseURL != "" {
		if hasAudio {
			// With audio: master=playlist.m3u8, sub-playlists=video.m3u8, audio.m3u8
			// Inject into both sub-playlists
			for _, pl := range []string{"video.m3u8", "audio.m3u8"} {
				plPath := filepath.Join(outputDir, pl)
				if _, err := os.Stat(plPath); err == nil {
					if err := injectClearKeyTag(plPath, licenseURL, keyID); err != nil {
						slog.Warn("inject clearkey tag failed", "playlist", pl, "error", err)
					}
				}
			}
		} else {
			// No audio: playlist.m3u8 is the single variant playlist
			if err := injectClearKeyTag(filepath.Join(outputDir, "playlist.m3u8"), licenseURL, keyID); err != nil {
				slog.Warn("inject clearkey tag failed", "error", err)
			}
		}
	}

	w.reportProgress(taskUUID, 80)

	// Rename .m4s segments to configured extension (.jpeg) for CDN caching
	if err := w.renameSegments(outputDir, w.cfg.HLS.SegmentExtension); err != nil {
		return fmt.Errorf("rename segments: %w", err)
	}

	// Step 3: Upload encrypted output to S3
	s3Prefix := fmt.Sprintf("videos/%s/variants/%s", videoUUID, resolution)
	if err := w.uploadDirectory(ctx, outputDir, s3Prefix); err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	w.reportProgress(taskUUID, 95)

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"resolution":      resolution,
		"playlist_s3_key": fmt.Sprintf("%s/playlist.m3u8", s3Prefix),
		"drm":             true,
	})
	w.reportComplete(taskUUID, resultJSON, videoID, videoUUID)
	return nil
}

// shakaPackage uses Shaka Packager to encrypt an MP4 into HLS with CENC (ClearKey/Widevine compatible).
// It outputs fMP4 segments (.m4s) with init segments (.mp4) so that --protection_scheme cenc
// is properly applied. TS segments do NOT support CENC — Shaka silently ignores the flag for TS.
func (w *Worker) shakaPackage(ctx context.Context, inputMP4, outputDir string, hasAudio bool, keyID, contentKey, iv string, segmentDuration int) error {
	playlistPath := filepath.Join(outputDir, "playlist.m3u8")

	// Use fMP4 (.m4s) segments with explicit init segments so CENC encryption works.
	videoSegTemplate := filepath.Join(outputDir, "segment_$Number$.m4s")
	videoInitSeg := filepath.Join(outputDir, "video_init.mp4")

	// Build Shaka Packager command.
	// We only package the video stream. Audio will be handled separately below.
	var args []string

	videoDesc := fmt.Sprintf(
		"in=%s,stream=video,init_segment=%s,segment_template=%s,playlist_name=video.m3u8",
		inputMP4, videoInitSeg, videoSegTemplate,
	)
	args = append(args, videoDesc)

	if hasAudio {
		// Package audio as a separate stream with its own init segment and segments.
		audioInitSeg := filepath.Join(outputDir, "audio_init.mp4")
		audioSegTemplate := filepath.Join(outputDir, "audio_$Number$.m4s")
		audioDesc := fmt.Sprintf(
			"in=%s,stream=audio,init_segment=%s,segment_template=%s,playlist_name=audio.m3u8,hls_group_id=audio,hls_name=audio",
			inputMP4, audioInitSeg, audioSegTemplate,
		)
		args = append(args, audioDesc)
	}

	args = append(args,
		"--enable_raw_key_encryption",
		"--keys", fmt.Sprintf("key_id=%s:key=%s", keyID, contentKey),
		"--iv", iv,
		"--protection_scheme", "cenc",
		"--hls_master_playlist_output", playlistPath,
		"--segment_duration", fmt.Sprintf("%d", segmentDuration),
		"--temp_dir", w.cfg.Worker.TempDir,
	)

	slog.Info("running shaka packager", "args_count", len(args))

	cmd := execCommandContext(ctx, w.cfg.Shaka.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shaka packager failed: %w\noutput: %s", err, string(output))
	}

	// Shaka outputs a master playlist that references video.m3u8 and optionally audio.m3u8.
	// We use the video playlist as our "playlist.m3u8" for the variant.
	// If audio exists, we merge the audio EXT-X-MAP and segments into the video playlist
	// so the player can handle everything from a single playlist.
	videoPlaylist := filepath.Join(outputDir, "video.m3u8")

	if hasAudio {
		// For fMP4 HLS, audio and video must be in separate playlists.
		// Keep Shaka's master playlist which correctly references both.
		// Just remove the individual video.m3u8 name collision.
		// Actually — our architecture expects a single playlist.m3u8 per variant.
		// The master playlist Shaka generates already references video.m3u8 and audio.m3u8
		// with proper EXT-X-MEDIA and EXT-X-STREAM-INF tags.
		// We keep the master as playlist.m3u8 (Shaka already wrote it there).
		// But we need video.m3u8 and audio.m3u8 to exist alongside it.
		// No renaming needed — Shaka's master playlist is already at playlistPath.
		// Just clean up: nothing to do.
		slog.Info("shaka: keeping master playlist with audio+video", "dir", outputDir)
	} else {
		// No audio: replace master with video variant playlist.
		if _, err := os.Stat(videoPlaylist); err == nil {
			data, err := os.ReadFile(videoPlaylist)
			if err == nil {
				_ = os.WriteFile(playlistPath, data, 0644)
			}
			os.Remove(videoPlaylist)
		}
	}

	return nil
}

// injectClearKeyTag reads an HLS playlist, replaces Shaka's EXT-X-KEY tag with
// a ClearKey-compatible one, and writes it back. Shaka generates its own EXT-X-KEY
// with a data: URI containing PSSH — we replace it with a ClearKey license URL
// that hls.js can use to fetch the decryption key via the W3C ClearKey protocol.
func injectClearKeyTag(playlistPath, licenseURL, keyIDHex string) error {
	data, err := os.ReadFile(playlistPath)
	if err != nil {
		return err
	}

	// Convert hex key ID to base64url for the URI query
	keyIDBytes, err := hex.DecodeString(keyIDHex)
	if err != nil {
		return fmt.Errorf("decode key_id: %w", err)
	}
	keyIDB64 := base64.RawURLEncoding.EncodeToString(keyIDBytes)

	// Build the ClearKey EXT-X-KEY tag.
	// KEYFORMAT must be "org.w3.clearkey" for hls.js to recognize it.
	// The URI points to our ClearKey license endpoint; the player will POST
	// a {"kids":[...]} request to it and receive the key back.
	clearKeyTag := fmt.Sprintf(
		`#EXT-X-KEY:METHOD=SAMPLE-AES-CTR,URI="%s?kid=%s",KEYFORMAT="org.w3.clearkey",KEYFORMATVERSIONS="1"`,
		licenseURL, keyIDB64,
	)

	// Replace Shaka's EXT-X-KEY tags with our ClearKey tag.
	// Shaka generates tags like:
	//   #EXT-X-KEY:METHOD=SAMPLE-AES-CTR,URI="data:text/plain;base64,...",KEYID=0x...,IV=0x...
	// We replace all of them with our single ClearKey tag.
	lines := strings.Split(string(data), "\n")
	var result []string
	clearKeyInjected := false
	for _, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-KEY:") {
			// Replace the first EXT-X-KEY with ours, skip subsequent ones
			if !clearKeyInjected {
				result = append(result, clearKeyTag)
				clearKeyInjected = true
			}
			continue
		}
		result = append(result, line)
	}

	// If no EXT-X-KEY was found (shouldn't happen), inject after #EXTM3U
	if !clearKeyInjected {
		for i, line := range result {
			if line == "#EXTM3U" {
				result = append(result[:i+1], append([]string{clearKeyTag}, result[i+1:]...)...)
				break
			}
		}
	}

	return os.WriteFile(playlistPath, []byte(strings.Join(result, "\n")), 0644)
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

// renameSegments renames .ts and .m4s files to the configured extension and rewrites the playlist.
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
		if strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".m4s") {
			ext := filepath.Ext(name)
			newName := strings.TrimSuffix(name, ext) + newExt
			oldPath := filepath.Join(dir, name)
			newPath := filepath.Join(dir, newName)
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("rename %s: %w", name, err)
			}
		}
	}

	// Rewrite all playlists in the directory (playlist.m3u8, video.m3u8, audio.m3u8)
	entries, err = os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".m3u8") {
			continue
		}
		plPath := filepath.Join(dir, entry.Name())
		f, err := os.Open(plPath)
		if err != nil {
			continue
		}
		rewritten, err := hls.RewritePlaylist(f, newExt)
		f.Close()
		if err != nil {
			return fmt.Errorf("rewrite playlist %s: %w", entry.Name(), err)
		}
		if err := os.WriteFile(plPath, []byte(rewritten), 0644); err != nil {
			return fmt.Errorf("write playlist %s: %w", entry.Name(), err)
		}
	}

	return nil
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
		case strings.HasSuffix(entry.Name(), ".mp4"), strings.HasSuffix(entry.Name(), ".m4s"):
			contentType = "video/mp4"
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
