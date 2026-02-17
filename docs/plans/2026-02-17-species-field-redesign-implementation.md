# Species Field Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace complex custom species dropdown with native HTML5 datalist, remove species table, use freeform text with autocomplete from user catch history.

**Architecture:** Database migration adds species_name to catches, backfills from species table, drops species table and species_id FK. New autocomplete endpoint returns frequency-sorted species from user history. Frontend uses native `<datalist>` instead of custom dropdown.

**Tech Stack:** Go 1.25, SQLite, Svelte 5, chi router, sqlx

---

## Task 1: Database Migration

**Files:**
- Modify: `internal/database/migrations.go`

**Step 1: Add migration function**

Add the new migration function after the existing `RunMigrations` function:

```go
func migrateSpeciesToFreeform(db *sqlx.DB) error {
	// Check if species_name column already exists (idempotent check)
	var colExists int
	_ = db.Get(&colExists, "SELECT COUNT(*) FROM pragma_table_info('catches') WHERE name='species_name'")
	if colExists > 0 {
		return nil // Migration already applied
	}

	// Step 1: Add species_name column
	if _, err := db.Exec("ALTER TABLE catches ADD COLUMN species_name TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	// Step 2: Backfill species_name from species table
	if _, err := db.Exec(`
		UPDATE catches
		SET species_name = (SELECT name FROM species WHERE species.id = catches.species_id)
		WHERE species_id IS NOT NULL
	`); err != nil {
		return err
	}

	// Step 3: Recreate catches table without species_id (SQLite requires table recreation to drop columns)
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Create new catches table without species_id
	if _, err := tx.Exec(`
		CREATE TABLE catches_new (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			trip_id INTEGER,
			species_name TEXT NOT NULL,
			caught_at DATETIME NOT NULL,
			latitude REAL,
			longitude REAL,
			location_name TEXT,
			length_in REAL,
			weight_lb REAL,
			kept BOOLEAN DEFAULT 0,
			bait_or_lure TEXT,
			rod_setup TEXT,
			line_info TEXT,
			hook_size TEXT,
			air_temp_f REAL,
			wind_mph REAL,
			wind_dir TEXT,
			conditions TEXT,
			pressure_mb REAL,
			humidity_pct REAL,
			water_temp_f REAL,
			water_clarity TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (trip_id) REFERENCES trips(id)
		)
	`); err != nil {
		return err
	}

	// Copy data from old table to new table
	if _, err := tx.Exec(`
		INSERT INTO catches_new SELECT
			id, user_id, trip_id, species_name, caught_at, latitude, longitude,
			location_name, length_in, weight_lb, kept, bait_or_lure, rod_setup,
			line_info, hook_size, air_temp_f, wind_mph, wind_dir, conditions,
			pressure_mb, humidity_pct, water_temp_f, water_clarity, notes,
			created_at, updated_at
		FROM catches
	`); err != nil {
		return err
	}

	// Drop old table
	if _, err := tx.Exec("DROP TABLE catches"); err != nil {
		return err
	}

	// Rename new table
	if _, err := tx.Exec("ALTER TABLE catches_new RENAME TO catches"); err != nil {
		return err
	}

	// Recreate indexes
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_catches_user_id ON catches(user_id);
		CREATE INDEX IF NOT EXISTS idx_catches_trip_id ON catches(trip_id);
	`); err != nil {
		return err
	}

	// Step 4: Drop species table
	if _, err := tx.Exec("DROP TABLE species"); err != nil {
		return err
	}

	// Step 5: Recreate user_settings table without species_filter
	if _, err := tx.Exec(`
		CREATE TABLE user_settings_new (
			id INTEGER PRIMARY KEY,
			user_id INTEGER UNIQUE NOT NULL,
			theme TEXT NOT NULL DEFAULT 'system',
			units TEXT NOT NULL DEFAULT 'imperial',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		return err
	}

	// Copy data from old table to new table
	if _, err := tx.Exec(`
		INSERT INTO user_settings_new SELECT id, user_id, theme, units, updated_at
		FROM user_settings
	`); err != nil {
		return err
	}

	// Drop old table
	if _, err := tx.Exec("DROP TABLE user_settings"); err != nil {
		return err
	}

	// Rename new table
	if _, err := tx.Exec("ALTER TABLE user_settings_new RENAME TO user_settings"); err != nil {
		return err
	}

	return tx.Commit()
}
```

**Step 2: Call migration in RunMigrations**

Find the `RunMigrations` function and add the migration call after `seedSpecies(db)`:

```go
func RunMigrations(db *sqlx.DB) error {
	schema := `
	// ... existing schema ...
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Add species_filter column to existing user_settings tables (ignore error if column already exists)
	_, _ = db.Exec("ALTER TABLE user_settings ADD COLUMN species_filter TEXT NOT NULL DEFAULT 'all'")

	if err := seedSpecies(db); err != nil {
		return err
	}

	// NEW: Run species-to-freeform migration
	return migrateSpeciesToFreeform(db)
}
```

**Step 3: Test migration locally**

Run: `FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./test.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale`

Expected: App starts without errors, migration runs successfully

**Step 4: Verify migration in database**

Run: `sqlite3 test.db "PRAGMA table_info(catches);" | grep species`

Expected: Only `species_name` column exists, no `species_id`

Run: `sqlite3 test.db "SELECT name FROM sqlite_master WHERE type='table' AND name='species';"`

Expected: Empty result (species table dropped)

**Step 5: Commit**

```bash
git add internal/database/migrations.go
git commit -m "feat(db): add migration to replace species_id with species_name

