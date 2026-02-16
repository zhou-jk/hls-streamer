package handler

import (
	"encoding/base64"
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

// GetKeys returns the DRM key info for a video (without the content key).
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

	response.OK(c, key)
}

// ClearKeyLicense handles W3C ClearKey license requests.
// The player sends a JSON request with "kids" (key IDs in base64url),
// and we respond with the matching keys in the ClearKey format.
//
// Request:  {"kids": ["<base64url key_id>"], "type": "temporary"}
// Response: {"keys": [{"kty":"oct","kid":"<base64url>","k":"<base64url>"}], "type":"temporary"}
func (h *DRMHandler) ClearKeyLicense(c *gin.Context) {
	type clearKeyRequest struct {
		Kids []string `json:"kids"`
		Type string   `json:"type"`
	}
	var req clearKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid ClearKey request"})
		return
	}

	type clearKeyEntry struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		K   string `json:"k"`
	}
	type clearKeyResponse struct {
		Keys []clearKeyEntry `json:"keys"`
		Type string          `json:"type"`
	}

	resp := clearKeyResponse{Type: "temporary"}

	for _, kidB64 := range req.Kids {
		// Decode base64url kid to raw bytes, then to hex for DB lookup
		kidBytes, err := base64.RawURLEncoding.DecodeString(kidB64)
		if err != nil || len(kidBytes) != 16 {
			continue
		}
		kidHex := hex.EncodeToString(kidBytes)

		key, err := h.drmSvc.GetKeyByID(kidHex)
		if err != nil {
			continue
		}

		// Decode content key from hex to raw bytes, then base64url encode
		contentKeyBytes, err := hex.DecodeString(key.ContentKey)
		if err != nil {
			continue
		}

		resp.Keys = append(resp.Keys, clearKeyEntry{
			Kty: "oct",
			Kid: base64.RawURLEncoding.EncodeToString(kidBytes),
			K:   base64.RawURLEncoding.EncodeToString(contentKeyBytes),
		})
	}

	if len(resp.Keys) == 0 {
		c.JSON(404, gin.H{"error": "no matching keys found"})
		return
	}

	c.JSON(200, resp)
}
