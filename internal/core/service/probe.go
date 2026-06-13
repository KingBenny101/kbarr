package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/parser"
)

// detectQualityAndSubs resolves the quality label and comma-separated subtitle
// languages for a downloaded episode file. ffprobe is the source of truth; when
// it is unavailable the quality falls back to parsing the filename (which kbarr
// stamps with the resolution at hardlink time) and subtitles to whatever sidecar
// files reveal.
func (c *AvailabilityChecker) detectQualityAndSubs(path string) (quality, subtitles string) {
	info, err := probeMedia(path)
	quality = info.Resolution
	if err != nil {
		slog.Debug("availability: ffprobe unavailable, falling back to name parse", "path", path, "error", err)
	}
	if quality == "" {
		quality = strings.ToLower(parser.Parse(filepath.Base(path)).ScreenSize)
	}
	subtitles = strings.Join(info.SubLangs, ",")
	return quality, subtitles
}

// MediaInfo is the subset of a media file's properties kbarr tracks: its video
// resolution as a quality label (e.g. "1080p") and the languages of every
// subtitle it carries (embedded tracks plus sidecar files), deduped.
type MediaInfo struct {
	Resolution string
	SubLangs   []string
}

// ffprobeOutput models the slice of ffprobe's `-show_streams` JSON we read.
type ffprobeOutput struct {
	Streams []struct {
		CodecType   string `json:"codec_type"`
		CodecName   string `json:"codec_name"`
		Height      int    `json:"height"`
		Disposition struct {
			AttachedPic int `json:"attached_pic"`
		} `json:"disposition"`
		Tags struct {
			Language string `json:"language"`
		} `json:"tags"`
	} `json:"streams"`
}

// imageCodecs are still-image codecs ffprobe reports as codec_type "video" when
// a release embeds cover art / a poster. Their dimensions must not be mistaken
// for the video resolution (e.g. a 2400px poster would otherwise read as "4k").
var imageCodecs = map[string]bool{"mjpeg": true, "png": true, "bmp": true, "gif": true, "webp": true}

// probeMedia inspects a single video file with ffprobe, returning its resolution
// label and the languages of any embedded subtitle tracks, merged with any
// sidecar subtitle files found next to it. It is intentionally called at most
// once per file (the availability tracker only probes monitors that have no
// quality/subtitle recorded yet), so the ffprobe cost is paid a single time.
//
// If ffprobe is not installed, probeMedia degrades gracefully: it returns an
// error for the caller to fall back on name-based parsing, but still merges in
// any sidecar subtitle languages it can detect without ffprobe.
func probeMedia(path string) (MediaInfo, error) {
	info := MediaInfo{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	langs := map[string]bool{}
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		// ffprobe missing or failed — still try sidecar files before giving up.
		for _, l := range sidecarSubLangs(path) {
			langs[l] = true
		}
		info.SubLangs = sortedKeys(langs)
		return info, err
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return info, err
	}

	maxHeight := 0
	for _, s := range parsed.Streams {
		switch s.CodecType {
		case "video":
			// Skip embedded cover art / posters: ffprobe tags them as video but
			// they carry an attached-picture disposition or a still-image codec,
			// and their dimensions are unrelated to the actual video resolution.
			if s.Disposition.AttachedPic == 1 || imageCodecs[strings.ToLower(s.CodecName)] {
				continue
			}
			if s.Height > maxHeight {
				maxHeight = s.Height
			}
		case "subtitle":
			if l := normalizeLang(s.Tags.Language); l != "" {
				langs[l] = true
			}
		}
	}
	info.Resolution = heightToQuality(maxHeight)

	for _, l := range sidecarSubLangs(path) {
		langs[l] = true
	}
	info.SubLangs = sortedKeys(langs)
	return info, nil
}

// subtitleExts are the sidecar subtitle file extensions kbarr recognises.
var subtitleExts = map[string]bool{".srt": true, ".ass": true, ".ssa": true, ".sub": true, ".vtt": true}

// sidecarSubLangs scans the directory of path for subtitle files that belong to
// the same video — e.g. "Show - S01E01.en.srt" next to "Show - S01E01.mkv" —
// and returns the languages encoded in their names. A subtitle file with no
// language segment (e.g. "Show - S01E01.srt") contributes "und" (undetermined).
func sidecarSubLangs(path string) []string {
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	langs := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !subtitleExts[ext] {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if stem == base {
			langs["und"] = true
			continue
		}
		// Expect "<videoBase>.<lang>" — strip the video base and read the suffix.
		if strings.HasPrefix(stem, base+".") {
			suffix := strings.TrimPrefix(stem, base+".")
			if l := normalizeLang(suffix); l != "" {
				langs[l] = true
			} else {
				langs["und"] = true
			}
		}
	}
	return sortedKeys(langs)
}

// heightToQuality maps a pixel height to the quality vocabulary used elsewhere
// (mirrors the indexer's qualityRank labels). Returns "" for unknown heights.
func heightToQuality(h int) string {
	switch {
	case h <= 0:
		return ""
	case h >= 2000:
		return "4k"
	case h >= 1000:
		return "1080p"
	case h >= 700:
		return "720p"
	case h >= 560:
		return "576p"
	case h >= 460:
		return "480p"
	default:
		return ""
	}
}

// normalizeLang lower-cases and trims a language tag, dropping clearly-bogus
// values ffprobe sometimes emits for untagged tracks.
func normalizeLang(s string) string {
	l := strings.ToLower(strings.TrimSpace(s))
	if l == "" || l == "und" || l == "unknown" {
		return ""
	}
	return l
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
