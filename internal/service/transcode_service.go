package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/queue"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
	"github.com/Zhou-JK/hls-streamer/pkg/hls"
)

type TranscodeService struct {
	taskRepo  *repository.TaskRepo
	videoRepo *repository.VideoRepo
	producer  *queue.Producer
	s3        *storage.S3Client
	drmSvc    *DRMService
}

func NewTranscodeService(taskRepo *repository.TaskRepo, videoRepo *repository.VideoRepo, producer *queue.Producer, s3 *storage.S3Client, drmSvc *DRMService) *TranscodeService {
	return &TranscodeService{
		taskRepo:  taskRepo,
		videoRepo: videoRepo,
		producer:  producer,
		s3:        s3,
		drmSvc:    drmSvc,
	}
}

// StartTranscodeInput is the handler-facing alias.
type StartTranscodeInput = TranscodeRequest

type TranscodeRequest struct {
	Resolutions []ResolutionSpec `json:"resolutions" binding:"required,min=1"`
	Codec       string           `json:"codec"`
	DRM         bool             `json:"drm"`
}

type ResolutionSpec struct {
	Name         string `json:"name" binding:"required"` // e.g. "720p"
	Width        int    `json:"width" binding:"required"`
	Height       int    `json:"height" binding:"required"`
	BitrateKbps  int    `json:"bitrate_kbps" binding:"required"`
	AudioBitrate int    `json:"audio_bitrate_kbps"`
}

func (s *TranscodeService) StartTranscode(ctx context.Context, videoUUID string, req TranscodeRequest, licenseBaseURL string) ([]model.TranscodeTask, error) {
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	if req.Codec == "" {
		req.Codec = "h264"
	}

	// If DRM requested, generate keys first
	var drmKey *model.DRMKey
	if req.DRM && s.drmSvc != nil {
		drmKey, err = s.drmSvc.GenerateKeys(video.ID, licenseBaseURL)
		if err != nil {
			return nil, fmt.Errorf("generate DRM keys: %w", err)
		}
	}

	// Update video status
	_ = s.videoRepo.UpdateStatus(videoUUID, "processing")

	var tasks []model.TranscodeTask

	for _, res := range req.Resolutions {
		if res.AudioBitrate == 0 {
			res.AudioBitrate = 128
		}

		paramMap := map[string]interface{}{
			"video_uuid":         videoUUID,
			"video_id":           video.ID,
			"original_s3_key":    video.OriginalS3Key,
			"resolution":         res.Name,
			"width":              res.Width,
			"height":             res.Height,
			"bitrate_kbps":       res.BitrateKbps,
			"audio_bitrate_kbps": res.AudioBitrate,
			"codec":              req.Codec,
			"drm":                req.DRM,
		}

		// Include DRM key material for the worker
		if drmKey != nil {
			paramMap["drm_key_id"] = drmKey.KeyID
			paramMap["drm_content_key"] = drmKey.ContentKey
			paramMap["drm_iv"] = drmKey.IV
		}

		params, _ := json.Marshal(paramMap)

		task := model.TranscodeTask{
			TaskUUID:    uuid.New().String(),
			VideoID:     video.ID,
			Type:        "transcode",
			Status:      "queued",
			Params:      model.JSON(params),
			MaxAttempts: 3,
		}

		if err := s.taskRepo.Create(&task); err != nil {
			return nil, fmt.Errorf("create task: %w", err)
		}

		// Create variant record
		variant := &model.VideoVariant{
			VideoID:        video.ID,
			ResolutionName: res.Name,
			Width:          uint(res.Width),
			Height:         uint(res.Height),
			BitrateKbps:    uint(res.BitrateKbps),
			Codec:          req.Codec,
			PlaylistS3Key:  fmt.Sprintf("videos/%s/variants/%s/playlist.m3u8", videoUUID, res.Name),
			Status:         "processing",
		}
		_ = s.videoRepo.CreateVariant(variant)

		// Publish to Redis
		msg := queue.TaskMessage{
			TaskUUID: task.TaskUUID,
			VideoID:  video.ID,
			Type:     "transcode",
			Params:   json.RawMessage(params),
		}
		if err := s.producer.Publish(ctx, msg); err != nil {
			return nil, fmt.Errorf("publish task: %w", err)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *TranscodeService) StartProbe(ctx context.Context, videoUUID string) (*model.TranscodeTask, error) {
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return nil, err
	}

	params, _ := json.Marshal(map[string]interface{}{
		"video_uuid":      videoUUID,
		"video_id":        video.ID,
		"original_s3_key": video.OriginalS3Key,
	})

	task := model.TranscodeTask{
		TaskUUID:    uuid.New().String(),
		VideoID:     video.ID,
		Type:        "probe",
		Status:      "queued",
		Params:      model.JSON(params),
		MaxAttempts: 3,
	}

	if err := s.taskRepo.Create(&task); err != nil {
		return nil, err
	}

	msg := queue.TaskMessage{
		TaskUUID: task.TaskUUID,
		VideoID:  video.ID,
		Type:     "probe",
		Params:   json.RawMessage(params),
	}
	if err := s.producer.Publish(ctx, msg); err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *TranscodeService) StartThumbnailGeneration(ctx context.Context, videoUUID string, count, width, height int) (*model.TranscodeTask, error) {
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return nil, err
	}

	params, _ := json.Marshal(map[string]interface{}{
		"video_uuid":      videoUUID,
		"video_id":        video.ID,
		"original_s3_key": video.OriginalS3Key,
		"count":           count,
		"width":           width,
		"height":          height,
	})

	task := model.TranscodeTask{
		TaskUUID:    uuid.New().String(),
		VideoID:     video.ID,
		Type:        "thumbnail",
		Status:      "queued",
		Params:      model.JSON(params),
		MaxAttempts: 3,
	}

	if err := s.taskRepo.Create(&task); err != nil {
		return nil, err
	}

	msg := queue.TaskMessage{
		TaskUUID: task.TaskUUID,
		VideoID:  video.ID,
		Type:     "thumbnail",
		Params:   json.RawMessage(params),
	}
	if err := s.producer.Publish(ctx, msg); err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *TranscodeService) GetTask(taskUUID string) (*model.TranscodeTask, error) {
	return s.taskRepo.FindByUUID(taskUUID)
}

