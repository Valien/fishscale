# Fishscale Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a self-hosted fishing tracker app that runs on Tailscale's tsnet, deployable as a single Docker image.

**Architecture:** Go backend with chi router serving a Svelte SPA embedded via `embed.FS`. SQLite database. tsnet for networking and auth via WhoIs. Photos stored on local filesystem.

**Tech Stack:** Go 1.25, Svelte 5, Vite, SQLite (via modernc.org/sqlite), chi router, tsnet, MapLibre GL JS, Open-Meteo API

---

## Task 1: Initialize Go Module and Project Structure

**Files:**
- Create: `go.mod`
- Create: `cmd/fishscale/main.go`
- Create: `internal/config/config.go`

**Step 1: Initialize the Go module**

```bash
cd /Users/allen/Documents/GitHub/fishscale
go mod init github.com/allen/fishscale
```

**Step 2: Create project directory structure**

```bash
mkdir -p cmd/fishscale
mkdir -p internal/{config,database,handler,middleware,model,storage}
mkdir -p frontend
```

**Step 3: Write the config loader**

Create `internal/config/config.go`:

```go
package config

import "os"

type Config struct {
	TSAuthKey  string
	TSHostname string
	TSStateDir string
	DBPath     string
	PhotoDir   string
	LogLevel   string
	DevMode    bool
}

func Load() *Config {
	return &Config{
		TSAuthKey:  os.Getenv("TS_AUTHKEY"),
		TSHostname: getEnv("TS_HOSTNAME", "fishscale"),
		TSStateDir: getEnv("TS_STATE_DIR", "/data/tsnet-state"),
		DBPath:     getEnv("FISHSCALE_DB_PATH", "/data/fish.db"),
		PhotoDir:   getEnv("FISHSCALE_PHOTO_DIR", "/data/photos"),
		LogLevel:   getEnv("FISHSCALE_LOG_LEVEL", "info"),
		DevMode:    os.Getenv("FISHSCALE_DEV_MODE") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

**Step 4: Write the minimal main.go entrypoint**

Create `cmd/fishscale/main.go`:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/allen/fishscale/internal/config"
)

func main() {
	cfg := config.Load()

	if cfg.LogLevel == "debug" {
		fmt.Fprintf(os.Stderr, "config: %+v\n", cfg)
	}

	log.Println("fishscale starting...")
	// Server setup will go here in later tasks.
}
```

**Step 5: Verify it compiles and commit**

```bash
go build ./cmd/fishscale
rm fishscale
git add -A
git commit -m "feat: initialize Go module and project structure"
```

---

## Task 2: SQLite Database Layer and Migrations

**Files:**
- Create: `internal/database/db.go`
- Create: `internal/database/migrations.go`
- Create: `internal/database/db_test.go`

**Step 1: Add SQLite dependency**

```bash
go get modernc.org/sqlite
go get github.com/jmoiron/sqlx
```

We use `modernc.org/sqlite` (pure Go, no CGO) and `sqlx` for ergonomic query mapping.

**Step 2: Write the failing test**

Create `internal/database/db_test.go`:

```go
package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify tables exist
	tables := []string{"users", "trips", "catches", "species", "photos", "user_settings"}
	for _, table := range tables {
		var count int
		err := db.Get(&count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table)
		if err != nil {
			t.Fatalf("query sqlite_master for %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestSeedSpecies(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var count int
	err = db.Get(&count, "SELECT count(*) FROM species")
	if err != nil {
		t.Fatalf("count species: %v", err)
	}
	if count == 0 {
		t.Error("expected species table to be seeded with data")
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("expected parent directory to be created")
	}
}
```

**Step 3: Run test to verify it fails**

```bash
go test ./internal/database/ -v
```

