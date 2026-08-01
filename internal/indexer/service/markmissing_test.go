package service

import "testing"

func TestCanMarkMissing(t *testing.T) {
	// The indexer must not mark a monitor missing when its file became
	// available while the search was in flight, or when the monitor is no
	// longer waiting on the indexer — otherwise it clobbers states set by the
	// availability scanner / downloader and the UI shows "missing" for
	// episodes that are on disk.
	cases := []struct {
		status    string
		available bool
		want      bool
	}{
		{"pending", false, true},
		{"searching", false, true},
		{"missing", false, true},
		{"downloaded", false, false},
		{"queued", false, false},
		{"downloading", false, false},
		{"unmonitored", false, false},
		{"pending", true, false},
		{"searching", true, false},
		{"missing", true, false},
	}
	for _, tc := range cases {
		if got := canMarkMissing(tc.status, tc.available); got != tc.want {
			t.Fatalf("canMarkMissing(%q, %v) = %v; want %v", tc.status, tc.available, got, tc.want)
		}
	}
}
