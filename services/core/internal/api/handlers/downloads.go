package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	downloaderpb "github.com/kingbenny101/kbarr/shared/proto/downloader"
)

func HandleAddTorrent(w http.ResponseWriter, r *http.Request, downloaderClient downloaderpb.DownloaderClient) {
	var request downloaderpb.AddTorrentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := downloaderClient.AddTorrent(ctx, &request)
	if err != nil {
		slog.Error("Downloader add torrent failed", "error", err)
		http.Error(w, "failed to add torrent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if response == nil {
		response = &downloaderpb.AddTorrentResponse{}
	}
	_ = json.NewEncoder(w).Encode(response)
}

func HandleListTorrents(w http.ResponseWriter, r *http.Request, downloaderClient downloaderpb.DownloaderClient) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := downloaderClient.ListTorrents(ctx, &downloaderpb.ListTorrentsRequest{Category: r.URL.Query().Get("category")})
	if err != nil {
		slog.Error("Downloader list torrents failed", "error", err)
		http.Error(w, "failed to list torrents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if response == nil {
		response = &downloaderpb.ListTorrentsResponse{Torrents: []*downloaderpb.TorrentResponse{}}
	}
	_ = json.NewEncoder(w).Encode(response)
}

func HandleGetTorrent(w http.ResponseWriter, r *http.Request, downloaderClient downloaderpb.DownloaderClient) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		http.Error(w, "missing torrent hash", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := downloaderClient.GetTorrent(ctx, &downloaderpb.TorrentRequest{Hash: hash})
	if err != nil {
		slog.Error("Downloader get torrent failed", "error", err)
		http.Error(w, "failed to get torrent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if response == nil {
		response = &downloaderpb.TorrentResponse{}
	}
	_ = json.NewEncoder(w).Encode(response)
}

func HandleRemoveTorrent(w http.ResponseWriter, r *http.Request, downloaderClient downloaderpb.DownloaderClient) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		http.Error(w, "missing torrent hash", http.StatusBadRequest)
		return
	}

	deleteFiles, err := strconv.ParseBool(r.URL.Query().Get("deleteFiles"))
	if err != nil && r.URL.Query().Get("deleteFiles") != "" {
		http.Error(w, "invalid deleteFiles value", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := downloaderClient.RemoveTorrent(ctx, &downloaderpb.RemoveTorrentRequest{Hash: hash, DeleteFiles: deleteFiles})
	if err != nil {
		slog.Error("Downloader remove torrent failed", "error", err)
		http.Error(w, "failed to remove torrent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if response == nil {
		response = &downloaderpb.RemoveTorrentResponse{}
	}
	_ = json.NewEncoder(w).Encode(response)
}
