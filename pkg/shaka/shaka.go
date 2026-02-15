package shaka

import (
	"context"
	"fmt"
	"os/exec"
)

type Packager struct {
	path string
}

func New(path string) *Packager {
	return &Packager{path: path}
}

// PackageParams contains parameters for DRM packaging.
type PackageParams struct {
	InputPlaylist  string
	OutputDir      string
	KeyID          string // hex
	ContentKey     string // hex
	IV             string // hex (optional)
	WidevineEnabled bool
	FairPlayEnabled bool
	PSSH           string // base64 PSSH box for Widevine
	FairPlayURI    string // skd:// URI
}

// Package encrypts HLS content using CENC with Widevine and/or FairPlay.
func (p *Packager) Package(ctx context.Context, params PackageParams) error {
	// Build Shaka Packager command
	// Input stream descriptor
	inputDesc := fmt.Sprintf(
		"in=%s,stream=video,output=%s/encrypted_video.mp4,drm_label=HD",
		params.InputPlaylist, params.OutputDir,
	)

	args := []string{
		inputDesc,
		"--enable_raw_key_encryption",
		"--keys", fmt.Sprintf("label=HD:key_id=%s:key=%s", params.KeyID, params.ContentKey),
	}

	if params.IV != "" {
		args = append(args, "--iv", params.IV)
	}

	// Protection scheme
	args = append(args, "--protection_scheme", "cbcs")

	if params.WidevineEnabled && params.PSSH != "" {
		args = append(args, "--pssh", params.PSSH)
	}

	// HLS output
	args = append(args,
		"--hls_master_playlist_output", fmt.Sprintf("%s/master_encrypted.m3u8", params.OutputDir),
	)

	cmd := exec.CommandContext(ctx, p.path, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shaka packager: %w\noutput: %s", err, string(output))
	}

	return nil
}
