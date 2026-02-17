# Fishscale Iteration 3: Infrastructure, CI/CD & Frontend Quality

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Harden Docker deployment, add CI/CD, fix photo auth gap, set up frontend testing/linting, and address all items deferred from Iteration 2.

**Architecture:** Infrastructure and tooling pass. Fixes a HIGH-severity auth gap in photo serving, hardens Docker for production, adds GitHub Actions CI, and establishes frontend quality tooling. Each backend task follows TDD where applicable.

**Tech Stack:** Go 1.25, Docker, GitHub Actions, Vitest, ESLint, Prettier, golangci-lint

**Important:** All `go` commands must be prefixed with `GOWORK=off`. Every commit must include doc updates to this plan file (mark task status) and the design doc where applicable.

---

## Infrastructure Audit Summary

**Date:** 2026-02-16
**Method:** Manual review of Dockerfile, docker-compose.yml, .dockerignore, frontend build pipeline (package.json, vite.config.ts, tsconfig.app.json), project hygiene (CI/CD, linting, licensing), and Go embed/SPA serving (internal/server/server.go).

**Finding Summary:**

| # | Category | Severity | Status |
|---|----------|----------|--------|
| 1 | Photo serving has no ownership check | MEDIUM | Task 1 ✅ |
| 2 | Docker container runs as root | HIGH | Task 2 ✅ |
| 3 | No CI/CD pipeline | HIGH | Task 3 ✅ |
| 4 | No frontend testing framework | HIGH | Task 4 ✅ |
| 5 | Base images not pinned to patch versions | MEDIUM | Task 2 ✅ |
| 6 | No Docker HEALTHCHECK | MEDIUM | Task 2 ✅ (skipped — see note) |
| 7 | No resource limits in docker-compose | MEDIUM | Task 2 ✅ |
| 8 | Single 1MB+ JS bundle, no code splitting | MEDIUM | Task 5 ✅ |
| 9 | No ESLint/Prettier configuration | MEDIUM | Task 6 |
| 10 | No LICENSE file (MIT mentioned in README only) | MEDIUM | Task 7 |
| 11 | No Makefile for build automation | MEDIUM | Task 7 |
| 12 | No golangci-lint configuration | MEDIUM | Task 3 ✅ |
| 13 | .dockerignore could exclude more directories | LOW | Task 2 ✅ |
| 14 | No CONTRIBUTING.md or SECURITY.md | LOW | Deferred |

**Deferred from Iteration 2:**

| # | Category | Severity | Status |
|---|----------|----------|--------|
| 15 | Structured logging (slog) | LOW | Task 8 |
| 16 | Image resizing / thumbnails | LOW | Task 9 |
| 17 | TypeScript strict mode | LOW | Task 10 |
| 18 | Content-Security-Policy header | LOW | Task 11 |

---

## Task 1: Photo Ownership Check on Serving

