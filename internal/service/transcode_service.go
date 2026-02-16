package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/queue"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type TranscodeService struct {
	taskRepo  *repository.TaskRepo
	videoRepo *repository.VideoRepo
	producer  *queue.Producer
}

func NewTranscodeService(taskRepo *repository.TaskRepo, videoRepo *repository.VideoRepo, producer *queue.Producer) *TranscodeService {
	return &TranscodeService{
		taskRepo:  taskRepo,
		videoRepo: videoRepo,
		producer:  producer,
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

func (s *TranscodeService) StartTranscode(ctx context.Context, videoUUID string, req TranscodeRequest) ([]model.TranscodeTask, error) {
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	if req.Codec == "" {
		req.Codec = "h264"
	}

	// Update video status
	_ = s.videoRepo.UpdateStatus(videoUUID, "processing")

	var tasks []model.TranscodeTask

	for _, res := range req.Resolutions {
		if res.AudioBitrate == 0 {
			res.AudioBitrate = 128
		}

		params, _ := json.Marshal(map[string]interface{}{
			"video_uuid":        videoUUID,
			"video_id":          video.ID,
			"original_s3_key":   video.OriginalS3Key,
			"resolution":        res.Name,
			"width":             res.Width,
			"height":            res.Height,
			"bitrate_kbps":      res.BitrateKbps,
			"audio_bitrate_kbps": res.AudioBitrate,
			"codec":             req.Codec,
			"drm":               req.DRM,
		})

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
	return s.taskRepo.Complete(taskUUID, result)
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
