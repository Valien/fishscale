# Fishscale Iteration 2: Security Hardening & Best Practices

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all security, performance, and Go best-practice issues found during the Iteration 2 codebase audit.

**Architecture:** Hardening pass across the existing codebase. No new features. Every handler gets context propagation, every DB call gets error handling, the server gets graceful shutdown and security headers, photo uploads get MIME validation, and the frontend XSS hole gets closed. Each task is TDD: write the failing test first, then implement.

**Tech Stack:** Go 1.25, sqlx, chi, MapLibre GL JS, Svelte 5, TypeScript

**Important:** All `go` commands must be prefixed with `GOWORK=off`. Every commit must include doc updates to this plan file (mark task status) and the design doc where applicable.

---

## Security Audit Summary

**Date:** 2026-02-16
**Method:** Manual static code analysis of all Go backend files (`internal/`), frontend source (`frontend/src/`), configuration, and build pipeline. No automated SAST tools were used — findings are from line-by-line code review.

**Scan Coverage:**
- All HTTP handlers (`internal/handler/*.go`) — SQL injection, auth scoping, input validation, error handling
- Middleware (`internal/middleware/*.go`) — authentication, context propagation
- Storage layer (`internal/storage/*.go`) — path traversal, file handling
- Database layer (`internal/database/*.go`) — migrations, indexes, connection pooling
- Server/router (`internal/server/server.go`) — CORS, security headers, cache headers
- Entry point (`cmd/fishscale/main.go`) — graceful shutdown, config validation
- Frontend (`frontend/src/lib/`) — XSS, CSP, TypeScript strictness, dependency audit

**Finding Summary:**

| # | Category | Severity | Status |
|---|----------|----------|--------|
| 1 | Graceful shutdown missing | CRITICAL | Task 1 |
| 2 | DB calls ignore request context | CRITICAL | Task 2 |
| 3 | No file upload MIME validation | CRITICAL | Task 3 |
| 4 | XSS in MapView popup | CRITICAL | Task 4 |
| 5 | Errors silently ignored in handlers | HIGH | Task 5 |
| 6 | No security headers | HIGH | Task 6 |
| 7 | No SQLite connection pool limits | HIGH | Task 7 |
| 8 | Missing database indexes | HIGH | Task 7 |
| 9 | No input validation / length limits | MEDIUM | Task 8 |
| 10 | No rate limiting | MEDIUM | Task 9 |
| 11 | Config not validated | MEDIUM | Task 10 |
| 12 | Multipart temp files not cleaned | MEDIUM | Task 3 |
| 13 | Unstructured logging | LOW | Deferred to Iteration 3 |
| 14 | Image resizing / thumbnails | LOW | Deferred to Iteration 3 |
| 15 | TypeScript strict mode | LOW | Deferred to Iteration 3 |

---

## Task 1: Graceful Shutdown

**Files:**
- Modify: `cmd/fishscale/main.go`
- Test: `cmd/fishscale/main_test.go` (create)

**Step 1: Write the failing test**

Create `cmd/fishscale/main_test.go`. We can't easily test `main()` itself, but we can extract the server setup into a testable function and verify shutdown behavior.

Actually, for graceful shutdown the most pragmatic approach is to extract a `run(ctx)` function from `main()` that respects context cancellation, then test that canceling the context causes the server to stop. However, because tsnet requires real Tailscale auth, we'll test only the dev-mode path.

```go
// cmd/fishscale/main_test.go
package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestGracefulShutdown_DevMode(t *testing.T) {
	// Start server in background with a cancel context
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- runDevServer(ctx, ":0") // :0 = random port
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context (simulate SIGTERM)
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./cmd/fishscale/ -run TestGracefulShutdown -v
```
Expected: FAIL — `runDevServer` function does not exist.

**Step 3: Implement graceful shutdown**

Refactor `cmd/fishscale/main.go`:

