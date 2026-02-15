package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/middleware"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type VideoHandler struct {
	videoSvc *service.VideoService
}

func NewVideoHandler(videoSvc *service.VideoService) *VideoHandler {
	return &VideoHandler{videoSvc: videoSvc}
}

func (h *VideoHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	categoryID, _ := strconv.ParseUint(c.Query("category"), 10, 32)

	params := repository.VideoListParams{
		Page:     page,
		PerPage:  perPage,
		Status:   c.Query("status"),
		Category: uint(categoryID),
		Query:    c.Query("q"),
		Lang:     c.GetHeader("Accept-Language"),
	}

	videos, total, err := h.videoSvc.List(params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OKWithMeta(c, videos, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

func (h *VideoHandler) Get(c *gin.Context) {
	video, err := h.videoSvc.Get(c.Param("uuid"))
	if err != nil {
		response.NotFound(c, "video not found")
		return
	}
	response.OK(c, video)
}

func (h *VideoHandler) Create(c *gin.Context) {
	var input service.CreateVideoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID, _ := c.Get(middleware.ContextKeyUserID)
	video, err := h.videoSvc.Create(input, userID.(uint))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, video)
}

func (h *VideoHandler) Update(c *gin.Context) {
	var input service.UpdateVideoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	video, err := h.videoSvc.Update(c.Param("uuid"), input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, video)
}

func (h *VideoHandler) Delete(c *gin.Context) {
	if err := h.videoSvc.Delete(c.Param("uuid")); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

func (h *VideoHandler) Restore(c *gin.Context) {
	if err := h.videoSvc.Restore(c.Param("uuid")); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "restored"})
}

// Translations

func (h *VideoHandler) ListTranslations(c *gin.Context) {
	translations, err := h.videoSvc.ListTranslations(c.Param("uuid"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, translations)
}

func (h *VideoHandler) UpsertTranslation(c *gin.Context) {
	var input service.TranslationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.videoSvc.UpsertTranslation(c.Param("uuid"), c.Param("lang"), input); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "translation saved"})
}

func (h *VideoHandler) DeleteTranslation(c *gin.Context) {
	if err := h.videoSvc.DeleteTranslation(c.Param("uuid"), c.Param("lang")); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// Variants

func (h *VideoHandler) ListVariants(c *gin.Context) {
	variants, err := h.videoSvc.ListVariants(c.Param("uuid"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, variants)
}

// Thumbnails

func (h *VideoHandler) ListThumbnails(c *gin.Context) {
	thumbs, err := h.videoSvc.ListThumbnails(c.Param("uuid"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, thumbs)
}

func (h *VideoHandler) SetDefaultThumbnail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid thumbnail id")
		return
	}

	if err := h.videoSvc.SetDefaultThumbnail(c.Param("uuid"), uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "default thumbnail set"})
}

func (h *VideoHandler) DeleteThumbnail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid thumbnail id")
		return
	}

	if err := h.videoSvc.DeleteThumbnail(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// Subtitles

func (h *VideoHandler) ListSubtitles(c *gin.Context) {
	subs, err := h.videoSvc.ListSubtitles(c.Param("uuid"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, subs)
}

func (h *VideoHandler) DeleteSubtitle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid subtitle id")
		return
	}

	if err := h.videoSvc.DeleteSubtitle(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// Cast

func (h *VideoHandler) ListCast(c *gin.Context) {
	cast, err := h.videoSvc.ListCast(c.Param("uuid"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, cast)
}

func (h *VideoHandler) DeleteCast(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid cast id")
		return
	}

	if err := h.videoSvc.DeleteCast(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}
