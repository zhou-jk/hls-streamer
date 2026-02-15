package model

import "time"

type DRMKey struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	VideoID     uint      `json:"video_id" gorm:"not null;index:idx_video_drm"`
	KeyID       string    `json:"key_id" gorm:"uniqueIndex;size:64;not null"`
	ContentKey  string    `json:"-" gorm:"size:64;not null"` // encrypted at rest
	IV          string    `json:"-" gorm:"size:64"`
	DRMSystem   string    `json:"drm_system" gorm:"size:20;not null;index:idx_video_drm"` // widevine, fairplay, common
	LicenseURL  string    `json:"license_url" gorm:"size:1000"`
	PSSHBox     string    `json:"pssh_box,omitempty" gorm:"type:text"` // Base64 PSSH for Widevine
	FairPlayURI string    `json:"fairplay_uri,omitempty" gorm:"size:1000"`
	CreatedAt   time.Time `json:"created_at"`
}
