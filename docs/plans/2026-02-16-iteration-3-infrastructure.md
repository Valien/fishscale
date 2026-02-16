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
| 1 | Photo serving has no authentication | HIGH | Task 1 |
| 2 | Docker container runs as root | HIGH | Task 2 |
| 3 | No CI/CD pipeline | HIGH | Task 3 |
| 4 | No frontend testing framework | HIGH | Task 4 |
| 5 | Base images not pinned to patch versions | MEDIUM | Task 2 |
| 6 | No Docker HEALTHCHECK | MEDIUM | Task 2 |
| 7 | No resource limits in docker-compose | MEDIUM | Task 2 |
| 8 | Single 1MB+ JS bundle, no code splitting | MEDIUM | Task 5 |
| 9 | No ESLint/Prettier configuration | MEDIUM | Task 6 |
| 10 | No LICENSE file (MIT mentioned in README only) | MEDIUM | Task 7 |
| 11 | No Makefile for build automation | MEDIUM | Task 7 |
| 12 | No golangci-lint configuration | MEDIUM | Task 3 |
| 13 | .dockerignore could exclude more directories | LOW | Task 2 |
| 14 | No CONTRIBUTING.md or SECURITY.md | LOW | Deferred |

**Deferred from Iteration 2:**

| # | Category | Severity | Status |
|---|----------|----------|--------|
| 15 | Structured logging (slog) | LOW | Task 8 |
| 16 | Image resizing / thumbnails | LOW | Task 9 |
| 17 | TypeScript strict mode | LOW | Task 10 |
| 18 | Content-Security-Policy header | LOW | Task 11 |

---

## Task 1: Authenticate Photo Serving

**Severity:** HIGH
**Finding:** `GET /photos/*` in `internal/server/server.go` serves photos directly via `http.FileServer` with no authentication middleware. Any device on the tailnet can access any photo by guessing or enumerating filenames.

**Files:**
- Modify: `internal/server/server.go`
- Test: `internal/server/server_test.go` (extend or create)

**Step 1: Write the failing test**

```go
func TestPhotoServingRequiresAuth(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	photoDir := dir + "/photos"
	os.MkdirAll(photoDir, 0o755)
	os.WriteFile(filepath.Join(photoDir, "test.jpg"), []byte("fake image"), 0o644)

	cfg := &config.Config{PhotoDir: photoDir, DevMode: true}
	store := storage.NewLocalStore(cfg.PhotoDir)

	// Router WITHOUT auth middleware (simulates unauthenticated request)
	router := NewRouter(cfg, db, store, nil)

	req := httptest.NewRequest("GET", "/photos/test.jpg", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Without auth, should get 401, not 200
	if rec.Code == http.StatusOK {
		t.Error("expected photo serving to require auth, got 200 without auth middleware")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
GOWORK=off go test ./internal/server/ -run TestPhotoServingRequiresAuth -v
```
Expected: FAIL — currently returns 200 without auth.

**Step 3: Implement fix**

In `internal/server/server.go`, move the photo serving route inside the authenticated route group (where the auth middleware is applied), or wrap it with the auth middleware directly:

```go
// BEFORE (outside auth group):
r.Get("/photos/*", http.StripPrefix("/photos/", http.FileServer(http.Dir(cfg.PhotoDir))).ServeHTTP)

// AFTER (inside the auth-protected group):
r.Group(func(r chi.Router) {
    if authMiddleware != nil {
        r.Use(authMiddleware)
    }
    // ... existing API routes ...
    r.Get("/photos/*", http.StripPrefix("/photos/", http.FileServer(http.Dir(cfg.PhotoDir))).ServeHTTP)
})
```

**Step 4: Run test to verify it passes**

```bash
GOWORK=off go test ./internal/server/ -run TestPhotoServingRequiresAuth -v
```
Expected: PASS

**Step 5: Run full test suite**

```bash
GOWORK=off go test ./... -v
```
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/server/ docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "security: require authentication for photo serving