```go
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tailscale.com/tsnet"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/server"
	"github.com/allen/fishscale/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)

	if cfg.DevMode {
		log.Println("DEV MODE: listening on http://localhost:8080")
		if err := runDevServer(ctx, ":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	} else {
		ts := &tsnet.Server{
			Hostname: cfg.TSHostname,
			AuthKey:  cfg.TSAuthKey,
			Dir:      cfg.TSStateDir,
		}
		defer ts.Close()

		lc, err := ts.LocalClient()
		if err != nil {
			log.Fatalf("tsnet local client: %v", err)
		}

		authMW := middleware.TailscaleAuth(lc, db)
		router := server.NewRouter(cfg, db, store, authMW)

		ln, err := ts.ListenTLS("tcp", ":443")
		if err != nil {
			log.Fatalf("tsnet listen: %v", err)
		}
		defer ln.Close()

		srv := &http.Server{Handler: router}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			srv.Shutdown(shutdownCtx)
		}()

		log.Printf("fishscale available at https://%s.<tailnet>.ts.net", cfg.TSHostname)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
		log.Println("server stopped gracefully")
	}
}

// runDevServer starts an HTTP server on addr that shuts down when ctx is canceled.
// Exported-style name but package main — used by tests.
func runDevServer(ctx context.Context, addr string) error {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)
	router := server.NewRouter(cfg, db, store, nil)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: router}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	return srv.Serve(ln)
}
```

**Note:** The `runDevServer` above creates its own DB/store for testability. In production `main()`, the existing `cfg`/`db`/`store` are reused — so refactor `main()` to call a similar pattern but reuse its own resources. The key change is replacing `log.Fatal(http.ListenAndServe(...))` and `log.Fatal(http.Serve(...))` with `http.Server` + `Shutdown()`.

**Step 4: Run test to verify it passes**

```bash
GOWORK=off go test ./cmd/fishscale/ -run TestGracefulShutdown -v
```
Expected: PASS

**Step 5: Run full test suite**

```bash
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 6: Commit**

```bash
git add cmd/fishscale/main.go cmd/fishscale/main_test.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "feat: add graceful shutdown with signal handling

Refactor main() to use signal.NotifyContext for SIGINT/SIGTERM.
Server now drains in-flight requests on shutdown (10s timeout).
Add runDevServer() for testable dev-mode startup.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Context Propagation to Database Calls

**Files:**
- Modify: `internal/handler/catches.go`
- Modify: `internal/handler/trips.go`
- Modify: `internal/handler/species.go`
- Modify: `internal/handler/settings.go`
- Modify: `internal/handler/stats.go`
- Modify: `internal/handler/export.go`
- Modify: `internal/handler/photos.go`
- Modify: `internal/middleware/tailscale.go`
- Test: `internal/handler/catches_test.go` (add context cancellation test)

Every `h.db.Select(...)` → `h.db.SelectContext(r.Context(), ...)`, every `h.db.Get(...)` → `h.db.GetContext(r.Context(), ...)`, every `h.db.Exec(...)` → `h.db.ExecContext(r.Context(), ...)`.

**Step 1: Write the failing test**

Add to `internal/handler/catches_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./internal/handler/ -run TestListCatches_CanceledContext -v
```
Expected: FAIL — currently returns 200 because `db.Select` ignores context.

**Step 3: Implement context propagation**

Mechanically replace in every handler file. Example from `catches.go`:

```go
// BEFORE:
if err := h.db.Select(&catches, query, user.ID); err != nil {

// AFTER:
if err := h.db.SelectContext(r.Context(), &catches, query, user.ID); err != nil {
```

```go
// BEFORE:
result, err := h.db.Exec(`INSERT INTO catches ...`, ...)

// AFTER:
result, err := h.db.ExecContext(r.Context(), `INSERT INTO catches ...`, ...)
```

```go
// BEFORE:
h.db.Get(&catch, `SELECT ...`, id)

// AFTER:
if err := h.db.GetContext(r.Context(), &catch, `SELECT ...`, id); err != nil {
    jsonError(w, http.StatusInternalServerError, "failed to fetch catch")
    return
}
```

Apply the same pattern to ALL files:
- `catches.go`: Lines 40, 70, 76, 101, 117, 120, 142, 159, 177, 199, 201
- `trips.go`: Lines 32, 58, 64, 102, 111, 136, 157, 165, 184, 186
- `species.go`: Lines 33, 35, 73, 81
- `settings.go`: Lines 29, 83, 93
- `stats.go`: Lines 30, 33, 36, 39, 48, 59, 67
- `export.go`: Line 37
- `photos.go`: Lines 39, 69, 110, 118
- `tailscale.go`: Lines 27, 37

