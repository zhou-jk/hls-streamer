package model

import "time"

type Language struct {
	Code       string `json:"code" gorm:"primaryKey;size:10"`
	Name       string `json:"name" gorm:"size:100;not null"`
	NativeName string `json:"native_name" gorm:"size:100;not null"`
	IsDefault  bool   `json:"is_default" gorm:"default:false;not null"`
	IsActive   bool   `json:"is_active" gorm:"default:true;not null"`
	SortOrder  int    `json:"sort_order" gorm:"default:0;not null"`
}

type Video struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	UUID              string     `json:"uuid" gorm:"uniqueIndex;size:36;not null"`
	Slug              string     `json:"slug" gorm:"uniqueIndex;size:255;not null"`
	Status            string     `json:"status" gorm:"size:20;default:draft;not null;index"` // draft, processing, ready, error, archived
	OriginalFilename  string     `json:"original_filename" gorm:"size:500;not null"`
	OriginalS3Key     string     `json:"-" gorm:"size:1000;not null"`
	DurationSeconds   *float64   `json:"duration_seconds" gorm:"type:decimal(10,2)"`
	Width             *uint      `json:"width"`
	Height            *uint      `json:"height"`
	FileSizeBytes     *int64     `json:"file_size_bytes"`
	Codec             string     `json:"codec" gorm:"size:50"`
	FPS               *float64   `json:"fps" gorm:"type:decimal(6,2)"`
	HasAudio          bool       `json:"has_audio" gorm:"default:true;not null"`
	HasDRM            bool       `json:"has_drm" gorm:"default:false;not null"`
	IsPublic          bool       `json:"is_public" gorm:"default:false;not null;index"`
	MasterPlaylistKey *string    `json:"master_playlist_key" gorm:"size:1000"`
	ReleaseDate       *time.Time `json:"release_date" gorm:"type:date"`
	Rating            string     `json:"rating" gorm:"size:10"`
	SortOrder         int        `json:"sort_order" gorm:"default:0;not null"`
	ViewCount         uint64     `json:"view_count" gorm:"default:0;not null"`
	CreatedBy         uint       `json:"created_by" gorm:"not null"`
	Creator           *User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// Relations (loaded on demand)
	Translations []VideoTranslation `json:"translations,omitempty" gorm:"foreignKey:VideoID"`
	Variants     []VideoVariant     `json:"variants,omitempty" gorm:"foreignKey:VideoID"`
	Thumbnails   []Thumbnail        `json:"thumbnails,omitempty" gorm:"foreignKey:VideoID"`
	Subtitles    []Subtitle         `json:"subtitles,omitempty" gorm:"foreignKey:VideoID"`
	Categories   []Category         `json:"categories,omitempty" gorm:"many2many:video_categories"`
	Tags         []Tag              `json:"tags,omitempty" gorm:"many2many:video_tags"`
	Cast         []VideoCast        `json:"cast,omitempty" gorm:"foreignKey:VideoID"`
}

type VideoTranslation struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	VideoID        uint      `json:"video_id" gorm:"uniqueIndex:uk_video_lang;not null"`
	LanguageCode   string    `json:"language_code" gorm:"uniqueIndex:uk_video_lang;size:10;not null"`
	Title          string    `json:"title" gorm:"size:500;not null"`
	Description    string    `json:"description" gorm:"type:text"`
	Synopsis       string    `json:"synopsis" gorm:"type:text"`
	SEOTitle       string    `json:"seo_title" gorm:"size:200"`
	SEODescription string    `json:"seo_description" gorm:"size:500"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
