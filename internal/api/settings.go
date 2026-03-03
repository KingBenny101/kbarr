package api

import (
	"encoding/json"
	"net/http"

	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/config"
)

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	

	settings, err := db.GetAllSettings() 
	if err != nil {
		http.Error(w, "Failed to retrieve settings", http.StatusInternalServerError)
		return
	}

	parsedSettings := make(map[string]string)

	for _, setting := range settings {
		parsedSettings[setting.Key] = setting.Value
	}

	settingsJSON, err := json.Marshal(parsedSettings)
	if err != nil {
		http.Error(w, "Failed to encode settings", http.StatusInternalServerError)
		return
	}

	
	w.Write(settingsJSON)
}

func handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var updatedSettings map[string]string
	err := json.NewDecoder(r.Body).Decode(&updatedSettings)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for key, value := range updatedSettings {
		err := db.SetSetting(key, value)
		if err != nil {
			http.Error(w, "Failed to update settings", http.StatusInternalServerError)
			return
		}
	}

	config.Reload()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Settings updated successfully"))
}