Adds species_name column, backfills from species table, drops species_id
FK and species table entirely. Removes species_filter from user_settings.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Update Model Structs

**Files:**
- Modify: `internal/model/models.go`

**Step 1: Remove Species struct**

Delete lines 23-27 (the entire Species struct):

```go
// DELETE THIS:
type Species struct {
	ID       int64  `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
}
```

**Step 2: Update Catch struct**

Remove `SpeciesID` field (line 33), keep only `SpeciesName` (already exists at line 57):

```go
type Catch struct {
	ID           int64      `db:"id" json:"id"`
	UserID       int64      `db:"user_id" json:"user_id"`
	TripID       *int64     `db:"trip_id" json:"trip_id"`
	// REMOVE: SpeciesID    *int64     `db:"species_id" json:"species_id"`
	SpeciesName  string     `db:"species_name" json:"species_name"` // MOVE THIS UP
	CaughtAt     time.Time  `db:"caught_at" json:"caught_at"`
	// ... rest unchanged
}
```

**Step 3: Update CreateCatchRequest struct**

Replace `SpeciesID *int64` with `SpeciesName string` (around line 81):

```go
type CreateCatchRequest struct {
	TripID       *int64   `json:"trip_id"`
	SpeciesName  string   `json:"species_name"` // CHANGED from SpeciesID
	CaughtAt     string   `json:"caught_at"`
	// ... rest unchanged
}
```

**Step 4: Update UserSettings struct**

Remove `SpeciesFilter` field (line 75):

```go
type UserSettings struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Theme     string    `db:"theme" json:"theme"`
	Units     string    `db:"units" json:"units"`
	// REMOVE: SpeciesFilter string    `db:"species_filter" json:"species_filter"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
```

**Step 5: Update SpeciesCount and PersonalBest structs**

Remove species_id fields (lines 115 and 121):

```go
type SpeciesCount struct {
	// REMOVE: SpeciesID   int64  `db:"species_id" json:"species_id"`
	SpeciesName string `db:"species_name" json:"species_name"`
	Count       int    `db:"count" json:"count"`
}

type PersonalBest struct {
	// REMOVE: SpeciesID   int64   `db:"species_id" json:"species_id"`
	SpeciesName string  `db:"species_name" json:"species_name"`
	MaxWeightLb float64 `db:"max_weight_lb" json:"max_weight_lb"`
	MaxLengthIn float64 `db:"max_length_in" json:"max_length_in"`
}
```

**Step 6: Commit**

```bash
git add internal/model/models.go
git commit -m "refactor(model): remove Species struct and species_id fields

Use species_name string instead of species_id FK. Remove Species struct
and SpeciesFilter from settings.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Create Autocomplete Handler

**Files:**
- Create: `internal/handler/autocomplete.go`

**Step 1: Write autocomplete handler**