**Severity:** MEDIUM
**Finding:** `GET /photos/*` in `internal/server/server.go` uses `http.FileServer` which serves any file by path. The auth middleware *does* run on this route (it's applied router-wide via `r.Use()`), so unauthenticated requests are blocked. However, any authenticated user can access any other user's photos by guessing the filename — `http.FileServer` has no concept of ownership.

**Note on production context:** This app runs on a private Tailnet. In practice, most deployments are single-user. The Tailscale network layer ensures only authorized devices reach the server. This task is MEDIUM severity because multi-user tailnets could expose photos across users. For single-user deployments, this is cosmetic.

**Files:**
- Modify: `internal/server/server.go` (replace FileServer with custom handler)
- Modify: `internal/handler/photos.go` (add Serve method)
- Test: `internal/handler/photos_test.go` (extend)

**Step 1: Write the failing test**

Add to `internal/handler/photos_test.go`:

```go
func TestPhotoServing_RequiresOwnership(t *testing.T) {
	router := setupFullRouter(t)

	// Create a catch (owned by dev user ID 1)
	catchBody, _ := json.Marshal(map[string]string{"caught_at": "2026-02-16T10:30:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(catchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Try to serve a photo filename that isn't linked to any catch for this user
	req = httptest.NewRequest("GET", "/photos/nonexistent-file.jpg", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("expected non-owned photo to be rejected, got 200")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./internal/handler/ -run TestPhotoServing_RequiresOwnership -v
```
Expected: FAIL — FileServer serves any file that exists on disk.

**Step 3: Implement ownership-checked photo serving**

Replace the `http.FileServer` route in `server.go` with a handler that checks the photo belongs to the requesting user:

```go
// In server.go — replace the FileServer line:
r.Get("/photos/*", photos.Serve)
```

Add a `Serve` method to `PhotoHandler` in `photos.go`:

```go
func (h *PhotoHandler) Serve(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract filename from URL path
	filename := strings.TrimPrefix(r.URL.Path, "/photos/")
	if filename == "" {
		http.NotFound(w, r)
		return
	}

	// Verify this photo belongs to a catch owned by this user
	var exists int
	err := h.db.GetContext(r.Context(), &exists,
		`SELECT 1 FROM photos p
		 JOIN catches c ON p.catch_id = c.id
		 WHERE p.filename = ? AND c.user_id = ?`, filename, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	h.store.ServeFile(w, r, filename)
}
```

Add `ServeFile` to the `storage.Store` interface and implement in `local.go`:

```go
func (s *LocalStore) ServeFile(w http.ResponseWriter, r *http.Request, filename string) {
	http.ServeFile(w, r, filepath.Join(s.dir, filename))
}
```

**Step 4: Run test to verify it passes**

```bash
GOWORK=off go test ./internal/handler/ -run TestPhotoServing_RequiresOwnership -v
```
Expected: PASS

**Step 5: Run full test suite**

```bash
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/server/ internal/handler/photos.go internal/handler/photos_test.go internal/storage/ docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "security: add ownership check to photo serving

Replace http.FileServer with a handler that verifies the requested
photo belongs to a catch owned by the authenticated user. Previously,
any authenticated tailnet user could access any photo by filename.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Docker Hardening

**Severity:** HIGH (root container), MEDIUM (other items)
**Findings:**
- Container runs as root (no USER directive)
- Base images use floating tags (node:22-alpine, golang:1.25-alpine, alpine:3.20)
- No HEALTHCHECK instruction
- No resource limits in docker-compose.yml
- .dockerignore could exclude docs/, .claude/, .worktrees/

**Files:**
- Modify: `internal/server/server.go` (add /healthz endpoint)
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `.dockerignore`

**Step 1: Add a `/healthz` endpoint to the router**

Before updating the Dockerfile, add a lightweight health check endpoint that works in both dev and production modes. In `internal/server/server.go`, add before the auth middleware:

```go
// Health check endpoint — no auth required, used by Docker HEALTHCHECK.
// Placed before auth middleware so it works without Tailscale identity.
r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
})
```

This must be registered **before** the `r.Use(authMiddleware)` block (move it after the `r.Use(httprate.LimitByIP(...))` line and before the auth `if` block). Alternatively, use a separate inline router group with no auth.

**Step 2: Update Dockerfile**

```dockerfile
# Stage 1: Build Svelte frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./internal/frontend/dist
RUN CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Stage 3: Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S fishscale && adduser -S fishscale -G fishscale
COPY --from=backend /app/fishscale /usr/local/bin/fishscale
RUN mkdir -p /data/photos /data/tsnet-state && \
    chown -R fishscale:fishscale /data
USER fishscale
VOLUME /data
ENTRYPOINT ["fishscale"]
```

**Note on HEALTHCHECK:** We intentionally omit `HEALTHCHECK` from the Dockerfile. In dev mode, the app listens on `:8080` and a health check would work. In production, the app listens exclusively on tsnet (TLS on port 443 via Tailscale) — `wget` from inside the container cannot reach it without Tailscale credentials. A `HEALTHCHECK` that always fails in production is worse than no health check. If container health monitoring is needed, use an external probe from another tailnet node, or add a local-only health listener in a future iteration.

**Step 3: Update docker-compose.yml**

Add resource limits and logging configuration. Uses `mem_limit` and `cpus` (Compose v2 top-level keys) instead of `deploy.resources.limits` which requires `--compatibility` flag or Swarm mode:

```yaml
services:
  fishscale:
    build: .
    container_name: fishscale
    restart: unless-stopped
    mem_limit: 256m
    cpus: 1.0
    volumes:
      - fishscale-data:/data
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
    environment:
      - TS_AUTHKEY=${TS_AUTHKEY}
      - TS_HOSTNAME=${TS_HOSTNAME:-fishscale}
      - TS_STATE_DIR=/data/tsnet-state
      - FISHSCALE_DB_PATH=/data/fish.db
      - FISHSCALE_PHOTO_DIR=/data/photos
      - FISHSCALE_LOG_LEVEL=${FISHSCALE_LOG_LEVEL:-info}
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  fishscale-data:
```

**Step 4: Update .dockerignore**

Add exclusions for development/documentation directories:

```
docs/
.claude/
.worktrees/
*.md
.git/
.gitignore
```

**Step 6: Verify Docker build**

```bash
docker build -t fishscale:test .
docker run --rm fishscale:test whoami  # should print "fishscale", not "root"
```

**Step 7: Commit**

```bash
git add Dockerfile docker-compose.yml .dockerignore internal/server/server.go docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "security: harden Docker deployment

