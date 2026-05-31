package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kingbenny101/kbarr/services/metadata/internal/service"
)

type PrepareRequest struct {
	AID       uint   `json:"aid"`
	Title     string `json:"title"`
	LibraryID uint   `json:"library_id"`
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

func (h *Handler) Prepare(w http.ResponseWriter, r *http.Request) {
	var req PrepareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	detailed, err := h.svc.PrepareDetailed(req.AID, req.Title, req.LibraryID)
	if err != nil {
		slog.Error("Prepare failed", "aid", req.AID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detailed)
}
