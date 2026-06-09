package main

import (
	"encoding/json"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	coreapi "github.com/kingbenny101/kbarr/internal/core/api"
)

func main() {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("kbarr API", "1.0.0")
	cfg.Info.Description = "Self-hosted anime management API."
	api := humachi.New(r, cfg)
	api.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "token"},
	}

	// nil deps are safe: handler closures are registered for schema reflection but never invoked
	coreapi.RegisterRoutes(api, nil, nil, "dev")

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(api.OpenAPI()); err != nil {
		os.Stderr.WriteString("genspec: " + err.Error() + "\n")
		os.Exit(1)
	}
}
