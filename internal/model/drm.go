package model

import "time"

type DRMKey struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	VideoID    uint      `json:"video_id" gorm:"not null;uniqueIndex:idx_video_drm"`
	KeyID      string    `json:"key_id" gorm:"size:64;not null"`       // hex-encoded 16 bytes
	ContentKey string    `json:"-" gorm:"size:64;not null"`            // hex-encoded 16 bytes, hidden from JSON
	IV         string    `json:"-" gorm:"size:64"`                     // hex-encoded 16 bytes
	PSSHBox    string    `json:"pssh_box,omitempty" gorm:"type:text"`  // Base64-encoded Widevine PSSH box
	LicenseURL string    `json:"license_url" gorm:"size:1000"`
	CreatedAt  time.Time `json:"created_at"`
}
