package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/storage"
)

func TestRateLimiting(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{PhotoDir: dir + "/photos", DevMode: true}
	store := storage.NewLocalStore(cfg.PhotoDir)
	router := NewRouter(cfg, db, store, middleware.DevAuth)

	// Send 101 rapid requests (limit should be 100/min)
	var lastCode int
	for i := 0; i < 101; i++ {
		req := httptest.NewRequest("GET", "/api/v1/species", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after rate limit exceeded, got %d", lastCode)
	}
}
