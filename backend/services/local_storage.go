package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// LocalStorage implements MediaStorage for local filesystem
type LocalStorage struct {
	basePath string // Base directory for file storage (e.g., ./storage/media)
	baseURL  string // Base URL for serving files (e.g., http://localhost:8080/media)
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Store saves a file to the local filesystem with a unique filename
func (s *LocalStorage) Store(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// Generate unique filename (UUID + original extension)
	ext := filepath.Ext(filename)
	uniqueFilename := uuid.New().String() + ext

	// Determine subdirectory based on content type
	var subdir string
	switch {
	case isImageType(contentType):
		subdir = "images/originals"
	case isVideoType(contentType):
		subdir = "videos/originals"
	default:
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}

	// Create full storage path
	fullPath := filepath.Join(s.basePath, subdir, uniqueFilename)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, file); err != nil {
		// Clean up partially written file
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return relative path from basePath
	relativePath := filepath.Join(subdir, uniqueFilename)
	return relativePath, nil
}

// Retrieve returns a reader for the file content
func (s *LocalStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from storage
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// URL returns a publicly accessible URL for the file
func (s *LocalStorage) URL(ctx context.Context, path string) (string, error) {
	return s.baseURL + "/" + path, nil
}

// Helper functions

func isImageType(contentType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

func isVideoType(contentType string) bool {
	videoTypes := []string{
		"video/mp4",
		"video/webm",
		"video/quicktime", // .mov files
	}
	for _, t := range videoTypes {
		if contentType == t {
			return true
		}
	}
	return false
}
