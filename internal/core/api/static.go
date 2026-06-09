package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:frontend
var frontendFS embed.FS

func staticHandler() http.Handler {
	ui, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		return http.NotFoundHandler()
	}

	return &spaHandler{fs: ui}
}

// spaHandler implements http.Handler and serves a single page app.
// It serves files from the embedded FS, and falls back to index.html for client-side routing.
type spaHandler struct {
	fs fs.FS
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	fileServer := http.FileServer(http.FS(h.fs))
	path := r.URL.Path
	if path == "/" {
		path = "index.html"
	} else if path != "" && path[0] == '/' {
		path = path[1:]
	}

	if _, err := fs.Stat(h.fs, path); err != nil {
		r.URL.Path = "/"
	}

	fileServer.ServeHTTP(w, r)
}
