package handler

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type UploadHandler struct {
	uploadSvc    *service.UploadService
	transcodeSvc *service.TranscodeService
	videoSvc     *service.VideoService
	s3           *storage.S3Client
}

func NewUploadHandler(uploadSvc *service.UploadService, transcodeSvc *service.TranscodeService, videoSvc *service.VideoService, s3 *storage.S3Client) *UploadHandler {
	return &UploadHandler{uploadSvc: uploadSvc, transcodeSvc: transcodeSvc, videoSvc: videoSvc, s3: s3}
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

// UploadSubtitle handles multipart form file upload for subtitles.
func (h *UploadHandler) UploadSubtitle(c *gin.Context) {
	videoUUID := c.Param("uuid")

	langCode := c.PostForm("language_code")
	label := c.PostForm("label")
	if langCode == "" || label == "" {
		response.BadRequest(c, "language_code and label are required")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	s3Key := fmt.Sprintf("videos/%s/subtitles/%s_%s", videoUUID, langCode, header.Filename)

	if err := h.s3.Upload(c.Request.Context(), s3Key, io.Reader(file), "text/vtt"); err != nil {
		response.InternalError(c, "failed to upload subtitle: "+err.Error())
		return
	}

	sub := &model.Subtitle{
		LanguageCode: langCode,
		Label:        label,
		S3Key:        s3Key,
	}
	if err := h.videoSvc.CreateSubtitle(videoUUID, sub); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, sub)
}