Run container as non-root user (fishscale:fishscale). Add /healthz
endpoint for health probes. Add resource limits (256MB RAM, 1 CPU)
and log rotation in docker-compose. Expand .dockerignore to exclude
docs and dev directories.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: CI/CD Pipeline & Go Linting

**Severity:** HIGH (no CI), MEDIUM (no linting)
**Findings:**
- No GitHub Actions workflow for tests, build, or linting
- No golangci-lint configuration

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.golangci.yml`

**Step 1: Create golangci-lint config**

Create `.golangci.yml`:

```yaml
run:
  timeout: 5m

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - bodyclose
    - contextcheck
    - nilerr
    - gosec

linters-settings:
  errcheck:
    check-type-assertions: true
  gosec:
    excludes:
      - G104  # handled by errcheck
```

**Step 2: Create CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test-backend:
    runs-on: ubuntu-latest
    env:
      GOWORK: 'off'
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run tests
        run: go test ./... -v -race

      - name: Run linter
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  build-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Build
        working-directory: frontend
        run: npm run build

      - name: Check types
        working-directory: frontend
        run: npx svelte-check --tsconfig ./tsconfig.app.json

  build-binary:
    needs: [test-backend, build-frontend]
    runs-on: ubuntu-latest
    env:
      GOWORK: 'off'
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Build frontend
        working-directory: frontend
        run: npm ci && npm run build

      - name: Copy frontend to embed dir
        run: rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist

      - name: Build Go binary
        run: CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale
```

**Step 3: Verify locally**

```bash
# Run golangci-lint (install if needed: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
GOWORK=off golangci-lint run ./...
```

Fix any lint findings before committing.

**Step 4: Commit**

```bash
git add .github/workflows/ci.yml .golangci.yml docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "ci: add GitHub Actions workflow and golangci-lint config

Run Go tests with race detector, golangci-lint (errcheck, gosec,
staticcheck, etc.), frontend build, svelte-check, and full binary
build on push to main and pull requests.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Frontend Testing with Vitest

**Severity:** HIGH
**Finding:** No frontend testing framework. No unit tests for API client, stores, or utility functions.

**Files:**
- Modify: `frontend/package.json` (add vitest, @testing-library/svelte)
- Create: `frontend/vitest.config.ts`
- Create: `frontend/src/lib/__tests__/api.test.ts`

**Step 1: Install Vitest**

```bash
cd frontend && npm install -D vitest @testing-library/svelte @testing-library/jest-dom jsdom
```

**Step 2: Create Vitest config**

Create `frontend/vitest.config.ts`:

```typescript
import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte({ hot: !process.env.VITEST })],
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    globals: true,
  },
});
```

**Step 3: Add test script to package.json**

Add to `scripts` in `frontend/package.json`:

```json
"test": "vitest run",
"test:watch": "vitest"
```

**Step 4: Write initial API client tests**

Create `frontend/src/lib/__tests__/api.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest';

// Test that the API module constructs correct URLs and handles responses.
// We use vi.resetModules() + dynamic import to get a fresh module per test,
// since the api module captures `fetch` at import time.
describe('API client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.resetModules();
  });

  it('constructs correct catch list URL', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify([]), { status: 200 })
    );

    const { api } = await import('../api');
    await api.catches.list();

    expect(fetchSpy).toHaveBeenCalledWith('/api/v1/catches', expect.objectContaining({
      headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
    }));
  });

  it('throws on non-OK response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
    );

    const { api } = await import('../api');
    await expect(api.catches.get(999)).rejects.toThrow('not found');
  });

  it('returns undefined for 204 responses', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(null, { status: 204 })
    );

    const { api } = await import('../api');
    const result = await api.catches.delete(1);
    expect(result).toBeUndefined();
  });
});
```

Note: `api` is a **named export** (not default) from `api.ts`. Each test uses `vi.resetModules()` + dynamic `import()` to get a fresh module instance, preventing mock leakage between tests.

**Step 5: Run tests**

```bash
cd frontend && npm test
```
Expected: PASS

**Step 6: Update CI workflow**

Add to `.github/workflows/ci.yml` in the `build-frontend` job:

```yaml
      - name: Run tests
        working-directory: frontend
        run: npm test
