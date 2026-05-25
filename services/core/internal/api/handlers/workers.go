package handlers

import (
	"encoding/json"
	"net/http"
)

func HandleGetWorkers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]any{})
}
