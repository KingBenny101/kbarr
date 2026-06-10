// Package naming derives filesystem-safe names for media folders and files.
package naming

import (
	"regexp"
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
