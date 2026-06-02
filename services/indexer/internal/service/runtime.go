package service

import (
	"os"
	"path/filepath"
	"strings"
)

// guessitExe returns the path to the guessit binary.
// Checks several candidate locations and returns the first one that exists.
func guessitExe() string {
	exe, _ := os.Executable()
	candidates := []string{
		// Docker: binary copied next to the service binary
		filepath.Join(filepath.Dir(exe), "guessit"),
		// Local dev: binary committed under the service source tree
		filepath.Join(filepath.Dir(exe), "internal", "service", "bin", "guessit"),
		// Explicit path relative to working directory (air / go run)
		"services/indexer/internal/service/bin/guessit",
		"internal/service/bin/guessit",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Last resort: assume it's on PATH
	return "guessit"
}

func DataRootDir() string {
	if d := strings.TrimSpace(os.Getenv("KBARR_DATA_DIR")); d != "" {
		return d
	}
	return "data"
}
