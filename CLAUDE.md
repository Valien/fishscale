# CLAUDE.md

Project-level guidance for Claude Code working on Fishscale.

## Project Overview

Fishscale is a self-hosted fishing tracker. Go backend with embedded Svelte 5 frontend, SQLite database, Tailscale authentication. Single binary, Docker deployment.

## Session Warmup

On project warmup, check GitHub for open issues and add any new ones to the backlog:

```bash
gh issue list --repo Valien/fishscale --state open
```

Review against `docs/plans/todo.md` and surface any new issues that aren't already tracked.

## Critical Build Requirement

**All `go` commands must be prefixed with `GOWORK=off`:**

```bash
GOWORK=off go test ./... -v
GOWORK=off go build -o fishscale ./cmd/fishscale
GOWORK=off go run ./cmd/fishscale
GOWORK=off go get github.com/some/dep
```

There is an external `go.work` file above this repo that interferes with module resolution.

## Running the Project

```bash
# Dev mode (no Tailscale, HTTP on :8080)
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale

# Frontend dev server with hot reload (separate terminal)
cd frontend && npm run dev

# Build frontend and embed into Go binary
cd frontend && npm run build && cd ..
rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist
GOWORK=off CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale
```

## Running Tests

```bash
# Go tests
GOWORK=off go test ./... -v

# Single package
GOWORK=off go test ./internal/handler/ -v

# Single test
GOWORK=off go test ./internal/handler/ -run TestListCatches -v
```

## Running CI Checks Locally

All integration and testing is done locally, not via GitHub Actions:

```bash
# Run all CI checks (tests, lints, format check, type check, build)
make ci
# or
make check

# Run individual checks
make test           # Go tests (with race detector) + frontend tests
make lint           # Go linter + frontend ESLint
cd frontend && npm run format:check   # Prettier format check
cd frontend && npm run check          # Svelte type check
```

**IMPORTANT:** After every major feature fix/enhancement, run `make ci` or `make check` before manually testing the app. This catches issues early and ensures code quality.

**Note:** `golangci-lint` is not installed locally. The `make lint` target will fail on the Go lint step. Frontend ESLint works fine. This does not block shipping.

## Project Structure

```
cmd/fishscale/          Entry point (main.go)
internal/
  config/               Environment variable configuration
  database/             SQLite connection, migrations, seed data (sqlx)
  frontend/             Embedded SPA assets (go:embed)
  handler/              HTTP handlers for all API endpoints
  middleware/            Tailscale auth, dev auth, security headers
  model/                Data model structs
  server/               Router assembly (chi) and SPA serving
  storage/              Photo storage interface + local filesystem impl
frontend/               Svelte 5 + TypeScript + Vite
  src/lib/
    api.ts              Typed API client
    components/         BottomNav, shared UI
    pages/              LogCatch, MapView, CatchLog, Stats, Settings
    stores/             Svelte stores for catches and settings
    theme.css           CSS custom properties for theming
docs/plans/             Design doc and iteration plans
```

## Go Conventions

### Handler Pattern

Every handler struct takes `*sqlx.DB` and optionally `storage.Store`. Every method starts with auth check:

```go
func (h *CatchHandler) List(w http.ResponseWriter, r *http.Request) {
    user := middleware.UserFromContext(r.Context())
    if user == nil {
        jsonError(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    // ...
}
```

### Response Helpers

Use `jsonResponse` and `jsonError` from `internal/handler/helpers.go`:

```go
jsonResponse(w, http.StatusOK, data)
jsonError(w, http.StatusBadRequest, "invalid id")
```

### Database Calls

Use sqlx with `?` parameter placeholders (SQLite). Use context-aware variants:

```go
h.db.SelectContext(r.Context(), &catches, query, user.ID)
h.db.GetContext(r.Context(), &catch, query, id, user.ID)
h.db.ExecContext(r.Context(), query, args...)
```

Every query must be scoped to `user_id` from the authenticated context.

### URL Parameters

```go
id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
```

### Test Helpers

Tests use `setupTestHandler(t)` or `setupFullRouter(t)` which create a temp SQLite DB, insert a dev user, and wire up the router with `DevAuth` middleware:

```go
func TestSomething(t *testing.T) {
    _, router := setupTestHandler(t)
    req := httptest.NewRequest("GET", "/api/v1/catches", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    // assert on rec.Code, rec.Body
}
```

## Frontend Conventions

### Svelte 5 Runes

This project uses Svelte 5 runes, not Svelte 4 syntax:

- `$state` for reactive state
- `$effect` for side effects
- `$derived` for computed values
- `$props()` for component props

### iOS Safari Compatibility

The species dropdown uses a specific touch event pattern that was hard-won through debugging. Do not change the `ontouchend`/`onmousedown`/`onblur` pattern in `LogCatch.svelte` without reading the detailed rationale in `docs/plans/2026-02-16-iteration-1-bugfixes.md`.

Key rules:
- Use `ontouchend` for iOS tap handling (not `onclick` or `onpointerup`)
- Use `onmousedown` for desktop fallback
- The `justSelected` flag must be a plain `let`, NOT `$state`, to avoid `$effect` re-trigger loops

### Map Preservation

`MapView.svelte` is always rendered (hidden via CSS `display: none`), not conditionally rendered with `{#if}`. This preserves map zoom/pan state across tab switches.

### Navigation Pattern

The app uses state-driven navigation (no router). `App.svelte` has an `activePage` state controlling which page shows. `CatchLog` has an internal `view` state (`'list' | 'detail' | 'edit'`) for sub-views. Cross-page navigation (e.g., map popup to catch detail) uses callback props and one-shot state that must be reset after consumption to prevent stale re-triggers on remount.

### API Client

`frontend/src/lib/api.ts` provides a typed wrapper around `fetch()`. All endpoints are under `/api/v1`.

## Commit Conventions

```
feat: add graceful shutdown with signal handling
fix: dropdown $effect race condition on iOS
security: validate upload MIME types via magic bytes
perf: add SQLite pool limits and missing indexes
docs: sync design doc with iteration 1 fixes
chore: add .worktrees to gitignore
test: add context cancellation tests for handlers
```

Every commit that changes code must also update relevant docs in the same commit (design doc, iteration plan). End all commits with:

```
Co-Authored-By: Claude <noreply@anthropic.com>
```

## Documentation

- `docs/plans/2026-02-16-fishscale-design.md` — Architecture, data model, API design, UX decisions
- `docs/plans/2026-02-16-iteration-1-bugfixes.md` — iOS Safari bugs, photo upload, species filter
- `docs/plans/2026-02-16-iteration-2-security-hardening.md` — Security audit findings and fixes
- `docs/plans/2026-02-16-iteration-3-infrastructure.md` — Docker, CI/CD, frontend quality

Update the design doc when changing architecture, data model, API surface, or security posture. Update the relevant iteration plan when completing tasks.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25, chi v5, sqlx |
| Database | SQLite (modernc.org/sqlite, pure Go, no CGO) |
| Frontend | Svelte 5, TypeScript, Vite |
| Maps | MapLibre GL JS + OpenStreetMap |
| Auth | Tailscale tsnet (WhoIs API) |
| Weather | Open-Meteo API (no key required) |