Move /photos/* route inside the auth-protected route group.
Previously, any tailnet device could access photos without
authentication by guessing filenames.

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
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `.dockerignore`

**Step 1: Update Dockerfile**

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
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/v1/species || exit 1
ENTRYPOINT ["fishscale"]
```

Note: HEALTHCHECK only works in dev mode (port 8080). In production, the app listens on tsnet. A more sophisticated health check could be added in the future — for now this catches container crashes.

**Step 2: Update docker-compose.yml**

Add resource limits and logging configuration:

```yaml
services:
  fishscale:
    image: fishscale:latest
    container_name: fishscale
    restart: unless-stopped
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
      - FISHSCALE_LOG_LEVEL=info
    deploy:
      resources:
        limits:
          memory: 256M
          cpus: '1.0'
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  fishscale-data:
```

**Step 3: Update .dockerignore**

Add exclusions for development/documentation directories:

```
docs/
.claude/
.worktrees/
*.md
.git/
.gitignore
```

**Step 4: Verify Docker build**

```bash
docker build -t fishscale:test .
docker run --rm fishscale:test whoami  # should print "fishscale", not "root"
```

**Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml .dockerignore docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "security: harden Docker deployment

Run container as non-root user (fishscale:fishscale). Add
HEALTHCHECK instruction for container orchestration. Add resource
limits (256MB RAM, 1 CPU) and log rotation in docker-compose.
Expand .dockerignore to exclude docs and dev directories.

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
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run tests
        run: GOWORK=off go test ./... -v -race

      - name: Run linter
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
        env:
          GOWORK: 'off'

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
        run: GOWORK=off CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale
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

// Test that the API module constructs correct URLs and handles responses
describe('API client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('constructs correct catch list URL', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify([]), { status: 200 })
    );

    const { default: api } = await import('../api');
    await api.catches.list();

    expect(fetchSpy).toHaveBeenCalledWith('/api/v1/catches', expect.any(Object));
  });

  it('throws on non-OK response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
    );

    const { default: api } = await import('../api');
    await expect(api.catches.get(999)).rejects.toThrow();
  });
});
```

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
**Finding:** Entire app ships as a single 1MB+ JS bundle. Map page loads MapLibre GL JS even when viewing settings.

**Files:**
- Modify: `frontend/src/App.svelte` (lazy-load pages)
- Modify: `frontend/vite.config.ts` (configure chunk splitting)

**Step 1: Configure Vite manual chunks**

In `frontend/vite.config.ts`, add build configuration:

```typescript
export default defineConfig({
  // ... existing config ...
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          maplibre: ['maplibre-gl'],
        },
      },
    },
  },
});
```

This separates MapLibre GL (~700KB) into its own chunk that only loads when the map page is visited.

**Step 2: Verify build**

```bash
cd frontend && npm run build
```

Check that the output shows multiple chunks with MapLibre separated.

**Step 3: Commit**

```bash
git add frontend/vite.config.ts docs/plans/2026-02-16-iteration-3-infrastructure.md
git commit -m "perf: split MapLibre GL into separate chunk

Configure Vite manual chunks to separate maplibre-gl (~700KB) from
the main bundle. The map chunk only loads when the map page is visited.

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
**Finding:** All logging uses `log.Printf`. The `FISHSCALE_LOG_LEVEL` config is defined but unused.

**Files:**
- Modify: `cmd/fishscale/main.go`
- Modify: `internal/config/config.go`
- Modify: `internal/middleware/tailscale.go`
- Modify: `internal/handler/*.go` (where log.Printf is used)

**Step 1: Write the failing test**

Add to config test:

```go
func TestLogLevelParsing(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},       // default
		{"invalid", slog.LevelInfo}, // fallback
	}
	for _, tt := range tests {
		got := ParseLogLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Implement**

Add `ParseLogLevel` to config package. In `main.go`, initialize `slog.SetDefault()` with a JSON handler using the parsed log level. Replace `log.Printf` calls in middleware and handlers with `slog.Info`, `slog.Error`, etc.

**Step 3: Run tests, commit**

```bash
GOWORK=off go test ./... -v
```

---

## Task 9: Image Resizing / Thumbnails

**Severity:** LOW (deferred from Iteration 2)
**Finding:** Full-resolution photos stored and served. The `thumbnail` field in the photos table is unused.

**Files:**
- Modify: `internal/handler/photos.go`
- Modify: `internal/storage/local.go`
- Modify: `go.mod` (add imaging library)

**Implementation notes:**
- Use `github.com/disintegration/imaging` for resize
- On upload, after MIME validation, generate a 400px-wide thumbnail
- Save thumbnail alongside original with `_thumb` suffix
- Populate `thumbnail` field in DB
- Serve thumbnails in list views, originals in detail views

This task requires careful testing with real image files. Plan specific test approach when implementing.

---

## Task 10: TypeScript Strict Mode

**Severity:** LOW (deferred from Iteration 2)
**Finding:** `tsconfig.app.json` lacks `strict: true`. 18+ instances of `any` type throughout frontend code.

**Files:**
- Modify: `frontend/tsconfig.app.json`
- Modify: `frontend/src/lib/api.ts` (replace `any` with interfaces)
- Modify: Various `.svelte` files (fix type errors)

**Implementation notes:**
- Enable `strict: true` in tsconfig.app.json
- Define response interfaces in `api.ts` for all API endpoints
- Replace `any` with proper types throughout
- Run `npx svelte-check` to find all remaining type errors

---

## Task 11: Content-Security-Policy Header

**Severity:** LOW (deferred from Iteration 2)
**Finding:** No CSP header. Requires careful configuration to allow MapLibre GL tile loading, inline styles from MapLibre, and the Open-Meteo weather API.

**Files:**
- Modify: `internal/middleware/security.go`
- Modify: `internal/middleware/security_test.go`

**Implementation notes:**
- Allow `*.tile.openstreetmap.org` for map tiles
- Allow `api.open-meteo.com` for weather
- Allow `'unsafe-inline'` for styles (MapLibre injects inline styles)
- Allow `blob:` for MapLibre WebGL
- Test thoroughly in both dev mode and production

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