Expected: FAIL (package doesn't exist yet)

**Step 4: Write the database open/migrate code**

Create `internal/database/db.go`:

```go
package database

import (
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func Open(dbPath string) (*sqlx.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sqlx.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
```

Create `internal/database/migrations.go`:

```go
package database

import "github.com/jmoiron/sqlx"

func migrate(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		tailscale_id  TEXT UNIQUE NOT NULL,
		display_name  TEXT NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trips (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id       INTEGER NOT NULL REFERENCES users(id),
		name          TEXT,
		started_at    DATETIME NOT NULL,
		ended_at      DATETIME,
		notes         TEXT,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS species (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		name     TEXT UNIQUE NOT NULL,
		category TEXT
	);

	CREATE TABLE IF NOT EXISTS catches (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id       INTEGER NOT NULL REFERENCES users(id),
		trip_id       INTEGER REFERENCES trips(id),
		species_id    INTEGER REFERENCES species(id),
		caught_at     DATETIME NOT NULL,
		latitude      REAL NOT NULL,
		longitude     REAL NOT NULL,
		location_name TEXT,
		length_in     REAL,
		weight_lb     REAL,
		kept          BOOLEAN DEFAULT 0,
		bait_or_lure  TEXT,
		rod_setup     TEXT,
		line_info     TEXT,
		hook_size     TEXT,
		air_temp_f    REAL,
		wind_mph      REAL,
		wind_dir      TEXT,
		conditions    TEXT,
		pressure_mb   REAL,
		humidity_pct  REAL,
		water_temp_f  REAL,
		water_clarity TEXT,
		notes         TEXT,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS photos (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		catch_id   INTEGER NOT NULL REFERENCES catches(id) ON DELETE CASCADE,
		filename   TEXT NOT NULL,
		thumbnail  TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_settings (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER UNIQUE NOT NULL REFERENCES users(id),
		theme      TEXT DEFAULT 'system',
		units      TEXT DEFAULT 'imperial',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_catches_user_id ON catches(user_id);
	CREATE INDEX IF NOT EXISTS idx_catches_caught_at ON catches(caught_at);
	CREATE INDEX IF NOT EXISTS idx_catches_species_id ON catches(species_id);
	CREATE INDEX IF NOT EXISTS idx_photos_catch_id ON photos(catch_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	return seedSpecies(db)
}

func seedSpecies(db *sqlx.DB) error {
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM species"); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	species := []struct {
		Name     string
		Category string
	}{
		// Freshwater
		{"Largemouth Bass", "Freshwater"},
		{"Smallmouth Bass", "Freshwater"},
		{"Spotted Bass", "Freshwater"},
		{"Striped Bass", "Freshwater"},
		{"Channel Catfish", "Freshwater"},
		{"Blue Catfish", "Freshwater"},
		{"Flathead Catfish", "Freshwater"},
		{"Bluegill", "Freshwater"},
		{"Crappie", "Freshwater"},
		{"White Crappie", "Freshwater"},
		{"Black Crappie", "Freshwater"},
		{"Walleye", "Freshwater"},
		{"Sauger", "Freshwater"},
		{"Northern Pike", "Freshwater"},
		{"Musky", "Freshwater"},
		{"Rainbow Trout", "Freshwater"},
		{"Brown Trout", "Freshwater"},
		{"Brook Trout", "Freshwater"},
		{"Lake Trout", "Freshwater"},
		{"Yellow Perch", "Freshwater"},
		{"White Bass", "Freshwater"},
		{"Carp", "Freshwater"},
		{"Drum", "Freshwater"},
		{"Gar", "Freshwater"},
		{"Bowfin", "Freshwater"},
		// Saltwater
		{"Redfish", "Saltwater"},
		{"Speckled Trout", "Saltwater"},
		{"Flounder", "Saltwater"},
		{"Snook", "Saltwater"},
		{"Tarpon", "Saltwater"},
		{"Mahi-Mahi", "Saltwater"},
		{"Red Snapper", "Saltwater"},
		{"Grouper", "Saltwater"},
		{"King Mackerel", "Saltwater"},
		{"Spanish Mackerel", "Saltwater"},
		{"Sheepshead", "Saltwater"},
		{"Pompano", "Saltwater"},
		{"Cobia", "Saltwater"},
		{"Amberjack", "Saltwater"},
		{"Tuna", "Saltwater"},
		{"Wahoo", "Saltwater"},
		{"Sailfish", "Saltwater"},
		{"Marlin", "Saltwater"},
		{"Striped Bass", "Saltwater"},
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, s := range species {
		_, err := tx.Exec("INSERT OR IGNORE INTO species (name, category) VALUES (?, ?)", s.Name, s.Category)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/database/ -v
```

Expected: All 3 tests PASS.

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: add SQLite database layer with migrations and species seed data"
```

---

## Task 3: Go Models

**Files:**
- Create: `internal/model/models.go`

**Step 1: Write the model structs**

Create `internal/model/models.go`:

```go
package model

import "time"

type User struct {
	ID          int64     `db:"id" json:"id"`
	TailscaleID string   `db:"tailscale_id" json:"tailscale_id"`
	DisplayName string   `db:"display_name" json:"display_name"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Trip struct {
	ID        int64      `db:"id" json:"id"`
	UserID    int64      `db:"user_id" json:"user_id"`
	Name      string     `db:"name" json:"name"`
	StartedAt time.Time  `db:"started_at" json:"started_at"`
	EndedAt   *time.Time `db:"ended_at" json:"ended_at,omitempty"`
	Notes     string     `db:"notes" json:"notes,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	Catches   []Catch    `db:"-" json:"catches,omitempty"`
}

type Species struct {
	ID       int64  `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
}

type Catch struct {
	ID           int64      `db:"id" json:"id"`
	UserID       int64      `db:"user_id" json:"user_id"`
	TripID       *int64     `db:"trip_id" json:"trip_id,omitempty"`
	SpeciesID    *int64     `db:"species_id" json:"species_id,omitempty"`
	CaughtAt     time.Time  `db:"caught_at" json:"caught_at"`
	Latitude     float64    `db:"latitude" json:"latitude"`
	Longitude    float64    `db:"longitude" json:"longitude"`
	LocationName string     `db:"location_name" json:"location_name,omitempty"`
	LengthIn     *float64   `db:"length_in" json:"length_in,omitempty"`
	WeightLb     *float64   `db:"weight_lb" json:"weight_lb,omitempty"`
	Kept         bool       `db:"kept" json:"kept"`
	BaitOrLure   string     `db:"bait_or_lure" json:"bait_or_lure,omitempty"`
	RodSetup     string     `db:"rod_setup" json:"rod_setup,omitempty"`
	LineInfo     string     `db:"line_info" json:"line_info,omitempty"`
	HookSize     string     `db:"hook_size" json:"hook_size,omitempty"`
	AirTempF     *float64   `db:"air_temp_f" json:"air_temp_f,omitempty"`
	WindMph      *float64   `db:"wind_mph" json:"wind_mph,omitempty"`
	WindDir      string     `db:"wind_dir" json:"wind_dir,omitempty"`
	Conditions   string     `db:"conditions" json:"conditions,omitempty"`
	PressureMb   *float64   `db:"pressure_mb" json:"pressure_mb,omitempty"`
	HumidityPct  *float64   `db:"humidity_pct" json:"humidity_pct,omitempty"`
	WaterTempF   *float64   `db:"water_temp_f" json:"water_temp_f,omitempty"`
	WaterClarity string     `db:"water_clarity" json:"water_clarity,omitempty"`
	Notes        string     `db:"notes" json:"notes,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	Photos       []Photo    `db:"-" json:"photos,omitempty"`
	SpeciesName  string     `db:"species_name" json:"species_name,omitempty"`
}

type Photo struct {
	ID        int64     `db:"id" json:"id"`
	CatchID   int64     `db:"catch_id" json:"catch_id"`
	Filename  string    `db:"filename" json:"filename"`
	Thumbnail string    `db:"thumbnail" json:"thumbnail,omitempty"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	URL       string    `db:"-" json:"url"`
}

type UserSettings struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Theme     string    `db:"theme" json:"theme"`
	Units     string    `db:"units" json:"units"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// API request/response types

type CreateCatchRequest struct {
	CaughtAt     string   `json:"caught_at"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	LocationName string   `json:"location_name"`
	SpeciesID    *int64   `json:"species_id"`
	TripID       *int64   `json:"trip_id"`
	LengthIn     *float64 `json:"length_in"`
	WeightLb     *float64 `json:"weight_lb"`
	Kept         bool     `json:"kept"`
	BaitOrLure   string   `json:"bait_or_lure"`
	RodSetup     string   `json:"rod_setup"`
	LineInfo     string   `json:"line_info"`
	HookSize     string   `json:"hook_size"`
	AirTempF     *float64 `json:"air_temp_f"`
	WindMph      *float64 `json:"wind_mph"`
	WindDir      string   `json:"wind_dir"`
	Conditions   string   `json:"conditions"`
	PressureMb   *float64 `json:"pressure_mb"`
	HumidityPct  *float64 `json:"humidity_pct"`
	WaterTempF   *float64 `json:"water_temp_f"`
	WaterClarity string   `json:"water_clarity"`
	Notes        string   `json:"notes"`
}

type StatsResponse struct {
	TotalCatches   int              `json:"total_catches"`
	CatchesThisYear int             `json:"catches_this_year"`
	CatchesThisMonth int            `json:"catches_this_month"`
	TopSpecies     []SpeciesCount   `json:"top_species"`
	PersonalBests  []PersonalBest   `json:"personal_bests"`
	TopBaits       []BaitCount      `json:"top_baits"`
	CatchesByMonth []MonthCount     `json:"catches_by_month"`
}

type SpeciesCount struct {
	Name  string `db:"name" json:"name"`
	Count int    `db:"count" json:"count"`
}

type PersonalBest struct {
	SpeciesName string  `db:"species_name" json:"species_name"`
	WeightLb    float64 `db:"weight_lb" json:"weight_lb"`
	LengthIn    float64 `db:"length_in" json:"length_in"`
	CatchID     int64   `db:"catch_id" json:"catch_id"`
}

type BaitCount struct {
	Name  string `db:"name" json:"name"`
	Count int    `db:"count" json:"count"`
}

type MonthCount struct {
	Month string `db:"month" json:"month"`
	Count int    `db:"count" json:"count"`
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/model/
```

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: add data model structs"
```

---

## Task 4: Photo Storage Interface and Local Filesystem Implementation

**Files:**
- Create: `internal/storage/storage.go`
- Create: `internal/storage/local.go`
- Create: `internal/storage/local_test.go`

**Step 1: Write the failing test**

Create `internal/storage/local_test.go`:

```go
package storage

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStore(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	data := []byte("fake image data")

	t.Run("save and retrieve", func(t *testing.T) {
		path, err := store.Save("test.jpg", bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Save: %v", err)
		}

		r, err := store.Get(path)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		defer r.Close()

		got, _ := io.ReadAll(r)
		if !bytes.Equal(got, data) {
			t.Errorf("got %q, want %q", got, data)
		}
	})

	t.Run("delete", func(t *testing.T) {
		path, err := store.Save("delete-me.jpg", bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Save: %v", err)
		}

		if err := store.Delete(path); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		fullPath := filepath.Join(dir, path)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("expected file to be deleted")
		}
	})

	t.Run("organizes by date directory", func(t *testing.T) {
		path, _ := store.Save("photo.jpg", bytes.NewReader(data))
		// Path should contain YYYY/MM structure
		if len(path) < 8 {
			t.Errorf("expected date-organized path, got %s", path)
		}
	})
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/storage/ -v
```

**Step 3: Write the storage interface and local implementation**

Create `internal/storage/storage.go`:

```go
package storage

import "io"

type Store interface {
	Save(filename string, r io.Reader) (path string, err error)
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
}
```

Create `internal/storage/local.go`:

```go
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"crypto/rand"
	"encoding/hex"
)

type LocalStore struct {
	baseDir string
}

func NewLocalStore(baseDir string) *LocalStore {
	return &LocalStore{baseDir: baseDir}
}

func (s *LocalStore) Save(filename string, r io.Reader) (string, error) {
	now := time.Now()
	dir := fmt.Sprintf("%d/%02d", now.Year(), now.Month())

	fullDir := filepath.Join(s.baseDir, dir)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// Generate unique filename to avoid collisions
	randBytes := make([]byte, 8)
	rand.Read(randBytes)
	ext := filepath.Ext(filename)
	uniqueName := hex.EncodeToString(randBytes) + ext

	relPath := filepath.Join(dir, uniqueName)
	fullPath := filepath.Join(s.baseDir, relPath)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(fullPath)
		return "", err
	}

	return relPath, nil
}

func (s *LocalStore) Get(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.baseDir, path))
}

