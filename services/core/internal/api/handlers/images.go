package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func HandleGetImage(w http.ResponseWriter, r *http.Request) {
	imageName := chi.URLParam(r, "imageName")
	cleanName := filepath.Base(strings.TrimSpace(imageName))
	if cleanName == "" || cleanName == "." || cleanName != imageName {
		http.NotFound(w, r)
		return
	}

	dataDir := strings.TrimSpace(os.Getenv("KBARR_DATA_DIR"))
	if dataDir == "" {
		dataDir = "data"
	}

	http.ServeFile(w, r, filepath.Join(dataDir, "images", cleanName))
}