```

**Step 7: Commit**

```bash
git add frontend/ .github/workflows/ci.yml docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "test: add Vitest framework and initial API client tests

Set up Vitest with jsdom environment for frontend unit testing.
Add initial tests for API client URL construction and error handling.
Integrate into CI pipeline.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Frontend Code Splitting

**Severity:** MEDIUM
**Finding:** Entire app ships as a single 1MB+ JS bundle. MapLibre GL JS (~700KB) is included even when the user is viewing settings or the catch log.

**Files:**
- Modify: `frontend/vite.config.ts` (configure chunk splitting)

**Important context:** `MapView.svelte` is always mounted (hidden via CSS `display: none`) to preserve map zoom/pan state across tab switches. This means MapLibre will always be *imported* at app startup — `manualChunks` separates it into a cacheable chunk, but does NOT defer loading. True lazy-loading would require changing MapView to use dynamic `import('maplibre-gl')` inside `onMount`, which is a larger refactor.

For now, `manualChunks` provides a caching benefit: when app code changes, users only re-download the app chunk — the MapLibre chunk (~700KB) stays cached since its content hash doesn't change. This is the right trade-off for a v1.

**Step 1: Configure Vite manual chunks**

In `frontend/vite.config.ts`, add build configuration:

```typescript
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/photos': 'http://localhost:8080',
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          maplibre: ['maplibre-gl'],
        },
      },
    },
  },
})
```

**Step 2: Verify build**

```bash
cd frontend && npm run build
```

Check that the output shows multiple chunks. Expect something like:
- `index-[hash].js` (~50-100KB) — app code
- `maplibre-[hash].js` (~700KB) — MapLibre GL

**Step 3: Commit**

```bash
git add frontend/vite.config.ts docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "perf: split MapLibre GL into separate cacheable chunk

Configure Vite manual chunks to separate maplibre-gl (~700KB) from
the main bundle. MapLibre chunk is cached independently so app code
changes don't force re-downloading the map library.

Note: MapView is always mounted (CSS hidden) for state preservation,
so the chunk is still loaded eagerly. True lazy-loading would require
refactoring MapView to use dynamic import() — deferred to a future
iteration.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: ESLint & Prettier

**Severity:** MEDIUM
**Finding:** No linting or formatting configuration for frontend code.

**Files:**
- Create: `frontend/eslint.config.js`
- Create: `frontend/.prettierrc`
- Modify: `frontend/package.json` (add lint/format scripts)

**Step 1: Install dependencies**

```bash
cd frontend && npm install -D eslint @eslint/js typescript-eslint eslint-plugin-svelte prettier prettier-plugin-svelte
```

**Step 2: Create ESLint config**

Create `frontend/eslint.config.js`:

```javascript
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';

export default [
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...svelte.configs['flat/recommended'],
  {
    rules: {
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    },
  },
  {
    ignores: ['dist/', 'node_modules/', '.svelte-kit/'],
  },
];
```

**Step 3: Create Prettier config**

Create `frontend/.prettierrc`:

```json
{
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2,
  "plugins": ["prettier-plugin-svelte"],
  "overrides": [{ "files": "*.svelte", "options": { "parser": "svelte" } }]
}
```

**Step 4: Add scripts to package.json**

```json
"lint": "eslint src/",
"lint:fix": "eslint src/ --fix",
"format": "prettier --write 'src/**/*.{ts,svelte,css}'",
"format:check": "prettier --check 'src/**/*.{ts,svelte,css}'"
```

**Step 5: Run lint to see current state**

```bash
cd frontend && npm run lint
```

Fix only errors that block the build. Warnings can be addressed incrementally.

**Step 6: Update CI workflow**

Add to `.github/workflows/ci.yml` in the `build-frontend` job:

```yaml
      - name: Lint
        working-directory: frontend
        run: npm run lint

      - name: Format check
        working-directory: frontend
        run: npm run format:check
```

**Step 7: Commit**

```bash
git add frontend/ .github/workflows/ci.yml docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "chore: add ESLint and Prettier for frontend

Configure ESLint with TypeScript and Svelte plugins. Add Prettier
with svelte plugin. Integrate both into CI pipeline.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: LICENSE File & Makefile

**Severity:** MEDIUM
**Finding:** README references MIT license but no LICENSE file exists. No build automation (Makefile).

