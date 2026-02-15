package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type DRMService struct {
	drmRepo *repository.DRMRepo
}

func NewDRMService(drmRepo *repository.DRMRepo) *DRMService {
	return &DRMService{drmRepo: drmRepo}
}

// GenerateKeys creates DRM keys for a video (both Widevine and FairPlay).
func (s *DRMService) GenerateKeys(videoID uint, licenseBaseURL string) ([]model.DRMKey, error) {
	keyID, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate key_id: %w", err)
	}
	contentKey, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate content_key: %w", err)
	}
	iv, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate iv: %w", err)
	}

	var keys []model.DRMKey

	// Widevine key
	wvKey := model.DRMKey{
		VideoID:    videoID,
		KeyID:      keyID,
		ContentKey: contentKey,
		IV:         iv,
		DRMSystem:  "widevine",
		LicenseURL: fmt.Sprintf("%s/api/v1/drm/widevine/license", licenseBaseURL),
	}
	if err := s.drmRepo.Create(&wvKey); err != nil {
		return nil, err
	}
	keys = append(keys, wvKey)

	// FairPlay key (same content key, different system entry)
	fpKey := model.DRMKey{
		VideoID:    videoID,
		KeyID:      keyID,
		ContentKey: contentKey,
		IV:         iv,
		DRMSystem:  "fairplay",
		LicenseURL: fmt.Sprintf("%s/api/v1/drm/fairplay/license", licenseBaseURL),
		FairPlayURI: fmt.Sprintf("skd://%s", keyID),
	}
	if err := s.drmRepo.Create(&fpKey); err != nil {
		return nil, err
	}
	keys = append(keys, fpKey)

	return keys, nil
}

func (s *DRMService) GetKeysByVideo(videoID uint) ([]model.DRMKey, error) {
	return s.drmRepo.FindByVideoID(videoID)
}

func (s *DRMService) GetKeyByID(keyID string) (*model.DRMKey, error) {
	return s.drmRepo.FindByKeyID(keyID)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