func (s *LocalStore) Delete(path string) error {
	return os.Remove(filepath.Join(s.baseDir, path))
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/storage/ -v
```

Expected: All 3 subtests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add photo storage interface with local filesystem implementation"
```

---

## Task 5: Tailscale Auth Middleware

**Files:**
- Create: `internal/middleware/auth.go`
- Create: `internal/middleware/auth_test.go`

**Step 1: Write the failing test**

Create `internal/middleware/auth_test.go`:

```go
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allen/fishscale/internal/model"
)

func TestUserFromContext(t *testing.T) {
	user := &model.User{ID: 1, TailscaleID: "test", DisplayName: "Test User"}
	ctx := context.WithValue(context.Background(), userContextKey, user)

	got := UserFromContext(ctx)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != 1 {
		t.Errorf("got ID %d, want 1", got.ID)
	}
}

func TestUserFromContextMissing(t *testing.T) {
	got := UserFromContext(context.Background())
	if got != nil {
		t.Error("expected nil for empty context")
	}
}

func TestDevModeMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			t.Fatal("expected dev user in context")
		}
		if user.DisplayName != "Dev User" {
			t.Errorf("got %q, want 'Dev User'", user.DisplayName)
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := DevAuth(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/middleware/ -v
```

**Step 3: Write the auth middleware**

Create `internal/middleware/auth.go`:

```go
package middleware

import (
	"context"
	"net/http"

	"github.com/allen/fishscale/internal/model"
)

type contextKey string

const userContextKey contextKey = "user"

func UserFromContext(ctx context.Context) *model.User {
	user, _ := ctx.Value(userContextKey).(*model.User)
	return user
}

func withUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// DevAuth is a development-mode middleware that injects a fake user.
func DevAuth(next http.Handler) http.Handler {
	devUser := &model.User{
		ID:          1,
		TailscaleID: "dev-user",
		DisplayName: "Dev User",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := withUser(r.Context(), devUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TailscaleAuth is a placeholder for the real tsnet WhoIs middleware.
// It will be implemented when tsnet integration is added (Task 12).
// The signature is: func TailscaleAuth(tsServer *tsnet.Server, db *sqlx.DB) func(http.Handler) http.Handler
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/middleware/ -v
```

Expected: All 3 tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add auth middleware with dev mode and Tailscale placeholder"
```

---

## Task 6: Catch CRUD Handlers

**Files:**
- Create: `internal/handler/catches.go`
- Create: `internal/handler/catches_test.go`
- Create: `internal/handler/helpers.go`

This is the largest task -- the core API for creating, reading, updating, and deleting catches.

**Step 1: Write the JSON helper**

Create `internal/handler/helpers.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
)

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}
```

**Step 2: Write the failing test for list and create**

Create `internal/handler/catches_test.go`:

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
	"github.com/allen/fishscale/internal/storage"
)

func setupTestHandler(t *testing.T) (*CatchHandler, *chi.Mux) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Insert a dev user
	_, err = db.Exec("INSERT INTO users (tailscale_id, display_name) VALUES ('dev-user', 'Dev User')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	store := storage.NewLocalStore(dir + "/photos")
	h := NewCatchHandler(db, store)

	r := chi.NewRouter()
	r.Use(middleware.DevAuth)
	r.Route("/api/v1/catches", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})

	return h, r
}

func TestCreateAndListCatch(t *testing.T) {
	_, router := setupTestHandler(t)

	// Create a catch
	body := model.CreateCatchRequest{
		CaughtAt:  "2026-02-16T10:30:00Z",
		Latitude:  32.7767,
		Longitude: -96.7970,
		Kept:      false,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	var created model.Catch
	json.NewDecoder(rec.Body).Decode(&created)
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// List catches
	req = httptest.NewRequest("GET", "/api/v1/catches", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: got %d, want 200", rec.Code)
	}

	var catches []model.Catch
	json.NewDecoder(rec.Body).Decode(&catches)
	if len(catches) != 1 {
		t.Errorf("expected 1 catch, got %d", len(catches))
	}
}

func TestGetCatch(t *testing.T) {
	_, router := setupTestHandler(t)

	// Create
	body := model.CreateCatchRequest{
		CaughtAt:  "2026-02-16T10:30:00Z",
		Latitude:  32.7767,
		Longitude: -96.7970,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var created model.Catch
	json.NewDecoder(rec.Body).Decode(&created)

	// Get by ID
	req = httptest.NewRequest("GET", "/api/v1/catches/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteCatch(t *testing.T) {
	_, router := setupTestHandler(t)

	// Create
	body := model.CreateCatchRequest{
		CaughtAt:  "2026-02-16T10:30:00Z",
		Latitude:  32.7767,
		Longitude: -96.7970,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/catches", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/v1/catches/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", rec.Code)
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/api/v1/catches/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("get after delete: got %d, want 404", rec.Code)
	}
}
```

