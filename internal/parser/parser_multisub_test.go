package parser

import (
	"slices"
	"testing"
)

func TestMultiSubDetection(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"MultiSub bracket", "[Erai-raws] Grand Blue S3 - 02 [1080p][MultiSub][CRC]", true},
		{"Multi-Subs bracket", "Show - 05 [1080p][Multi-Subs]", true},
		{"Multi Sub unbracketed", "Show S01E01 1080p WEB-DL Multi Sub", true},
		{"Multi-Audio and Multi-Subs", "[SubsPlease] Show - 01 [1080p][Multi-Audio][Multi-Subs]", true},
		{"Multiple Subtitles", "Show - 01 [1080p][Multiple Subtitles]", true},
		{"Multiple Sub", "Show - 01 [1080p][Multiple Sub]", true},
		{"Eng Sub only", "[Group] Title - 01 [1080p][Eng Sub]", false},
		{"Jap Sub only", "[Group] Title - 02 [1080p][Jap Sub]", false},
		{"plain audio description", "Multi-Audio Release - 01", false},
		{"no subtitle info", "Just a regular torrent name", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Parse(tt.filename)
			got := slices.Contains(r.SubtitleLangs, "multi")
			if got != tt.want {
				t.Errorf("Parse(%q).SubtitleLangs = %v, contains multi = %v, want %v", tt.filename, r.SubtitleLangs, got, tt.want)
			}
		})
	}
}
