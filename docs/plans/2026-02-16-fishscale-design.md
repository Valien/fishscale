# Fishscale - Design Document

**Date:** 2026-02-16
**Status:** Approved

## Overview

Fishscale is an open-source, self-hosted fishing tracker app. It runs on a user's Tailnet via Tailscale's `tsnet` library, accessible from any device with a browser. It is deployed as a Docker image with a SQLite backend.

This is a personal tracking and logging tool for fishermen to record where they fish, what they catch, how they catch fish, and what techniques were used. It is not a social media platform. A future v2 may add optional sharing via Tailscale Funnel or social media for individual catches.

## Tech Stack

- **Backend:** Go with `tsnet` for Tailscale networking, `chi` router
- **Frontend:** Svelte + Vite, compiled and embedded into the Go binary via `embed.FS`
- **Database:** SQLite
- **Maps:** MapLibre GL JS + OpenStreetMap (with provider abstraction for future swap to Google Maps, etc.)
- **Weather:** Open-Meteo API (free, no API key required) for auto-fetching conditions
- **Photo Storage:** Local filesystem (with storage interface for future S3-compatible backend)
- **Auth:** Tailscale WhoIs API (no login screen, network is the auth boundary)

## Architecture

Single Go binary with everything embedded. Serves the Svelte SPA and JSON API over tsnet. Deployed as a minimal Docker container.

```
┌─────────────────────────────────────────────────┐
│                 Docker Container                 │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │           Go Binary (fishscale)            │  │
│  │                                            │  │
│  │  ┌──────────┐  ┌───────────────────────┐   │  │
│  │  │  tsnet   │──│   HTTP Router (chi)   │   │  │
│  │  │ listener │  │                       │   │  │
│  │  └──────────┘  │  /api/catches         │   │  │
│  │                │  /api/locations        │   │  │
│  │  ┌──────────┐  │  /api/species         │   │  │
│  │  │ embed.FS │──│  /api/weather         │   │  │
│  │  │ (Svelte) │  │  /api/photos          │   │  │
│  │  └──────────┘  │  /api/export          │   │  │
│  │                │  /* (SPA fallback)     │   │  │
│  │                └───────────────────────┘   │  │
│  │                                            │  │
│  │  ┌──────────────┐  ┌───────────────────┐   │  │
│  │  │   SQLite DB   │  │  Photo Storage   │   │  │
│  │  │ /data/fish.db │  │  /data/photos/   │   │  │
│  │  └──────────────┘  └───────────────────┘   │  │
│  └────────────────────────────────────────────┘  │
│                                                  │
│  Volumes: /data (DB + photos + tsnet state)      │
└─────────────────────────────────────────────────┘
```

### Key Architecture Decisions

- **Embedded SPA:** Single artifact, trivial deployment, no reverse proxy needed. Frontend rebuilds require recompiling the Go binary (mitigated by dev mode serving from disk).
- **SPA cache headers:** `index.html` is served with `Cache-Control: no-cache, no-store, must-revalidate` so deploys take effect immediately (critical for Safari which aggressively caches HTML). Vite's hashed assets under `/assets/` are served with `Cache-Control: public, max-age=31536000, immutable`.
- **Provider interfaces:** Map provider and photo storage have abstraction layers so defaults (MapLibre, local filesystem) can be swapped for alternatives (Google Maps, S3) without architectural changes.
- **No ports exposed:** tsnet handles all networking within the Tailnet. No public internet exposure.

## Data Model

### users
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| tailscale_id | TEXT UNIQUE NOT NULL | Stable Tailscale user ID |
| display_name | TEXT NOT NULL | From Tailscale profile |
| created_at | DATETIME | Auto-populated |

### trips (optional grouping of catches)
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | INTEGER FK | References users(id) |
| name | TEXT | e.g., "Saturday at Lake Fork" |
| started_at | DATETIME NOT NULL | |
| ended_at | DATETIME | |
| notes | TEXT | |
| created_at | DATETIME | Auto-populated |