**Step 4: Run test to verify it passes**

```bash
GOWORK=off go test ./internal/handler/ -run TestListCatches_CanceledContext -v
```
Expected: PASS (non-200 status for canceled context)

**Step 5: Run full test suite**

```bash
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/handler/ internal/middleware/tailscale.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "security: propagate request context to all database calls

Replace db.Select/Get/Exec with SelectContext/GetContext/ExecContext
in all handlers and middleware. Canceled client connections now
properly abort in-flight database operations.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: File Upload MIME Validation & Cleanup

**Files:**
- Modify: `internal/handler/photos.go`
- Test: `internal/handler/photos_test.go` (create)

**Step 1: Write the failing tests**

Create `internal/handler/photos_test.go`:

```go
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	// Upload a .txt file disguised as photo
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "malicious.txt")
	part.Write([]byte("this is not an image"))
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

	// Upload a real JPEG (minimal valid JPEG: FF D8 FF header)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "fish.jpg")
	// Minimal JPEG: SOI marker + APP0 marker (enough for http.DetectContentType)
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00})
	// Pad to 512 bytes for DetectContentType
	part.Write(make([]byte, 501))
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/catches/1/photos", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for JPEG upload, got %d: %s", rec.Code, rec.Body.String())
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

	// Upload a PNG (magic bytes)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photos", "fish.png")
	// PNG magic header
	part.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	part.Write(make([]byte, 504))
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/catches/1/photos", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for PNG upload, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
GOWORK=off go test ./internal/handler/ -run TestPhotoUpload -v
```
Expected: `TestPhotoUpload_RejectsNonImage` FAILS (currently accepts all files as 201).

**Step 3: Implement MIME validation and temp file cleanup**

Replace the `Add` method in `internal/handler/photos.go`:

```go
func (h *PhotoHandler) Add(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	catchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid catch id")
		return
	}

	// Verify catch ownership
	var exists int
	if err := h.db.GetContext(r.Context(), &exists, "SELECT 1 FROM catches WHERE id = ? AND user_id = ?", catchID, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, http.StatusBadRequest, "failed to parse form")
		return
	}
	defer r.MultipartForm.RemoveAll() // clean up temp files

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		jsonError(w, http.StatusBadRequest, "no photos provided")
		return
	}

	// Allowed MIME types (checked via magic bytes, not file extension)
	allowedMIME := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"image/heic": true,
		"image/heif": true,
	}

	// Force safe extensions based on detected MIME type
	mimeToExt := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
		"image/heic": ".heic",
		"image/heif": ".heif",
	}

	var photos []model.Photo
	for i, fh := range files {
		f, err := fh.Open()
		if err != nil {
			continue
		}

		// Read first 512 bytes for MIME detection
		head := make([]byte, 512)
		n, _ := f.Read(head)
		mime := http.DetectContentType(head[:n])

		if !allowedMIME[mime] {
			f.Close()
			continue // skip non-image files silently
		}

		// Reset reader to beginning: wrap head + remaining data
		combined := io.MultiReader(bytes.NewReader(head[:n]), f)

		// Use safe extension based on detected content, not user-supplied filename
		safeFilename := "photo" + mimeToExt[mime]
		path, err := h.store.Save(safeFilename, combined)
		f.Close()
		if err != nil {
			continue
		}

		result, err := h.db.ExecContext(r.Context(),
			"INSERT INTO photos (catch_id, filename, sort_order) VALUES (?, ?, ?)",
			catchID, path, i,
		)
		if err != nil {
			h.store.Delete(path)
			continue
		}

		id, _ := result.LastInsertId()
		photos = append(photos, model.Photo{
			ID:        id,
			CatchID:   catchID,
			Filename:  path,
			SortOrder: i,
		})
	}

	if len(photos) == 0 {
		jsonError(w, http.StatusBadRequest, "no valid image files provided")
		return
	}

	jsonResponse(w, http.StatusCreated, photos)
}
```

New imports needed in `photos.go`: `"bytes"`, `"io"`.

**Step 4: Run tests to verify they pass**

```bash
GOWORK=off go test ./internal/handler/ -run TestPhotoUpload -v
```
Expected: All 3 PASS

**Step 5: Run full test suite**

```bash
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/handler/photos.go internal/handler/photos_test.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "security: validate upload MIME types via magic bytes