**Step 3: Run tests to verify they fail**

```bash
go get github.com/go-chi/chi/v5
go test ./internal/handler/ -v
```

**Step 4: Implement the catch handler**

Create `internal/handler/catches.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
	"github.com/allen/fishscale/internal/storage"
)

type CatchHandler struct {
	db    *sqlx.DB
	store storage.Store
}

func NewCatchHandler(db *sqlx.DB, store storage.Store) *CatchHandler {
	return &CatchHandler{db: db, store: store}
}

func (h *CatchHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.user_id = ?
		ORDER BY c.caught_at DESC`

	var catches []model.Catch
	if err := h.db.Select(&catches, query, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query catches")
		return
	}

	if catches == nil {
		catches = []model.Catch{}
	}

	jsonResponse(w, http.StatusOK, catches)
}

func (h *CatchHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var catch model.Catch
	query := `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.id = ? AND c.user_id = ?`
	if err := h.db.Get(&catch, query, id, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	var photos []model.Photo
	h.db.Select(&photos, "SELECT * FROM photos WHERE catch_id = ? ORDER BY sort_order", id)
	catch.Photos = photos

	jsonResponse(w, http.StatusOK, catch)
}

func (h *CatchHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreateCatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	caughtAt, err := time.Parse(time.RFC3339, req.CaughtAt)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid caught_at format, use RFC3339")
		return
	}

	result, err := h.db.Exec(`INSERT INTO catches (
		user_id, trip_id, species_id, caught_at, latitude, longitude, location_name,
		length_in, weight_lb, kept, bait_or_lure, rod_setup, line_info, hook_size,
		air_temp_f, wind_mph, wind_dir, conditions, pressure_mb, humidity_pct,
		water_temp_f, water_clarity, notes
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, req.TripID, req.SpeciesID, caughtAt, req.Latitude, req.Longitude, req.LocationName,
		req.LengthIn, req.WeightLb, req.Kept, req.BaitOrLure, req.RodSetup, req.LineInfo, req.HookSize,
		req.AirTempF, req.WindMph, req.WindDir, req.Conditions, req.PressureMb, req.HumidityPct,
		req.WaterTempF, req.WaterClarity, req.Notes,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create catch")
		return
	}

	id, _ := result.LastInsertId()

	var catch model.Catch
	h.db.Get(&catch, `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.id = ?`, id)

	jsonResponse(w, http.StatusCreated, catch)
}

func (h *CatchHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Verify ownership
	var exists int
	if err := h.db.Get(&exists, "SELECT 1 FROM catches WHERE id = ? AND user_id = ?", id, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	var req model.CreateCatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	caughtAt, err := time.Parse(time.RFC3339, req.CaughtAt)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid caught_at format")
		return
	}

	_, err = h.db.Exec(`UPDATE catches SET
		trip_id=?, species_id=?, caught_at=?, latitude=?, longitude=?, location_name=?,
		length_in=?, weight_lb=?, kept=?, bait_or_lure=?, rod_setup=?, line_info=?, hook_size=?,
		air_temp_f=?, wind_mph=?, wind_dir=?, conditions=?, pressure_mb=?, humidity_pct=?,
		water_temp_f=?, water_clarity=?, notes=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND user_id=?`,
		req.TripID, req.SpeciesID, caughtAt, req.Latitude, req.Longitude, req.LocationName,
		req.LengthIn, req.WeightLb, req.Kept, req.BaitOrLure, req.RodSetup, req.LineInfo, req.HookSize,
		req.AirTempF, req.WindMph, req.WindDir, req.Conditions, req.PressureMb, req.HumidityPct,
		req.WaterTempF, req.WaterClarity, req.Notes,
		id, user.ID,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update catch")
		return
	}

	var catch model.Catch
	h.db.Get(&catch, `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.id = ?`, id)

	jsonResponse(w, http.StatusOK, catch)
}

func (h *CatchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Get photos to clean up files
	var photos []model.Photo
	h.db.Select(&photos, "SELECT * FROM photos WHERE catch_id = ?", id)

	result, err := h.db.Exec("DELETE FROM catches WHERE id = ? AND user_id = ?", id, user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to delete")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	// Clean up photo files
	for _, p := range photos {
		h.store.Delete(p.Filename)
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/handler/ -v
```

Expected: All tests PASS.

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: add catch CRUD handlers with tests"
```

---

## Task 7: Remaining API Handlers (Trips, Species, Photos, Settings, Weather, Stats, Export)

**Files:**
- Create: `internal/handler/trips.go`
- Create: `internal/handler/species.go`
- Create: `internal/handler/photos.go`
- Create: `internal/handler/settings.go`
- Create: `internal/handler/weather.go`
- Create: `internal/handler/stats.go`
- Create: `internal/handler/export.go`

These follow the same pattern as the catch handler. I'll provide the key implementations. Each handler gets its own file.

**Step 1: Write trips handler**

Create `internal/handler/trips.go` -- standard CRUD following the catch handler pattern. List, Get (with catches), Create, Update, Delete (unlinks catches).

**Step 2: Write species handler**

Create `internal/handler/species.go` -- List with `?q=` search query, Create for custom species.

**Step 3: Write photos handler**

Create `internal/handler/photos.go`:
- `AddPhotos`: accepts multipart form upload, saves via `storage.Store`, creates DB records, generates thumbnail path.
- `DeletePhoto`: removes DB record and file.

**Step 4: Write settings handler**

Create `internal/handler/settings.go`:
- `GetSettings`: returns user_settings or defaults if none exist.
- `UpdateSettings`: upserts theme and units.

**Step 5: Write weather proxy handler**

Create `internal/handler/weather.go`:
- `GetWeather`: accepts `?lat=&lon=` query params, calls `https://api.open-meteo.com/v1/forecast` with `current=temperature_2m,wind_speed_10m,wind_direction_10m,relative_humidity_2m,surface_pressure,weather_code`, maps WMO weather codes to condition strings, returns JSON.

**Step 6: Write stats handler**

Create `internal/handler/stats.go`:
- Total catches, catches this year/month.
- Top 5 species by count: `SELECT s.name, COUNT(*) as count FROM catches c JOIN species s ... GROUP BY ... ORDER BY count DESC LIMIT 5`
- Personal bests by species (heaviest).
- Top baits/lures by count.
- Catches by month for the last 12 months.

**Step 7: Write export handler**

Create `internal/handler/export.go`:
- Queries all user catches with species name joined.
- `?format=csv`: writes CSV with headers and `Content-Disposition: attachment`.
- `?format=json`: writes JSON array with `Content-Disposition: attachment`.
- Respects user's unit preference for conversion.

**Step 8: Write tests for each handler and verify all pass**

```bash
go test ./internal/handler/ -v
```

**Step 9: Commit**

```bash
git add -A
git commit -m "feat: add trip, species, photo, settings, weather, stats, and export handlers"
```

---

## Task 8: HTTP Router and Server Assembly

**Files:**
- Create: `internal/server/server.go`
- Modify: `cmd/fishscale/main.go`

**Step 1: Write the server assembly**

Create `internal/server/server.go`:

```go
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/handler"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/storage"
)

func NewRouter(cfg *config.Config, db *sqlx.DB, store storage.Store) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Compress(5))

	// Auth middleware -- dev mode uses fake user, production uses tsnet WhoIs
	if cfg.DevMode {
		r.Use(middleware.DevAuth)
	}
	// In production, TailscaleAuth middleware is added in main.go when tsnet is available

	catches := handler.NewCatchHandler(db, store)
	// trips, species, photos, settings, weather, stats, export handlers initialized similarly

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/catches", func(r chi.Router) {
			r.Get("/", catches.List)
			r.Post("/", catches.Create)
			r.Get("/{id}", catches.Get)
			r.Put("/{id}", catches.Update)
			r.Delete("/{id}", catches.Delete)
			// r.Post("/{id}/photos", photos.Add)
		})
		// Mount remaining handlers for /trips, /species, /photos, /settings, /weather, /stats, /export
	})

	// SPA fallback -- serve embedded frontend (added in Task 11)

	return r
}
```

**Step 2: Update main.go to wire everything together in dev mode**

Update `cmd/fishscale/main.go` to:
- Load config
- Open database
- Create local store
- Build router
- Start HTTP server on `:8080` in dev mode (tsnet in production, Task 12)

```go
package main

import (
	"log"
	"net/http"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/server"
	"github.com/allen/fishscale/internal/storage"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)

	router := server.NewRouter(cfg, db, store)

	if cfg.DevMode {
		log.Println("DEV MODE: listening on http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", router))
	} else {
		log.Println("fishscale starting... (tsnet mode not yet implemented)")
		// tsnet startup will be added in Task 12
		log.Fatal(http.ListenAndServe(":8080", router))
	}
}
```

**Step 3: Verify it compiles and starts**

```bash
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=/tmp/fishscale-dev.db FISHSCALE_PHOTO_DIR=/tmp/fishscale-photos go run ./cmd/fishscale &
curl -s http://localhost:8080/api/v1/catches | head
kill %1
```

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: assemble HTTP router and server with dev mode"
```

---

## Task 9: Initialize Svelte Frontend

**Files:**
- Create: `frontend/` (Svelte project via `npm create svelte`)
- Create: `frontend/src/lib/api.ts`

**Step 1: Scaffold the Svelte project**

```bash
cd /Users/allen/Documents/GitHub/fishscale
npm create vite@latest frontend -- --template svelte-ts
cd frontend
npm install
npm install maplibre-gl
npm install -D @types/maplibre-gl
```

**Step 2: Create the API client**

Create `frontend/src/lib/api.ts`:

```typescript
const BASE = '/api/v1';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  catches: {
    list: () => request<any[]>('/catches'),
    get: (id: number) => request<any>(`/catches/${id}`),
    create: (data: any) => request<any>('/catches', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: any) => request<any>(`/catches/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request<void>(`/catches/${id}`, { method: 'DELETE' }),
  },
  species: {
    list: (q?: string) => request<any[]>(`/species${q ? `?q=${q}` : ''}`),
    create: (data: any) => request<any>('/species', { method: 'POST', body: JSON.stringify(data) }),
  },
  weather: {
    get: (lat: number, lon: number) => request<any>(`/weather?lat=${lat}&lon=${lon}`),
  },
  settings: {
    get: () => request<any>('/settings'),
    update: (data: any) => request<any>('/settings', { method: 'PUT', body: JSON.stringify(data) }),
  },
  stats: {
    get: () => request<any>('/stats'),
  },
  trips: {
    list: () => request<any[]>('/trips'),
    get: (id: number) => request<any>(`/trips/${id}`),
    create: (data: any) => request<any>('/trips', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: any) => request<any>(`/trips/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request<void>(`/trips/${id}`, { method: 'DELETE' }),
  },
};
```

**Step 3: Configure Vite proxy for dev mode**

Update `frontend/vite.config.ts` to proxy `/api` to `localhost:8080`:

```typescript
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
});
```

**Step 4: Verify frontend builds**

```bash
cd /Users/allen/Documents/GitHub/fishscale/frontend
npm run build
```

**Step 5: Commit**

```bash
cd /Users/allen/Documents/GitHub/fishscale
echo "frontend/node_modules" >> .gitignore
echo "frontend/dist" >> .gitignore
git add -A
git commit -m "feat: scaffold Svelte frontend with API client and Vite dev proxy"
```

---

## Task 10: Svelte UI -- Core Pages

**Files:**
- Create/modify: `frontend/src/App.svelte`
- Create: `frontend/src/lib/components/BottomNav.svelte`
- Create: `frontend/src/lib/pages/LogCatch.svelte`
- Create: `frontend/src/lib/pages/MapView.svelte`
- Create: `frontend/src/lib/pages/CatchLog.svelte`
- Create: `frontend/src/lib/pages/Stats.svelte`
- Create: `frontend/src/lib/pages/Settings.svelte`
- Create: `frontend/src/lib/stores/settings.ts`
- Create: `frontend/src/lib/stores/catches.ts`

This is the largest frontend task. Build each page incrementally.

**Step 1: Create the settings store (theme + units)**

`frontend/src/lib/stores/settings.ts` -- Svelte store that loads from `/api/v1/settings`, applies theme class to `<html>`, and provides reactive `units` value.

**Step 2: Create the bottom nav component**

`frontend/src/lib/components/BottomNav.svelte` -- 5 tabs: Map, Log, [+] (floating action button), Stats, Settings. Active tab highlighted.

**Step 3: Create App.svelte with routing**

Use a simple hash-based router or `svelte-spa-router`. Renders the active page based on route, always shows BottomNav.

**Step 4: Build the LogCatch page (the critical UX)**

`frontend/src/lib/pages/LogCatch.svelte`:
- On mount: request camera/file picker, request GPS via `navigator.geolocation`, fetch weather from API.
- Two-tier form: quick fields (species autocomplete, bait autocomplete, kept toggle) always visible. "More Detail" expandable section.
- Save button calls `api.catches.create()`.
- Species and bait autocomplete fields query the API / local history.

**Step 5: Build the MapView page**

`frontend/src/lib/pages/MapView.svelte`:
- Initialize MapLibre GL map with OSM tile source.
- Load catches as GeoJSON markers.
- Popup on marker click shows catch summary card.
- Filter panel (date range, species).

**Step 6: Build the CatchLog page**

`frontend/src/lib/pages/CatchLog.svelte`:
- Fetch catches list from API.
- Render cards with thumbnail, species, date, location, bait.
- Click card navigates to detail/edit view.
- Search bar and filter dropdowns.

**Step 7: Build the Stats page**

`frontend/src/lib/pages/Stats.svelte`:
- Fetch from `/api/v1/stats`.
- Render total counts, top species list, catches-by-month bar chart (use a lightweight chart lib or SVG), personal bests, top baits.

**Step 8: Build the Settings page**

`frontend/src/lib/pages/Settings.svelte`:
- Theme picker: Light / Dark / System radio buttons.
- Units picker: Imperial / Metric radio buttons.
- Display user profile (name from Tailscale, read-only).
- Save calls `api.settings.update()`.

**Step 9: Add CSS theming**

Create `frontend/src/lib/theme.css`:
- CSS custom properties for colors: `--bg`, `--text`, `--primary`, `--card-bg`, etc.
- `.theme-light` and `.theme-dark` classes on `<html>`.
- `@media (prefers-color-scheme: dark)` for `system` mode.

**Step 10: Verify frontend builds and commit**

```bash
cd /Users/allen/Documents/GitHub/fishscale/frontend
npm run build
cd ..
git add -A
git commit -m "feat: add Svelte UI with all core pages"
```

---

## Task 11: Embed Frontend in Go Binary

**Files:**
- Create: `internal/frontend/embed.go`
- Modify: `internal/server/server.go`

**Step 1: Create the embed file**

Create `internal/frontend/embed.go`:

```go
package frontend

