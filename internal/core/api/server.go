package api

import (
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kingbenny101/kbarr/internal/core/api/handlers"
	"github.com/kingbenny101/kbarr/internal/core/auth"
	"github.com/kingbenny101/kbarr/internal/core/clients"
)

func NewRouter(metadataClient *clients.MetadataClient, version string, authStore *auth.Store) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))
	r.Use(exemptMiddleware(authStore.Middleware, "/api/health", "/api/auth/login", "/api/images/"))

	// Images served as plain chi handler — http.ServeFile handles range/etag/304
	r.Get("/api/images/{imageName}", handlers.HandleGetImage)

	cfg := huma.DefaultConfig("kbarr API", "1.0.0")
	cfg.Info.Description = "Self-hosted anime management API."
	api := humachi.New(r, cfg)
	api.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "token"},
	}

	RegisterRoutes(api, metadataClient, authStore, version)

	r.NotFound(staticHandler().ServeHTTP)
	return r
}

func exemptMiddleware(mw func(http.Handler) http.Handler, prefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range prefixes {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			mw(next).ServeHTTP(w, r)
		})
	}
}
