package service

import "testing"

func TestMagnetHash(t *testing.T) {
	cases := map[string]string{
		"magnet:?xt=urn:btih:ABCDEF0123456789&dn=Some.Name": "ABCDEF0123456789",
		"magnet:?dn=Name&xt=urn:btih:deadbeef&tr=udp://x":   "deadbeef",
		"magnet:?xt=urn:btih:cafef00d":                      "cafef00d",
		"https://indexer.example/file.torrent":              "",
		"magnet:?dn=NoHashHere":                             "",
	}
	for in, want := range cases {
		if got := magnetHash(in); got != want {
			t.Errorf("magnetHash(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormPath(t *testing.T) {
	cases := map[string]string{
		`/downloads/`:   "/downloads",
		`/downloads`:    "/downloads",
		`C:\Downloads\`: "C:/Downloads",
		`C:\Downloads`:  "C:/Downloads",
		`/a/b/c/`:       "/a/b/c",
	}
	for in, want := range cases {
		if got := normPath(in); got != want {
			t.Errorf("normPath(%q) = %q, want %q", in, got, want)
		}
	}
}
