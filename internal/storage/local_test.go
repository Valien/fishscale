package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	content := "hello, fish photo data"
	relPath, err := store.Save("photo.jpg", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !strings.HasSuffix(relPath, ".jpg") {
		t.Errorf("expected .jpg extension, got path %q", relPath)
	}

	rc, err := store.Get(relPath)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(got) != content {
		t.Errorf("content mismatch: got %q, want %q", string(got), content)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	relPath, err := store.Save("delete-me.png", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.Delete(relPath); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(relPath)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected IsNotExist error, got %v", err)
	}
}

func TestDateOrganizedPath(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	relPath, err := store.Save("test.jpg", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	now := time.Now()
	expectedPrefix := filepath.Join(fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	if !strings.HasPrefix(relPath, expectedPrefix) {
		t.Errorf("expected path to start with %q, got %q", expectedPrefix, relPath)
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) != 3 {
		t.Errorf("expected 3 path components (YYYY/MM/file), got %d: %v", len(parts), parts)
	}
}
