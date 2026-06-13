package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/parser"
	"github.com/kingbenny101/kbarr/internal/subtitle"
)

// detectQualityAndSubs resolves the quality label and comma-separated subtitle
// languages for a downloaded episode file. ffprobe is the source of truth; when
// it is unavailable the quality falls back to parsing the filename (which kbarr
// stamps with the resolution at hardlink time) and subtitles to whatever sidecar
// files reveal.
func (c *AvailabilityChecker) detectQualityAndSubs(path string) (quality, subtitles string) {
	videoEpisode, _ := parseSubEpisode(filepath.Base(path))
	info, err := probeMedia(path, videoEpisode)
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
func probeMedia(path string, videoEpisode int64) (MediaInfo, error) {
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
		for _, l := range sidecarSubLangs(path, videoEpisode) {
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

	for _, l := range sidecarSubLangs(path, videoEpisode) {
		langs[l] = true
	}
	info.SubLangs = sortedKeys(langs)
	return info, nil
}

// sidecarSubLangs scans the directory of path for subtitle files that belong to
// the same video and returns their languages. It handles three layouts:
//   - the kbarr sidecar convention, "<videoBase>.<lang>.srt" next to the video;
//   - an unqualified sidecar, "<videoBase>.srt" (contributes "und");
//   - a manually-dropped release-named file that carries the same episode number
//     as the video (e.g. "Show_[crc].eng_Fansub.ass" next to "Show - S01E01.mkv"),
//     whose language is guessed from the filename.
//
// The episode fallback is what lets a subtitle a user moved in by hand — which
// almost never matches the renamed video stem — still be detected.
func sidecarSubLangs(path string, videoEpisode int64) []string {
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
		if !subtitle.IsSubtitle(name) {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		switch {
		case stem == base:
			langs["und"] = true
		case strings.HasPrefix(stem, base+"."):
			// "<videoBase>.<lang>[.n]" — the language is the first segment after the base.
			suffix := strings.TrimPrefix(stem, base+".")
			suffix = strings.SplitN(suffix, ".", 2)[0]
			if l := normalizeLang(suffix); l != "" {
				langs[l] = true
			} else {
				langs["und"] = true
			}
		default:
			// Not named after the video. Accept it only when its episode number
			// matches (so a folder of mixed episodes can't cross-contaminate), or
			// when the video has no episode number at all (a movie / single file).
			subEp, hasEp := parseSubEpisode(name)
			if videoEpisode == 0 || (hasEp && subEp == videoEpisode) {
				langs[subtitle.GuessLang(nil, name)] = true
			}
		}
	}
	return sortedKeys(langs)
}

// parseSubEpisode extracts an episode number from a subtitle filename, reusing
// the same SxxExx fast path / anitogo parser the video scanner uses.
func parseSubEpisode(name string) (int64, bool) {
	if m := episodePattern.FindStringSubmatch(name); m != nil {
		if n, err := strconv.ParseInt(m[1], 10, 64); err == nil && n > 0 {
			return n, true
		}
	}
	if ep := parser.Parse(name).Episode; ep > 0 {
		return int64(ep), true
	}
	return 0, false
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
