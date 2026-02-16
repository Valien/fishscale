# Fishscale

A self-hosted fishing tracker. Log catches, view them on a map, track personal bests, and get weather data — all from a single binary accessible over your Tailscale network.

## Features

- **Catch logging** with GPS coordinates, species, bait/lure, weight, length, and photos
- **Interactive map** showing all catch locations with locate-me button (MapLibre GL + OpenStreetMap)
- **Automatic weather** fetched from Open-Meteo at time of logging (no API key needed)
- **Statistics dashboard** with species breakdown, personal bests, top baits, and monthly trends
- **Trip tracking** to group catches by outing
- **Export** your data as JSON or CSV
- **44 pre-loaded species** covering freshwater, saltwater, and fly fishing
- **Light/dark/system theme** with imperial or metric units
- **Single binary** — Go backend with embedded Svelte frontend, no external services
- **SQLite database** — zero configuration, WAL mode for performance
- **Tailscale authentication** — accessible only on your tailnet, user identity via WhoIs

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go, chi router, sqlx |
| Database | SQLite (modernc.org/sqlite, pure Go, no CGO) |
| Frontend | Svelte 5, TypeScript, Vite, MapLibre GL JS |
| Auth | Tailscale tsnet |
| Weather | Open-Meteo API |

## Quick Start with Docker

1. Generate a [Tailscale auth key](https://login.tailscale.com/admin/settings/keys) (reusable, with appropriate tags).

2. Create a `.env` file:

```
TS_AUTHKEY=tskey-auth-...
TS_HOSTNAME=fishscale
```

3. Run:

```bash
docker compose up -d
```

Fishscale will be available at `https://fishscale.<your-tailnet>.ts.net`.

## Development

### Prerequisites

- Go 1.25+
- Node.js 22+

### Run locally

```bash
# Install frontend dependencies and build
cd frontend && npm install && npm run build && cd ..

# Copy built frontend into embed directory
cp -r frontend/dist internal/frontend/dist

# Run in dev mode (no Tailscale required, listens on :8080)
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos go run ./cmd/fishscale
```

Open http://localhost:8080.

For frontend development with hot reload, run the Vite dev server separately:

```bash
# Terminal 1: Go backend
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos go run ./cmd/fishscale

# Terminal 2: Vite dev server (proxies API requests to :8080)
cd frontend && npm run dev
```

### Run tests

```bash
go test ./... -v
```

### Build the binary

```bash
cd frontend && npm run build && cd ..
cp -r frontend/dist internal/frontend/dist
CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale
```

## Configuration

All configuration is through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `TS_AUTHKEY` | *(required in production)* | Tailscale auth key |
| `TS_HOSTNAME` | `fishscale` | Tailscale hostname |
| `TS_STATE_DIR` | `/data/tsnet-state` | Tailscale state directory |
| `FISHSCALE_DB_PATH` | `/data/fish.db` | SQLite database path |
| `FISHSCALE_PHOTO_DIR` | `/data/photos` | Photo storage directory |
| `FISHSCALE_LOG_LEVEL` | `info` | Log level |
| `FISHSCALE_DEV_MODE` | `false` | Enable dev mode (HTTP on :8080, no Tailscale) |

## API

All endpoints are under `/api/v1`. Responses are JSON.

### Catches

```
GET    /api/v1/catches          List catches (?species_id=&trip_id=&q=)
POST   /api/v1/catches          Create a catch
GET    /api/v1/catches/:id      Get a catch
PUT    /api/v1/catches/:id      Update a catch
DELETE /api/v1/catches/:id      Delete a catch
POST   /api/v1/catches/:id/photos  Upload a photo (multipart)
```

### Trips

```
GET    /api/v1/trips            List trips
POST   /api/v1/trips            Create a trip
GET    /api/v1/trips/:id        Get a trip
PUT    /api/v1/trips/:id        Update a trip
DELETE /api/v1/trips/:id        Delete a trip
```

### Other

```
GET    /api/v1/species          List species (?q= for search)
POST   /api/v1/species          Add custom species
DELETE /api/v1/photos/:id       Delete a photo
GET    /api/v1/settings         Get user settings
PUT    /api/v1/settings         Update settings (theme, units)
GET    /api/v1/weather          Get weather (?lat=&lon=)
GET    /api/v1/stats            Get statistics
GET    /api/v1/export           Export data (?format=json|csv)
```

### Examples

Create a catch:

```bash
curl -X POST http://localhost:8080/api/v1/catches \
  -H 'Content-Type: application/json' \
  -d '{
    "caught_at": "2026-02-16T10:30:00Z",
    "latitude": 32.7767,
    "longitude": -96.797,
    "location_name": "Lake Fork - Cove",
    "bait_or_lure": "Senko",
    "kept": true
  }'
```

Get weather for a location:

```bash
curl 'http://localhost:8080/api/v1/weather?lat=32.77&lon=-96.79'
# {"air_temp_f":54.9,"wind_mph":7.6,"wind_dir":"SSE","conditions":"Partly Cloudy","pressure_mb":1000.6,"humidity_pct":89}
```

Export as CSV:

```bash
curl 'http://localhost:8080/api/v1/export?format=csv' -o catches.csv
```

## Project Structure

```
cmd/fishscale/          Entry point
internal/
  config/               Environment variable configuration
  database/             SQLite connection, migrations, seed data
  frontend/             Embedded SPA assets (go:embed)
  handler/              HTTP handlers for all API endpoints
  middleware/            Tailscale auth and dev-mode auth
  model/                Data model structs
  server/               Router assembly and SPA serving
  storage/              Photo storage interface and local filesystem impl
frontend/               Svelte 5 + TypeScript + Vite
  src/lib/
    api.ts              Typed API client
    components/         BottomNav, shared UI
    pages/              LogCatch, MapView, CatchLog, Stats, Settings
    stores/             Svelte stores for catches and settings
    theme.css           CSS custom properties for theming
```

## License

MIT
