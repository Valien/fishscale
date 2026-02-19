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
	autocomplete := NewAutocompleteHandler(db)
	photos := NewPhotoHandler(db, store)
	settings := NewSettingsHandler(db)
	stats := NewStatsHandler(db)
	export := NewExportHandler(db)
	user := NewUserHandler()

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
		r.Get("/autocomplete/species", autocomplete.Species)
		r.Delete("/photos/{id}", photos.Delete)
		r.Get("/settings", settings.Get)
		r.Put("/settings", settings.Update)
		r.Get("/stats", stats.Get)
		r.Get("/export", export.Export)
		r.Get("/me", user.GetMe)
	})

	// Serve photos with ownership check (mirrors server.go)
	r.Get("/photos/*", photos.Serve)

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
	_ = json.NewDecoder(rec.Body).Decode(&trips)
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

func TestAutocompleteSpecies(t *testing.T) {
	router := setupFullRouter(t)

	// Initially, autocomplete should return empty array (no catches yet)
	req := httptest.NewRequest("GET", "/api/v1/autocomplete/species", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("autocomplete species: got %d, want 200", rec.Code)
	}

	var species []string
	_ = json.NewDecoder(rec.Body).Decode(&species)
	if len(species) != 0 {
		t.Errorf("expected empty species list for new user, got %d", len(species))
	}

	// Create some catches with species names
	for _, sp := range []string{"Largemouth Bass", "Largemouth Bass", "Bluegill"} {
		body, _ := json.Marshal(map[string]any{
			"caught_at":    "2026-02-16T10:30:00Z",
			"species_name": sp,
		})
		req = httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create catch: got %d, want 201", rec.Code)
		}
	}

	// Now autocomplete should return species sorted by frequency
	req = httptest.NewRequest("GET", "/api/v1/autocomplete/species", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("autocomplete species: got %d, want 200", rec.Code)
	}

	_ = json.NewDecoder(rec.Body).Decode(&species)
	if len(species) != 2 {
		t.Errorf("expected 2 species, got %d", len(species))
	}
	if len(species) > 0 && species[0] != "Largemouth Bass" {
		t.Errorf("expected 'Largemouth Bass' first (most frequent), got %q", species[0])
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
	_ = json.NewDecoder(rec.Body).Decode(&settings)
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
	_ = json.NewDecoder(rec.Body).Decode(&updated)
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
		CaughtAt:    "2026-02-16T10:30:00Z",
		SpeciesName: "Largemouth Bass",
		BaitOrLure:  "Texas Rig",
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
	_ = json.NewDecoder(rec.Body).Decode(&stats)
	if stats.TotalCatches != 1 {
		t.Errorf("expected 1 total catch, got %d", stats.TotalCatches)
	}
	if stats.TotalSpecies != 1 {
		t.Errorf("expected 1 total species, got %d", stats.TotalSpecies)
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

func TestGetMe(t *testing.T) {
	router := setupFullRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get me: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	// Verify basic user fields
	if resp["display_name"] != "Dev User" {
		t.Errorf("expected display_name 'Dev User', got %q", resp["display_name"])
	}
	if resp["tailscale_id"] != "dev-user" {
		t.Errorf("expected tailscale_id 'dev-user', got %q", resp["tailscale_id"])
	}

	// Verify tailscale_info is present
	tsInfo, ok := resp["tailscale_info"].(map[string]any)
	if !ok {
		t.Fatal("expected tailscale_info object in response")
	}
	if tsInfo["login_name"] != "dev@localhost" {
		t.Errorf("expected login_name 'dev@localhost', got %q", tsInfo["login_name"])
	}
	if tsInfo["display_name"] != "Dev User" {
		t.Errorf("expected ts display_name 'Dev User', got %q", tsInfo["display_name"])
	}
	if tsInfo["node_name"] != "fishscale.dev.ts.net" {
		t.Errorf("expected node_name 'fishscale.dev.ts.net', got %q", tsInfo["node_name"])
	}
}
