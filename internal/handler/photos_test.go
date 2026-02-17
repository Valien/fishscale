package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhotoUpload_RejectsNonImage(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch first
	catchBody, _ := json.Marshal(map[string]string{"caught_at": "2026-02-16T10:30:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create catch: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	// Upload a .txt file disguised as photo
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "malicious.txt")
	_, _ = part.Write([]byte("this is not an image"))
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/catches/1/photos", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should reject — no valid images saved
	if rec.Code == http.StatusCreated {
		t.Error("expected non-image upload to be rejected, got 201")
	}
}

func TestPhotoUpload_AcceptsJPEG(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch first
	catchBody, _ := json.Marshal(map[string]string{"caught_at": "2026-02-16T10:30:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create catch: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	// Upload a real JPEG (minimal valid JPEG: FF D8 FF header)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "fish.jpg")
	// Minimal JPEG: SOI marker + APP0 marker (enough for http.DetectContentType)
	_, _ = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00})
	// Pad to 512 bytes for DetectContentType
	_, _ = part.Write(make([]byte, 501))
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/catches/1/photos", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for JPEG upload, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPhotoServing_RequiresOwnership(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch and upload a photo (owned by dev user ID 1)
	catchBody, _ := json.Marshal(map[string]string{"caught_at": "2026-02-16T10:30:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create catch: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	// Upload a JPEG to catch 1
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "fish.jpg")
	_, _ = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00})
	_, _ = part.Write(make([]byte, 501))
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/catches/1/photos", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("upload photo: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	// Extract the saved filename from the response
	var photos []struct {
		Filename string `json:"filename"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&photos)
	if len(photos) == 0 {
		t.Fatal("expected at least one photo in response")
	}
	savedFilename := photos[0].Filename

	// Serve the photo as the same user (dev user ID 1) — should succeed
	req = httptest.NewRequest("GET", "/photos/"+savedFilename, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for owned photo, got %d", rec.Code)
	}

	// Try to serve a filename that isn't linked to any catch — should return 404
	req = httptest.NewRequest("GET", "/photos/nonexistent-file.jpg", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-owned photo, got %d", rec.Code)
	}

	// Try path traversal — should return 404
	req = httptest.NewRequest("GET", "/photos/../etc/passwd", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for path traversal attempt, got %d", rec.Code)
	}
}

func TestPhotoUpload_AcceptsPNG(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch first
	catchBody, _ := json.Marshal(map[string]string{"caught_at": "2026-02-16T10:30:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create catch: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	// Upload a PNG (magic bytes)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "fish.png")
	// PNG magic header
	_, _ = part.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	_, _ = part.Write(make([]byte, 504))
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/catches/1/photos", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for PNG upload, got %d: %s", rec.Code, rec.Body.String())
	}
}
