package service

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type VideoService struct {
	videoRepo *repository.VideoRepo
}

func NewVideoService(videoRepo *repository.VideoRepo) *VideoService {
	return &VideoService{videoRepo: videoRepo}
}

type CreateVideoInput struct {
	Slug             string `json:"slug" binding:"required"`
	OriginalFilename string `json:"original_filename" binding:"required"`
	ReleaseDate      string `json:"release_date"`
	Rating           string `json:"rating"`
	// Initial translation
	Language    string `json:"language" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Synopsis    string `json:"synopsis"`
}

func (s *VideoService) Create(input CreateVideoInput, userID uint) (*model.Video, error) {
	videoUUID := uuid.New().String()
	s3Key := fmt.Sprintf("videos/%s/original/%s", videoUUID, input.OriginalFilename)

	video := &model.Video{
		UUID:             videoUUID,
		Slug:             input.Slug,
		Status:           "draft",
		OriginalFilename: input.OriginalFilename,
		OriginalS3Key:    s3Key,
		Rating:           input.Rating,
		CreatedBy:        userID,
	}

	if err := s.videoRepo.Create(video); err != nil {
		return nil, err
	}

	// Create initial translation
	trans := &model.VideoTranslation{
		VideoID:      video.ID,
		LanguageCode: input.Language,
		Title:        input.Title,
		Description:  input.Description,
		Synopsis:     input.Synopsis,
	}
	if err := s.videoRepo.UpsertTranslation(trans); err != nil {
		return nil, err
	}

	return s.videoRepo.FindByUUID(videoUUID)
}

func (s *VideoService) Get(uuid string) (*model.Video, error) {
	return s.videoRepo.FindByUUID(uuid)
}

func (s *VideoService) List(params repository.VideoListParams) ([]model.Video, int64, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 20
	}
	return s.videoRepo.List(params)
}

type UpdateVideoInput struct {
	Slug        *string `json:"slug"`
	Rating      *string `json:"rating"`
	ReleaseDate *string `json:"release_date"`
	HasDRM      *bool   `json:"has_drm"`
	SortOrder   *int    `json:"sort_order"`
}

func (s *VideoService) Update(uuid string, input UpdateVideoInput) (*model.Video, error) {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return nil, err
	}

	if input.Slug != nil {
		video.Slug = *input.Slug
	}
	if input.Rating != nil {
		video.Rating = *input.Rating
	}
	if input.HasDRM != nil {
		video.HasDRM = *input.HasDRM
	}
	if input.SortOrder != nil {
		video.SortOrder = *input.SortOrder
	}

	if err := s.videoRepo.Update(video); err != nil {
		return nil, err
	}

	return s.videoRepo.FindByUUID(uuid)
}

func (s *VideoService) Delete(uuid string) error {
	return s.videoRepo.SoftDelete(uuid)
}

func (s *VideoService) Restore(uuid string) error {
	return s.videoRepo.Restore(uuid)
}

// Translation operations

type TranslationInput struct {
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description"`
	Synopsis       string `json:"synopsis"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
}

func (s *VideoService) UpsertTranslation(uuid, lang string, input TranslationInput) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}

	trans := &model.VideoTranslation{
		VideoID:        video.ID,
		LanguageCode:   lang,
		Title:          input.Title,
		Description:    input.Description,
		Synopsis:       input.Synopsis,
		SEOTitle:       input.SEOTitle,
		SEODescription: input.SEODescription,
	}

	return s.videoRepo.UpsertTranslation(trans)
}

func (s *VideoService) DeleteTranslation(uuid, lang string) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}
	return s.videoRepo.DeleteTranslation(video.ID, lang)
}

func (s *VideoService) ListTranslations(uuid string) ([]model.VideoTranslation, error) {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return nil, err
	}
	return s.videoRepo.ListTranslations(video.ID)
}

// Variant operations

func (s *VideoService) ListVariants(uuid string) ([]model.VideoVariant, error) {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return nil, err
	}
	return s.videoRepo.ListVariants(video.ID)
}

// Thumbnail operations

func (s *VideoService) ListThumbnails(uuid string) ([]model.Thumbnail, error) {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return nil, err
	}
	return s.videoRepo.ListThumbnails(video.ID)
}

func (s *VideoService) SetDefaultThumbnail(uuid string, thumbID uint) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}
	return s.videoRepo.SetDefaultThumbnail(video.ID, thumbID)
}

func (s *VideoService) DeleteThumbnail(id uint) error {
	return s.videoRepo.DeleteThumbnail(id)
}

// Subtitle operations

func (s *VideoService) ListSubtitles(uuid string) ([]model.Subtitle, error) {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return nil, err
	}
	return s.videoRepo.ListSubtitles(video.ID)
}

func (s *VideoService) CreateSubtitle(uuid string, sub *model.Subtitle) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}
	sub.VideoID = video.ID
	return s.videoRepo.CreateSubtitle(sub)
}

func (s *VideoService) DeleteSubtitle(id uint) error {
	return s.videoRepo.DeleteSubtitle(id)
}

func (s *VideoService) FindSubtitle(id uint) (*model.Subtitle, error) {
	return s.videoRepo.FindSubtitle(id)
}

// Cast operations

func (s *VideoService) ListCast(uuid string) ([]model.VideoCast, error) {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return nil, err
	}
	return s.videoRepo.ListCast(video.ID)
}

func (s *VideoService) AddCast(uuid string, cast *model.VideoCast) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}
	cast.VideoID = video.ID
	return s.videoRepo.AddCast(cast)
}

func (s *VideoService) DeleteCast(id uint) error {
	return s.videoRepo.DeleteCast(id)
}

// Category/Tag association

func (s *VideoService) SetCategories(uuid string, categoryIDs []uint) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}
	return s.videoRepo.SetCategories(video.ID, categoryIDs)
}

func (s *VideoService) SetTags(uuid string, tagIDs []uint) error {
	video, err := s.videoRepo.FindByUUID(uuid)
	if err != nil {
		return err
	}
	return s.videoRepo.SetTags(video.ID, tagIDs)
}

func (s *VideoService) UpdateStatus(uuid, status string) error {
	return s.videoRepo.UpdateStatus(uuid, status)
}

func (s *VideoService) GetRepo() *repository.VideoRepo {
	return s.videoRepo
}
