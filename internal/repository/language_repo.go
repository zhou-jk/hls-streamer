package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type LanguageRepo struct {
	db *gorm.DB
}

func NewLanguageRepo(db *gorm.DB) *LanguageRepo {
	return &LanguageRepo{db: db}
}

func (r *LanguageRepo) List() ([]model.Language, error) {
	var langs []model.Language
	err := r.db.Where("is_active = ?", true).Order("sort_order").Find(&langs).Error
	return langs, err
}

func (r *LanguageRepo) GetDefault() (*model.Language, error) {
	var lang model.Language
	err := r.db.Where("is_default = ?", true).First(&lang).Error
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) FindByCode(code string) (*model.Language, error) {
	var lang model.Language
	err := r.db.First(&lang, "code = ?", code).Error
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) Create(lang *model.Language) error {
	return r.db.Create(lang).Error
}

// ResolutionPreset operations

func (r *LanguageRepo) ListPresets() ([]model.ResolutionPreset, error) {
	var presets []model.ResolutionPreset
	err := r.db.Where("is_active = ?", true).Order("sort_order").Find(&presets).Error
	return presets, err
}
