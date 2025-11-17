package services

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// ImageProcessor handles image processing operations
type ImageProcessor struct {
	thumbnailWidth  int
	thumbnailHeight int
}

// NewImageProcessor creates a new ImageProcessor instance
func NewImageProcessor(thumbnailWidth, thumbnailHeight int) *ImageProcessor {
	return &ImageProcessor{
		thumbnailWidth:  thumbnailWidth,
		thumbnailHeight: thumbnailHeight,
	}
}

// GenerateThumbnail creates a thumbnail from the source image file
// Returns the path to the generated thumbnail
func (p *ImageProcessor) GenerateThumbnail(srcPath, dstPath string) error {
	// Open source image
	src, err := imaging.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Generate thumbnail using Lanczos resampling (high quality)
	thumbnail := imaging.Thumbnail(src, p.thumbnailWidth, p.thumbnailHeight, imaging.Lanczos)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Save thumbnail
	if err := imaging.Save(thumbnail, dstPath); err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return nil
}

// GetImageDimensions returns the width and height of an image
func (p *ImageProcessor) GetImageDimensions(imagePath string) (width, height int, err error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}

	return img.Width, img.Height, nil
}

// ValidateImage checks if a file is a valid image
func (p *ImageProcessor) ValidateImage(imagePath string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	_, _, err = image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	return nil
}
