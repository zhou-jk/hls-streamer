package repository

import (
	"time"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type WorkerRepo struct {
	db *gorm.DB
}

func NewWorkerRepo(db *gorm.DB) *WorkerRepo {
	return &WorkerRepo{db: db}
}

func (r *WorkerRepo) Upsert(w *model.Worker) error {
	return r.db.Where("id = ?", w.ID).Assign(w).FirstOrCreate(w).Error
}

func (r *WorkerRepo) UpdateStatus(id, status string) error {
	return r.db.Model(&model.Worker{}).Where("id = ?", id).Update("status", status).Error
}

func (r *WorkerRepo) UpdateHeartbeat(id string) error {
	return r.db.Model(&model.Worker{}).Where("id = ?", id).
		Update("last_heartbeat", gorm.Expr("NOW()")).Error
}

func (r *WorkerRepo) SetCurrentTask(workerID string, taskID *uint) error {
	return r.db.Model(&model.Worker{}).Where("id = ?", workerID).
		Update("current_task_id", taskID).Error
}

func (r *WorkerRepo) List() ([]model.Worker, error) {
	var workers []model.Worker
	err := r.db.Order("registered_at DESC").Find(&workers).Error
	return workers, err
}

// CleanupStale marks workers as offline if no heartbeat for given duration,
// and deletes workers that have been offline for over 24 hours.
func (r *WorkerRepo) CleanupStale(timeout time.Duration) error {
	cutoff := time.Now().Add(-timeout)
	if err := r.db.Model(&model.Worker{}).
		Where("status != ? AND last_heartbeat < ?", "offline", cutoff).
		Update("status", "offline").Error; err != nil {
		return err
	}
	// Delete workers offline for over 24h
	deleteCutoff := time.Now().Add(-24 * time.Hour)
	return r.db.Where("status = ? AND last_heartbeat < ?", "offline", deleteCutoff).
		Delete(&model.Worker{}).Error
}

func (r *WorkerRepo) FindByID(id string) (*model.Worker, error) {
	var w model.Worker
	err := r.db.First(&w, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}