```go
package handler

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
)

type AutocompleteHandler struct {
	db *sqlx.DB
}

func NewAutocompleteHandler(db *sqlx.DB) *AutocompleteHandler {
	return &AutocompleteHandler{db: db}
}

func (h *AutocompleteHandler) Species(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	type speciesResult struct {
		SpeciesName string `db:"species_name"`
	}

	query := `
		SELECT species_name, COUNT(*) as catch_count
		FROM catches
		WHERE user_id = ? AND species_name != ''
		GROUP BY species_name
		ORDER BY catch_count DESC
		LIMIT 50
	`

	var results []speciesResult
	if err := h.db.SelectContext(r.Context(), &results, query, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query species")
		return
	}

	// Extract just the names as a string array
	names := make([]string, 0, len(results))
	for _, r := range results {
		names = append(names, r.SpeciesName)
	}

	jsonResponse(w, http.StatusOK, names)
}
```

**Step 2: Commit**

```bash
git add internal/handler/autocomplete.go
git commit -m "feat(handler): add autocomplete handler for species

Returns frequency-sorted unique species names from user catch history.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Update Catch Handler

**Files:**
- Modify: `internal/handler/catches.go`

**Step 1: Update List query**

Replace lines 33-37 to remove species JOIN:

```go
// OLD:
query := `SELECT c.*, COALESCE(s.name, '') as species_name
	FROM catches c
	LEFT JOIN species s ON c.species_id = s.id
	WHERE c.user_id = ?
	ORDER BY c.caught_at DESC`

// NEW:
query := `SELECT * FROM catches WHERE user_id = ? ORDER BY caught_at DESC`
```

**Step 2: Update Get query**

Replace lines 66-69 to remove species JOIN:

```go
// OLD:
query := `SELECT c.*, COALESCE(s.name, '') as species_name
	FROM catches c
	LEFT JOIN species s ON c.species_id = s.id
	WHERE c.id = ? AND c.user_id = ?`

