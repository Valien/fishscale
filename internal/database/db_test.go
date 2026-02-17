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
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	tables := []string{"users", "trips", "catches", "photos", "user_settings"}
	for _, table := range tables {
		var name string
		err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestConnectionPoolLimits(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 2 {
		t.Errorf("expected MaxOpenConnections=2, got %d", stats.MaxOpenConnections)
	}
}

func TestIndexesExist(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	indexes := []string{
		"idx_catches_user_id",
		"idx_catches_caught_at",
		"idx_catches_species_name",
		"idx_catches_trip_id",
		"idx_photos_catch_id",
		"idx_trips_user_id",
	}
	for _, idx := range indexes {
		var name string
		err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx)
		if err != nil {
			t.Errorf("index %q not found: %v", idx, err)
		}
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	dbPath := filepath.Join(nested, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("expected nested directory to be created")
	}
}
