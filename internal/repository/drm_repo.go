package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type DRMRepo struct {
	db *gorm.DB
}

func NewDRMRepo(db *gorm.DB) *DRMRepo {
	return &DRMRepo{db: db}
}

func (r *DRMRepo) Create(key *model.DRMKey) error {
	return r.db.Create(key).Error
}

func (r *DRMRepo) FindByVideoID(videoID uint) (*model.DRMKey, error) {
	var key model.DRMKey
	err := r.db.Where("video_id = ?", videoID).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *DRMRepo) FindByKeyID(keyID string) (*model.DRMKey, error) {
	var key model.DRMKey
	err := r.db.Where("key_id = ?", keyID).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *DRMRepo) DeleteByVideoID(videoID uint) error {
	return r.db.Where("video_id = ?", videoID).Delete(&model.DRMKey{}).Error
}
