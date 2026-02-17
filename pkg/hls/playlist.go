package hls

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// RewritePlaylist reads an M3U8 playlist and rewrites segment references
// to use the specified extension (e.g., ".jpeg" instead of ".ts").
func RewritePlaylist(r io.Reader, newExt string) (string, error) {
	var b strings.Builder
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// Rewrite segment file references (non-comment, non-empty lines that look like filenames)
		if !strings.HasPrefix(line, "#") && line != "" && isSegmentLine(line) {
			ext := filepath.Ext(line)
			if ext == ".ts" || ext == ".m4s" {
				line = strings.TrimSuffix(line, ext) + newExt
			}
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan playlist: %w", err)
	}

	return b.String(), nil
}

// GenerateMasterPlaylist creates a master M3U8 playlist from variant info.
func GenerateMasterPlaylist(variants []VariantInfo) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")

	for _, v := range variants {
		b.WriteString(fmt.Sprintf(
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=\"%s\"\n",
			v.Bandwidth, v.Width, v.Height, v.Name,
		))
		b.WriteString(v.PlaylistPath)
		b.WriteString("\n")
	}

	return b.String()
}

type VariantInfo struct {
	Name         string
	Bandwidth    int
	Width        int
	Height       int
	PlaylistPath string
}

func isSegmentLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.HasSuffix(lower, ".ts") ||
		strings.HasSuffix(lower, ".m4s") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".jpg") ||
		strings.Contains(lower, "segment")
}
