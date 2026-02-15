package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type CategoryHandler struct {
	categorySvc *service.CategoryService
}

func NewCategoryHandler(categorySvc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categorySvc: categorySvc}
}

// Categories

func (h *CategoryHandler) ListCategories(c *gin.Context) {
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		lang = "en"
	}

	categories, err := h.categorySvc.ListCategories(lang)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, categories)
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var input service.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	cat, err := h.categorySvc.CreateCategory(input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, cat)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid category id")
		return
	}

	var input service.UpdateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	cat, err := h.categorySvc.UpdateCategory(uint(id), input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, cat)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid category id")
		return
	}

	if err := h.categorySvc.DeleteCategory(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.NoContent(c)
}

func (h *CategoryHandler) UpsertCategoryTranslation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid category id")
		return
	}
	lang := c.Param("lang")

	var input struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.categorySvc.UpsertCategoryTranslation(uint(id), lang, input.Name, input.Description); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "translation saved"})
}

// Tags

func (h *CategoryHandler) ListTags(c *gin.Context) {
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		lang = "en"
	}

	tags, err := h.categorySvc.ListTags(lang)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, tags)
}

func (h *CategoryHandler) CreateTag(c *gin.Context) {
	var input service.CreateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tag, err := h.categorySvc.CreateTag(input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, tag)
}

func (h *CategoryHandler) DeleteTag(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid tag id")
		return
	}

	if err := h.categorySvc.DeleteTag(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.NoContent(c)
}
