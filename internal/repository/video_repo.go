package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type VideoRepo struct {
	db *gorm.DB
}

func NewVideoRepo(db *gorm.DB) *VideoRepo {
	return &VideoRepo{db: db}
}

func (r *VideoRepo) Create(video *model.Video) error {
	return r.db.Create(video).Error
}

func (r *VideoRepo) FindByUUID(uuid string) (*model.Video, error) {
	var video model.Video
	err := r.db.Where("uuid = ? AND deleted_at IS NULL", uuid).
		Preload("Translations").
		Preload("Variants").
		Preload("Thumbnails").
		Preload("Subtitles").
		Preload("Categories.Translations").
		Preload("Tags.Translations").
		Preload("Cast.Person.Translations").
		First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *VideoRepo) FindByID(id uint) (*model.Video, error) {
	var video model.Video
	err := r.db.Where("deleted_at IS NULL").First(&video, id).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

type VideoListParams struct {
	Page     int
	PerPage  int
	Status   string
	Category uint
	Query    string
	Lang     string
}

func (r *VideoRepo) List(params VideoListParams) ([]model.Video, int64, error) {
	var videos []model.Video
	var total int64

	q := r.db.Model(&model.Video{}).Where("deleted_at IS NULL")

	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.Category > 0 {
		q = q.Joins("JOIN video_categories ON video_categories.video_id = videos.id").
			Where("video_categories.category_id = ?", params.Category)
	}
	if params.Query != "" {
		q = q.Joins("JOIN video_translations ON video_translations.video_id = videos.id").
			Where("video_translations.title LIKE ?", "%"+params.Query+"%")
	}

	q.Count(&total)

	err := q.Preload("Translations").
		Preload("Thumbnails", "is_default = ?", true).
		Offset((params.Page - 1) * params.PerPage).
		Limit(params.PerPage).
		Order("videos.created_at DESC").
		Find(&videos).Error

	return videos, total, err
}

func (r *VideoRepo) ListPublic(params VideoListParams) ([]model.Video, int64, error) {
	var videos []model.Video
	var total int64

	q := r.db.Model(&model.Video{}).Where("deleted_at IS NULL AND is_public = ? AND status = ?", true, "ready")

	if params.Category > 0 {
		q = q.Joins("JOIN video_categories ON video_categories.video_id = videos.id").
			Where("video_categories.category_id = ?", params.Category)
	}
	if params.Query != "" {
		q = q.Joins("JOIN video_translations ON video_translations.video_id = videos.id").
			Where("video_translations.title LIKE ?", "%"+params.Query+"%")
	}

	q.Count(&total)

	err := q.Preload("Translations").
		Preload("Thumbnails", "is_default = ?", true).
		Preload("Categories.Translations").
		Preload("Tags.Translations").
		Offset((params.Page - 1) * params.PerPage).
		Limit(params.PerPage).
		Order("videos.sort_order DESC, videos.created_at DESC").
		Find(&videos).Error

	return videos, total, err
}

func (r *VideoRepo) FindPublicByUUID(uuid string) (*model.Video, error) {
	var video model.Video
	err := r.db.Where("uuid = ? AND deleted_at IS NULL AND is_public = ? AND status = ?", uuid, true, "ready").
		Preload("Translations").
		Preload("Variants", "status = ?", "ready").
		Preload("Thumbnails").
		Preload("Subtitles").
		Preload("Categories.Translations").
		Preload("Tags.Translations").
		First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *VideoRepo) Update(video *model.Video) error {
	return r.db.Omit("Translations", "Variants", "Thumbnails", "Subtitles", "Categories", "Tags", "Cast").Save(video).Error
}

func (r *VideoRepo) SoftDelete(uuid string) error {
	return r.db.Model(&model.Video{}).
		Where("uuid = ? AND deleted_at IS NULL", uuid).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

// HardDelete permanently removes a video and all related records.
func (r *VideoRepo) HardDelete(videoID uint) error {
	tx := r.db.Begin()
	tx.Where("video_id = ?", videoID).Delete(&model.VideoTranslation{})
	tx.Where("video_id = ?", videoID).Delete(&model.VideoVariant{})
	tx.Where("video_id = ?", videoID).Delete(&model.Thumbnail{})
	tx.Where("video_id = ?", videoID).Delete(&model.Subtitle{})
	tx.Where("video_id = ?", videoID).Delete(&model.VideoCast{})
	// Clear many-to-many associations
	video := model.Video{ID: videoID}
	tx.Model(&video).Association("Categories").Clear()
	tx.Model(&video).Association("Tags").Clear()
	// Delete the video itself (hard delete)
	tx.Unscoped().Delete(&model.Video{}, videoID)
	return tx.Commit().Error
}

func (r *VideoRepo) Restore(uuid string) error {
	return r.db.Model(&model.Video{}).
		Where("uuid = ?", uuid).
		Update("deleted_at", nil).Error
}

func (r *VideoRepo) UpdateStatus(uuid, status string) error {
	return r.db.Model(&model.Video{}).
		Where("uuid = ?", uuid).
		Update("status", status).Error
}

// Translation operations

func (r *VideoRepo) UpsertTranslation(t *model.VideoTranslation) error {
	return r.db.Where("video_id = ? AND language_code = ?", t.VideoID, t.LanguageCode).
		Assign(t).FirstOrCreate(t).Error
}

func (r *VideoRepo) DeleteTranslation(videoID uint, lang string) error {
	return r.db.Where("video_id = ? AND language_code = ?", videoID, lang).
		Delete(&model.VideoTranslation{}).Error
}

func (r *VideoRepo) ListTranslations(videoID uint) ([]model.VideoTranslation, error) {
	var translations []model.VideoTranslation
	err := r.db.Where("video_id = ?", videoID).Find(&translations).Error
	return translations, err
}

// Variant operations

func (r *VideoRepo) CreateOrUpdateVariant(v *model.VideoVariant) error {
	return r.db.Where("video_id = ? AND resolution_name = ? AND codec = ?", v.VideoID, v.ResolutionName, v.Codec).
		Assign(v).FirstOrCreate(v).Error
}

func (r *VideoRepo) UpdateVariant(v *model.VideoVariant) error {
	return r.db.Save(v).Error
}

func (r *VideoRepo) ListVariants(videoID uint) ([]model.VideoVariant, error) {
	var variants []model.VideoVariant
	err := r.db.Where("video_id = ?", videoID).Find(&variants).Error
	return variants, err
}

func (r *VideoRepo) DeleteVariants(videoID uint) error {
	return r.db.Where("video_id = ?", videoID).Delete(&model.VideoVariant{}).Error
}

func (r *VideoRepo) FindVariantByResolution(videoID uint, resolution string) (*model.VideoVariant, error) {
	var v model.VideoVariant
	err := r.db.Where("video_id = ? AND resolution_name = ?", videoID, resolution).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepo) FindVariant(id uint) (*model.VideoVariant, error) {
	var v model.VideoVariant
	err := r.db.First(&v, id).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepo) DeleteVariant(id uint) error {
	return r.db.Delete(&model.VideoVariant{}, id).Error
}

// Thumbnail operations

func (r *VideoRepo) CreateThumbnail(t *model.Thumbnail) error {
	return r.db.Create(t).Error
}

func (r *VideoRepo) ListThumbnails(videoID uint) ([]model.Thumbnail, error) {
	var thumbs []model.Thumbnail
	err := r.db.Where("video_id = ?", videoID).Order("sort_order").Find(&thumbs).Error
	return thumbs, err
}

func (r *VideoRepo) SetDefaultThumbnail(videoID, thumbID uint) error {
	tx := r.db.Begin()
	tx.Model(&model.Thumbnail{}).Where("video_id = ?", videoID).Update("is_default", false)
	tx.Model(&model.Thumbnail{}).Where("id = ? AND video_id = ?", thumbID, videoID).Update("is_default", true)
	return tx.Commit().Error
}

func (r *VideoRepo) DeleteThumbnail(id uint) error {
	return r.db.Delete(&model.Thumbnail{}, id).Error
}

// Subtitle operations

func (r *VideoRepo) CreateSubtitle(s *model.Subtitle) error {
	return r.db.Create(s).Error
}

func (r *VideoRepo) ListSubtitles(videoID uint) ([]model.Subtitle, error) {
	var subs []model.Subtitle
	err := r.db.Where("video_id = ?", videoID).Order("sort_order").Find(&subs).Error
	return subs, err
}

func (r *VideoRepo) UpdateSubtitle(s *model.Subtitle) error {
	return r.db.Save(s).Error
}

func (r *VideoRepo) DeleteSubtitle(id uint) error {
	return r.db.Delete(&model.Subtitle{}, id).Error
}

func (r *VideoRepo) FindSubtitle(id uint) (*model.Subtitle, error) {
	var sub model.Subtitle
	err := r.db.First(&sub, id).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// Cast operations

func (r *VideoRepo) AddCast(c *model.VideoCast) error {
	return r.db.Create(c).Error
}

func (r *VideoRepo) ListCast(videoID uint) ([]model.VideoCast, error) {
	var cast []model.VideoCast
	err := r.db.Where("video_id = ?", videoID).
		Preload("Person.Translations").
		Order("sort_order").
		Find(&cast).Error
	return cast, err
}

func (r *VideoRepo) DeleteCast(id uint) error {
	return r.db.Delete(&model.VideoCast{}, id).Error
}

// Category/Tag association

func (r *VideoRepo) SetCategories(videoID uint, categoryIDs []uint) error {
	video := model.Video{ID: videoID}
	categories := make([]model.Category, len(categoryIDs))
	for i, id := range categoryIDs {
		categories[i] = model.Category{ID: id}
	}
	return r.db.Model(&video).Association("Categories").Replace(categories)
}

func (r *VideoRepo) SetTags(videoID uint, tagIDs []uint) error {
	video := model.Video{ID: videoID}
	tags := make([]model.Tag, len(tagIDs))
	for i, id := range tagIDs {
		tags[i] = model.Tag{ID: id}
	}
	return r.db.Model(&video).Association("Tags").Replace(tags)
}