func (s *TranscodeService) ListTasks(videoUUID string) ([]model.TranscodeTask, error) {
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return nil, err
	}
	return s.taskRepo.ListByVideoID(video.ID)
}

func (s *TranscodeService) UpdateProgress(taskUUID string, progress uint8) error {
	return s.taskRepo.UpdateProgress(taskUUID, progress)
}

func (s *TranscodeService) CompleteTask(taskUUID string, result model.JSON) error {
	task, err := s.taskRepo.FindByUUID(taskUUID)
	if err != nil {
		return err
	}

	if err := s.taskRepo.Complete(taskUUID, result); err != nil {
		return err
	}

	// Post-completion processing based on task type
	switch task.Type {
	case "probe":
		s.handleProbeResult(task, result)
	case "thumbnail":
		s.handleThumbnailResult(task, result)
	case "transcode":
		s.handleTranscodeResult(task, result)
	}

	return nil
}

func (s *TranscodeService) handleProbeResult(task *model.TranscodeTask, result model.JSON) {
	var r struct {
		Duration float64 `json:"duration"`
		Width    int     `json:"width"`
		Height   int     `json:"height"`
		Codec    string  `json:"codec"`
		FPS      float64 `json:"fps"`
		FileSize int64   `json:"file_size"`
	}
	if err := json.Unmarshal(result, &r); err != nil {
		return
	}

	video, err := s.videoRepo.FindByID(task.VideoID)
	if err != nil {
		return
	}

	if r.Duration > 0 {
		video.DurationSeconds = &r.Duration
	}
	if r.Width > 0 {
		w := uint(r.Width)
		video.Width = &w
	}
	if r.Height > 0 {
		h := uint(r.Height)
		video.Height = &h
	}
	if r.Codec != "" {
		video.Codec = r.Codec
	}
	if r.FPS > 0 {
		video.FPS = &r.FPS
	}
	if r.FileSize > 0 {
		video.FileSizeBytes = &r.FileSize
	}

	// After probe, move from draft to uploaded (ready for transcoding)
	if video.Status == "draft" {
		video.Status = "uploaded"
	}

	_ = s.videoRepo.Update(video)
}