Reject non-image uploads by reading file header and checking against
allowed MIME types (jpeg, png, gif, webp, heic). Use safe file
extensions based on detected content type, not user-supplied filename.
Reduce max upload size from 32MB to 10MB. Add defer RemoveAll() for
multipart temp file cleanup.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Fix XSS in MapView Popup

**Files:**
- Modify: `frontend/src/lib/pages/MapView.svelte`

This is a frontend-only change. The `setHTML()` call in MapView renders user-controlled data (`species_name`, `location_name`, `bait_or_lure`) without escaping. Replace with `setDOMContent()` using programmatically created DOM elements, or use a text-escaping helper.

**Step 1: Implement the fix**

Replace the popup creation in `MapView.svelte` (lines 44-52):

```typescript
// Helper to escape HTML entities
function escapeHTML(str: string): string {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}
```

Then in `syncMarkers`:

```typescript
const popupEl = document.createElement('div');
popupEl.style.cssText = 'padding:4px;font-family:sans-serif;font-size:0.85rem;';

const name = document.createElement('strong');
name.textContent = c.species_name || 'Unknown';
popupEl.appendChild(name);

popupEl.appendChild(document.createElement('br'));

const loc = document.createTextNode(c.location_name || '');
popupEl.appendChild(loc);

popupEl.appendChild(document.createElement('br'));

const date = document.createElement('small');
date.textContent = new Date(c.caught_at).toLocaleDateString();
popupEl.appendChild(date);

if (c.weight_lb) {
  popupEl.appendChild(document.createElement('br'));
  const weight = document.createElement('small');
  weight.textContent = `${c.weight_lb} lb`;
  popupEl.appendChild(weight);
}

if (c.bait_or_lure) {
  popupEl.appendChild(document.createElement('br'));
  const bait = document.createElement('small');
  bait.textContent = c.bait_or_lure;
  popupEl.appendChild(bait);
}

const popup = new maplibregl.Popup({ offset: 10 }).setDOMContent(popupEl);
```

**Step 2: Build frontend to verify no compile errors**

```bash
cd frontend && npm run build
```
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/lib/pages/MapView.svelte docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "security: fix XSS in map popup by using setDOMContent

Replace setHTML() with setDOMContent() and programmatic DOM element
creation. User-controlled data (species_name, location_name,
bait_or_lure) is now set via textContent which auto-escapes HTML.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Fix Silently Ignored Errors in Handlers

**Files:**
- Modify: `internal/handler/catches.go`
- Modify: `internal/handler/trips.go`
- Modify: `internal/handler/species.go`
- Modify: `internal/handler/settings.go`
- Modify: `internal/handler/stats.go`
- Modify: `internal/handler/photos.go`
- Test: `internal/handler/handlers_test.go` (extend)

Every `h.db.Get(...)` / `h.db.Select(...)` / `h.db.Exec(...)` call whose error is currently discarded must now be checked and return an appropriate HTTP error.

**Step 1: Write failing tests**

Add to `internal/handler/handlers_test.go`:

```go
func TestStatsHandler_ReturnsValidResponse(t *testing.T) {
	router := setupFullRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("stats: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	// Verify the response is valid JSON with expected fields
	var stats model.StatsResponse
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}
}
```

**Step 2: Identify and fix all ignored errors**

In each handler, find every DB call where the error return is `_` or unchecked:

**catches.go:**
- Line 76: `h.db.Select(&photos, ...)` — add error check, return 500 if fails
- Line 117: `id, _ := result.LastInsertId()` — check error
- Line 120: `h.db.Get(&catch, ...)` — add error check, return 500 if fails
- Line 177: `h.db.Get(&catch, ...)` — same
- Line 199: `h.db.Select(&photos, ...)` — same
- Line 207: `rows, _ := result.RowsAffected()` — check error

**trips.go:**
- Line 64: `h.db.Select(&catches, ...)` — add error check
- Line 109: `id, _ := result.LastInsertId()` — check error
- Line 111: `h.db.Get(&trip, ...)` — add error check
- Line 165: `h.db.Get(&trip, ...)` — same
- Line 184: `h.db.Exec(...)` — add error check (catch unlink)

