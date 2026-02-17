package service

import (
	"strconv"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type SettingService struct {
	repo *repository.SettingRepo
}

func NewSettingService(repo *repository.SettingRepo) *SettingService {
	return &SettingService{repo: repo}
}

func (s *SettingService) List() ([]model.AppSetting, error) {
	return s.repo.List()
}

func (s *SettingService) Get(key string) (*model.AppSetting, error) {
	return s.repo.Get(key)
}

func (s *SettingService) GetInt(key string, fallback int) int {
	setting, err := s.repo.Get(key)
	if err != nil {
		return fallback
	}
	v, err := strconv.Atoi(setting.Value)
	if err != nil {
		return fallback
	}
	return v
}

func (s *SettingService) GetBool(key string, fallback bool) bool {
	setting, err := s.repo.Get(key)
	if err != nil {
		return fallback
	}
	v, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return fallback
	}
	return v
}

func (s *SettingService) Update(key, value string) error {
	return s.repo.Set(key, value)
}