// NEW:
query := `SELECT * FROM catches WHERE id = ? AND user_id = ?`
```

**Step 3: Update Create handler**

Replace lines 109-119 to use species_name instead of species_id:

```go
// OLD:
result, err := h.db.ExecContext(r.Context(), `INSERT INTO catches (
	user_id, trip_id, species_id, caught_at, latitude, longitude, location_name,
	length_in, weight_lb, kept, bait_or_lure, rod_setup, line_info, hook_size,
	air_temp_f, wind_mph, wind_dir, conditions, pressure_mb, humidity_pct,
	water_temp_f, water_clarity, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	user.ID, req.TripID, req.SpeciesID, caughtAt, req.Latitude, req.Longitude, req.LocationName,
	// ...

// NEW:
result, err := h.db.ExecContext(r.Context(), `INSERT INTO catches (
	user_id, trip_id, species_name, caught_at, latitude, longitude, location_name,
	length_in, weight_lb, kept, bait_or_lure, rod_setup, line_info, hook_size,
	air_temp_f, wind_mph, wind_dir, conditions, pressure_mb, humidity_pct,
	water_temp_f, water_clarity, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	user.ID, req.TripID, req.SpeciesName, caughtAt, req.Latitude, req.Longitude, req.LocationName,
	req.LengthIn, req.WeightLb, req.Kept, req.BaitOrLure, req.RodSetup, req.LineInfo, req.HookSize,
	req.AirTempF, req.WindMph, req.WindDir, req.Conditions, req.PressureMb, req.HumidityPct,
	req.WaterTempF, req.WaterClarity, req.Notes,
)
```

**Step 4: Update Create fetch query**

Replace lines 132-135 to remove species JOIN:

```go
// OLD:
if err := h.db.GetContext(r.Context(), &catch, `SELECT c.*, COALESCE(s.name, '') as species_name
	FROM catches c
	LEFT JOIN species s ON c.species_id = s.id
	WHERE c.id = ?`, id); err != nil {

// NEW:
if err := h.db.GetContext(r.Context(), &catch, `SELECT * FROM catches WHERE id = ?`, id); err != nil {
```

**Step 5: Update Update handler**

Find the Update method (around line 143) and update the query to use species_name instead of species_id. Replace the UPDATE query:

```go
// OLD:
_, err = h.db.ExecContext(r.Context(), `UPDATE catches SET
	trip_id = ?, species_id = ?, caught_at = ?, latitude = ?, longitude = ?,
	// ...
	WHERE id = ? AND user_id = ?`,
	req.TripID, req.SpeciesID, caughtAt, req.Latitude, req.Longitude,
	// ...

// NEW:
_, err = h.db.ExecContext(r.Context(), `UPDATE catches SET
	trip_id = ?, species_name = ?, caught_at = ?, latitude = ?, longitude = ?,
	location_name = ?, length_in = ?, weight_lb = ?, kept = ?,
	bait_or_lure = ?, rod_setup = ?, line_info = ?, hook_size = ?,
	air_temp_f = ?, wind_mph = ?, wind_dir = ?, conditions = ?,
	pressure_mb = ?, humidity_pct = ?, water_temp_f = ?, water_clarity = ?,
	notes = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ? AND user_id = ?`,
	req.TripID, req.SpeciesName, caughtAt, req.Latitude, req.Longitude,
	req.LocationName, req.LengthIn, req.WeightLb, req.Kept,
	req.BaitOrLure, req.RodSetup, req.LineInfo, req.HookSize,
	req.AirTempF, req.WindMph, req.WindDir, req.Conditions,
	req.PressureMb, req.HumidityPct, req.WaterTempF, req.WaterClarity,
	req.Notes, id, user.ID,
)
```

**Step 6: Update fetch query in Update handler**

Replace the SELECT query in Update handler to remove species JOIN.

**Step 7: Commit**

```bash
git add internal/handler/catches.go
git commit -m "refactor(handler): update catch handler to use species_name

Replace species_id with species_name in create/update/list/get operations.
Remove species table JOINs.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Update Validate Function

**Files:**
- Modify: `internal/handler/validate.go`

**Step 1: Update validation**

Find `validateCatchRequest` function and update to validate species_name instead of species_id:

```go
func validateCatchRequest(req *model.CreateCatchRequest) error {
	// Remove species_id validation, add species_name validation
	if req.SpeciesName == "" {
		return fmt.Errorf("species_name is required")
	}
	if len(req.SpeciesName) > 100 {
		return fmt.Errorf("species_name must be 100 characters or less")
	}

	// ... rest of validation unchanged
}
```

**Step 2: Commit**

```bash
git add internal/handler/validate.go
git commit -m "refactor(validate): validate species_name instead of species_id

Require species_name string, max 100 characters.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Update Settings Handler

**Files:**
- Modify: `internal/handler/settings.go`

**Step 1: Remove species_filter from Get handler**

Find the `Get` method and ensure the query selects only theme and units (no species_filter).

**Step 2: Remove species_filter from Update handler**

Find the `Update` method and remove species_filter from the UPDATE query.

**Step 3: Commit**

```bash
git add internal/handler/settings.go
git commit -m "refactor(handler): remove species_filter from settings

No longer reading or writing species_filter field.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: Update Stats Handler

**Files:**
- Modify: `internal/handler/stats.go`

**Step 1: Update stats queries**

Find all queries that reference species table and update to use species_name directly from catches table. Remove JOINs with species table.

For example, species counts query should become:

```go
// OLD:
query := `SELECT s.id as species_id, s.name as species_name, COUNT(*) as count
	FROM catches c
	JOIN species s ON c.species_id = s.id
	WHERE c.user_id = ?
	GROUP BY s.id, s.name
	ORDER BY count DESC
	LIMIT 10`

// NEW:
query := `SELECT species_name, COUNT(*) as count
	FROM catches
	WHERE user_id = ? AND species_name != ''
	GROUP BY species_name
	ORDER BY count DESC
	LIMIT 10`
```

Update personal bests query similarly.

**Step 2: Commit**

```bash
git add internal/handler/stats.go
git commit -m "refactor(handler): update stats handler to use species_name

Query species_name directly from catches, remove species table JOINs.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: Update Export Handler

**Files:**
- Modify: `internal/handler/export.go`

**Step 1: Update export queries**

Find queries that JOIN with species table and remove the JOIN, use species_name from catches directly.

**Step 2: Commit**

```bash
git add internal/handler/export.go
git commit -m "refactor(handler): update export handler to use species_name

Query species_name directly from catches, remove species table JOINs.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: Delete Species Handler

**Files:**
- Delete: `internal/handler/species.go`

**Step 1: Delete the file**

Run: `rm internal/handler/species.go`

**Step 2: Commit**

```bash
git rm internal/handler/species.go
git commit -m "refactor(handler): remove species handler

No longer needed - species is freeform text now.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: Update Routes

**Files:**
- Modify: `internal/server/server.go`

**Step 1: Remove species handler initialization**

Remove line 44:

```go
// DELETE THIS:
species := handler.NewSpeciesHandler(db)
```

**Step 2: Add autocomplete handler initialization**

Add after line 49:

```go
autocomplete := handler.NewAutocompleteHandler(db)
```

**Step 3: Remove species routes**

Remove lines 67-70:

```go
// DELETE THIS:
r.Route("/species", func(r chi.Router) {
	r.Get("/", species.List)
	r.Post("/", species.Create)
})
```

**Step 4: Add autocomplete routes**

Add after the trips routes (after line 66):

```go
r.Get("/autocomplete/species", autocomplete.Species)
```

**Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "refactor(server): replace species routes with autocomplete

Remove species endpoints, add autocomplete/species endpoint.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 11: Update Backend Tests

**Files:**
- Modify: `internal/handler/catches_test.go`

**Step 1: Update test fixtures to use species_name**

Find all test cases that use `species_id` and replace with `species_name`:

```go
// OLD:
body := `{"species_id": 1, "caught_at": "2024-01-01T12:00:00Z", ...}`

// NEW:
body := `{"species_name": "Largemouth Bass", "caught_at": "2024-01-01T12:00:00Z", ...}`
```

**Step 2: Run tests**

Run: `GOWORK=off go test ./internal/handler/ -v`

Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/handler/catches_test.go
git commit -m "test(handler): update catch tests to use species_name

Replace species_id with species_name in test fixtures.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 12: Add Autocomplete Test

**Files:**
- Create: `internal/handler/autocomplete_test.go`

**Step 1: Write test**

```go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAutocompleteSpecies(t *testing.T) {
	_, router := setupFullRouter(t)

	// Create some catches with species names
	createCatch(t, router, `{"species_name": "Largemouth Bass", "caught_at": "2024-01-01T12:00:00Z"}`)
	createCatch(t, router, `{"species_name": "Largemouth Bass", "caught_at": "2024-01-02T12:00:00Z"}`)
	createCatch(t, router, `{"species_name": "Bluegill", "caught_at": "2024-01-03T12:00:00Z"}`)

	// Request autocomplete
	req := httptest.NewRequest("GET", "/api/v1/autocomplete/species", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var names []string
	if err := json.NewDecoder(rec.Body).Decode(&names); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return frequency-sorted: Largemouth Bass (2 catches) before Bluegill (1 catch)
	if len(names) != 2 {
		t.Fatalf("expected 2 species, got %d", len(names))
	}
	if names[0] != "Largemouth Bass" {
		t.Errorf("expected first species to be Largemouth Bass, got %s", names[0])
	}
	if names[1] != "Bluegill" {
		t.Errorf("expected second species to be Bluegill, got %s", names[1])
	}
}

func createCatch(t *testing.T, router http.Handler, body string) {
	req := httptest.NewRequest("POST", "/api/v1/catches", nil)
	req.Body = http.NoBody
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	// Simplified helper - expand as needed
}
```

**Step 2: Run test**

Run: `GOWORK=off go test ./internal/handler/ -run TestAutocompleteSpecies -v`

Expected: Test passes

**Step 3: Commit**

```bash
git add internal/handler/autocomplete_test.go
git commit -m "test(handler): add autocomplete species test

Test frequency sorting of species from user catch history.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 13: Update Frontend API Client

**Files:**
- Modify: `frontend/src/lib/api.ts`

**Step 1: Remove species methods**

Find and delete the species section:

```typescript
// DELETE THIS:
species: {
  list: async (): Promise<Species[]> => {
    const res = await fetch('/api/v1/species');
    if (!res.ok) throw new Error('Failed to fetch species');
    return res.json();
  },
  create: async (name: string, category: string): Promise<Species> => {
    const res = await fetch('/api/v1/species', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, category }),
    });
    if (!res.ok) throw new Error('Failed to create species');
    return res.json();
  },
},
```

**Step 2: Add autocomplete methods**

Add new autocomplete section:

```typescript
autocomplete: {
  species: async (): Promise<string[]> => {
    const res = await fetch('/api/v1/autocomplete/species');
    if (!res.ok) throw new Error('Failed to fetch species autocomplete');
    return res.json();
  },
},
```

**Step 3: Update Catch interface**

Find the Catch interface and update:

```typescript
interface Catch {
  // ... other fields
  species_name: string;  // CHANGE from species_id: number
  // ... rest
}
```

**Step 4: Update catches.create and catches.update**

Update the request body type to use species_name:

```typescript
catches: {
  create: async (data: {
    species_name: string;  // CHANGED from species_id: number
    // ... rest
  }): Promise<Catch> => {
    // ... implementation
  },
  update: async (id: number, data: {
    species_name: string;  // CHANGED from species_id: number
    // ... rest
  }): Promise<Catch> => {
    // ... implementation
  },
},
```

**Step 5: Update Settings interface**

Remove species_filter:

```typescript
interface Settings {
  theme: string;
  units: string;
  // REMOVE: species_filter: string;
}
```

**Step 6: Commit**

```bash
git add frontend/src/lib/api.ts
git commit -m "refactor(frontend): update API client for species redesign

Add autocomplete.species(), update catches to use species_name, remove
species endpoints and species_filter from settings.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 14: Update LogCatch Component

**Files:**
- Modify: `frontend/src/lib/pages/LogCatch.svelte`

**Step 1: Remove species dropdown state and logic**

Delete lines 8-12 (species state variables):

```typescript
// DELETE:
let speciesList = $state<any[]>([]);
let speciesQuery = $state('');
let filteredSpecies = $state<any[]>([]);
let showSpeciesDropdown = $state(false);
let justSelected = false;
```

**Step 2: Add species autocomplete state**

Add after line 7:

```typescript
let speciesSuggestions = $state<string[]>([]);
```

**Step 3: Remove species $effect**

Delete lines 48-76 (species loading and filtering effects):

```typescript
// DELETE:
$effect(() => {
  api.species.list().then((s) => {
    speciesList = s;
  });
});

$effect(() => {
  if (justSelected) {
    justSelected = false;
    return;
  }
  const filter = $settings.species_filter || 'all';
  if (speciesQuery.length > 0) {
    filteredSpecies = speciesList
      .filter((s) => {
        const matchesQuery = s.name.toLowerCase().includes(speciesQuery.toLowerCase());
        const matchesCategory = filter === 'all' || s.category === filter;
        return matchesQuery && matchesCategory;
      })
      .slice(0, 8);
    showSpeciesDropdown = filteredSpecies.length > 0;
  } else {
    showSpeciesDropdown = false;
    form.species_id = null;
    form.species_name = '';
  }
});
```

**Step 4: Add autocomplete loading effect**

Add after the GPS effect:

```typescript
// Load species autocomplete
$effect(() => {
  api.autocomplete.species().then(s => {
    speciesSuggestions = s;
  });
});
```

**Step 5: Update form state**

Replace lines 21-46, change species fields in form state:

```typescript
let form = $state({
  caught_at: new Date(Date.now() - new Date().getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16),
  latitude: null as number | null,
  longitude: null as null,
  location_name: '',
  species_name: '',  // CHANGED: single field instead of species_id + species_name
  bait_or_lure: '',
  // ... rest unchanged
});
```

**Step 6: Remove selectSpecies and dismiss functions**

Delete lines 106-129:

```typescript
// DELETE:
function selectSpecies(s: any, e?: Event) {
  e?.preventDefault();
  justSelected = true;
  form.species_id = s.id;
  form.species_name = s.name;
  speciesQuery = s.name;
  showSpeciesDropdown = false;
}

let dismissTimer: ReturnType<typeof setTimeout> | null = null;

function dismissDropdown() {
  dismissTimer = setTimeout(() => {
    showSpeciesDropdown = false;
  }, 200);
}

function cancelDismiss() {
  if (dismissTimer) {
    clearTimeout(dismissTimer);
    dismissTimer = null;
  }
}
```

**Step 7: Update species field markup**

Replace lines 216-241 (species field):

```svelte
<!-- OLD: -->
<div class="form-group species-field">
  <label>Species</label>
  <input
    type="text"
    placeholder="Search species..."
    bind:value={speciesQuery}
    onfocus={() => {
      if (speciesQuery.length > 0 && !justSelected) showSpeciesDropdown = true;
    }}
    onblur={dismissDropdown}
  />
  {#if showSpeciesDropdown}
    <div class="dropdown" ontouchstart={cancelDismiss}>
      {#each filteredSpecies as s}
        <button
          class="dropdown-item"
          ontouchend={(e) => selectSpecies(s, e)}
          onmousedown={() => selectSpecies(s)}
        >
          {s.name}
          <span class="chip">{s.category}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- NEW: -->
<div class="form-group">
  <label>Species</label>
  <input
    type="text"
    list="species-datalist"
    placeholder="e.g. Largemouth Bass"
    bind:value={form.species_name}
  />
  <datalist id="species-datalist">
    {#each speciesSuggestions as species}
      <option value={species}></option>
    {/each}
  </datalist>
</div>
```

**Step 8: Update save function**

Update to send species_name instead of species_id (around line 147):

```typescript
// OLD:
const created = await api.catches.create({
  // ...
  species_id: form.species_id,
  // ...
});

// NEW:
const created = await api.catches.create({
  // ...
  species_name: form.species_name,
  // ...
});
```

**Step 9: Remove custom dropdown CSS**

Delete lines 376-410 (dropdown styles):

```css
/* DELETE:
.species-field {
  position: relative;
}

.dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  max-height: 200px;
  overflow-y: auto;
  z-index: 10;
  box-shadow: 0 4px 12px var(--shadow);
}

.dropdown-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding: 10px 12px;
  border: none;
  background: none;
  color: var(--text);
  cursor: pointer;
  text-align: left;
  font-size: 0.9rem;
}

.dropdown-item:hover {
  background: var(--bg-secondary);
}
*/
```

**Step 10: Commit**

```bash
git add frontend/src/lib/pages/LogCatch.svelte
git commit -m "refactor(frontend): replace custom dropdown with native datalist

Remove ~80 lines of custom dropdown code. Use HTML5 datalist with
autocomplete from user catch history. Species is now freeform text.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 15: Update Settings Component

**Files:**
- Modify: `frontend/src/lib/pages/Settings.svelte`

**Step 1: Remove species filter from form**

Find the species filter radio group and delete it (look for "Species Filter" label and the radio buttons for All/Freshwater/Saltwater).

**Step 2: Remove species_filter from form state**

Remove species_filter field from the local form state.

**Step 3: Commit**

```bash
git add frontend/src/lib/pages/Settings.svelte
git commit -m "refactor(frontend): remove species filter from settings

No longer needed with freeform species field.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 16: Run Full CI Check

**Files:**
- None (validation step)

**Step 1: Run all checks**

Run: `make ci`

Expected: All tests pass, linting passes, frontend builds successfully

**Step 2: Fix any issues**

If any tests or lints fail, fix them and commit.

---

## Task 17: Update Design Doc

**Files:**
- Modify: `docs/plans/2026-02-16-fishscale-design.md`

**Step 1: Update data model section**

Update the catches table schema to show species_name instead of species_id. Remove species table documentation. Remove species_filter from user_settings.

**Step 2: Update API design section**

Remove species endpoints documentation. Add autocomplete endpoint documentation.

**Step 3: Update UX design section**

Update the species dropdown description to mention native datalist instead of custom dropdown.

**Step 4: Update Ideas & Enhancements**

Mark the species dropdown investigation as completed:

```markdown
- [x] ~~Investigate species dropdown~~ (Completed in 2026-02-17: replaced with native datalist)
```

**Step 5: Commit**

```bash
git add docs/plans/2026-02-16-fishscale-design.md
git commit -m "docs: update design doc to reflect species field redesign

Species is now freeform text with native datalist autocomplete.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 18: Manual Testing

**Files:**
- None (testing step)

**Step 1: Start dev server**

Run: `make dev`

**Step 2: Test species field**

- Navigate to Log Catch page
- Type in species field, verify autocomplete appears
- Select from autocomplete, verify it fills the field
- Type freeform text, verify it works
- Save a catch with species name
- Verify it appears in catch log with species name

**Step 3: Test on iOS Safari (if available)**

- Open app on iPhone
- Test species field interaction
- Verify no UI lockups
- Verify field is responsive and dismissible

**Step 4: Verify migration**

- Check database: `sqlite3 fish.db "PRAGMA table_info(catches);" | grep species`
- Should show only species_name, no species_id
- Check that existing catches have species_name populated

---

## Summary

**Total Tasks:** 18 tasks across backend, frontend, and testing

**Lines of Code:**
- Backend: +150 (migration, autocomplete), -200 (species handler, JOINs)
- Frontend: +15 (datalist), -80 (custom dropdown)
- **Net: -115 lines** (significant simplification)

**Key Changes:**
1. Database migration preserves all data while removing species table
2. Autocomplete endpoint returns frequency-sorted species from user history
3. Native HTML5 datalist replaces complex custom dropdown
4. All species_id references replaced with species_name
5. Species filter setting removed entirely

**Testing:**
- Backend: Update existing tests, add autocomplete test
- Frontend: Manual testing on desktop and iOS Safari
- Integration: Full `make ci` check before deployment
