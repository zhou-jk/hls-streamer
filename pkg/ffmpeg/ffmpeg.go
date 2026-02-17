package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type FFmpeg struct {
	ffmpegPath  string
	ffprobePath string
	threads     int
}

func New(ffmpegPath, ffprobePath string, threads int) *FFmpeg {
	return &FFmpeg{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		threads:     threads,
	}
}

// ProbeResult contains media file metadata from ffprobe.
type ProbeResult struct {
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Codec    string  `json:"codec"`
	FPS      float64 `json:"fps"`
	FileSize int64   `json:"file_size"`
	HasAudio bool    `json:"has_audio"`
}

// Probe runs ffprobe on the input file and returns metadata.
func (f *FFmpeg) Probe(ctx context.Context, input string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, f.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		input,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var probe struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	result := &ProbeResult{}

	if d, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		result.Duration = d
	}
	if s, err := strconv.ParseInt(probe.Format.Size, 10, 64); err == nil {
		result.FileSize = s
	}

	for _, s := range probe.Streams {
		if s.CodecType == "video" {
			result.Width = s.Width
			result.Height = s.Height
			result.Codec = s.CodecName
			result.FPS = parseFrameRate(s.RFrameRate)
		}
		if s.CodecType == "audio" {
			result.HasAudio = true
		}
	}

	return result, nil
}

// TranscodeHLSParams contains parameters for HLS transcoding.
type TranscodeHLSParams struct {
	Input           string
	OutputDir       string
	PlaylistName    string
	SegmentPattern  string
	Width           int
	Height          int
	VideoBitrate    int // kbps
	AudioBitrate    int // kbps
	Codec           string // h264, h265
	HasAudio        bool
	SegmentDuration int
}

// TranscodeToHLS transcodes the input to HLS format with the given parameters.
func (f *FFmpeg) TranscodeToHLS(ctx context.Context, p TranscodeHLSParams) error {
	codec := "libx264"
	if p.Codec == "h265" || p.Codec == "hevc" {
		codec = "libx265"
	}

	args := []string{
		"-i", p.Input,
		"-c:v", codec,
		"-b:v", fmt.Sprintf("%dk", p.VideoBitrate),
	}
	if p.HasAudio {
		args = append(args, "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", p.AudioBitrate))
	} else {
		args = append(args, "-an")
	}
	args = append(args,
		"-vf", fmt.Sprintf("scale=%d:%d", p.Width, p.Height),
		"-preset", "medium",
		"-g", fmt.Sprintf("%d", p.SegmentDuration*30), // keyframe interval
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", p.SegmentDuration),
		"-hls_list_size", "0",
		"-hls_segment_filename", fmt.Sprintf("%s/%s", p.OutputDir, p.SegmentPattern),
		"-hls_playlist_type", "vod",
		"-y",
	)

	if f.threads > 0 {
		args = append([]string{"-threads", fmt.Sprintf("%d", f.threads)}, args...)
	}

	args = append(args, fmt.Sprintf("%s/%s", p.OutputDir, p.PlaylistName))

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg transcode: %w\noutput: %s", err, string(output))
	}

	return nil
}

// ExtractThumbnails extracts thumbnail images at evenly spaced intervals.
func (f *FFmpeg) ExtractThumbnails(ctx context.Context, input, outputDir string, count, width, height int, duration float64) error {
	if count <= 0 || duration <= 0 {
		return fmt.Errorf("invalid count or duration")
	}

	interval := duration / float64(count+1)

	for i := 1; i <= count; i++ {
		timestamp := interval * float64(i)
		output := fmt.Sprintf("%s/thumb_%03d.jpg", outputDir, i)

		cmd := exec.CommandContext(ctx, f.ffmpegPath,
			"-ss", fmt.Sprintf("%.2f", timestamp),
			"-i", input,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale=%d:%d", width, height),
			"-q:v", "2",
			"-y",
			output,
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("extract thumbnail %d: %w\noutput: %s", i, err, string(output))
		}
	}

	return nil
}

// TranscodeEncryptedHLSParams contains parameters for AES-128 encrypted HLS transcoding.
type TranscodeEncryptedHLSParams struct {
	Input           string
	OutputDir       string
	PlaylistName    string
	SegmentPattern  string
	Width           int
	Height          int
	VideoBitrate    int // kbps
	AudioBitrate    int // kbps
	Codec           string
	HasAudio        bool
	SegmentDuration int
	KeyInfoFile     string // path to key_info file for -hls_key_info_file
}

// TranscodeToEncryptedHLS transcodes to AES-128 encrypted HLS using FFmpeg's native encryption.
// The KeyInfoFile format:
//
//	Line 1: Key URI (URL the player will fetch the raw key from)
//	Line 2: Key file path (local path to the 16-byte raw key file)
//	Line 3: IV in hex (32 hex chars)
func (f *FFmpeg) TranscodeToEncryptedHLS(ctx context.Context, p TranscodeEncryptedHLSParams) error {
	codec := "libx264"
	if p.Codec == "h265" || p.Codec == "hevc" {
		codec = "libx265"
	}

	args := []string{
		"-i", p.Input,
		"-c:v", codec,
		"-b:v", fmt.Sprintf("%dk", p.VideoBitrate),
	}
	if p.HasAudio {
		args = append(args, "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", p.AudioBitrate))
	} else {
		args = append(args, "-an")
	}
	args = append(args,
		"-vf", fmt.Sprintf("scale=%d:%d", p.Width, p.Height),
		"-preset", "medium",
		"-g", fmt.Sprintf("%d", p.SegmentDuration*30),
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", p.SegmentDuration),
		"-hls_list_size", "0",
		"-hls_segment_filename", fmt.Sprintf("%s/%s", p.OutputDir, p.SegmentPattern),
		"-hls_playlist_type", "vod",
		"-hls_key_info_file", p.KeyInfoFile,
		"-y",
	)

	if f.threads > 0 {
		args = append([]string{"-threads", fmt.Sprintf("%d", f.threads)}, args...)
	}

	args = append(args, fmt.Sprintf("%s/%s", p.OutputDir, p.PlaylistName))

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg transcode encrypted hls: %w\noutput: %s", err, string(output))
	}

	return nil
}

func parseFrameRate(rate string) float64 {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return 0
	}
	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	return num / den
}
