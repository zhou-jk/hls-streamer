package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) *CategoryRepo {
	return &CategoryRepo{db: db}
}

// Category operations

func (r *CategoryRepo) CreateCategory(c *model.Category) error {
	return r.db.Create(c).Error
}

func (r *CategoryRepo) FindCategoryByID(id uint) (*model.Category, error) {
	var cat model.Category
	err := r.db.Preload("Translations").Preload("Children.Translations").First(&cat, id).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepo) ListCategories() ([]model.Category, error) {
	var cats []model.Category
	err := r.db.Where("parent_id IS NULL").
		Preload("Translations").
		Preload("Children.Translations").
		Order("sort_order").
		Find(&cats).Error
	return cats, err
}

func (r *CategoryRepo) UpdateCategory(c *model.Category) error {
	return r.db.Save(c).Error
}

func (r *CategoryRepo) DeleteCategory(id uint) error {
	return r.db.Delete(&model.Category{}, id).Error
}

func (r *CategoryRepo) UpsertCategoryTranslation(t *model.CategoryTranslation) error {
	return r.db.Where("category_id = ? AND language_code = ?", t.CategoryID, t.LanguageCode).
		Assign(t).FirstOrCreate(t).Error
}

// Tag operations

func (r *CategoryRepo) CreateTag(t *model.Tag) error {
	return r.db.Create(t).Error
}

func (r *CategoryRepo) ListTags() ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Translations").Find(&tags).Error
	return tags, err
}

func (r *CategoryRepo) DeleteTag(id uint) error {
	return r.db.Delete(&model.Tag{}, id).Error
}

func (r *CategoryRepo) UpdateTag(t *model.Tag) error {
	return r.db.Save(t).Error
}

func (r *CategoryRepo) UpsertTagTranslation(t *model.TagTranslation) error {
	return r.db.Where("tag_id = ? AND language_code = ?", t.TagID, t.LanguageCode).
		Assign(t).FirstOrCreate(t).Error
}

// Person operations

func (r *CategoryRepo) CreatePerson(p *model.Person) error {
	return r.db.Create(p).Error
}

func (r *CategoryRepo) FindPersonByID(id uint) (*model.Person, error) {
	var person model.Person
	err := r.db.Preload("Translations").First(&person, id).Error
	if err != nil {
		return nil, err
	}
	return &person, nil
}

func (r *CategoryRepo) ListPeople(page, perPage int) ([]model.Person, int64, error) {
	var people []model.Person
	var total int64

	r.db.Model(&model.Person{}).Count(&total)

	err := r.db.Preload("Translations").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&people).Error

	return people, total, err
}

func (r *CategoryRepo) UpdatePerson(p *model.Person) error {
	return r.db.Save(p).Error
}

func (r *CategoryRepo) UpsertPersonTranslation(t *model.PersonTranslation) error {
	return r.db.Where("person_id = ? AND language_code = ?", t.PersonID, t.LanguageCode).
		Assign(t).FirstOrCreate(t).Error
}
