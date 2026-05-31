package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kingbenny101/kbarr/services/core/internal/auth"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/config"
)

func HandleLogin(store *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if !auth.ValidateCredentials(db.DB, req.Username, req.Password) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := store.Issue(req.Username)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

func HandleLogout(store *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.Revoke(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}
}

func HandleMe(store *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := store.UsernameFromRequest(r)
		if username == "" {
			username = config.Get(db.DB, "authUsername", "admin")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": username})
	}
}

func HandleChangeCredentials() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CurrentPassword string `json:"currentPassword"`
			NewUsername     string `json:"newUsername"`
			NewPassword     string `json:"newPassword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := auth.UpdateCredentials(db.DB, req.CurrentPassword, req.NewUsername, req.NewPassword); err != nil {
			if _, ok := err.(*auth.ErrUnauthorized); ok {
				http.Error(w, "Invalid current password", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Failed to update credentials", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