import "embed"

//go:embed all:dist
var Assets embed.FS
```

Note: The `dist/` directory is the Vite build output copied into this package during the Docker build (or symlinked during dev).

**Step 2: Add SPA file server to the router**

In `internal/server/server.go`, add a file server that:
- Serves static files from the embedded `frontend.Assets` FS.
- Falls back to `index.html` for any non-API, non-file route (SPA routing).
- In dev mode (`cfg.DevMode`), optionally serve from disk instead of embed for hot reload.

**Step 3: Verify the full stack works**

```bash
cd /Users/allen/Documents/GitHub/fishscale/frontend && npm run build
cp -r dist ../internal/frontend/dist
cd ..
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=/tmp/fishscale-dev.db FISHSCALE_PHOTO_DIR=/tmp/fishscale-photos go run ./cmd/fishscale
# Visit http://localhost:8080 in browser
```

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: embed Svelte frontend in Go binary"
```

---

## Task 12: Tailscale tsnet Integration

**Files:**
- Modify: `cmd/fishscale/main.go`
- Create: `internal/middleware/tailscale.go`

**Step 1: Add tsnet dependency**

```bash
go get tailscale.com/tsnet
```

**Step 2: Implement the TailscaleAuth middleware**

Create `internal/middleware/tailscale.go`:

