package handler

import (
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	svc *service.SettingService
}

func NewSettingHandler(svc *service.SettingService) *SettingHandler {
	return &SettingHandler{svc: svc}
}

func (h *SettingHandler) List(c *gin.Context) {
	settings, err := h.svc.List()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, settings)
}

type updateSettingInput struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

func (h *SettingHandler) Update(c *gin.Context) {
	var input updateSettingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.Update(input.Key, input.Value); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, nil)
}
