package model

import "time"

type Thumbnail struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	VideoID    uint      `json:"video_id" gorm:"not null;index"`
	S3Key      string    `json:"s3_key" gorm:"size:1000;not null"`
	Width      uint      `json:"width" gorm:"not null"`
	Height     uint      `json:"height" gorm:"not null"`
	SizeBytes  uint      `json:"size_bytes" gorm:"not null"`
	TimestampS float64   `json:"timestamp_s" gorm:"type:decimal(10,2);not null"`
	IsDefault  bool      `json:"is_default" gorm:"default:false;not null;index:idx_video_default"`
	SortOrder  int       `json:"sort_order" gorm:"default:0;not null"`
	CreatedAt  time.Time `json:"created_at"`
}
