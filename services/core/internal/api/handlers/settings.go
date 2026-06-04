package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/config"
)

func HandleTestKbdex(indexerAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, err := http.Post(indexerAddr+"/test/kbdex", "application/json", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func HandleTestIndexer(indexerAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, err := http.Post(indexerAddr+"/test", "application/json", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func HandleTestDownloader(downloaderAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, err := http.Post(downloaderAddr+"/test", "application/json", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func HandleTriggerDownloader(downloaderAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, err := http.Post(downloaderAddr+"/trigger", "application/json", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	settings, err := config.GetSettingsMap(db.DB)
	if err != nil {
		http.Error(w, "Failed to retrieve settings", http.StatusInternalServerError)
		return
	}
	delete(settings, "authPasswordHash")
	delete(settings, "authUsername")

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		http.Error(w, "Failed to encode settings", http.StatusInternalServerError)
		return
	}

	w.Write(settingsJSON)
}

func HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var updatedSettings map[string]string
	err := json.NewDecoder(r.Body).Decode(&updatedSettings)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for key, value := range updatedSettings {
		err := config.SetSetting(db.DB, key, value)
		if err != nil {
			http.Error(w, "Failed to update settings", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)

	res := make(map[string]string)
	res["success"] = "true"
	res["message"] = "Settings updated successfully"

	resJSON, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Write(resJSON)
}