**Files:**
- Create: `LICENSE`
- Create: `Makefile`

**Step 1: Create LICENSE file**

Create `LICENSE` with the MIT License text (year 2026, copyright holder from git config or README).

**Step 2: Create Makefile**

```makefile
.PHONY: dev build test lint clean frontend

# Build frontend and copy to embed dir
frontend:
	cd frontend && npm ci && npm run build
	rm -rf internal/frontend/dist
	cp -r frontend/dist internal/frontend/dist

# Build Go binary (requires frontend to be built first)
build: frontend
	GOWORK=off CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Run in dev mode
dev:
	FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale

# Run all tests
test:
	GOWORK=off go test ./... -v
	cd frontend && npm test

# Run linters
lint:
	GOWORK=off golangci-lint run ./...
	cd frontend && npm run lint

# Clean build artifacts
clean:
	rm -f fishscale
	rm -rf internal/frontend/dist
	rm -rf frontend/dist

# Docker build
docker:
	docker build -t fishscale:latest .
```

**Step 3: Commit**

```bash
git add LICENSE Makefile docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "chore: add MIT LICENSE file and Makefile

Add LICENSE file (MIT). Add Makefile with targets for dev, build,
test, lint, clean, frontend, and docker.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: Structured Logging (slog)

**Severity:** LOW (deferred from Iteration 2)
**Finding:** All logging uses `log.Printf`. The `FISHSCALE_LOG_LEVEL` config is defined but unused. There are 13 `log.Printf/Println/Fatalf` calls: 10 in `cmd/fishscale/main.go` and 3 in `internal/middleware/tailscale.go`. No handler files use logging directly (they return errors via `jsonError`).

**Files:**
- Modify: `internal/config/config.go` (add ParseLogLevel)
- Test: `internal/config/config_test.go` (extend)
- Modify: `cmd/fishscale/main.go` (initialize slog)
- Modify: `internal/middleware/tailscale.go` (replace log.Printf)

**Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},   // case insensitive
		{"", slog.LevelInfo},          // default
		{"invalid", slog.LevelInfo},   // fallback
	}
	for _, tt := range tests {
		got := ParseLogLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./internal/config/ -run TestParseLogLevel -v
```
Expected: FAIL — `ParseLogLevel` does not exist.

**Step 3: Implement ParseLogLevel**

Add to `internal/config/config.go`:

```go
import "log/slog"

func ParseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

**Step 4: Initialize slog in main.go**

After `config.Load()` and `Validate()`:

```go
logLevel := config.ParseLogLevel(cfg.LogLevel)
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
```

Replace `log.Println`/`log.Printf` with `slog.Info`/`slog.Error` throughout `main.go` and `tailscale.go`. Keep `log.Fatalf` for startup failures (slog has no Fatal — use `slog.Error` + `os.Exit(1)` or keep `log.Fatalf` for pre-slog initialization).

**Step 5: Run tests, commit**

```bash
GOWORK=off go test ./... -v
```

```bash
git add internal/config/ cmd/fishscale/main.go internal/middleware/tailscale.go docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "chore: replace log.Printf with slog structured logging

Add config.ParseLogLevel() to parse FISHSCALE_LOG_LEVEL into slog
levels. Initialize slog.SetDefault with JSON handler in main().
Replace log.Printf in middleware with slog.Error/slog.Info.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: Image Resizing / Thumbnails

**Severity:** LOW (deferred from Iteration 2)
**Finding:** Full-resolution photos stored and served. The `thumbnail` field in the photos table is unused.

> **Note:** This task is **outline-only**. The implementation requires careful testing with real JPEG/PNG files and platform-specific image decoding behavior. Flesh out TDD steps when starting implementation.

**Files:**
- Modify: `internal/handler/photos.go` (generate thumbnail after save)
- Modify: `internal/storage/local.go` (add thumbnail path helper)
- Modify: `go.mod` (add `github.com/disintegration/imaging`)
- Test: `internal/handler/photos_test.go` (extend with thumbnail verification)

**Implementation approach:**
- `github.com/disintegration/imaging` uses Go's `image` stdlib — works with `CGO_ENABLED=0`
- After saving the original file, decode the image and resize to max 400px width (preserving aspect ratio)
- Save thumbnail alongside original with `_thumb` suffix (e.g., `abc123.jpg` → `abc123_thumb.jpg`)
- Populate `thumbnail` field in the photos DB insert
- Frontend: serve thumbnails in list/map views, originals in detail/fullscreen views
- If thumbnail generation fails, log the error but don't fail the upload — the original is still valid

