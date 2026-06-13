// Package subtitle holds the logic shared by the downloader (which links subtitle
// files into the media library) and the core availability checker (which detects
// the languages of subtitles already on disk). Releases store subtitles in wildly
// inconsistent ways — sidecar files named after the video, language subfolders
// (ENG/, FRE/, CHI/…), nested folders (RUS/SUB/), or filename segments like
// ".eng_SubDESU-H" — so the language guesser here is deliberately lenient.
package subtitle

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Exts is the set of sidecar subtitle file extensions kbarr recognises.
var Exts = map[string]bool{".srt": true, ".ass": true, ".ssa": true, ".sub": true, ".vtt": true, ".idx": true}

// IsSubtitle reports whether a filename has a recognised subtitle extension.
func IsSubtitle(name string) bool {
	return Exts[strings.ToLower(filepath.Ext(name))]
}

// canonical maps every alias we might see — ISO 639-1 (2-letter), ISO 639-2
// (3-letter), and the English language name — to the 3-letter tag kbarr uses
// elsewhere (matching the preferredSubtitleLanguage options and ffprobe-derived
// tags). Two-letter codes are only trusted when they appear as a whole path
// segment (a language subfolder); inside filenames they cause too many false
// positives ("it", "de", "is"), so guessFromTokens ignores entries shorter than
// three characters.
var canonical = map[string]string{
	"en": "eng", "eng": "eng", "english": "eng",
	"ja": "jpn", "jp": "jpn", "jpn": "jpn", "japanese": "jpn",
	"zh": "chi", "chi": "chi", "zho": "chi", "chinese": "chi", "chs": "chi", "cht": "chi",
	"ko": "kor", "kor": "kor", "korean": "kor",
	"fr": "fre", "fra": "fre", "fre": "fre", "french": "fre",
	"de": "ger", "deu": "ger", "ger": "ger", "german": "ger",
	"es": "spa", "spa": "spa", "spanish": "spa",
	"pt": "por", "por": "por", "portuguese": "por",
	"ru": "rus", "rus": "rus", "russian": "rus",
	"ar": "ara", "ara": "ara", "arabic": "ara",
	"it": "ita", "ita": "ita", "italian": "ita",
}

// tokenSep splits a filename or path segment into candidate words on any run of
// characters that are not letters or digits.
var tokenSep = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// GuessLang infers a 3-letter subtitle language tag from where a subtitle file
// lives and what it is named. dirSegments are the path components between the
// scan root and the file (e.g. ["ENG"] or ["RUS", "SUB"]); filename is the base
// name. Directory segments are the most reliable signal for the subfolder
// convention, so they are tried first (and may match 2-letter codes). Filename
// tokens are tried second and only match codes/names of three or more characters.
// Returns "und" when nothing recognisable is found.
func GuessLang(dirSegments []string, filename string) string {
	for _, seg := range dirSegments {
		if tag, ok := canonical[strings.ToLower(strings.TrimSpace(seg))]; ok {
			return tag
		}
	}
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	for _, tok := range tokenSep.Split(stem, -1) {
		t := strings.ToLower(tok)
		if len(t) < 3 {
			continue
		}
		if tag, ok := canonical[t]; ok {
			return tag
		}
	}
	return "und"
}
