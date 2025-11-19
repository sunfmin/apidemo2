package services

import (
	"context"
	"io"
)

// MediaStorage defines the interface for storing and retrieving media files
type MediaStorage interface {
	// Store saves a file and returns its storage path
	Store(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)

	// Retrieve returns a reader for the file content
	Retrieve(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, path string) error

	// URL returns a publicly accessible URL for the file
	URL(ctx context.Context, path string) (string, error)
}
