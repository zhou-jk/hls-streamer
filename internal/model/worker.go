package model

import "time"

type Worker struct {
	ID              string     `json:"id" gorm:"primaryKey;size:100"`
	Hostname        string     `json:"hostname" gorm:"size:255;not null"`
	IPAddress       string     `json:"ip_address" gorm:"size:45;not null"`
	Capabilities    JSON       `json:"capabilities" gorm:"type:json;not null"`
	Status          string     `json:"status" gorm:"size:20;default:offline;not null;index"` // online, busy, offline
	CurrentTaskID   *uint      `json:"current_task_id" gorm:"index"`
	LastHeartbeat   time.Time  `json:"last_heartbeat" gorm:"not null"`
	RegisteredAt    time.Time  `json:"registered_at"`
}
