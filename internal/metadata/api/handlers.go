package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kingbenny101/kbarr/internal/metadata/service"
)

type PrepareRequest struct {
	Source    string `json:"source"`
	SourceID  string `json:"source_id"`
	Title     string `json:"title"`
	LibraryID uint   `json:"library_id"`
	// Force bypasses the per-aid details cache, used when refreshing an airing show.
	Force bool `json:"force"`
}

type Handler struct {
	svc *service.AniDBService
}

func NewHandler(svc *service.AniDBService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	results, err := h.svc.SearchTitles(query)
	if err != nil {
		slog.Error("Search failed", "error", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func (h *Handler) GetAnimeDetails(w http.ResponseWriter, r *http.Request) {
	aidStr := r.PathValue("aid")
	aid, err := strconv.ParseUint(aidStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid aid", http.StatusBadRequest)
		return
	}

	details, err := h.svc.GetAnimeDetails(uint(aid))
	if err != nil {
		slog.Error("GetAnimeDetails failed", "aid", aid, "error", err)
		http.Error(w, "failed to get anime details", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(details)
}

func (h *Handler) ResolveAniList(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("anilistID")
	anilistID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid anilistID", http.StatusBadRequest)
		return
	}

	// Optional title hints (repeated ?title=) drive the fuzzy fallback when no
	// direct Fribb mapping exists.
	titles := r.URL.Query()["title"]

	aid, matched := h.svc.ResolveAniList(anilistID, titles)
	if matched == "" {
		http.Error(w, "no AniDB match for AniList ID", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"aid": aid, "matched": matched})
}

func (h *Handler) Prepare(w http.ResponseWriter, r *http.Request) {
	var req PrepareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	aid, err := strconv.ParseUint(req.SourceID, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid source_id %q: must be a numeric AniDB ID", req.SourceID), http.StatusBadRequest)
		return
	}

	metadata, err := h.svc.PrepareDetailed(uint(aid), req.Title, req.LibraryID, req.Force)
	if err != nil {
		slog.Error("Prepare failed", "source", req.Source, "source_id", req.SourceID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}