func (s *TranscodeService) handleThumbnailResult(task *model.TranscodeTask, result model.JSON) {
	var thumbResult struct {
		Thumbnails []string `json:"thumbnails"`
		Count      int      `json:"count"`
	}
	if err := json.Unmarshal(result, &thumbResult); err != nil {
		return
	}

	var params struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	_ = json.Unmarshal(task.Params, &params)

	video, _ := s.videoRepo.FindByID(task.VideoID)
	duration := 0.0
	if video != nil && video.DurationSeconds != nil {
		duration = *video.DurationSeconds
	}

	for i, s3Key := range thumbResult.Thumbnails {
		timestamp := 0.0
		if duration > 0 && thumbResult.Count > 0 {
			timestamp = duration * float64(i+1) / float64(thumbResult.Count+1)
		}

		thumb := &model.Thumbnail{
			VideoID:    task.VideoID,
			S3Key:      s3Key,
			Width:      uint(params.Width),
			Height:     uint(params.Height),
			TimestampS: timestamp,
			IsDefault:  i == 0,
			SortOrder:  i,
		}
		_ = s.videoRepo.CreateThumbnail(thumb)
	}
}

func (s *TranscodeService) handleTranscodeResult(task *model.TranscodeTask, result model.JSON) {
	var r struct {
		Resolution    string `json:"resolution"`
		PlaylistS3Key string `json:"playlist_s3_key"`
	}
	if err := json.Unmarshal(result, &r); err != nil {
		return
	}

	var params struct {
		VideoUUID string `json:"video_uuid"`
	}
	_ = json.Unmarshal(task.Params, &params)

	// Update variant status to ready
	variants, _ := s.videoRepo.ListVariants(task.VideoID)
	for _, v := range variants {
		if v.ResolutionName == r.Resolution && v.Status == "processing" {
			v.Status = "ready"
			_ = s.videoRepo.UpdateVariant(&v)
			break
		}
	}

	// Check if all transcode tasks are done, build master playlist
	s.checkAndBuildMasterPlaylist(task.VideoID, params.VideoUUID)
}

func (s *TranscodeService) checkAndBuildMasterPlaylist(videoID uint, videoUUID string) {
	pending, err := s.taskRepo.CountPendingByVideo(videoID)
	if err != nil || pending > 0 {
		return
	}

	variants, err := s.videoRepo.ListVariants(videoID)
	if err != nil {
		return
	}

	var readyVariants []model.VideoVariant
	for _, v := range variants {
		if v.Status == "ready" {
			readyVariants = append(readyVariants, v)
		}
	}

	if len(readyVariants) == 0 {
		_ = s.videoRepo.UpdateStatus(videoUUID, "error")
		return
	}

	// Generate master playlist content
	variantInfos := make([]hls.VariantInfo, len(readyVariants))
	for i, v := range readyVariants {
		variantInfos[i] = hls.VariantInfo{
			Name:         v.ResolutionName,
			Bandwidth:    int(v.BitrateKbps) * 1000,
			Width:        int(v.Width),
			Height:       int(v.Height),
			PlaylistPath: fmt.Sprintf("%s/playlist.m3u8", v.ResolutionName),
		}
	}
	masterContent := hls.GenerateMasterPlaylist(variantInfos)

	// Upload master playlist to S3
	masterKey := fmt.Sprintf("videos/%s/master.m3u8", videoUUID)
	if err := s.s3.Upload(context.Background(), masterKey, strings.NewReader(masterContent), "application/vnd.apple.mpegurl"); err != nil {
		_ = s.videoRepo.UpdateStatus(videoUUID, "error")
		return
	}

	video, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		return
	}
	video.MasterPlaylistKey = &masterKey
	video.Status = "ready"
	_ = s.videoRepo.Update(video)
}