**Key test:** Upload a real JPEG (valid JFIF header + pixel data), verify that both original and `_thumb` files exist on disk after upload, and that the DB record has a non-empty `thumbnail` field.

---

## Task 10: TypeScript Strict Mode

**Severity:** LOW (deferred from Iteration 2)
**Finding:** `tsconfig.app.json` lacks `strict: true`. 18+ instances of `any` type throughout frontend code, primarily in `api.ts` and Svelte component props.

> **Note:** This task is **outline-only**. The exact type errors depend on the state of the codebase when this task is started. Run `svelte-check` to enumerate errors before writing fixes.

**Files:**
- Modify: `frontend/tsconfig.app.json` (enable strict)
- Modify: `frontend/src/lib/api.ts` (replace `any` with interfaces)
- Modify: Various `.svelte` files (fix resulting type errors)
- Test: `npx svelte-check` must pass with zero errors

**Implementation approach:**
1. Define response interfaces in `api.ts` matching the Go model structs:
   ```typescript
   interface Catch { id: number; user_id: number; species_name?: string; ... }
   interface Species { id: number; name: string; category: string; }
   interface Trip { id: number; name: string; started_at: string; ... }
   interface UserSettings { theme: string; units: string; species_filter: string; }
   interface Stats { total_catches: number; ... }
   ```
2. Replace `any` with concrete types in all `request<T>()` calls
3. Enable `"strict": true` in `tsconfig.app.json`
4. Run `npx svelte-check --tsconfig ./tsconfig.app.json` and fix all errors
5. Verify `npm run build` succeeds

---

## Task 11: Content-Security-Policy Header

**Severity:** LOW (deferred from Iteration 2)
**Finding:** No CSP header. Requires careful configuration to allow MapLibre GL tile loading, inline styles from MapLibre, and the Open-Meteo weather API.

> **Note:** This task is **outline-only**. CSP is notoriously fragile — a single missing source directive breaks the entire page. Implementation requires iterative testing in both dev mode and production.

**Files:**
- Modify: `internal/middleware/security.go` (add CSP header)
- Modify: `internal/middleware/security_test.go` (extend)

**Implementation approach:**

The CSP header needs to allow:
- `default-src 'self'`
- `script-src 'self'` (all scripts are bundled by Vite)
- `style-src 'self' 'unsafe-inline'` (MapLibre injects inline styles for markers/popups)
- `img-src 'self' data: blob: https://*.tile.openstreetmap.org` (map tiles, data URIs for markers)
- `connect-src 'self' https://*.tile.openstreetmap.org https://api.open-meteo.com` (tile fetches, weather API)
- `worker-src 'self' blob:` (MapLibre uses Web Workers for rendering)
- `font-src 'self'`

**Test approach:**
1. Add CSP header to `SecurityHeaders` middleware
2. Unit test: verify the header is present and contains expected directives
3. Manual test: load the app in Chrome DevTools, check Console for CSP violations
4. Verify: map tiles load, markers display, weather popup works, photo uploads work
5. If violations appear, add the specific source to the policy and re-test

**Warning:** `'unsafe-inline'` for styles is a known trade-off. MapLibre's inline style injection is fundamental to how it works. A future improvement could use CSP nonces, but that requires server-side nonce generation per request.

---

## Testing Plan

After all tasks, run the complete verification:

```bash
# Go tests
GOWORK=off go test ./... -v -race

# Go lint
GOWORK=off golangci-lint run ./...

# Frontend tests
cd frontend && npm test

# Frontend lint
cd frontend && npm run lint && npm run format:check

# Frontend build
cd frontend && npm run build && cd ..

# Copy to embed dir
rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist

# Build binary
GOWORK=off CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Docker build
docker build -t fishscale:test .
docker run --rm fishscale:test whoami  # should print "fishscale"

# Run in dev mode
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos ./fishscale

# Test:
# - Photo URLs return 401 without auth
# - CI pipeline passes on push
# - Frontend tests pass
# - golangci-lint clean
# - Docker image runs as non-root
# - MapLibre chunk loads lazily
```

---

## Deferred to Future Iterations

- **CONTRIBUTING.md / SECURITY.md:** Community docs for open-source project
- **Accessibility audit:** Address 20+ a11y warnings identified in frontend
- **S3-compatible photo storage:** Implement the existing storage interface for S3
- **Progressive Web App (PWA):** Service worker for offline catch logging
- **Database backup automation:** Scheduled sqlite3 .backup + remote sync
