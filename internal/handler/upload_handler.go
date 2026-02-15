package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type UploadHandler struct {
	uploadSvc    *service.UploadService
	transcodeSvc *service.TranscodeService
}

func NewUploadHandler(uploadSvc *service.UploadService, transcodeSvc *service.TranscodeService) *UploadHandler {
	return &UploadHandler{uploadSvc: uploadSvc, transcodeSvc: transcodeSvc}
}

func (h *UploadHandler) Initiate(c *gin.Context) {
	var input struct {
		Filename    string `json:"filename" binding:"required"`
		ContentType string `json:"content_type" binding:"required"`
		PartCount   int    `json:"part_count" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.uploadSvc.InitiateUpload(c.Request.Context(), c.Param("uuid"), input.Filename, input.ContentType, input.PartCount)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, result)
}

func (h *UploadHandler) Complete(c *gin.Context) {
	var input service.CompleteUploadRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	videoUUID := c.Param("uuid")
	if err := h.uploadSvc.CompleteUpload(c.Request.Context(), videoUUID, input); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Trigger probe task
	task, err := h.transcodeSvc.StartProbe(c.Request.Context(), videoUUID)
	if err != nil {
		response.InternalError(c, "upload complete but probe failed: "+err.Error())
		return
	}

	response.OK(c, gin.H{
		"message":   "upload complete, probe started",
		"task_uuid": task.TaskUUID,
	})
}
