package handler

import (
	"bytes"
	"context"
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

func setupTestHandler(t *testing.T) (*CatchHandler, *chi.Mux) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Insert a dev user
	_, err = db.Exec("INSERT INTO users (tailscale_id, display_name) VALUES ('dev-user', 'Dev User')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	store := storage.NewLocalStore(dir + "/photos")
	h := NewCatchHandler(db, store)

	r := chi.NewRouter()
	r.Use(middleware.DevAuth)
	r.Route("/api/v1/catches", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})

	return h, r
}

func TestCreateAndListCatch(t *testing.T) {
	_, router := setupTestHandler(t)

	body := model.CreateCatchRequest{
		CaughtAt: "2026-02-16T10:30:00Z",
		Kept:     false,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	var created model.Catch
	json.NewDecoder(rec.Body).Decode(&created)
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// List catches
	req = httptest.NewRequest("GET", "/api/v1/catches", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: got %d, want 200", rec.Code)
	}

	var catches []model.Catch
	json.NewDecoder(rec.Body).Decode(&catches)
	if len(catches) != 1 {
		t.Errorf("expected 1 catch, got %d", len(catches))
	}
}

func TestGetCatch(t *testing.T) {
	_, router := setupTestHandler(t)

	body := model.CreateCatchRequest{
		CaughtAt: "2026-02-16T10:30:00Z",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Get by ID
	req = httptest.NewRequest("GET", "/api/v1/catches/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteCatch(t *testing.T) {
	_, router := setupTestHandler(t)

	body := model.CreateCatchRequest{
		CaughtAt: "2026-02-16T10:30:00Z",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/v1/catches/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", rec.Code)
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/api/v1/catches/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("get after delete: got %d, want 404", rec.Code)
	}
}

func TestListCatches_CanceledContext(t *testing.T) {
	_, router := setupTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req := httptest.NewRequest("GET", "/api/v1/catches", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// With context propagation, a canceled context should result in an error response
	// (500 or similar), not a successful 200 with empty data
	if rec.Code == http.StatusOK {
		t.Error("expected non-200 response for canceled context, got 200")
	}
}
