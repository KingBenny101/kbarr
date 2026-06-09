package parser

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/nssteinbrenner/anitogo"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

type ParseResult struct {
	Title        string
	Season       int
	Episode      int
	ScreenSize   string
	VideoCodec   string
	ColorDepth   string
	Source       string
	ReleaseGroup string
	Type         string
}

var memo sync.Map // filename → ParseResult

// episodeFallbackRe catches patterns anitogo misses:
//   - #1 / ＃1 (hash-prefixed, common in adult/doujin releases)
//   - 第1話 / 第01話 (Japanese episode notation)
var episodeFallbackRe = regexp.MustCompile(`[#＃](\d+)|第(\d+)話`)

// normalizeWidth folds full-width Unicode characters to their ASCII equivalents
// before passing to anitogo, so patterns like ＃1 (U+FF03) are recognised as #1.
func normalizeWidth(s string) string {
	return norm.NFC.String(width.Narrow.String(s))
}

func Parse(filename string) ParseResult {
	if filename == "" {
		return ParseResult{}
	}
	if v, ok := memo.Load(filename); ok {
		return v.(ParseResult)
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
	if r.Episode == 0 {
		if m := episodeFallbackRe.FindStringSubmatch(filename); m != nil {
			if m[1] != "" {
				r.Episode, _ = strconv.Atoi(m[1])
			} else if m[2] != "" {
				r.Episode, _ = strconv.Atoi(m[2])
			}
		}
	}
	memo.Store(filename, r)
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
