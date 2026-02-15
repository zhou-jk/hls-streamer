package model

import "time"

type Subtitle struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	VideoID      uint      `json:"video_id" gorm:"uniqueIndex:uk_video_subtitle_lang;not null"`
	LanguageCode string    `json:"language_code" gorm:"uniqueIndex:uk_video_subtitle_lang;size:10;not null"`
	Label        string    `json:"label" gorm:"size:100;not null"`
	S3Key        string    `json:"s3_key" gorm:"size:1000;not null"`
	IsDefault    bool      `json:"is_default" gorm:"default:false;not null"`
	SortOrder    int       `json:"sort_order" gorm:"default:0;not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
