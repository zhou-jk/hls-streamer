package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type DRMService struct {
	drmRepo   *repository.DRMRepo
	videoRepo *repository.VideoRepo
}

func NewDRMService(drmRepo *repository.DRMRepo, videoRepo *repository.VideoRepo) *DRMService {
	return &DRMService{drmRepo: drmRepo, videoRepo: videoRepo}
}

// GenerateKeys creates a DRM key for a video.
// If keys already exist for this video, they are returned as-is.
func (s *DRMService) GenerateKeys(videoID uint, baseURL string) (*model.DRMKey, error) {
	// Check if key already exists
	existing, err := s.drmRepo.FindByVideoID(videoID)
	if err == nil && existing != nil {
		dirty := false
		// Backfill KeyURL if empty or pointing to old endpoint
		expectedURL := fmt.Sprintf("%s/api/v1/drm/key/%s", baseURL, existing.KeyID)
		if existing.KeyURL == "" || strings.Contains(existing.KeyURL, "/clearkey/") {
			existing.KeyURL = expectedURL
			dirty = true
		}
		// Fix truncated IV (must be exactly 32 hex chars = 16 bytes)
		if len(existing.IV) != 32 {
			newIV, err := randomHex(16)
			if err == nil {
				existing.IV = newIV
				dirty = true
			}
		}
		if dirty {
			_ = s.drmRepo.Update(existing)
		}
		return existing, nil
	}

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

	key := &model.DRMKey{
		VideoID:    videoID,
		KeyID:      keyID,
		ContentKey: contentKey,
		IV:         iv,
		KeyURL:     fmt.Sprintf("%s/api/v1/drm/key/%s", baseURL, keyID),
	}

	if err := s.drmRepo.Create(key); err != nil {
		return nil, err
	}

	// Mark video as DRM-enabled
	video, err := s.videoRepo.FindByID(videoID)
	if err == nil && video != nil {
		video.HasDRM = true
		_ = s.videoRepo.Update(video)
	}

	return key, nil
}

func (s *DRMService) GetKeyByVideo(videoID uint) (*model.DRMKey, error) {
	return s.drmRepo.FindByVideoID(videoID)
}

func (s *DRMService) GetKeyByID(keyID string) (*model.DRMKey, error) {
	return s.drmRepo.FindByKeyID(keyID)
}

func (s *DRMService) DeleteByVideoID(videoID uint) error {
	return s.drmRepo.DeleteByVideoID(videoID)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