### catches (core record)
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | INTEGER FK | References users(id) |
| trip_id | INTEGER FK | References trips(id), nullable (standalone catch) |
| species_id | INTEGER FK | References species(id) |
| caught_at | DATETIME NOT NULL | Auto-filled, editable |
| latitude | REAL NOT NULL | |
| longitude | REAL NOT NULL | |
| location_name | TEXT | e.g., "Lake Fork, boat ramp cove" |
| length_in | REAL | Inches (or cm if metric) |
| weight_lb | REAL | Pounds (or kg if metric) |
| kept | BOOLEAN | Default FALSE (released) |
| bait_or_lure | TEXT | Freeform with autocomplete from history |
| rod_setup | TEXT | e.g., "7' MH spinning" |
| line_info | TEXT | e.g., "15lb braid + 12lb fluoro leader" |
| hook_size | TEXT | e.g., "3/0 EWG" |
| air_temp_f | REAL | Auto-fetched from Open-Meteo, editable |
| wind_mph | REAL | Auto-fetched |
| wind_dir | TEXT | "NW", "SE", etc. |
| conditions | TEXT | "Partly Cloudy", "Rain", etc. |
| pressure_mb | REAL | Auto-fetched |
| humidity_pct | REAL | Auto-fetched |
| water_temp_f | REAL | Manual entry |
| water_clarity | TEXT | "Clear", "Stained", "Muddy" |
| notes | TEXT | |
| created_at | DATETIME | Auto-populated |
| updated_at | DATETIME | Auto-populated |

**Note:** Values are stored in their original unit system. Conversion to imperial/metric is handled at the display layer based on user settings.

### species (pre-seeded + user-addable)
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| name | TEXT UNIQUE NOT NULL | e.g., "Largemouth Bass" |
| category | TEXT | "Freshwater", "Saltwater" |

### photos (multiple per catch)
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| catch_id | INTEGER FK | References catches(id) ON DELETE CASCADE |
| filename | TEXT NOT NULL | Path relative to /data/photos/ |
| thumbnail | TEXT | Path to generated thumbnail |
| sort_order | INTEGER | Default 0 |
| created_at | DATETIME | Auto-populated |

### user_settings (per-user preferences)
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | INTEGER UNIQUE FK | References users(id) |
| theme | TEXT | 'light', 'dark', 'system' (default: 'system') |
| units | TEXT | 'imperial', 'metric' (default: 'imperial') |
| species_filter | TEXT | 'all', 'freshwater', 'saltwater' (default: 'all') |
| updated_at | DATETIME | Auto-populated |

### Design Notes

- **Trips are optional.** A catch can be standalone or grouped under a trip. Quick capture does not require creating a trip first.
- **Gear fields are freeform text** with autocomplete populated from the user's history. Avoids complex gear management while making repeat entries fast.
- **Species table is pre-seeded** with common freshwater/saltwater fish. Users can add custom entries.
- **Photos cascade delete.** Deleting a catch removes its photos from the DB and triggers cleanup of files on disk.
- **Deleting a trip unlinks catches** (sets trip_id = NULL) rather than deleting them.

## API Design

RESTful JSON API. All routes prefixed with `/api/v1`. Every request authenticated via tsnet WhoIs middleware -- user ID extracted from Tailscale identity automatically.

### Endpoints

```
Auth (automatic via middleware):
  WhoIs lookup -> user_id injected into request context

Catches:
  GET    /api/v1/catches              -- list (filterable: species, date range, location)
  GET    /api/v1/catches/:id          -- single catch with photos
  POST   /api/v1/catches              -- create (multipart: JSON fields + photo files)
  PUT    /api/v1/catches/:id          -- update
  DELETE /api/v1/catches/:id          -- delete (cascades photos)

Trips:
  GET    /api/v1/trips                -- list
  GET    /api/v1/trips/:id            -- single trip with its catches
  POST   /api/v1/trips                -- create
  PUT    /api/v1/trips/:id            -- update
  DELETE /api/v1/trips/:id            -- delete (unlinks catches, does not delete them)

Photos:
  POST   /api/v1/catches/:id/photos   -- add photos to existing catch
  DELETE /api/v1/photos/:id            -- remove single photo

Species:
  GET    /api/v1/species               -- list (searchable via ?q=)
  POST   /api/v1/species               -- add custom species

Weather:
  GET    /api/v1/weather?lat=&lon=     -- proxy to Open-Meteo, returns current conditions

Settings:
  GET    /api/v1/settings              -- get current user's settings
  PUT    /api/v1/settings              -- update theme, units, etc.

Export:
  GET    /api/v1/export?format=csv     -- download catch log as CSV
  GET    /api/v1/export?format=json    -- download catch log as JSON

Stats:
  GET    /api/v1/stats                 -- total catches, top species, catches by month, etc.
```