```go
package middleware

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"tailscale.com/tsnet"

	"github.com/allen/fishscale/internal/model"
)

func TailscaleAuth(ts *tsnet.Server, db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			whois, err := ts.WhoIs(r.Context(), r.RemoteAddr)
			if err != nil {
				log.Printf("WhoIs error: %v", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			tailscaleID := whois.UserProfile.ID.String()
			displayName := whois.UserProfile.DisplayName

			// Upsert user
			_, err = db.Exec(`INSERT INTO users (tailscale_id, display_name) VALUES (?, ?)
				ON CONFLICT(tailscale_id) DO UPDATE SET display_name = ?`,
				tailscaleID, displayName, displayName)
			if err != nil {
				log.Printf("upsert user error: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			var user model.User
			err = db.Get(&user, "SELECT * FROM users WHERE tailscale_id = ?", tailscaleID)
			if err != nil {
				log.Printf("get user error: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			ctx := withUser(r.Context(), &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

**Step 3: Update main.go for tsnet startup**

Update `cmd/fishscale/main.go` to:
- If NOT dev mode: create `tsnet.Server` with `Hostname` and `AuthKey` from config, call `ts.ListenTLS("tcp", ":443")`, serve HTTP over the tsnet listener.
- If dev mode: keep the plain HTTP server on `:8080`.

```go
// In production mode:
ts := &tsnet.Server{
    Hostname: cfg.TSHostname,
    AuthKey:  cfg.TSAuthKey,
    Dir:      cfg.TSStateDir,
}
defer ts.Close()

