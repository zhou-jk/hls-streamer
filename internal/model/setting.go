package model

type AppSetting struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Key         string `json:"key" gorm:"uniqueIndex;size:100;not null"`
	Value       string `json:"value" gorm:"size:500;not null"`
	Description string `json:"description" gorm:"size:500"`
}
