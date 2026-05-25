package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/config"
)

func HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	settings, err := config.GetSettingsMap(db.DB)
	if err != nil {
		http.Error(w, "Failed to retrieve settings", http.StatusInternalServerError)
		return
	}

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
