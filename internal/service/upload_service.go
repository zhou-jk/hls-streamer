package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/Zhou-JK/hls-streamer/internal/repository"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
)

type UploadService struct {
	s3        *storage.S3Client
	videoRepo *repository.VideoRepo
}

func NewUploadService(s3 *storage.S3Client, videoRepo *repository.VideoRepo) *UploadService {
	return &UploadService{s3: s3, videoRepo: videoRepo}
}

type InitiateUploadResponse struct {
	UploadID  string            `json:"upload_id"`
	PartURLs  map[int]string    `json:"part_urls"`
	S3Key     string            `json:"s3_key"`
}

// InitiateUpload creates a multipart upload and returns presigned URLs for each part.
func (s *UploadService) InitiateUpload(ctx context.Context, videoUUID, filename, contentType string, partCount int) (*InitiateUploadResponse, error) {
	// Randomize the S3 key to prevent guessing the original filename
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	ext := filepath.Ext(filename)
	s3Key := fmt.Sprintf("videos/%s/original/%s%s", videoUUID, hex.EncodeToString(randBytes), ext)

	// Update the video's original_filename with the real filename
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return nil, fmt.Errorf("find video: %w", err)
	}
	video.OriginalFilename = filename
	if err := s.videoRepo.Update(video); err != nil {
		return nil, fmt.Errorf("update original filename: %w", err)
	}

	uploadID, err := s.s3.CreateMultipartUpload(ctx, s3Key, contentType)
	if err != nil {
		return nil, fmt.Errorf("create multipart upload: %w", err)
	}

	partURLs := make(map[int]string, partCount)
	for i := 1; i <= partCount; i++ {
		url, err := s.s3.PresignUploadPart(ctx, s3Key, uploadID, int32(i), 1*time.Hour)
		if err != nil {
			return nil, fmt.Errorf("presign part %d: %w", i, err)
		}
		partURLs[i] = url
	}

	return &InitiateUploadResponse{
		UploadID: uploadID,
		PartURLs: partURLs,
		S3Key:    s3Key,
	}, nil
}

type CompleteUploadRequest struct {
	UploadID string         `json:"upload_id" binding:"required"`
	S3Key    string         `json:"s3_key" binding:"required"`
	Parts    []PartInfo     `json:"parts" binding:"required,min=1"`
}

type PartInfo struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

// CompleteUpload finalizes the multipart upload.
func (s *UploadService) CompleteUpload(ctx context.Context, videoUUID string, req CompleteUploadRequest) error {
	parts := make([]s3types.CompletedPart, len(req.Parts))
	for i, p := range req.Parts {
		partNum := int32(p.PartNumber)
		parts[i] = s3types.CompletedPart{
			PartNumber: &partNum,
			ETag:       &p.ETag,
		}
	}

	if err := s.s3.CompleteMultipartUpload(ctx, req.S3Key, req.UploadID, parts); err != nil {
		return fmt.Errorf("complete multipart upload: %w", err)
	}

	// Update video's S3 key
	video, err := s.videoRepo.FindByUUID(videoUUID)
	if err != nil {
		return err
	}
	video.OriginalS3Key = req.S3Key
	return s.videoRepo.Update(video)
}

// PresignDownload returns a presigned URL for downloading a file from S3.
func (s *UploadService) PresignDownload(ctx context.Context, s3Key string, expires time.Duration) (string, error) {
	return s.s3.PresignGetObject(ctx, s3Key, expires)
}