**species.go:**
- Line 79: `id, _ := result.LastInsertId()` — check error
- Line 81: `h.db.Get(&species, ...)` — add error check

**settings.go:**
- Line 93: `h.db.Get(&settings, ...)` — add error check

**stats.go:**
- Lines 30, 33, 36, 39, 48, 59, 67: ALL `h.db.Get()`/`h.db.Select()` calls — add error checks, return 500

**photos.go:**
- Line 78: `id, _ := result.LastInsertId()` — check error
- Line 118: `h.db.Exec(...)` — add error check

Pattern for each fix:

```go
// BEFORE (ignored error):
h.db.Get(&stats.TotalCatches, "SELECT COUNT(*) ...", user.ID)

// AFTER:
if err := h.db.GetContext(r.Context(), &stats.TotalCatches, "SELECT COUNT(*) ...", user.ID); err != nil {
    jsonError(w, http.StatusInternalServerError, "failed to query stats")
    return
}
```

For `LastInsertId()` — log but don't fail (it's informational and the INSERT already succeeded):

```go
id, err := result.LastInsertId()
if err != nil {
    jsonError(w, http.StatusInternalServerError, "failed to get created ID")
    return
}
```

**Step 3: Run full test suite**

```bash
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/handler/ docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "fix: check all database operation errors in handlers

Every silently-ignored db.Get/Select/Exec/LastInsertId call now has
proper error checking. Failed queries return 500 to the client
instead of silently returning partial or empty data.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Security Headers Middleware

**Files:**
- Create: `internal/middleware/security.go`
- Test: `internal/middleware/security_test.go` (create)
- Modify: `internal/server/server.go` (add middleware)

**Step 1: Write the failing test**

Create `internal/middleware/security_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "camera=(self), geolocation=(self)"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.header, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./internal/middleware/ -run TestSecurityHeaders -v
```
Expected: FAIL — `SecurityHeaders` function does not exist.

**Step 3: Implement security headers middleware**

Create `internal/middleware/security.go`:

```go
package middleware

import "net/http"

// SecurityHeaders adds standard security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(self), geolocation=(self)")
		next.ServeHTTP(w, r)
	})
}
```

Wire it into `internal/server/server.go`:

```go
r.Use(appMiddleware.SecurityHeaders)
```

Add after the existing `r.Use(chiMiddleware.Compress(5))` line.

**Step 4: Run tests**

```bash
GOWORK=off go test ./internal/middleware/ -run TestSecurityHeaders -v
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/middleware/security.go internal/middleware/security_test.go internal/server/server.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "security: add security headers middleware

Add X-Frame-Options: DENY, X-Content-Type-Options: nosniff,
Referrer-Policy, and Permissions-Policy headers to all responses.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: SQLite Connection Pool Limits & Missing Indexes

**Files:**
- Modify: `internal/database/db.go`
- Modify: `internal/database/migrations.go`
- Test: `internal/database/db_test.go` (extend)

**Step 1: Write the failing tests**

Add to `internal/database/db_test.go`:

```go
func TestConnectionPoolLimits(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 2 {
		t.Errorf("expected MaxOpenConnections=2, got %d", stats.MaxOpenConnections)
	}
}

func TestIndexesExist(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	indexes := []string{
		"idx_catches_user_id",
		"idx_catches_caught_at",
		"idx_catches_species_id",
		"idx_catches_trip_id",
		"idx_photos_catch_id",
		"idx_trips_user_id",
	}
	for _, idx := range indexes {
		var name string
		err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx)
		if err != nil {
			t.Errorf("index %q not found: %v", idx, err)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
GOWORK=off go test ./internal/database/ -run "TestConnectionPoolLimits|TestIndexesExist" -v
```
Expected: FAIL — no pool limits set, missing indexes.

**Step 3: Implement fixes**

In `internal/database/db.go`, after `db.Ping()`:

```go
// SQLite supports limited concurrency. Set conservative pool limits.
db.SetMaxOpenConns(2)
db.SetMaxIdleConns(1)
```

