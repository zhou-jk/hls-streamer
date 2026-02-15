package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type DRMHandler struct {
	drmSvc   *service.DRMService
	videoSvc *service.VideoService
}

func NewDRMHandler(drmSvc *service.DRMService, videoSvc *service.VideoService) *DRMHandler {
	return &DRMHandler{drmSvc: drmSvc, videoSvc: videoSvc}
}

// GenerateKeys generates DRM keys for a video.
func (h *DRMHandler) GenerateKeys(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoSvc.Get(uuid)
	if err != nil {
		response.NotFound(c, "video not found")
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host

	keys, err := h.drmSvc.GenerateKeys(video.ID, baseURL)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, keys)
}

// WidevineLicense handles Widevine license requests.
// In production, this would integrate with a proper Widevine license server.
func (h *DRMHandler) WidevineLicense(c *gin.Context) {
	// Widevine license protocol is binary (protobuf).
	// A full implementation requires the Widevine SDK.
	// This is a placeholder that returns the content key for development.
	var input struct {
		KeyID string `json:"key_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	key, err := h.drmSvc.GetKeyByID(input.KeyID)
	if err != nil {
		response.NotFound(c, "key not found")
		return
	}

	// In production: process the Widevine license request protobuf
	// and return a proper license response.
	response.OK(c, gin.H{
		"key_id":      key.KeyID,
		"drm_system":  key.DRMSystem,
		"license_url": key.LicenseURL,
	})
}

// FairPlayCertificate serves the FairPlay certificate.
func (h *DRMHandler) FairPlayCertificate(c *gin.Context) {
	// In production: serve the Apple-issued FairPlay certificate
	response.OK(c, gin.H{"message": "FairPlay certificate endpoint - configure fairplay_cert_path"})
}

// FairPlayLicense handles FairPlay license requests.
func (h *DRMHandler) FairPlayLicense(c *gin.Context) {
	var input struct {
		KeyID string `json:"key_id" binding:"required"`
		SPC   string `json:"spc"` // Server Playback Context (base64)
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	key, err := h.drmSvc.GetKeyByID(input.KeyID)
	if err != nil {
		response.NotFound(c, "key not found")
		return
	}

	// In production: process the SPC and return a CKC (Content Key Context)
	response.OK(c, gin.H{
		"key_id":     key.KeyID,
		"drm_system": key.DRMSystem,
	})
}
