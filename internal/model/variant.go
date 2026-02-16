package model

import "time"

type VideoVariant struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	VideoID        uint      `json:"video_id" gorm:"uniqueIndex:uk_video_resolution;not null"`
	ResolutionName string    `json:"resolution_name" gorm:"uniqueIndex:uk_video_resolution;size:20;not null"`
	Codec          string    `json:"codec" gorm:"uniqueIndex:uk_video_resolution;size:50;default:h264;not null"`
	Width          uint      `json:"width" gorm:"not null"`
	Height         uint      `json:"height" gorm:"not null"`
	BitrateKbps    uint      `json:"bitrate_kbps" gorm:"not null"`
	PlaylistS3Key  string    `json:"playlist_s3_key" gorm:"size:1000;not null"`
	SegmentCount   uint      `json:"segment_count" gorm:"default:0;not null"`
	TotalSizeBytes int64     `json:"total_size_bytes" gorm:"default:0;not null"`
	DRM            bool      `json:"drm" gorm:"default:false;not null"`
	Status         string    `json:"status" gorm:"size:20;default:pending;not null"` // pending, processing, ready, error
	CreatedAt      time.Time `json:"created_at"`
}

type ResolutionPreset struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	Name            string `json:"name" gorm:"uniqueIndex;size:20;not null"`
	Width           uint   `json:"width" gorm:"not null"`
	Height          uint   `json:"height" gorm:"not null"`
	BitrateKbps     uint   `json:"bitrate_kbps" gorm:"not null"`
	AudioBitrateKbps uint  `json:"audio_bitrate_kbps" gorm:"default:128;not null"`
	IsActive        bool   `json:"is_active" gorm:"default:true;not null"`
	SortOrder       int    `json:"sort_order" gorm:"default:0;not null"`
}
