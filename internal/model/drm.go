package model

import "time"

type DRMKey struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	VideoID    uint      `json:"video_id" gorm:"not null;uniqueIndex:idx_video_drm"`
	KeyID      string    `json:"key_id" gorm:"size:64;not null"`  // hex-encoded 16 bytes
	ContentKey string    `json:"-" gorm:"size:64;not null"`       // hex-encoded 16 bytes, hidden from JSON
	IV         string    `json:"-" gorm:"size:64"`                // hex-encoded 16 bytes
	KeyURL     string    `json:"key_url" gorm:"size:1000"`        // URL to fetch the raw AES key
	CreatedAt  time.Time `json:"created_at"`
}
