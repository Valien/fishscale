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

	CREATE TABLE IF NOT EXISTS catches (
		id            INTEGER  PRIMARY KEY AUTOINCREMENT,
		user_id       INTEGER  NOT NULL REFERENCES users(id),
		trip_id       INTEGER  REFERENCES trips(id),
		species_name  TEXT     NOT NULL DEFAULT '',
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
		updated_at     DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_catches_user_id      ON catches(user_id);
	CREATE INDEX IF NOT EXISTS idx_catches_caught_at    ON catches(caught_at);
	CREATE INDEX IF NOT EXISTS idx_catches_species_name ON catches(species_name);
	CREATE INDEX IF NOT EXISTS idx_catches_trip_id      ON catches(trip_id);
	CREATE INDEX IF NOT EXISTS idx_photos_catch_id      ON photos(catch_id);
	CREATE INDEX IF NOT EXISTS idx_trips_user_id        ON trips(user_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Run migration for existing databases that have the old species-based schema
	if err := migrateSpeciesToFreeform(db); err != nil {
		return err
	}

	return nil
}

// migrateSpeciesToFreeform migrates from species_id to species_name for existing databases.
// This is idempotent - it checks if migration is needed before running.
func migrateSpeciesToFreeform(db *sqlx.DB) error {
	// Check if the old species table exists
	var speciesTableExists int
	err := db.Get(&speciesTableExists, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='species'")
	if err != nil {
		return err
	}
	if speciesTableExists == 0 {
		// No species table = fresh database or already migrated
		return nil
	}

	// Check if catches table still has species_id column
	var hasSpeciesID int
	err = db.Get(&hasSpeciesID, "SELECT COUNT(*) FROM pragma_table_info('catches') WHERE name='species_id'")
	if err != nil {
		return err
	}
	if hasSpeciesID == 0 {
		// Already migrated, just need to drop species table
		_, _ = db.Exec("DROP TABLE IF EXISTS species")
		return nil
	}

	// Start migration transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Step 1: Add species_name column if it doesn't exist
	_, _ = tx.Exec("ALTER TABLE catches ADD COLUMN species_name TEXT NOT NULL DEFAULT ''")

	// Step 2: Backfill species_name from species table
	_, err = tx.Exec(`UPDATE catches SET species_name = COALESCE(
		(SELECT name FROM species WHERE species.id = catches.species_id), ''
	) WHERE species_name = ''`)
	if err != nil {
		return err
	}

	// Step 3: Recreate catches table without species_id (SQLite requires table recreation to drop columns)
	_, err = tx.Exec(`
		CREATE TABLE catches_new (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			user_id       INTEGER  NOT NULL REFERENCES users(id),
			trip_id       INTEGER  REFERENCES trips(id),
			species_name  TEXT     NOT NULL DEFAULT '',
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
		)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO catches_new (
		id, user_id, trip_id, species_name, caught_at, latitude, longitude, location_name,
		length_in, weight_lb, kept, bait_or_lure, rod_setup, line_info, hook_size,
		air_temp_f, wind_mph, wind_dir, conditions, pressure_mb, humidity_pct,
		water_temp_f, water_clarity, notes, created_at, updated_at
	) SELECT
		id, user_id, trip_id, species_name, caught_at, latitude, longitude, location_name,
		length_in, weight_lb, kept, bait_or_lure, rod_setup, line_info, hook_size,
		air_temp_f, wind_mph, wind_dir, conditions, pressure_mb, humidity_pct,
		water_temp_f, water_clarity, notes, created_at, updated_at
	FROM catches`)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DROP TABLE catches")
	if err != nil {
		return err
	}

	_, err = tx.Exec("ALTER TABLE catches_new RENAME TO catches")
	if err != nil {
		return err
	}

	// Recreate indexes for catches
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_catches_user_id ON catches(user_id)")
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_catches_caught_at ON catches(caught_at)")
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_catches_species_name ON catches(species_name)")
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_catches_trip_id ON catches(trip_id)")

	// Step 4: Drop the species table
	_, err = tx.Exec("DROP TABLE species")
	if err != nil {
		return err
	}

	// Step 5: Remove species_filter from user_settings (recreate table)
	var hasSpeciesFilter int
	err = tx.Get(&hasSpeciesFilter, "SELECT COUNT(*) FROM pragma_table_info('user_settings') WHERE name='species_filter'")
	if err != nil {
		return err
	}
	if hasSpeciesFilter > 0 {
		_, err = tx.Exec(`
			CREATE TABLE user_settings_new (
				id             INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id        INTEGER NOT NULL UNIQUE REFERENCES users(id),
				theme          TEXT    NOT NULL DEFAULT 'system',
				units          TEXT    NOT NULL DEFAULT 'imperial',
				updated_at     DATETIME NOT NULL DEFAULT (datetime('now'))
			)`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`INSERT INTO user_settings_new (id, user_id, theme, units, updated_at)
			SELECT id, user_id, theme, units, updated_at FROM user_settings`)
		if err != nil {
			return err
		}

		_, err = tx.Exec("DROP TABLE user_settings")
		if err != nil {
			return err
		}

		_, err = tx.Exec("ALTER TABLE user_settings_new RENAME TO user_settings")
		if err != nil {
			return err
		}
	}

	// Drop the old species_id index if it exists
	_, _ = tx.Exec("DROP INDEX IF EXISTS idx_catches_species_id")

	return tx.Commit()
}

