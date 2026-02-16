package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/repository"
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type TranscodeHandler struct {
	transcodeSvc *service.TranscodeService
}

func NewTranscodeHandler(transcodeSvc *service.TranscodeService) *TranscodeHandler {
	return &TranscodeHandler{transcodeSvc: transcodeSvc}
}

func (h *TranscodeHandler) StartTranscode(c *gin.Context) {
	uuid := c.Param("uuid")

	var input service.StartTranscodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Build license base URL for DRM key generation
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	licenseBaseURL := scheme + "://" + c.Request.Host

	tasks, err := h.transcodeSvc.StartTranscode(c.Request.Context(), uuid, input, licenseBaseURL)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, tasks)
}

func (h *TranscodeHandler) ListTasks(c *gin.Context) {
	uuid := c.Param("uuid")

	tasks, err := h.transcodeSvc.ListTasks(uuid)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, tasks)
}

func (h *TranscodeHandler) GetTask(c *gin.Context) {
	taskUUID := c.Param("task_uuid")

	task, err := h.transcodeSvc.GetTask(taskUUID)
	if err != nil {
		response.NotFound(c, "task not found")
		return
	}

	response.OK(c, task)
}

func (h *TranscodeHandler) GenerateThumbnails(c *gin.Context) {
	uuid := c.Param("uuid")

	var input struct {
		Count int `json:"count"`
		Width int `json:"width"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if input.Count == 0 {
		input.Count = 5
	}
	if input.Width == 0 {
		input.Width = 320
	}

	task, err := h.transcodeSvc.GenerateThumbnails(c.Request.Context(), uuid, input.Count, input.Width)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, task)
}

// CancelTask cancels a pending or queued task.
func (h *TranscodeHandler) CancelTask(c *gin.Context) {
	taskUUID := c.Param("task_uuid")

	if err := h.transcodeSvc.CancelTask(taskUUID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "task cancelled"})
}

// RetryTask retries a failed task.
func (h *TranscodeHandler) RetryTask(c *gin.Context) {
	taskUUID := c.Param("task_uuid")

	task, err := h.transcodeSvc.RetryTask(c.Request.Context(), taskUUID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, task)
}

// ListAllTasks returns a global paginated task list.
func (h *TranscodeHandler) ListAllTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	params := repository.TaskListParams{
		Page:    page,
		PerPage: perPage,
		Status:  c.Query("status"),
		Type:    c.Query("type"),
	}

	tasks, total, err := h.transcodeSvc.ListAllTasks(params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OKWithMeta(c, tasks, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}