In `internal/database/migrations.go`, add to the schema string after existing indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_catches_trip_id ON catches(trip_id);
CREATE INDEX IF NOT EXISTS idx_trips_user_id   ON trips(user_id);
```

Note: `user_settings.user_id` already has a UNIQUE constraint which implicitly creates an index.

**Step 4: Run tests**

```bash
GOWORK=off go test ./internal/database/ -v
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/database/db.go internal/database/migrations.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "perf: add SQLite pool limits and missing indexes

Set MaxOpenConns=2 and MaxIdleConns=1 for SQLite (limited write
concurrency). Add missing indexes on catches(trip_id) and
trips(user_id) for efficient query filtering.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: Input Validation

**Files:**
- Modify: `internal/handler/catches.go`
- Modify: `internal/handler/trips.go`
- Modify: `internal/handler/species.go`
- Create: `internal/handler/validate.go`
- Test: `internal/handler/validate_test.go` (create)

**Step 1: Write the failing tests**

Create `internal/handler/validate_test.go`:

```go
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

	lat := 200.0  // invalid: must be -90 to 90
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

func TestCreateSpecies_RejectsOversizedName(t *testing.T) {
	router := setupFullRouter(t)

	huge := strings.Repeat("x", 201) // over 200 char limit
	body, _ := json.Marshal(map[string]string{"name": huge, "category": "freshwater"})

	req := httptest.NewRequest("POST", "/api/v1/species", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized species name, got %d", rec.Code)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
GOWORK=off go test ./internal/handler/ -run "TestCreateCatch_Rejects|TestCreateSpecies_Rejects" -v
```
Expected: FAIL — no validation exists.

**Step 3: Implement validation**

Create `internal/handler/validate.go`:

```go
package handler

import "fmt"

const (
	maxTextFieldLen    = 2000
	maxShortFieldLen   = 200
	maxNotesLen        = 5000
)

func validateStringLen(field, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}

func validateLatitude(lat *float64) error {
	if lat != nil && (*lat < -90 || *lat > 90) {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	return nil
}

func validateLongitude(lon *float64) error {
	if lon != nil && (*lon < -180 || *lon > 180) {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func validatePositive(field string, val *float64) error {
	if val != nil && *val < 0 {
		return fmt.Errorf("%s must not be negative", field)
	}
	return nil
}
```

Then add validation in `catches.go` Create/Update, before the INSERT/UPDATE:

```go
// Validate field lengths
for _, check := range []struct{ name, val string; max int }{
    {"location_name", req.LocationName, maxTextFieldLen},
    {"bait_or_lure", req.BaitOrLure, maxShortFieldLen},
    {"rod_setup", req.RodSetup, maxShortFieldLen},
    {"line_info", req.LineInfo, maxShortFieldLen},
    {"hook_size", req.HookSize, maxShortFieldLen},
    {"water_clarity", req.WaterClarity, maxShortFieldLen},
    {"wind_dir", req.WindDir, maxShortFieldLen},
    {"conditions", req.Conditions, maxShortFieldLen},
    {"notes", req.Notes, maxNotesLen},
} {
    if err := validateStringLen(check.name, check.val, check.max); err != nil {
        jsonError(w, http.StatusBadRequest, err.Error())
        return
    }
}

// Validate coordinates
if err := validateLatitude(req.Latitude); err != nil {
    jsonError(w, http.StatusBadRequest, err.Error())
    return
}
if err := validateLongitude(req.Longitude); err != nil {
    jsonError(w, http.StatusBadRequest, err.Error())
    return
}

// Validate positive numeric fields
for _, check := range []struct{ name string; val *float64 }{
    {"weight_lb", req.WeightLb},
    {"length_in", req.LengthIn},
} {
    if err := validatePositive(check.name, check.val); err != nil {
        jsonError(w, http.StatusBadRequest, err.Error())
        return
    }
}
```

Similarly add name length validation in `species.go` Create:

```go
if err := validateStringLen("name", req.Name, maxShortFieldLen); err != nil {
    jsonError(w, http.StatusBadRequest, err.Error())
    return
}
```

And in `trips.go` Create/Update for name and notes fields.

**Step 4: Run tests**