### API Notes

- Catch creation accepts multipart form data so photos upload in the same request -- one tap to submit from mobile.
- Weather endpoint is a thin proxy so the frontend avoids CORS issues and API provider details stay server-side.
- Export respects the user's unit preference.

## Frontend / UX Design

### Guiding Principle

**"Capture in 30 seconds, detail later."**

### Navigation

Bottom tab bar on mobile, sidebar on desktop. Four main views plus the primary action:

- **[+] Log Catch** -- always accessible, primary action button
- **Map** -- full-screen map with catch pins
- **Log** -- chronological list of catches
- **Stats** -- dashboard with basic analytics
- **Settings** -- theme, units, profile

### Quick Capture Flow

The most important screen in the app. Optimized for speed on the water.

**Quick fields (shown immediately):**
- Date/time set to now, editable
- GPS location auto-detected, tap to adjust pin on map
- Species: searchable autocomplete from species list (custom touch-friendly dropdown, not native datalist)
- Bait/Lure: autocomplete from user's history
- Kept / Released: toggle
- Photo: optional camera/file picker button within the form (uses `accept="image/*" capture="environment"` for native camera on mobile)
- Weather auto-fetched from Open-Meteo based on GPS coordinates, shown as preview chip
- **[Save]** and **[+ More Detail]** buttons

**"+ More Detail" expands (all optional):**
- Length, Weight
- Rod setup, Line info, Hook size
- Water temp, Water clarity
- Trip assignment (none / select existing / create new)
- Notes

**Key UX decisions:**
- Photo is optional, not the entry point. Camera/file picker is a button in the form, not the first thing shown. This avoids blocking quick logging when you just want to record a catch fast.
- Auto-fill aggressively. GPS, time, and weather are pre-populated.
- Autocomplete from history. Bait/lure and gear fields learn from previous entries.
- Two-tier form. Quick save needs only species + bait + kept/released. Everything else is behind "More Detail."
- Save is always visible. You can save with just auto-filled data if in a rush.
- Species dropdown must work on iOS Safari. Uses `ontouchend` on dropdown items (fires reliably on finger lift), `onmousedown` for desktop, and `onblur` with a 200ms delay on the input for dismiss. A `cancelDismiss` on `ontouchstart` prevents the blur timer from hiding the dropdown before touch completes. No backdrop overlay (caused z-index stacking issues on Safari). The `justSelected` flag is a plain `let` (not `$state`) to avoid Svelte 5 `$effect` re-trigger loops.

### Map View

Full-screen MapLibre GL map showing catch pins. Pins color-coded by species or sized by weight (user toggle). Tapping a pin shows a summary card with photo thumbnail, species, date, and link to full catch detail.

The map instance is preserved across tab switches so zoom/pan position is remembered. On first load, the map shows a continental US view, then instantly fits to catch bounds if catches exist. A locate-me button (MapLibre GeolocateControl) in the top-right lets the user center on their GPS position on demand. No automatic GPS animations on load — positioning is either instant (catch bounds) or user-initiated (locate button).

Filter panel slides in: date range, species, bait, trip.

### Log View

Chronological list, most recent first. Each card shows thumbnail photo, species, size/weight, location name, date, and bait/lure used. Searchable and filterable.

### Stats Dashboard (v1)

- Total catches (all time / this year / this month)
- Top 5 species by count
- Catches by month bar chart
- Personal bests (biggest by species)
- Most-used baits/lures

### Settings

- Theme: Light / Dark / System
- Units: Imperial / Metric
- Species filter: All / Freshwater / Saltwater (controls which species appear in the species autocomplete when logging a catch; default: All)
- User profile display (from Tailscale identity, read-only)

## Docker & Deployment

### Docker Image (multi-stage build)

```dockerfile
# Stage 1: Build Svelte frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/ .
RUN npm ci && npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN go build -o fishscale ./cmd/fishscale

# Stage 3: Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend /app/fishscale /usr/local/bin/fishscale
VOLUME /data
ENTRYPOINT ["fishscale"]
```

Final image: ~30-40MB.

### Docker Compose

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
      - TS_HOSTNAME=fishscale
      - TS_STATE_DIR=/data/tsnet-state
      - FISHSCALE_DB_PATH=/data/fish.db
      - FISHSCALE_PHOTO_DIR=/data/photos
      - FISHSCALE_LOG_LEVEL=info

