package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/frontend"
	"github.com/allen/fishscale/internal/handler"
	appMiddleware "github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/storage"
)

func NewRouter(cfg *config.Config, db *sqlx.DB, store storage.Store) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Compress(5))

	if cfg.DevMode {
		r.Use(appMiddleware.DevAuth)
	}

	catches := handler.NewCatchHandler(db, store)
	trips := handler.NewTripHandler(db)
	species := handler.NewSpeciesHandler(db)
	photos := handler.NewPhotoHandler(db, store)
	settings := handler.NewSettingsHandler(db)
	weather := handler.NewWeatherHandler()
	stats := handler.NewStatsHandler(db)
	export := handler.NewExportHandler(db)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/catches", func(r chi.Router) {
			r.Get("/", catches.List)
			r.Post("/", catches.Create)
			r.Get("/{id}", catches.Get)
			r.Put("/{id}", catches.Update)
			r.Delete("/{id}", catches.Delete)
			r.Post("/{id}/photos", photos.Add)
		})
		r.Route("/trips", func(r chi.Router) {
			r.Get("/", trips.List)
			r.Post("/", trips.Create)
			r.Get("/{id}", trips.Get)
			r.Put("/{id}", trips.Update)
			r.Delete("/{id}", trips.Delete)
		})
		r.Route("/species", func(r chi.Router) {
			r.Get("/", species.List)
			r.Post("/", species.Create)
		})
		r.Delete("/photos/{id}", photos.Delete)
		r.Get("/settings", settings.Get)
		r.Put("/settings", settings.Update)
		r.Get("/weather", weather.Get)
		r.Get("/stats", stats.Get)
		r.Get("/export", export.Export)
	})

	// Serve photos from storage directory
	r.Get("/photos/*", http.StripPrefix("/photos/", http.FileServer(http.Dir(cfg.PhotoDir))).ServeHTTP)

	// Serve embedded SPA frontend with fallback to index.html
	distFS, err := fs.Sub(frontend.Assets, "dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(distFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Try to serve the exact file first
			if path != "/" && !strings.HasSuffix(path, "/") {
				if f, err := distFS.Open(strings.TrimPrefix(path, "/")); err == nil {
					f.Close()
					fileServer.ServeHTTP(w, r)
					return
				}
			}

			// SPA fallback: serve index.html for all non-file routes
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}
