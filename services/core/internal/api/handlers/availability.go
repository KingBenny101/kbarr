package handlers

import (
	"net/http"

	coreservice "github.com/kingbenny101/kbarr/services/core/internal/service"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
)

func HandleCheckAvailability(w http.ResponseWriter, r *http.Request) {
	coreservice.CheckAvailability(r.Context(), db.DB)
	w.WriteHeader(http.StatusNoContent)
}
