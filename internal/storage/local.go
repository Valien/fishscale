package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LocalStore implements Store using the local filesystem.
type LocalStore struct {
	BaseDir string
}

// NewLocalStore creates a new LocalStore rooted at baseDir.
func NewLocalStore(baseDir string) *LocalStore {
	return &LocalStore{BaseDir: baseDir}
}

// Save stores the contents from r into a date-organized directory structure
// (YYYY/MM) with a random filename, preserving the original file extension.
// It returns the relative path from BaseDir.
func (s *LocalStore) Save(filename string, r io.Reader) (string, error) {
	now := time.Now()
	dir := filepath.Join(s.BaseDir, fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(filename)
	randName, err := randomHex(16)
	if err != nil {
		return "", err
	}

	newName := randName + ext
	fullPath := filepath.Join(dir, newName)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(fullPath)
		return "", err
	}

	relPath, err := filepath.Rel(s.BaseDir, fullPath)
	if err != nil {
		return "", err
	}

	return relPath, nil
}

// Get returns a ReadCloser for the file at the given relative path.
func (s *LocalStore) Get(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.BaseDir, path)
	return os.Open(fullPath)
}

// Delete removes the file at the given relative path.
func (s *LocalStore) Delete(path string) error {
	fullPath := filepath.Join(s.BaseDir, path)
	return os.Remove(fullPath)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
