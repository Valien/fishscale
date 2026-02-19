# Fishscale

A self-hosted fishing tracker. Log catches, view them on a map, track personal bests, and get weather data — all from a single binary accessible over your Tailscale network.

## Author's note (not AI written..rest of this README is with my corrections)

So I haven't done any serious coding in a good number of years. And when you're not actively coding a lot you get really rusty (at least for me that's the case). But using Claude Code (for this project) has been really fun. I've spent more hours poring over it's code and notes than I've done in a long time.

Yes it has flavor of AgenticAI coding, I'm ok with that. Hope you are as well. This is an OSS project to showcase fun AgenticAI coding using Tailscale Aperture on the backend and `tsnet` to make this app work for anyone with a Tailnet. :)

No warranties or support is available for this project. So if something is broken, please file an Issue and I'll get Claude on it! :)  

Why did I build this project? I love to fish and it's starting to get warmer (it's February at the time of this writing) so I'm itching to get back on the lake to fish. But all the current apps out there are full of great features but a lot are gated to make you pay. Blech. I just wanted something simple where I can take a pic, post a location, add a few notes, and boom. Go back to fishing.

That's Fishscale :)

Hope you enjoy it! 

## Features

- **Catch logging** with GPS coordinates, species, bait/lure, weight, length, and photos
- **Interactive map** showing all catch locations with locate-me button (MapLibre GL + OpenStreetMap)
- **Automatic weather** fetched from Open-Meteo at time of logging (no API key needed)
- **Statistics dashboard** with species breakdown, personal bests, top baits, and monthly trends
- **Trip tracking** to group catches by outing
- **Export** your data as JSON or CSV
- ~**44 pre-loaded species** covering freshwater and saltwater, with a species filter setting~ --> got rid of this as it was to complex for now. Will revisit later. Check out the `TODO.md` file for some other ideas I'm thinking through.
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

**Pre-built image** (no build required):

```bash
docker pull ghcr.io/valien/fishscale:latest
```

Or use the included `docker-compose.yml` to build from source.

**Setup:**

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

## Deployment

### Prerequisites

- Docker and Docker Compose
- A [Tailscale](https://tailscale.com) account
- A Tailscale auth key

> **Synology NAS?** See the dedicated [Synology NAS Deployment Guide](docs/guides/synology-nas-deployment.md) for step-by-step instructions using Container Manager.

### Auth Key Setup

Generate an auth key from the [Tailscale admin console](https://login.tailscale.com/admin/settings/keys):

- **Reusable** — recommended so the container can restart without a new key.
- **Ephemeral** — optional. The node will be automatically removed from your tailnet when the container stops. Good for testing.
- **Tagged** — if you use [ACL tags](https://tailscale.com/kb/1068/acl-tags), tag the key (e.g., `tag:fishscale`) so it gets the right permissions.

Auth keys expire after 90 days by default. If the container fails to start after that, generate a new key and update your `.env` file.

### Production Setup

1. Clone the repo and create a `.env` file:

```bash
git clone https://github.com/Valien/fishscale.git
cd fishscale

cat > .env <<'EOF'
TS_AUTHKEY=tskey-auth-...
TS_HOSTNAME=fishscale
EOF
```

2. Start the container:

```bash
docker compose up -d
```

Fishscale will join your tailnet and be available at `https://fishscale.<your-tailnet>.ts.net`.

The `docker-compose.yml` includes:

- **Named volume** (`fishscale-data`) mounted at `/data` — stores the SQLite database, photos, and Tailscale state.
- **`/dev/net/tun`** device + **`NET_ADMIN`** capability — required for Tailscale's userspace networking.
- **Resource limits** — 256 MB memory, 1 CPU. Adjust in `docker-compose.yml` if needed.
- **Log rotation** — JSON file driver, 10 MB max, 3 files.

### Data Persistence

All data lives in the `fishscale-data` Docker volume:

```
/data/
  fish.db           SQLite database (catches, trips, settings)
  photos/           Uploaded catch photos
  tsnet-state/      Tailscale node identity and keys
```

**Back up your data:**

```bash
# Copy the database and photos out of the container
docker cp fishscale:/data/fish.db ./fish.db.backup
docker cp fishscale:/data/photos ./photos-backup

# Or find the volume on disk
docker volume inspect fishscale_fishscale-data --format '{{ .Mountpoint }}'
```

**Restore from backup:**

```bash
docker compose down
docker cp ./fish.db.backup fishscale:/data/fish.db
docker cp ./photos-backup/. fishscale:/data/photos/
docker compose up -d
```

### Updating

Pull the latest code and rebuild:

```bash
git pull
docker compose up -d --build
```

Your data volume is preserved across rebuilds.

### Logs & Troubleshooting

```bash
# Follow logs
docker compose logs -f

# Check container status
docker compose ps
```

**Common issues:**

| Symptom | Cause | Fix |
|---------|-------|-----|
| Container exits immediately | Auth key expired or invalid | Generate a new key, update `.env`, restart |
| `TUN device not found` | Missing `/dev/net/tun` mount | Ensure `docker-compose.yml` has the `/dev/net/tun:/dev/net/tun` volume and `NET_ADMIN` cap |
| Not reachable on tailnet | Firewall or ACL blocking | Check [Tailscale admin](https://login.tailscale.com/admin/machines) for the node, verify ACLs |
| `permission denied` on `/data` | Volume ownership mismatch | The container runs as user `fishscale` (non-root). Ensure the volume has correct permissions |

### Testing Locally Without Tailscale

To test Fishscale without joining a tailnet, use dev mode. This skips Tailscale entirely and serves over plain HTTP on port 8080:

```bash
docker run --rm -p 8080:8080 \
  -e FISHSCALE_DEV_MODE=true \
  -e FISHSCALE_DB_PATH=/data/fish.db \
  -e FISHSCALE_PHOTO_DIR=/data/photos \
  -v fishscale-data:/data \
  $(docker compose build --quiet fishscale && echo fishscale-fishscale)
```

Or build and run directly without Docker:

```bash
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos go run ./cmd/fishscale
```

Open http://localhost:8080. Dev mode uses a fake user identity — no authentication is required.

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
GOWORK=off go test ./... -v
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
GET    /api/v1/me               Get current authenticated user + Tailscale info
GET    /api/v1/autocomplete/species  Autocomplete species names
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
