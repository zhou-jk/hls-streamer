package service

import (
	"strconv"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type CategoryService struct {
	repo *repository.CategoryRepo
}

func NewCategoryService(repo *repository.CategoryRepo) *CategoryService {
	return &CategoryService{repo: repo}
}

// Input types

type CreateCategoryInput struct {
	Slug     string `json:"slug"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateCategoryInput struct {
	Slug     string `json:"slug"`
	ParentID *uint  `json:"parent_id"`
	IsActive *bool  `json:"is_active"`
}

type CreateTagInput struct {
	Slug string `json:"slug"`
}

// Category

func (s *CategoryService) CreateCategory(input CreateCategoryInput) (*model.Category, error) {
	cat := &model.Category{Slug: "tmp", ParentID: input.ParentID, IsActive: true}
	if err := s.repo.CreateCategory(cat); err != nil {
		return nil, err
	}
	if input.Slug == "" {
		cat.Slug = strconv.FormatUint(uint64(cat.ID), 10)
	} else {
		cat.Slug = input.Slug
	}
	if err := s.repo.UpdateCategory(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *CategoryService) GetCategory(id uint) (*model.Category, error) {
	return s.repo.FindCategoryByID(id)
}

func (s *CategoryService) ListCategories(lang string) ([]model.Category, error) {
	return s.repo.ListCategories()
}

func (s *CategoryService) UpdateCategory(id uint, input UpdateCategoryInput) (*model.Category, error) {
	cat, err := s.repo.FindCategoryByID(id)
	if err != nil {
		return nil, err
	}
	if input.Slug != "" {
		cat.Slug = input.Slug
	}
	if input.ParentID != nil {
		cat.ParentID = input.ParentID
	}
	if input.IsActive != nil {
		cat.IsActive = *input.IsActive
	}
	if err := s.repo.UpdateCategory(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *CategoryService) DeleteCategory(id uint) error {
	return s.repo.DeleteCategory(id)
}

func (s *CategoryService) UpsertCategoryTranslation(categoryID uint, lang, name, description string) error {
	t := &model.CategoryTranslation{
		CategoryID:   categoryID,
		LanguageCode: lang,
		Name:         name,
		Description:  description,
	}
	return s.repo.UpsertCategoryTranslation(t)
}

// Tag

func (s *CategoryService) CreateTag(input CreateTagInput) (*model.Tag, error) {
	tag := &model.Tag{Slug: "tmp"}
	if err := s.repo.CreateTag(tag); err != nil {
		return nil, err
	}
	if input.Slug == "" {
		tag.Slug = strconv.FormatUint(uint64(tag.ID), 10)
	} else {
		tag.Slug = input.Slug
	}
	if err := s.repo.UpdateTag(tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *CategoryService) ListTags(lang string) ([]model.Tag, error) {
	return s.repo.ListTags()
}

func (s *CategoryService) DeleteTag(id uint) error {
	return s.repo.DeleteTag(id)
}

// Person

func (s *CategoryService) CreatePerson(slug string) (*model.Person, error) {
	p := &model.Person{Slug: slug}
	if err := s.repo.CreatePerson(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *CategoryService) GetPerson(id uint) (*model.Person, error) {
	return s.repo.FindPersonByID(id)
}

func (s *CategoryService) ListPeople(page, perPage int) ([]model.Person, int64, error) {
	return s.repo.ListPeople(page, perPage)
}

func (s *CategoryService) UpsertPersonTranslation(personID uint, lang, name, biography string) error {
	t := &model.PersonTranslation{
		PersonID:     personID,
		LanguageCode: lang,
		Name:         name,
		Biography:    biography,
	}
	return s.repo.UpsertPersonTranslation(t)
}
