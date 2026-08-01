package service

import "testing"

func TestParseEpisodeNumber(t *testing.T) {
	exts := map[string]bool{".mkv": true, ".mp4": true}

	cases := []struct {
		name    string
		file    string
		wantEp  int64
		wantOk  bool
	}{
		{"sxxexx fast path", "Grand Blue Dreaming Season 2 - S01E05.mkv", 5, true},
		{"sxxexx season ignored", "Show - S02E12.mkv", 12, true},
		{"absolute numbering via parser", "[Group] Some Show - 07 [1080p].mkv", 7, true},
		{"non-video skipped", "tvshow.nfo", 0, false},
		{"image skipped", "poster.jpg", 0, false},
		{"no episode", "Show - extras.mkv", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ep, ok := parseEpisodeNumber(tc.file, exts)
			if ok != tc.wantOk || ep != tc.wantEp {
				t.Fatalf("parseEpisodeNumber(%q) = (%d, %v); want (%d, %v)", tc.file, ep, ok, tc.wantEp, tc.wantOk)
			}
		})
	}
}

func TestEpisodePatternIgnoresSeason(t *testing.T) {
	// The realistic Grand Blue S2 case: a season-2 show whose file was stamped
	// S01 by the hardlinker must still resolve to episode 5 (not be season-gated).
	exts := map[string]bool{".mkv": true}
	ep, ok := parseEpisodeNumber("Grand Blue Dreaming Season 2 - S01E05.mkv", exts)
	if !ok || ep != 5 {
		t.Fatalf("got (%d, %v); want (5, true)", ep, ok)
	}
}

func TestStatusForFoundFile(t *testing.T) {
	// A monitor whose file is found on disk must have stale statuses promoted
	// to downloaded, otherwise the UI keeps showing "missing" even though the
	// episode is available. Statuses owned by other actors (an in-flight
	// download, an explicit unmonitor) are left untouched.
	cases := []struct {
		status string
		want   string
	}{
		{"missing", "downloaded"},
		{"pending", "downloaded"},
		{"searching", "downloaded"},
		{"monitored", "downloaded"},
		{"downloaded", "downloaded"},
		{"available", "available"},
		{"queued", "queued"},
		{"downloading", "downloading"},
		{"unmonitored", "unmonitored"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := statusForFoundFile(tc.status); got != tc.want {
			t.Fatalf("statusForFoundFile(%q) = %q; want %q", tc.status, got, tc.want)
		}
	}
}
