package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCatch_RejectsOversizedFields(t *testing.T) {
	_, router := setupTestHandler(t)

	huge := strings.Repeat("x", 2001) // over 2000 char limit
	body, _ := json.Marshal(map[string]interface{}{
		"caught_at":     "2026-02-16T10:30:00Z",
		"location_name": huge,
	})

	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized field, got %d", rec.Code)
	}
}

func TestCreateCatch_RejectsInvalidCoordinates(t *testing.T) {
	_, router := setupTestHandler(t)

	lat := 200.0 // invalid: must be -90 to 90
	lon := -96.0
	body, _ := json.Marshal(map[string]interface{}{
		"caught_at": "2026-02-16T10:30:00Z",
		"latitude":  lat,
		"longitude": lon,
	})

	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid latitude, got %d", rec.Code)
	}
}

func TestCreateCatch_RejectsOversizedSpeciesName(t *testing.T) {
	_, router := setupTestHandler(t)

	huge := strings.Repeat("x", 201) // over 200 char limit
	body, _ := json.Marshal(map[string]interface{}{
		"caught_at":    "2026-02-16T10:30:00Z",
		"species_name": huge,
	})

	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized species name, got %d", rec.Code)
	}
}
