package model

import "time"

type TranscodeTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	TaskUUID     string     `json:"task_uuid" gorm:"uniqueIndex;size:36;not null"`
	VideoID      uint       `json:"video_id" gorm:"not null;index:idx_video_type"`
	Type         string     `json:"type" gorm:"size:20;not null;index:idx_video_type"` // transcode, thumbnail, drm_package, probe
	Status       string     `json:"status" gorm:"size:20;default:pending;not null;index:idx_status_priority"` // pending, queued, processing, completed, failed, cancelled
	Priority     int        `json:"priority" gorm:"default:0;not null;index:idx_status_priority"`
	WorkerID     *string    `json:"worker_id" gorm:"size:100;index"`
	Params       JSON       `json:"params" gorm:"type:json;not null"`
	Result       JSON       `json:"result,omitempty" gorm:"type:json"`
	ErrorMessage *string    `json:"error_message,omitempty" gorm:"type:text"`
	Progress     uint8      `json:"progress" gorm:"default:0;not null"` // 0-100
	Attempts     uint       `json:"attempts" gorm:"default:0;not null"`
	MaxAttempts  uint       `json:"max_attempts" gorm:"default:3;not null"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type TaskLog struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	TaskID    uint      `json:"task_id" gorm:"not null;index"`
	Level     string    `json:"level" gorm:"size:10;default:info;not null"` // info, warn, error
	Message   string    `json:"message" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
}
