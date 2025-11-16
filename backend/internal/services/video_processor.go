package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// VideoProcessor handles video processing operations using ffmpeg
type VideoProcessor struct{}

// NewVideoProcessor creates a new VideoProcessor instance
func NewVideoProcessor() *VideoProcessor {
	return &VideoProcessor{}
}

// ExtractPreviewFrame extracts a single frame from the video at the specified timestamp
// Returns the path to the generated preview frame
func (p *VideoProcessor) ExtractPreviewFrame(videoPath, outputPath string, timestamp string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run ffmpeg to extract frame
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,           // Input video
		"-ss", timestamp,          // Seek to timestamp (e.g., "00:00:01")
		"-vframes", "1",           // Extract 1 frame
		"-q:v", "2",               // Quality (2 = high quality)
		"-y",                      // Overwrite output file if exists
		outputPath,                // Output image path
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetVideoDuration returns the duration of the video in seconds
func (p *VideoProcessor) GetVideoDuration(videoPath string) (int, error) {
	// Run ffprobe to get duration
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		// If ffprobe is not available, return 0 (duration unknown)
		return 0, nil
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return int(duration), nil
}

// GetVideoDimensions returns the width and height of the video
func (p *VideoProcessor) GetVideoDimensions(videoPath string) (width, height int, err error) {
	// Run ffprobe to get dimensions
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		// If ffprobe is not available, return 0,0 (dimensions unknown)
		return 0, 0, nil
	}

	dimensions := strings.TrimSpace(string(output))
	parts := strings.Split(dimensions, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output format: %s", dimensions)
	}

	width, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse width: %w", err)
	}

	height, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse height: %w", err)
	}

	return width, height, nil
}

// ValidateVideo checks if ffmpeg can process the video file
func (p *VideoProcessor) ValidateVideo(videoPath string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-v", "error",
		"-i", videoPath,
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("invalid video format: %w\nOutput: %s", err, string(output))
	}

	return nil
}