ln, err := ts.ListenTLS("tcp", ":443")
if err != nil {
    log.Fatalf("tsnet listen: %v", err)
}
defer ln.Close()

// Add TailscaleAuth middleware to router
log.Printf("fishscale available at https://%s.<tailnet>.ts.net", cfg.TSHostname)
log.Fatal(http.Serve(ln, router))
```

**Step 4: Verify it compiles**

```bash
go build ./cmd/fishscale
```

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add Tailscale tsnet integration with WhoIs auth"
```

---

## Task 13: Dockerfile and Docker Compose

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`

**Step 1: Write the Dockerfile**

Create `Dockerfile`:

```dockerfile
# Stage 1: Build Svelte frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.23-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./internal/frontend/dist
RUN CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Stage 3: Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend /app/fishscale /usr/local/bin/fishscale
VOLUME /data
ENTRYPOINT ["fishscale"]
```

**Step 2: Write .dockerignore**

Create `.dockerignore`:

```
frontend/node_modules
frontend/dist
*.db
/data
.git
```

**Step 3: Write docker-compose.yml**

Create `docker-compose.yml`:

```yaml
services:
  fishscale:
    build: .
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
      - FISHSCALE_LOG_LEVEL=${FISHSCALE_LOG_LEVEL:-info}

volumes:
  fishscale-data:
