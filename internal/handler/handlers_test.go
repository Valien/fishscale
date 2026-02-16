package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
	"github.com/allen/fishscale/internal/storage"
)

func setupFullRouter(t *testing.T) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec("INSERT INTO users (tailscale_id, display_name) VALUES ('dev-user', 'Dev User')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	store := storage.NewLocalStore(dir + "/photos")
	catches := NewCatchHandler(db, store)
	trips := NewTripHandler(db)
	species := NewSpeciesHandler(db)
	photos := NewPhotoHandler(db, store)
	settings := NewSettingsHandler(db)
	stats := NewStatsHandler(db)
	export := NewExportHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.DevAuth)
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
		r.Get("/stats", stats.Get)
		r.Get("/export", export.Export)
	})

	return r
}

func TestTrips(t *testing.T) {
	router := setupFullRouter(t)

	// Create trip
	body, _ := json.Marshal(map[string]string{"name": "Lake Fork Saturday", "started_at": "2026-02-16T08:00:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/trips", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create trip: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	// List trips
	req = httptest.NewRequest("GET", "/api/v1/trips", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list trips: got %d, want 200", rec.Code)
	}

	var trips []model.Trip
	json.NewDecoder(rec.Body).Decode(&trips)
	if len(trips) != 1 {
		t.Errorf("expected 1 trip, got %d", len(trips))
	}

	// Delete trip
	req = httptest.NewRequest("DELETE", "/api/v1/trips/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete trip: got %d, want 204", rec.Code)
	}
}

func TestSpecies(t *testing.T) {
	router := setupFullRouter(t)

	// List species (should have seed data)
	req := httptest.NewRequest("GET", "/api/v1/species", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list species: got %d, want 200", rec.Code)
	}

	var species []model.Species
	json.NewDecoder(rec.Body).Decode(&species)
	if len(species) == 0 {
		t.Error("expected seeded species")
	}

	// Search species
	req = httptest.NewRequest("GET", "/api/v1/species?q=Bass", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("search species: got %d, want 200", rec.Code)
	}

	var filtered []model.Species
	json.NewDecoder(rec.Body).Decode(&filtered)
	if len(filtered) == 0 {
		t.Error("expected at least one Bass species")
	}
	if len(filtered) >= len(species) {
		t.Error("expected filtered results to be fewer than all species")
	}

	// Create custom species
	body, _ := json.Marshal(map[string]string{"name": "Giant Trevally", "category": "saltwater"})
	req = httptest.NewRequest("POST", "/api/v1/species", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create species: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}
}

func TestSettings(t *testing.T) {
	router := setupFullRouter(t)

	// Get default settings
	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get settings: got %d, want 200", rec.Code)
	}

	var settings model.UserSettings
	json.NewDecoder(rec.Body).Decode(&settings)
	if settings.Theme != "system" {
		t.Errorf("expected default theme 'system', got %q", settings.Theme)
	}

	// Update settings
	body, _ := json.Marshal(map[string]string{"theme": "dark", "units": "metric"})
	req = httptest.NewRequest("PUT", "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update settings: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var updated model.UserSettings
	json.NewDecoder(rec.Body).Decode(&updated)
	if updated.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %q", updated.Theme)
	}
	if updated.Units != "metric" {
		t.Errorf("expected units 'metric', got %q", updated.Units)
	}
}

func TestStats(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch first
	catchBody, _ := json.Marshal(model.CreateCatchRequest{
		CaughtAt:   "2026-02-16T10:30:00Z",
		BaitOrLure: "Texas Rig",
	})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Get stats
	req = httptest.NewRequest("GET", "/api/v1/stats", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get stats: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var stats model.StatsResponse
	json.NewDecoder(rec.Body).Decode(&stats)
	if stats.TotalCatches != 1 {
		t.Errorf("expected 1 total catch, got %d", stats.TotalCatches)
	}
}

func TestExport(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch
	catchBody, _ := json.Marshal(model.CreateCatchRequest{
		CaughtAt: "2026-02-16T10:30:00Z",
	})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Export JSON
	req = httptest.NewRequest("GET", "/api/v1/export?format=json", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export json: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type application/json, got %q", ct)
	}

	// Export CSV
	req = httptest.NewRequest("GET", "/api/v1/export?format=csv", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export csv: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("expected content-type text/csv, got %q", ct)
	}
}
