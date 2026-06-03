package service

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type GuessitResult struct {
	Title        string
	Season       int
	Episode      int
	Type         string // "episode", "movie", etc.
	ReleaseGroup string
}

type rawGuessit struct {
	Title        string          `json:"title"`
	Season       json.RawMessage `json:"season"`
	Episode      json.RawMessage `json:"episode"`
	Type         string          `json:"type"`
	ReleaseGroup string          `json:"release_group"`
}

func guessitExe() string {
	exe, _ := os.Executable()
	candidates := []string{
		filepath.Join(filepath.Dir(exe), "guessit"),
		filepath.Join(filepath.Dir(exe), "internal", "service", "bin", "guessit"),
		"services/downloader/internal/service/bin/guessit",
		"internal/service/bin/guessit",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "guessit"
}

func runGuessit(filename string) GuessitResult {
	if filename == "" {
		return GuessitResult{}
	}
	out, err := exec.Command(guessitExe(), "-j", "-n", filename).Output()
	if err != nil {
		slog.Warn("guessit failed", "filename", filename, "error", err)
		return GuessitResult{}
	}
	var raw rawGuessit
	if err := json.Unmarshal(out, &raw); err != nil {
		return GuessitResult{}
	}
	return GuessitResult{
		Title:        raw.Title,
		Season:       intFromRaw(raw.Season),
		Episode:      intFromRaw(raw.Episode),
		Type:         raw.Type,
		ReleaseGroup: raw.ReleaseGroup,
	}
}

func intFromRaw(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	var arr []int
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0]
	}
	return 0
}
