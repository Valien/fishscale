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

	tables := []string{"users", "trips", "species", "catches", "photos", "user_settings"}
	for _, table := range tables {
		var name string
		err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestSeedSpecies(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM species"); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count == 0 {
		t.Error("expected species count > 0, got 0")
	}
	t.Logf("seeded %d species", count)
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