func (s *TranscodeService) FailTask(taskUUID, errMsg string) error {
	return s.taskRepo.Fail(taskUUID, errMsg)
}

func (s *TranscodeService) SetWorker(taskUUID, workerID string) error {
	return s.taskRepo.SetWorker(taskUUID, workerID)
}

func (s *TranscodeService) CancelTask(taskUUID string) error {
	return s.taskRepo.UpdateStatus(taskUUID, "cancelled")
}

func (s *TranscodeService) ListAllTasks(params repository.TaskListParams) ([]model.TranscodeTask, int64, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 20
	}
	return s.taskRepo.ListAll(params)
}

func (s *TranscodeService) RetryTask(ctx context.Context, taskUUID string) (*model.TranscodeTask, error) {
	task, err := s.taskRepo.FindByUUID(taskUUID)
	if err != nil {
		return nil, err
	}
	if task.Status != "failed" {
		return nil, fmt.Errorf("can only retry failed tasks, current status: %s", task.Status)
	}

	_ = s.taskRepo.UpdateStatus(taskUUID, "queued")
	_ = s.taskRepo.IncrementAttempts(taskUUID)

	msg := queue.TaskMessage{
		TaskUUID: task.TaskUUID,
		VideoID:  task.VideoID,
		Type:     task.Type,
		Params:   json.RawMessage(task.Params),
	}
	if err := s.producer.Publish(ctx, msg); err != nil {
		return nil, err
	}

	return s.taskRepo.FindByUUID(taskUUID)
}

// GenerateThumbnails is the handler-facing wrapper for StartThumbnailGeneration.
func (s *TranscodeService) GenerateThumbnails(ctx context.Context, videoUUID string, count, width int) (*model.TranscodeTask, error) {
	height := width * 9 / 16 // assume 16:9
	return s.StartThumbnailGeneration(ctx, videoUUID, count, width, height)
}

// ReprocessPendingResults finds completed probe/thumbnail tasks where the video
// metadata was never applied (e.g. tasks completed before the post-processing
// code was deployed) and reprocesses them.
func (s *TranscodeService) ReprocessPendingResults() {
	// Reprocess probe results for videos still missing metadata
	probeTasks, err := s.taskRepo.FindCompletedWithResult("probe")
	if err != nil {
		return
	}
	for _, task := range probeTasks {
		video, err := s.videoRepo.FindByID(task.VideoID)
		if err != nil || video == nil {
			continue
		}
		// Skip if metadata already populated
		if video.DurationSeconds != nil && video.Width != nil {
			continue
		}
		s.handleProbeResult(&task, task.Result)
	}

	// Reprocess thumbnail results for videos with no thumbnails
	thumbTasks, err := s.taskRepo.FindCompletedWithResult("thumbnail")
	if err != nil {
		return
	}
	for _, task := range thumbTasks {
		thumbs, _ := s.videoRepo.ListThumbnails(task.VideoID)
		if len(thumbs) > 0 {
			continue
		}
		s.handleThumbnailResult(&task, task.Result)
	}
}

// CheckVideoComplete checks if all tasks for a video are done and updates status.
func (s *TranscodeService) CheckVideoComplete(videoID uint, videoUUID string) error {
	pending, err := s.taskRepo.CountPendingByVideo(videoID)
	if err != nil {
		return err
	}
	if pending == 0 {
		return s.videoRepo.UpdateStatus(videoUUID, "ready")
	}
	return nil
}