```bash
GOWORK=off go test ./internal/handler/ -run "TestCreateCatch_Rejects|TestCreateSpecies_Rejects" -v
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/handler/ docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "security: add input validation for all text and numeric fields

Validate string field lengths (200 for short fields, 2000 for text,
5000 for notes), coordinate ranges (-90/90 lat, -180/180 lon), and
reject negative weight/length values.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: Rate Limiting

**Files:**
- Modify: `internal/server/server.go`
- Modify: `go.mod` (add httprate dependency)
- Test: `internal/server/server_test.go` (create)

**Step 1: Write the failing test**

Create `internal/server/server_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./internal/server/ -run TestRateLimiting -v
```
Expected: FAIL — no rate limiting, all return 200.

**Step 3: Implement rate limiting**

Add dependency:

```bash
GOWORK=off go get github.com/go-chi/httprate
```

In `internal/server/server.go`, add rate limiting middleware:

```go
import "github.com/go-chi/httprate"

// Inside NewRouter, after other middleware:
r.Use(httprate.LimitByIP(100, 1*time.Minute))
```

This limits all requests to 100 per minute per IP. Since this runs behind Tailscale, IP-based limiting is per-device.

**Step 4: Run tests**

```bash
GOWORK=off go test ./internal/server/ -run TestRateLimiting -v
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/server/server.go go.mod go.sum internal/server/server_test.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "security: add rate limiting (100 req/min per IP)

Add go-chi/httprate middleware to limit API requests to 100 per
minute per IP address. Returns 429 Too Many Requests when exceeded.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: Config Validation

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"testing"
)

func TestValidate_RequiresAuthKeyInProduction(t *testing.T) {
	cfg := &Config{
		DevMode:   false,
		TSAuthKey: "",
		DBPath:    "/data/fish.db",
		PhotoDir:  "/data/photos",
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing TS_AUTHKEY in production mode, got nil")
	}
}

func TestValidate_AllowsMissingAuthKeyInDevMode(t *testing.T) {
	cfg := &Config{
		DevMode:  true,
		DBPath:   "/data/fish.db",
		PhotoDir: "/data/photos",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error in dev mode: %v", err)
	}
}

