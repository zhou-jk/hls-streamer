package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type TaskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(task *model.TranscodeTask) error {
	return r.db.Create(task).Error
}

func (r *TaskRepo) FindByUUID(uuid string) (*model.TranscodeTask, error) {
	var task model.TranscodeTask
	err := r.db.Where("task_uuid = ?", uuid).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepo) ListByVideoID(videoID uint) ([]model.TranscodeTask, error) {
	var tasks []model.TranscodeTask
	err := r.db.Where("video_id = ?", videoID).Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepo) UpdateStatus(taskUUID, status string) error {
	return r.db.Model(&model.TranscodeTask{}).
		Where("task_uuid = ?", taskUUID).
		Update("status", status).Error
}

func (r *TaskRepo) UpdateProgress(taskUUID string, progress uint8) error {
	return r.db.Model(&model.TranscodeTask{}).
		Where("task_uuid = ?", taskUUID).
		Update("progress", progress).Error
}

func (r *TaskRepo) SetWorker(taskUUID, workerID string) error {
	return r.db.Model(&model.TranscodeTask{}).
		Where("task_uuid = ?", taskUUID).
		Updates(map[string]interface{}{
			"worker_id":  workerID,
			"status":     "processing",
			"started_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *TaskRepo) Complete(taskUUID string, result model.JSON) error {
	return r.db.Model(&model.TranscodeTask{}).
		Where("task_uuid = ?", taskUUID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"progress":     100,
			"result":       result,
			"completed_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *TaskRepo) Fail(taskUUID, errMsg string) error {
	return r.db.Model(&model.TranscodeTask{}).
		Where("task_uuid = ?", taskUUID).
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": errMsg,
			"completed_at":  gorm.Expr("NOW()"),
		}).Error
}

func (r *TaskRepo) IncrementAttempts(taskUUID string) error {
	return r.db.Model(&model.TranscodeTask{}).
		Where("task_uuid = ?", taskUUID).
		Update("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *TaskRepo) AddLog(log *model.TaskLog) error {
	return r.db.Create(log).Error
}

func (r *TaskRepo) GetLogs(taskID uint) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	err := r.db.Where("task_id = ?", taskID).Order("created_at").Find(&logs).Error
	return logs, err
}

// CountPendingByVideo returns the number of non-completed tasks for a video.
func (r *TaskRepo) CountPendingByVideo(videoID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.TranscodeTask{}).
		Where("video_id = ? AND status NOT IN ?", videoID, []string{"completed", "failed", "cancelled"}).
		Count(&count).Error
	return count, err
}
