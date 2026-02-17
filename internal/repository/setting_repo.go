package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type SettingRepo struct {
	db *gorm.DB
}

func NewSettingRepo(db *gorm.DB) *SettingRepo {
	return &SettingRepo{db: db}
}

func (r *SettingRepo) List() ([]model.AppSetting, error) {
	var settings []model.AppSetting
	err := r.db.Order("key").Find(&settings).Error
	return settings, err
}

func (r *SettingRepo) Get(key string) (*model.AppSetting, error) {
	var s model.AppSetting
	err := r.db.Where("`key` = ?", key).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingRepo) Set(key, value string) error {
	var s model.AppSetting
	err := r.db.Where("`key` = ?", key).First(&s).Error
	if err != nil {
		return err
	}
	s.Value = value
	return r.db.Save(&s).Error
}
