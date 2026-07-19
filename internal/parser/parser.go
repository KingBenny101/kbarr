package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/nssteinbrenner/anitogo"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

type ParseResult struct {
	Title         string
	Season        int
	Episode       int
	ScreenSize    string
	VideoCodec    string
	ColorDepth    string
	Source        string
	ReleaseGroup  string
	Type          string
	SubtitleLangs []string // subtitle/language hints from the release name (lower-cased)
}

// multiSubRe catches multi-subtitle indicators that anitogo misses or
// normalizes to a single "multi" token so the subtitle language preference
// works with any common variant (MultiSub, Multi-Subs, Multiple Subtitles, …).
var multiSubRe = regexp.MustCompile(`(?i)\bmulti(?:[- ]?subs?|ple\s*sub(?:s|titles)?)\b`)

// episodeFallbackRe catches patterns anitogo misses:
//   - #1 / ＃1 (hash-prefixed, common in adult/doujin releases)
//   - 第1話 / 第01話 (Japanese episode notation)
var episodeFallbackRe = regexp.MustCompile(`[#＃](\d+)|第(\d+)話`)

// seasonEpisodeRe catches Western-style SxxExx markers. anitogo only recognises
// these when a title precedes them, so title-less scene names such as
// "S01E01-Deep Blue [crc].mkv" fall through with no episode. This fallback
// recovers both the season and episode from anywhere in the name.
var seasonEpisodeRe = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,3})`)

// normalizeWidth folds full-width Unicode characters to their ASCII equivalents
// before passing to anitogo, so patterns like ＃1 (U+FF03) are recognised as #1.
func normalizeWidth(s string) string {
	return norm.NFC.String(width.Narrow.String(s))
}

func Parse(filename string) ParseResult {
	if filename == "" {
		return ParseResult{}
	}
	e := anitogo.Parse(normalizeWidth(filename), anitogo.DefaultOptions)
	r := ParseResult{
		Title:        e.AnimeTitle,
		Season:       firstInt(e.AnimeSeason),
		Episode:      firstInt(e.EpisodeNumber),
		ScreenSize:   e.VideoResolution,
		VideoCodec:   firstStr(e.VideoTerm),
		Source:       firstStr(e.Source),
		ReleaseGroup: e.ReleaseGroup,
		Type:         "episode",
	}
	for _, t := range e.VideoTerm {
		if strings.Contains(strings.ToLower(t), "bit") {
			r.ColorDepth = t
			break
		}
	}
	for _, t := range e.AnimeType {
		if strings.EqualFold(t, "movie") {
			r.Type = "movie"
			break
		}
	}
	// Subtitle/language hints anitogo extracts from the name. These are only as
	// reliable as the release naming, but feed the soft subtitle-language
	// preference at download time (when the file does not yet exist to probe).
	seenLang := map[string]bool{}
	for _, l := range append(append([]string{}, e.Subtitles...), e.Language...) {
		l = strings.ToLower(strings.TrimSpace(l))
		if l != "" && !seenLang[l] {
			seenLang[l] = true
			r.SubtitleLangs = append(r.SubtitleLangs, l)
		}
	}
	// Normalise multi-subtitle indicators (MultiSub, Multi-Subs, Multiple Subtitles, …)
	// to "multi" for the subtitle language preference.
	if !seenLang["multi"] && (seenLang["multisub"] || multiSubRe.MatchString(filename)) {
		seenLang["multi"] = true
		r.SubtitleLangs = append(r.SubtitleLangs, "multi")
	}
	if r.Episode == 0 {
		if m := seasonEpisodeRe.FindStringSubmatch(filename); m != nil {
			r.Season, _ = strconv.Atoi(m[1])
			r.Episode, _ = strconv.Atoi(m[2])
			// A title-less scene name (e.g. "S01E01-Deep Blue") makes anitogo
			// treat the whole string as the title. Drop it so callers fall back
			// to folder/monitor context for the series name.
			if seasonEpisodeRe.MatchString(r.Title) {
				r.Title = ""
			}
		}
	}
	if r.Episode == 0 {
		if m := episodeFallbackRe.FindStringSubmatch(filename); m != nil {
			if m[1] != "" {
				r.Episode, _ = strconv.Atoi(m[1])
			} else if m[2] != "" {
				r.Episode, _ = strconv.Atoi(m[2])
			}
		}
	}
	return r
}

func firstInt(ss []string) int {
	if len(ss) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(ss[0])
	return n
}

func firstStr(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return ss[0]
}
