package handler

import (
	"encoding/hex"

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

	key, err := h.drmSvc.GenerateKeys(video.ID, baseURL)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, key)
}

// GetKeys returns the DRM key info for a video (including content key for admin).
func (h *DRMHandler) GetKeys(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoSvc.Get(uuid)
	if err != nil {
		response.NotFound(c, "video not found")
		return
	}

	key, err := h.drmSvc.GetKeyByVideo(video.ID)
	if err != nil {
		response.NotFound(c, "no DRM keys for this video")
		return
	}

	// Return full key info including content_key and iv (admin endpoint)
	response.OK(c, gin.H{
		"id":          key.ID,
		"video_id":    key.VideoID,
		"key_id":      key.KeyID,
		"content_key": key.ContentKey,
		"iv":          key.IV,
		"key_url":     key.KeyURL,
		"created_at":  key.CreatedAt,
	})
}

// RegenerateKeys deletes existing DRM keys and generates new ones.
func (h *DRMHandler) RegenerateKeys(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoSvc.Get(uuid)
	if err != nil {
		response.NotFound(c, "video not found")
		return
	}

	// Delete old keys
	_ = h.drmSvc.DeleteByVideoID(video.ID)

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host

	key, err := h.drmSvc.GenerateKeys(video.ID, baseURL)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"id":          key.ID,
		"video_id":    key.VideoID,
		"key_id":      key.KeyID,
		"content_key": key.ContentKey,
		"iv":          key.IV,
		"key_url":     key.KeyURL,
		"created_at":  key.CreatedAt,
	})
}

// ServeKey serves the raw 16-byte AES key for HLS AES-128 decryption.
// hls.js fetches this URI automatically when it encounters EXT-X-KEY:METHOD=AES-128.
func (h *DRMHandler) ServeKey(c *gin.Context) {
	keyID := c.Param("key_id")

	key, err := h.drmSvc.GetKeyByID(keyID)
	if err != nil {
		c.Status(404)
		return
	}

	// Decode hex content key to raw 16 bytes
	keyBytes, err := hex.DecodeString(key.ContentKey)
	if err != nil {
		c.Status(500)
		return
	}

	c.Data(200, "application/octet-stream", keyBytes)
}
