package database

import "github.com/jmoiron/sqlx"

func migrate(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		tailscale_id  TEXT    NOT NULL UNIQUE,
		display_name  TEXT    NOT NULL DEFAULT '',
		created_at    DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS trips (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users(id),
		name       TEXT    NOT NULL DEFAULT '',
		started_at DATETIME,
		ended_at   DATETIME,
		notes      TEXT    NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS species (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		name     TEXT    NOT NULL UNIQUE,
		category TEXT    NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS catches (
		id            INTEGER  PRIMARY KEY AUTOINCREMENT,
		user_id       INTEGER  NOT NULL REFERENCES users(id),
		trip_id       INTEGER  REFERENCES trips(id),
		species_id    INTEGER  REFERENCES species(id),
		caught_at     DATETIME NOT NULL DEFAULT (datetime('now')),
		latitude      REAL,
		longitude     REAL,
		location_name TEXT     NOT NULL DEFAULT '',
		length_in     REAL,
		weight_lb     REAL,
		kept          INTEGER  NOT NULL DEFAULT 0,
		bait_or_lure  TEXT     NOT NULL DEFAULT '',
		rod_setup     TEXT     NOT NULL DEFAULT '',
		line_info     TEXT     NOT NULL DEFAULT '',
		hook_size     TEXT     NOT NULL DEFAULT '',
		air_temp_f    REAL,
		wind_mph      REAL,
		wind_dir      TEXT     NOT NULL DEFAULT '',
		conditions    TEXT     NOT NULL DEFAULT '',
		pressure_mb   REAL,
		humidity_pct  REAL,
		water_temp_f  REAL,
		water_clarity TEXT     NOT NULL DEFAULT '',
		notes         TEXT     NOT NULL DEFAULT '',
		created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at    DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS photos (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		catch_id   INTEGER NOT NULL REFERENCES catches(id) ON DELETE CASCADE,
		filename   TEXT    NOT NULL,
		thumbnail  TEXT    NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS user_settings (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id        INTEGER NOT NULL UNIQUE REFERENCES users(id),
		theme          TEXT    NOT NULL DEFAULT 'system',
		units          TEXT    NOT NULL DEFAULT 'imperial',
		species_filter TEXT    NOT NULL DEFAULT 'all',
		updated_at     DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_catches_user_id    ON catches(user_id);
	CREATE INDEX IF NOT EXISTS idx_catches_caught_at  ON catches(caught_at);
	CREATE INDEX IF NOT EXISTS idx_catches_species_id ON catches(species_id);
	CREATE INDEX IF NOT EXISTS idx_catches_trip_id    ON catches(trip_id);
	CREATE INDEX IF NOT EXISTS idx_photos_catch_id    ON photos(catch_id);
	CREATE INDEX IF NOT EXISTS idx_trips_user_id      ON trips(user_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Add species_filter column to existing user_settings tables (ignore error if column already exists)
	_, _ = db.Exec("ALTER TABLE user_settings ADD COLUMN species_filter TEXT NOT NULL DEFAULT 'all'")

	return seedSpecies(db)
}

func seedSpecies(db *sqlx.DB) error {
	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM species"); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	species := []struct {
		Name     string
		Category string
	}{
		// Freshwater - Bass
		{"Largemouth Bass", "freshwater"},
		{"Smallmouth Bass", "freshwater"},
		{"Spotted Bass", "freshwater"},
		{"Striped Bass", "freshwater"},
		{"White Bass", "freshwater"},
		// Freshwater - Panfish
		{"Bluegill", "freshwater"},
		{"Crappie (Black)", "freshwater"},
		{"Crappie (White)", "freshwater"},
		{"Pumpkinseed", "freshwater"},
		{"Rock Bass", "freshwater"},
		{"Warmouth", "freshwater"},
		{"Yellow Perch", "freshwater"},
		// Freshwater - Trout & Salmon
		{"Rainbow Trout", "freshwater"},
		{"Brown Trout", "freshwater"},
		{"Brook Trout", "freshwater"},
		{"Lake Trout", "freshwater"},
		{"Cutthroat Trout", "freshwater"},
		{"Chinook Salmon", "freshwater"},
		{"Coho Salmon", "freshwater"},
		// Freshwater - Catfish
		{"Channel Catfish", "freshwater"},
		{"Blue Catfish", "freshwater"},
		{"Flathead Catfish", "freshwater"},
		// Freshwater - Pike & Muskie
		{"Northern Pike", "freshwater"},
		{"Muskellunge", "freshwater"},
		{"Chain Pickerel", "freshwater"},
		// Freshwater - Walleye & Sauger
		{"Walleye", "freshwater"},
		{"Sauger", "freshwater"},
		// Freshwater - Carp
		{"Common Carp", "freshwater"},
		{"Grass Carp", "freshwater"},
		// Saltwater
		{"Redfish (Red Drum)", "saltwater"},
		{"Speckled Trout", "saltwater"},
		{"Flounder", "saltwater"},
		{"Snook", "saltwater"},
		{"Tarpon", "saltwater"},
		{"Mahi-Mahi", "saltwater"},
		{"Red Snapper", "saltwater"},
		{"Grouper", "saltwater"},
		{"King Mackerel", "saltwater"},
		{"Spanish Mackerel", "saltwater"},
		{"Cobia", "saltwater"},
		{"Sheepshead", "saltwater"},
		{"Pompano", "saltwater"},
		{"Tuna (Yellowfin)", "saltwater"},
		{"Wahoo", "saltwater"},
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	for _, s := range species {
		if _, err := tx.Exec("INSERT INTO species (name, category) VALUES (?, ?)", s.Name, s.Category); err != nil {
			return err
		}
	}

	return tx.Commit()
}
