package service

import (
	"os"
	"strings"
)

func DataRootDir() string {
	if d := strings.TrimSpace(os.Getenv("KBARR_DATA_DIR")); d != "" {
		return d
	}
	return "data"
}