volumes:
  fishscale-data:
```

### Configuration

All config via environment variables with sensible defaults:

| Variable | Default | Description |
|---|---|---|
| TS_AUTHKEY | (required) | Tailscale auth key |
| TS_HOSTNAME | fishscale | Device name on Tailnet |
| TS_STATE_DIR | /data/tsnet-state | Tailscale state persistence |
| FISHSCALE_DB_PATH | /data/fish.db | SQLite database path |
| FISHSCALE_PHOTO_DIR | /data/photos | Photo storage directory |
| FISHSCALE_LOG_LEVEL | info | debug, info, warn, error |

### First-Run Experience

1. User generates a Tailscale auth key (one-time or reusable)
2. `docker compose up -d`
3. tsnet authenticates, "fishscale" appears on their Tailnet
4. Browse to `https://fishscale.<tailnet-name>.ts.net`
5. WhoIs identifies the user, user record auto-created, immediately in the app

No ports exposed. No reverse proxy. No DNS config.

### Backup

Everything lives under `/data`:
- SQLite DB: `/data/fish.db`
- Photos: `/data/photos/`
- Tailscale state: `/data/tsnet-state/`

A single `tar` of the Docker volume or a scheduled `sqlite3 .backup` + `rsync` covers it.

## Security

### Authentication & Authorization
- All requests authenticated via Tailscale WhoIs middleware (network is the auth boundary)
- Every database query scoped to `user_id` from authenticated context — no cross-user access
- Dev mode uses a hardcoded user (ID 1) — development only, never production
- Config validation requires `TS_AUTHKEY` in production mode (fails fast at startup)

### Request Security
- Security headers on all responses: X-Frame-Options DENY, X-Content-Type-Options nosniff, Referrer-Policy, Permissions-Policy
- Rate limiting: 100 requests/minute per IP via go-chi/httprate
- Request context propagated to all database calls — canceled client connections abort queries
- Input validation: string length limits (200 short / 2000 text / 5000 notes), coordinate range checks, positive numeric enforcement
- All handler errors checked and returned as proper HTTP status codes (no silently ignored errors)

### File Upload Security
- MIME type validated via magic bytes (first 512 bytes), not file extension
- Allowed types: image/jpeg, image/png, image/gif, image/webp
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
- Indexes on all foreign key columns for efficient joins and filtering

### Caching
- index.html: Cache-Control no-cache, no-store, must-revalidate (always fresh)
- /assets/*: Cache-Control public, max-age=31536000, immutable (Vite content-hashed)

## Future Considerations (v2+)

These are explicitly out of scope for v1 but the architecture is designed to accommodate them:

- **Fish species auto-ID from photos:** Hook point reserved in the photo upload handler. Most practical path is an optional ONNX model running server-side behind a feature flag.
- **Sharing via Tailscale Funnel:** tsnet already supports Funnel. Sharing a single catch page publicly is a flag flip when ready.
- **Social media sharing:** Generate a shareable image/card for a catch record.
- **S3-compatible photo storage:** Storage interface is abstracted. Add an S3 backend implementation.
- **Alternative map providers:** Map provider is abstracted behind an interface. Swap MapLibre for Google Maps or others.
- **Rich analytics:** Weather correlation, heatmaps, moon phase tracking, seasonal pattern analysis. The data model already captures enough to support this.
- **Import from CSV/JSON:** v1 supports export only. Import can be added using the same format.

## Ideas & Enhancements

Quick capture list for future work. Move items to "Future Considerations" once scoped, or into an iteration plan when ready to build.

- [ ] Photo picker should open device photo album by default instead of camera, so users can upload pictures already taken (remove `capture="environment"` from file input, keep `accept="image/*"`)
- [ ] Run security iteration plan 3
- [ ] Determine whether CI/CD with GitHub Actions is needed
- [ ] Catch log entries should be clickable/editable — tap to view full catch details, edit fields inline
- [ ] Redo bottom nav icons (map, log, stats, settings) — current icons need improvement
- [ ] Investigate species dropdown — still broken, may be overcomplicating it. Look for simpler alternatives or remove entirely for now
- [ ] Fish-log page should remember user's location and auto-update without prompting
- [ ] Investigate cancel button on fish-log page — may not be working
- [ ]
- [ ]
