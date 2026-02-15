package model

type Tag struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Slug string `json:"slug" gorm:"uniqueIndex;size:255;not null"`

	Translations []TagTranslation `json:"translations,omitempty" gorm:"foreignKey:TagID"`
}

type TagTranslation struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	TagID        uint   `json:"tag_id" gorm:"uniqueIndex:uk_tag_lang;not null"`
	LanguageCode string `json:"language_code" gorm:"uniqueIndex:uk_tag_lang;size:10;not null"`
	Name         string `json:"name" gorm:"size:255;not null"`
}
