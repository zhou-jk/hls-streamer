package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type WorkerHandler struct {
	workerSvc    *service.WorkerService
	transcodeSvc *service.TranscodeService
}

func NewWorkerHandler(workerSvc *service.WorkerService, transcodeSvc *service.TranscodeService) *WorkerHandler {
	return &WorkerHandler{workerSvc: workerSvc, transcodeSvc: transcodeSvc}
}

func (h *WorkerHandler) Register(c *gin.Context) {
	var input service.RegisterWorkerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.workerSvc.Register(input); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"message": "worker registered"})
}

func (h *WorkerHandler) Heartbeat(c *gin.Context) {
	var input struct {
		WorkerID string `json:"worker_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.workerSvc.Heartbeat(input.WorkerID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "ok"})
}

func (h *WorkerHandler) ReportProgress(c *gin.Context) {
	taskUUID := c.Param("task_uuid")

	var input struct {
		WorkerID string `json:"worker_id" binding:"required"`
		Progress uint8  `json:"progress" binding:"max=100"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	status, err := h.transcodeSvc.UpdateProgress(taskUUID, input.Progress)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "progress updated", "status": status})
}

func (h *WorkerHandler) CompleteTask(c *gin.Context) {
	taskUUID := c.Param("task_uuid")

	var input struct {
		WorkerID string    `json:"worker_id" binding:"required"`
		Result   model.JSON `json:"result"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.transcodeSvc.CompleteTask(taskUUID, input.Result); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	if err := h.workerSvc.SetOnline(input.WorkerID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "task completed"})
}

func (h *WorkerHandler) FailTask(c *gin.Context) {
	taskUUID := c.Param("task_uuid")

	var input struct {
		WorkerID string `json:"worker_id" binding:"required"`
		Error    string `json:"error" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.transcodeSvc.FailTask(taskUUID, input.Error); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	if err := h.workerSvc.SetOnline(input.WorkerID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "task marked as failed"})
}

func (h *WorkerHandler) ListWorkers(c *gin.Context) {
	workers, err := h.workerSvc.List()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, workers)
}
