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

// MasterPlaylist serves the master M3U8 playlist for a video.
func (h *PlaybackHandler) MasterPlaylist(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoRepo.FindByUUID(uuid)
	if err != nil || video.MasterPlaylistKey == nil {
		response.NotFound(c, "video or playlist not found")
		return
	}

	url, err := h.s3.PresignGetObject(c.Request.Context(), *video.MasterPlaylistKey, 1*time.Hour)
	if err != nil {
		response.InternalError(c, "failed to generate playlist URL")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// VariantPlaylist serves a variant M3U8 playlist.
func (h *PlaybackHandler) VariantPlaylist(c *gin.Context) {
	uuid := c.Param("uuid")
	variant := c.Param("variant")

	s3Key := fmt.Sprintf("videos/%s/variants/%s/playlist.m3u8", uuid, variant)
	url, err := h.s3.PresignGetObject(c.Request.Context(), s3Key, 1*time.Hour)
	if err != nil {
		response.NotFound(c, "variant playlist not found")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Segment serves a video segment (disguised as .jpeg).
func (h *PlaybackHandler) Segment(c *gin.Context) {
	uuid := c.Param("uuid")
	variant := c.Param("variant")
	segment := c.Param("segment")

	s3Key := fmt.Sprintf("videos/%s/variants/%s/%s", uuid, variant, segment)
	url, err := h.s3.PresignGetObject(c.Request.Context(), s3Key, 1*time.Hour)
	if err != nil {
		response.NotFound(c, "segment not found")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// SubtitleFile serves a WebVTT subtitle file.
func (h *PlaybackHandler) SubtitleFile(c *gin.Context) {
	uuid := c.Param("uuid")
	lang := c.Param("lang")

	s3Key := fmt.Sprintf("videos/%s/subtitles/%s.vtt", uuid, lang)
	url, err := h.s3.PresignGetObject(c.Request.Context(), s3Key, 1*time.Hour)
	if err != nil {
		response.NotFound(c, "subtitle not found")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// ThumbnailImage serves a thumbnail image.
func (h *PlaybackHandler) ThumbnailImage(c *gin.Context) {
	uuid := c.Param("uuid")
	filename := c.Param("filename")

	s3Key := fmt.Sprintf("videos/%s/thumbnails/%s", uuid, filename)
	url, err := h.s3.PresignGetObject(c.Request.Context(), s3Key, 1*time.Hour)
	if err != nil {
		response.NotFound(c, "thumbnail not found")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// DownloadOriginal redirects to a presigned URL for downloading the original video file.
func (h *PlaybackHandler) DownloadOriginal(c *gin.Context) {
	uuid := c.Param("uuid")
	video, err := h.videoRepo.FindByUUID(uuid)
	if err != nil || video.OriginalS3Key == "" {
		response.NotFound(c, "video not found")
		return
	}

	url, err := h.s3.PresignGetObject(c.Request.Context(), video.OriginalS3Key, 1*time.Hour)
	if err != nil {
		response.InternalError(c, "failed to generate download URL")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}