func TestLoad_SetsDefaults(t *testing.T) {
	// Clear env to test defaults
	os.Unsetenv("TS_HOSTNAME")
	os.Unsetenv("FISHSCALE_LOG_LEVEL")

	cfg := Load()
	if cfg.TSHostname != "fishscale" {
		t.Errorf("expected default hostname 'fishscale', got %q", cfg.TSHostname)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got %q", cfg.LogLevel)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
GOWORK=off go test ./internal/config/ -v
```
Expected: FAIL — `Validate` method does not exist.

**Step 3: Implement config validation**

Add to `internal/config/config.go`:

```go
import (
	"fmt"
	"os"
)

func (c *Config) Validate() error {
	if !c.DevMode && c.TSAuthKey == "" {
		return fmt.Errorf("TS_AUTHKEY is required in production mode")
	}
	if c.DBPath == "" {
		return fmt.Errorf("FISHSCALE_DB_PATH must not be empty")
	}
	if c.PhotoDir == "" {
		return fmt.Errorf("FISHSCALE_PHOTO_DIR must not be empty")
	}
	return nil
}
```

Call it in `cmd/fishscale/main.go` right after `config.Load()`:

```go
cfg := config.Load()
if err := cfg.Validate(); err != nil {
    log.Fatalf("invalid configuration: %v", err)
}
```

**Step 4: Run tests**

```bash
GOWORK=off go test ./internal/config/ -v
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/config/ cmd/fishscale/main.go docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "fix: validate config on startup, require TS_AUTHKEY in production

Add Config.Validate() that requires TS_AUTHKEY in production mode
and ensures DB/photo paths are non-empty. Exits immediately with
a clear error message instead of failing later during connection.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 11: Update Design Doc with Security Architecture

**Files:**
- Modify: `docs/plans/2026-02-16-fishscale-design.md`

Add a **Security** section to the design doc documenting the security architecture after all fixes:

```markdown
## Security

### Authentication & Authorization
- All requests authenticated via Tailscale WhoIs middleware (network is the auth boundary)
- Every database query scoped to `user_id` from authenticated context — no cross-user access
- Dev mode uses a hardcoded user (ID 1) — development only, never production

### Request Security
- Security headers on all responses: X-Frame-Options DENY, X-Content-Type-Options nosniff, Referrer-Policy, Permissions-Policy
- Rate limiting: 100 requests/minute per IP via go-chi/httprate
- Request context propagated to all database calls — canceled client connections abort queries
- Input validation: string length limits, coordinate range checks, positive numeric enforcement

### File Upload Security
- MIME type validated via magic bytes (first 512 bytes), not file extension
- Allowed types: image/jpeg, image/png, image/gif, image/webp, image/heic, image/heif
- File extensions forced to match detected MIME type (not user-supplied)
- Maximum upload size: 10MB per multipart form
- Temporary multipart files cleaned up via defer RemoveAll()

### Frontend Security
- Map popups use setDOMContent() with textContent (not setHTML) to prevent XSS
- No localStorage/sessionStorage token storage (Tailscale handles auth at network layer)
- No {@html} directives in Svelte components

### Database Security
- All queries use parameterized placeholders (?) — no SQL string concatenation
- SQLite connection pool limited to 2 max open / 1 idle connection
- Foreign key constraints enabled, cascade deletes configured
- WAL mode with 5-second busy timeout

### Caching
- index.html: Cache-Control no-cache, no-store, must-revalidate (always fresh)
- /assets/*: Cache-Control public, max-age=31536000, immutable (Vite content-hashed)

### Configuration
- TS_AUTHKEY required in production mode (validated on startup)
- No secrets in source code — all configuration via environment variables
```

Also update the Key Architecture Decisions section to mention rate limiting, input validation, and MIME validation.

**Step 1: Make edits**

Apply the changes to the design doc.

**Step 2: Commit**

```bash
git add docs/plans/2026-02-16-fishscale-design.md docs/plans/2026-02-16-iteration-2-security-hardening.md
git commit -m "docs: add Security section to design doc

Document authentication, request security, file upload validation,
frontend XSS prevention, database security, caching strategy, and
configuration validation. Also documents the audit methodology.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Testing Plan

After all tasks, run the complete verification:

```bash
# Go tests
GOWORK=off go test ./... -v

# Frontend build
cd frontend && npm run build && cd ..

# Copy to embed dir
rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist

# Build binary
GOWORK=off CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Run in dev mode
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos ./fishscale

# Test:
# - Map popups display correctly (no raw HTML)
# - Photo upload works for images, rejects non-images
# - Species dropdown still works on iOS Safari
# - Settings persist across sessions
# - SIGINT cleanly shuts down server (no "connection reset" errors)
# - Rate limiting kicks in after rapid requests (curl loop)
# - Oversized field values rejected with 400
# - Security headers present (check with curl -I)
```

---

## Deferred to Iteration 3

These items were identified in the audit but are lower priority. They are tracked in the **[Iteration 3: Infrastructure, CI/CD & Frontend Quality](2026-02-16-iteration-3-infrastructure.md)** plan along with additional findings from Docker, CI/CD, and frontend quality audits.

- **Structured logging (slog):** Replace `log.Printf` with Go 1.21+ `slog` package. Use LogLevel from config. Add request correlation IDs.
- **Image resizing / thumbnails:** Generate thumbnails on upload, serve compressed versions. Populate the `thumbnail` field in photos table.
- **TypeScript strict mode:** Enable `strict: true` in tsconfig.app.json, replace 18 instances of `any` with proper types, add API response interfaces.
- **Content-Security-Policy header:** Requires careful configuration to allow MapLibre GL tile loading, inline styles, etc. Needs dedicated testing.

Additional findings added to Iteration 3:
- **Photo ownership gap (MEDIUM):** `GET /photos/*` has auth but no per-user ownership check
- **Docker hardening (HIGH):** Container runs as root, no HEALTHCHECK, unpinned base images
- **CI/CD pipeline (HIGH):** No GitHub Actions, no golangci-lint
- **Frontend testing (HIGH):** No Vitest or any test framework
- **Code splitting (MEDIUM):** Single 1MB+ JS bundle
- **ESLint/Prettier (MEDIUM):** No frontend linting or formatting
- **LICENSE & Makefile (MEDIUM):** MIT license referenced but no file, no build automation
