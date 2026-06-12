// Package naming derives filesystem-safe names for media folders and files.
package naming

import (
	"regexp"
	"strconv"
	"strings"
)

// disallowed matches any run of characters that are not letters, digits, or
// spaces. Letters are matched in any script (\p{L}), so non-Latin titles such as
// Japanese are preserved — these are valid in Windows, NTFS, and POSIX paths.
var disallowed = regexp.MustCompile(`[^\p{L}\p{N} ]+`)

// multiSpace collapses runs of spaces left behind after stripping punctuation.
var multiSpace = regexp.MustCompile(` {2,}`)

// SanitizeFilename strips every character that is not a letter, digit, or space,
// then collapses repeated spaces and trims the ends. Returns an empty string if
// name contains no letters or digits (e.g. "...") — callers should supply a
// fallback in that case.
func SanitizeFilename(name string) string {
	cleaned := disallowed.ReplaceAllString(name, "")
	cleaned = multiSpace.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

var (
	seasonInTitleRe = regexp.MustCompile(`(?i)\bseason\s*(\d+)\b`)
	ordinalSeasonRe = regexp.MustCompile(`(?i)\b(\d+)(?:st|nd|rd|th)\s+season\b`)
)

// StripSeasonFromTitle removes trailing "Season N" and "Nth Season" phrases from
// a show title so that season-qualified titles match season-agnostic release names.
func StripSeasonFromTitle(title string) string {
	s := ordinalSeasonRe.ReplaceAllString(title, "")
	s = seasonInTitleRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// ExtractSeasonFromTitle returns the season number embedded in a show title
// (e.g. "Attack on Titan Season 3" → 3). Returns 0 if no season phrase is found.
func ExtractSeasonFromTitle(title string) int {
	if m := ordinalSeasonRe.FindStringSubmatch(title); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	if m := seasonInTitleRe.FindStringSubmatch(title); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}
