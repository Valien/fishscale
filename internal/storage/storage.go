package storage

import "io"

// Store defines the interface for photo storage.
type Store interface {
	// Save stores the contents from r and returns the relative path to the file.
	Save(filename string, r io.Reader) (string, error)

	// Get returns a ReadCloser for the file at the given relative path.
	Get(path string) (io.ReadCloser, error)

	// Delete removes the file at the given relative path.
	Delete(path string) error
}
