package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"

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

// GenerateKeys creates a DRM key for a video and generates a Widevine PSSH box.
// If keys already exist for this video, they are returned as-is.
func (s *DRMService) GenerateKeys(videoID uint, licenseBaseURL string) (*model.DRMKey, error) {
	// Check if key already exists
	existing, err := s.drmRepo.FindByVideoID(videoID)
	if err == nil && existing != nil {
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

	// Generate Widevine PSSH box
	psshBase64, err := generateWidevinePSSH(keyID)
	if err != nil {
		return nil, fmt.Errorf("generate PSSH: %w", err)
	}

	key := &model.DRMKey{
		VideoID:    videoID,
		KeyID:      keyID,
		ContentKey: contentKey,
		IV:         iv,
		PSSHBox:    psshBase64,
		LicenseURL: fmt.Sprintf("%s/api/v1/drm/clearkey/license", licenseBaseURL),
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

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateWidevinePSSH builds a Widevine PSSH box containing the given key ID.
// The PSSH box format follows the Common Encryption (CENC) spec:
//
//	Box header (8 bytes): size (4) + "pssh" (4)
//	Version + flags (4 bytes)
//	SystemID (16 bytes): Widevine = edef8ba9-79d6-4ace-a3c8-27dcd51d21ed
//	Data size (4 bytes)
//	Data: Widevine-specific protobuf containing the key ID
func generateWidevinePSSH(keyIDHex string) (string, error) {
	keyIDBytes, err := hex.DecodeString(keyIDHex)
	if err != nil {
		return "", fmt.Errorf("decode key_id: %w", err)
	}

	// Widevine SystemID
	widevineSystemID := []byte{
		0xed, 0xef, 0x8b, 0xa9, 0x79, 0xd6, 0x4a, 0xce,
		0xa3, 0xc8, 0x27, 0xdc, 0xd5, 0x1d, 0x21, 0xed,
	}

	// Widevine PSSH data is a simple protobuf:
	// field 2 (key_id), wire type 2 (length-delimited) = tag 0x12
	// length = 16 (0x10)
	// followed by the 16-byte key ID
	psshData := make([]byte, 0, 2+len(keyIDBytes))
	psshData = append(psshData, 0x12, byte(len(keyIDBytes)))
	psshData = append(psshData, keyIDBytes...)

	// Build the full PSSH box
	// version 0, no flags
	dataSize := len(psshData)
	boxSize := 4 + 4 + 4 + 16 + 4 + dataSize // size + "pssh" + version+flags + systemID + dataSize + data

	box := make([]byte, boxSize)
	binary.BigEndian.PutUint32(box[0:4], uint32(boxSize))
	copy(box[4:8], "pssh")
	// version 0, flags 0 (bytes 8-11 are already zero)
	copy(box[12:28], widevineSystemID)
	binary.BigEndian.PutUint32(box[28:32], uint32(dataSize))
	copy(box[32:], psshData)

	return base64.StdEncoding.EncodeToString(box), nil
}