```

**Step 4: Test Docker build**

```bash
docker build -t fishscale:latest .
```

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add Dockerfile and docker-compose.yml"
```

---

## Task 14: End-to-End Dev Mode Smoke Test

**Files:** None new -- this is a verification task.

**Step 1: Start the backend in dev mode**

```bash
FISHSCALE_DEV_MODE=true \
FISHSCALE_DB_PATH=/tmp/fishscale-e2e.db \
FISHSCALE_PHOTO_DIR=/tmp/fishscale-e2e-photos \
go run ./cmd/fishscale &
```

**Step 2: Start the frontend dev server**

```bash
cd frontend && npm run dev &
```

**Step 3: Verify the following flows work**

Using `curl` against `http://localhost:8080`:

1. `GET /api/v1/species` -- returns seeded species list
2. `POST /api/v1/catches` with JSON body -- creates a catch, returns 201
3. `GET /api/v1/catches` -- returns the created catch
4. `GET /api/v1/catches/1` -- returns catch detail
5. `PUT /api/v1/catches/1` -- updates catch
6. `DELETE /api/v1/catches/1` -- returns 204
7. `GET /api/v1/weather?lat=32.77&lon=-96.79` -- returns weather data from Open-Meteo
8. `GET /api/v1/settings` -- returns default settings
9. `PUT /api/v1/settings` with `{"theme":"dark","units":"metric"}` -- updates settings
10. `GET /api/v1/stats` -- returns stats
11. `GET /api/v1/export?format=json` -- returns JSON export

**Step 4: Clean up**

```bash
kill %1 %2
rm -rf /tmp/fishscale-e2e*
```

**Step 5: Run all Go tests**

```bash
go test ./... -v
```

Expected: All tests pass.

**Step 6: Commit any fixes**

```bash
git add -A
git commit -m "fix: address issues found during smoke test"
```

---

## Summary of Task Order and Dependencies

```
Task  1: Go module + project structure
Task  2: SQLite database + migrations        (depends on 1)
Task  3: Go model structs                    (depends on 1)
Task  4: Photo storage interface             (depends on 1)
Task  5: Auth middleware                     (depends on 3)
Task  6: Catch CRUD handlers                 (depends on 2, 3, 4, 5)
Task  7: Remaining API handlers              (depends on 6)
Task  8: HTTP router + server assembly       (depends on 6, 7)
Task  9: Svelte frontend scaffold            (depends on 1)
Task 10: Svelte UI pages                     (depends on 9)
Task 11: Embed frontend in Go binary         (depends on 8, 10)
Task 12: Tailscale tsnet integration         (depends on 8)
Task 13: Dockerfile + Docker Compose         (depends on 11, 12)
Task 14: End-to-end smoke test               (depends on 13)
```

Tasks 2-5 can be done in parallel. Tasks 9-10 (frontend) can be done in parallel with Tasks 6-8 (backend API).
