package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/repository"
	"github.com/Zhou-JK/hls-streamer/internal/storage"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type PlaybackHandler struct {
	videoRepo *repository.VideoRepo
	s3        *storage.S3Client
}

func NewPlaybackHandler(videoRepo *repository.VideoRepo, s3 *storage.S3Client) *PlaybackHandler {
	return &PlaybackHandler{videoRepo: videoRepo, s3: s3}
}

// redirectToS3 redirects to a public URL or presigned URL depending on config.
func (h *PlaybackHandler) redirectToS3(c *gin.Context, s3Key string) {
	if h.s3.IsPublicRead() {
		c.Redirect(http.StatusTemporaryRedirect, h.s3.PublicURL(s3Key))
		return
	}
	url, err := h.s3.PresignGetObject(c.Request.Context(), s3Key, 1*time.Hour)
	if err != nil {
		response.NotFound(c, "object not found")
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// MasterPlaylist serves the master M3U8 playlist for a video.
func (h *PlaybackHandler) MasterPlaylist(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoRepo.FindByUUID(uuid)
	if err != nil || video.MasterPlaylistKey == nil {
		response.NotFound(c, "video or playlist not found")
		return
	}
	h.redirectToS3(c, *video.MasterPlaylistKey)
}

// VariantPlaylist serves a variant M3U8 playlist.
func (h *PlaybackHandler) VariantPlaylist(c *gin.Context) {
	uuid := c.Param("uuid")
	variant := c.Param("variant")
	h.redirectToS3(c, fmt.Sprintf("videos/%s/variants/%s/playlist.m3u8", uuid, variant))
}

// Segment serves a video segment (disguised as .jpeg).
func (h *PlaybackHandler) Segment(c *gin.Context) {
	uuid := c.Param("uuid")
	variant := c.Param("variant")
	segment := c.Param("segment")
	h.redirectToS3(c, fmt.Sprintf("videos/%s/variants/%s/%s", uuid, variant, segment))
}

// SubtitleFile serves a WebVTT subtitle file.
func (h *PlaybackHandler) SubtitleFile(c *gin.Context) {
	uuid := c.Param("uuid")
	lang := c.Param("lang")
	h.redirectToS3(c, fmt.Sprintf("videos/%s/subtitles/%s.vtt", uuid, lang))
}

// ThumbnailImage serves a thumbnail image.
func (h *PlaybackHandler) ThumbnailImage(c *gin.Context) {
	uuid := c.Param("uuid")
	filename := c.Param("filename")
	h.redirectToS3(c, fmt.Sprintf("videos/%s/thumbnails/%s", uuid, filename))
}

// DownloadOriginal redirects to a presigned URL for downloading the original video file.
func (h *PlaybackHandler) DownloadOriginal(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoRepo.FindByUUID(uuid)
	if err != nil || video.OriginalS3Key == "" {
		response.NotFound(c, "video not found")
		return
	}
	h.redirectToS3(c, video.OriginalS3Key)
}
