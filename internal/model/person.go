package model

import "time"

type Person struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Slug      string     `json:"slug" gorm:"uniqueIndex;size:255;not null"`
	PhotoURL  string     `json:"photo_url" gorm:"size:1000"`
	BirthDate *time.Time `json:"birth_date" gorm:"type:date"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Translations []PersonTranslation `json:"translations,omitempty" gorm:"foreignKey:PersonID"`
}

type PersonTranslation struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	PersonID     uint   `json:"person_id" gorm:"uniqueIndex:uk_person_lang;not null"`
	LanguageCode string `json:"language_code" gorm:"uniqueIndex:uk_person_lang;size:10;not null"`
	Name         string `json:"name" gorm:"size:255;not null"`
	Biography    string `json:"biography" gorm:"type:text"`
}

type VideoCast struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	VideoID       uint   `json:"video_id" gorm:"not null;index:idx_video_role"`
	PersonID      uint   `json:"person_id" gorm:"not null"`
	Role          string `json:"role" gorm:"size:20;not null;index:idx_video_role"` // director, actor, writer, producer, other
	CharacterName string `json:"character_name" gorm:"size:255"`
	SortOrder     int    `json:"sort_order" gorm:"default:0;not null"`

	Person *Person `json:"person,omitempty" gorm:"foreignKey:PersonID"`
}
