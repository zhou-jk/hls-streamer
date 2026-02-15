package model

import "time"

type Category struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Slug      string    `json:"slug" gorm:"uniqueIndex;size:255;not null"`
	ParentID  *uint     `json:"parent_id"`
	SortOrder int       `json:"sort_order" gorm:"default:0;not null"`
	IsActive  bool      `json:"is_active" gorm:"default:true;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Translations []CategoryTranslation `json:"translations,omitempty" gorm:"foreignKey:CategoryID"`
	Children     []Category            `json:"children,omitempty" gorm:"foreignKey:ParentID"`
}

type CategoryTranslation struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	CategoryID   uint   `json:"category_id" gorm:"uniqueIndex:uk_category_lang;not null"`
	LanguageCode string `json:"language_code" gorm:"uniqueIndex:uk_category_lang;size:10;not null"`
	Name         string `json:"name" gorm:"size:255;not null"`
	Description  string `json:"description" gorm:"type:text"`
}
