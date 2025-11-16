# Research & Technical Decisions: Flexible PIM System

**Feature**: Flexible Product Information Management (PIM) System  
**Branch**: `001-flexible-pim`  
**Created**: November 16, 2025

## Overview

This document resolves all technical uncertainties identified in the Technical Context section of the implementation plan. Each decision includes rationale, alternatives considered, and implementation notes.

## Research Tasks

### 1. Image Processing Library for Thumbnails

**Decision**: Use `github.com/disintegration/imaging` for image processing

**Rationale**:
- Pure Go implementation (no CGO dependencies, easier cross-platform deployment)
- Simple and intuitive API for common operations (resize, crop, thumbnail generation)
- Good performance for typical thumbnail generation use cases
- Active maintenance and community support
- Supports all common image formats (JPEG, PNG, GIF, WebP) via Go's standard `image` package
- MIT license (permissive)
- Well-documented with clear examples

**Alternatives Considered**:

1. **Standard Library `image` package**
   - Pros: No external dependencies, part of standard library
   - Cons: Lower-level API requires more boilerplate code, no built-in thumbnail generation helpers, requires manual aspect ratio calculations
   - Rejected because: Would require significant custom code for thumbnail generation, aspect ratio preservation, and format conversion

2. **`github.com/nfnt/resize`**
   - Pros: Simple API, lightweight
   - Cons: Limited to resizing only, no other image operations, less active maintenance
   - Rejected because: Lacks thumbnail helper functions and additional operations we may need (crop, format conversion)

3. **`github.com/h2non/bimg`** (libvips bindings)
   - Pros: Extremely fast, handles large images efficiently
   - Cons: Requires CGO and libvips system library, more complex deployment
   - Rejected because: Added deployment complexity (system dependencies) not justified for our scale (10K products, 50 media files max)

**Implementation Notes**:
```go
import "github.com/disintegration/imaging"

// Generate thumbnail from uploaded image
func generateThumbnail(src image.Image, width, height int) image.Image {
    return imaging.Thumbnail(src, width, height, imaging.Lanczos)
}

// Save thumbnail in appropriate format
func saveThumbnail(img image.Image, path string) error {
    return imaging.Save(img, path)
}
```

**Performance Characteristics**:
- Thumbnail generation for 10MB image: ~100-300ms on modern hardware
- Memory usage: ~2-3x the image size during processing
- Acceptable for our performance goals (< 10 seconds for upload including processing)

---

### 2. Video Processing for Preview Frame Extraction

**Decision**: Use `ffmpeg` command-line tool via `os/exec` package

**Rationale**:
- Industry-standard tool for video processing
- Reliable and battle-tested
- Supports all common video formats (MP4, WebM, MOV, AVI, etc.)
- Simple command-line interface for frame extraction
- No need for complex Go bindings or CGO
- Available in all Linux distributions via package managers
- Docker image can include ffmpeg in base image
- Low frequency operation (only on video upload)

**Alternatives Considered**:

1. **`github.com/giorgisio/goav`** (FFmpeg Go bindings)
   - Pros: Direct Go API to FFmpeg libraries
   - Cons: Requires CGO, complex build process, FFmpeg development headers required
   - Rejected because: Added build complexity not justified for simple frame extraction

2. **`github.com/3d0c/gmf`** (Go FFmpeg bindings)
   - Pros: Go bindings to FFmpeg libraries
   - Cons: Requires CGO, limited documentation, less maintained
   - Rejected because: Build complexity and maintenance concerns

3. **Pure Go video libraries** (e.g., `github.com/nareix/joy4`)
   - Pros: Pure Go, no system dependencies
   - Cons: Limited format support, less mature than FFmpeg, potential codec issues
   - Rejected because: Risk of format compatibility issues, FFmpeg is more reliable

**Implementation Notes**:
```go
import (
    "os/exec"
    "fmt"
)

// Extract preview frame from video at 1 second mark
func extractVideoPreview(videoPath, outputPath string) error {
    cmd := exec.Command(
        "ffmpeg",
        "-i", videoPath,              // Input video
        "-ss", "00:00:01",            // Seek to 1 second
        "-vframes", "1",              // Extract 1 frame
        "-q:v", "2",                  // Quality (2 = high quality)
        outputPath,                    // Output image path
    )
    return cmd.Run()
}

// Check if ffmpeg is available
func checkFFmpeg() error {
    cmd := exec.Command("ffmpeg", "-version")
    return cmd.Run()
}
```

**Deployment Requirements**:
- Docker image must include ffmpeg: `RUN apt-get update && apt-get install -y ffmpeg`
- Development environment: Developers must have ffmpeg installed locally
- Mac: `brew install ffmpeg`
- Ubuntu/Debian: `apt-get install ffmpeg`
- Startup check: Application should verify ffmpeg availability on startup

**Performance Characteristics**:
- Frame extraction from 100MB video: ~1-2 seconds
- Memory usage: Minimal (ffmpeg handles its own memory management)
- Acceptable for our performance goals (< 10 seconds for upload including processing)

---

### 3. File Storage Strategy

**Decision**: Start with local filesystem storage, design for easy migration to cloud storage later

**Rationale**:
- **Simplicity first**: Local filesystem is simpler for initial development and testing
- **No external dependencies**: No cloud provider accounts, API keys, or network dependencies for development
- **Easy abstraction**: Interface-based design allows swapping storage backends without changing business logic
- **Future-proof**: Can migrate to S3/Azure Blob/GCS when scale demands it
- **Cost-effective for initial scale**: 100K products × 10 images × 1MB = ~1TB (affordable on local storage)
- **Performance**: Local filesystem is faster than network storage for small to medium scale

**Alternatives Considered**:

1. **Amazon S3 from day one**
   - Pros: Scalable, durable, CDN integration, no local storage management
   - Cons: Requires AWS account, adds cloud costs immediately, network latency, development complexity
   - Rejected because: Premature optimization for initial scale, adds complexity without immediate benefit

2. **Azure Blob Storage from day one**
   - Pros: Scalable, durable, Microsoft ecosystem integration
   - Cons: Same concerns as S3, plus less common in Go ecosystem
   - Rejected because: Same reasons as S3

3. **Google Cloud Storage from day one**
   - Pros: Scalable, durable, Google ecosystem integration
   - Cons: Same concerns as S3
   - Rejected because: Same reasons as S3

4. **MinIO (S3-compatible self-hosted)**
   - Pros: S3-compatible API, can run locally or in cloud, familiar interface
   - Cons: Another service to deploy and manage, overkill for simple file storage
   - Rejected because: Added operational complexity for simple requirement

**Implementation Strategy**:

Define storage interface that can be implemented by multiple backends:

```go
// Storage interface for media files
type MediaStorage interface {
    // Store saves a file and returns its storage path/URL
    Store(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)
    
    // Retrieve returns a reader for the file content
    Retrieve(ctx context.Context, path string) (io.ReadCloser, error)
    
    // Delete removes a file from storage
    Delete(ctx context.Context, path string) error
    
    // URL returns a publicly accessible URL for the file
    URL(ctx context.Context, path string) (string, error)
}

// Local filesystem implementation
type LocalStorage struct {
    basePath string
    baseURL  string
}

func (s *LocalStorage) Store(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
    // Generate unique filename (UUID + original extension)
    storagePath := filepath.Join(s.basePath, filename)
    
    // Create file
    f, err := os.Create(storagePath)
    if err != nil {
        return "", err
    }
    defer f.Close()
    
    // Copy content
    if _, err := io.Copy(f, file); err != nil {
        return "", err
    }
    
    return filename, nil
}

func (s *LocalStorage) URL(ctx context.Context, path string) (string, error) {
    return s.baseURL + "/" + path, nil
}

// Future S3 implementation
type S3Storage struct {
    client *s3.Client
    bucket string
}

func (s *S3Storage) Store(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
    // S3 upload implementation
    // ...
}
```

**Directory Structure for Local Storage**:
```
storage/
├── media/
│   ├── images/
│   │   ├── originals/
│   │   │   └── {uuid}.{ext}
│   │   └── thumbnails/
│   │       └── {uuid}_thumb.{ext}
│   └── videos/
│       ├── originals/
│       │   └── {uuid}.{ext}
│       └── previews/
│           └── {uuid}_preview.jpg
```

**Migration Path to Cloud Storage**:
1. Implement S3Storage that satisfies MediaStorage interface
2. Update dependency injection to use S3Storage instead of LocalStorage
3. Migrate existing files using background job (copy local → S3, update database paths)
4. Switch storage backend via configuration
5. Remove local files after verification

**Configuration**:
```go
type StorageConfig struct {
    Type     string // "local" or "s3"
    BasePath string // Local filesystem base path
    BaseURL  string // Base URL for serving files
    
    // S3 specific (when Type = "s3")
    S3Bucket string
    S3Region string
    S3Prefix string
}
```

**Serving Files**:
- Local storage: Use http.FileServer to serve files directly
- S3 storage: Return pre-signed URLs or configure bucket for public read

**Backup Strategy** (for local storage):
- Regular filesystem backups
- Database stores only relative paths (not absolute)
- Easy to restore by copying storage directory

---

## Summary of Decisions

| Concern | Decision | Key Dependencies |
|---------|----------|------------------|
| Image Processing | `github.com/disintegration/imaging` | Pure Go, no system dependencies |
| Video Processing | `ffmpeg` via `os/exec` | System package: `ffmpeg` binary |
| File Storage | Local filesystem with abstraction for future migration | Standard library `os`, `io` |

## Implementation Checklist

- [ ] Add `github.com/disintegration/imaging` to go.mod
- [ ] Add ffmpeg availability check to application startup
- [ ] Document ffmpeg installation in development setup guide
- [ ] Include ffmpeg in Docker image (Dockerfile)
- [ ] Implement MediaStorage interface
- [ ] Implement LocalStorage with proper directory structure
- [ ] Create storage initialization in main.go (create directories)
- [ ] Add file serving HTTP handler for local storage
- [ ] Add configuration for storage paths
- [ ] Write integration tests for image thumbnail generation
- [ ] Write integration tests for video preview extraction
- [ ] Write integration tests for file storage operations

## External Dependencies

### Go Packages
- `github.com/disintegration/imaging` - Image processing (MIT license)

### System Dependencies
- `ffmpeg` - Video processing (GPL/LGPL, available in all Linux distributions)

### Development Setup
Developers need to install:
```bash
# Mac
brew install ffmpeg

# Ubuntu/Debian
sudo apt-get install ffmpeg

# Alpine Linux (for Docker)
apk add ffmpeg
```

## Performance Validation

All decisions meet the performance requirements from the specification:

- **Image thumbnail generation**: 100-300ms (well under 10 second upload goal)
- **Video preview extraction**: 1-2 seconds (well under 10 second upload goal)
- **File storage (local)**: Disk I/O limited, typically < 100ms for 10MB file
- **Total upload time**: < 5 seconds for 10MB media file (network + storage + processing)

## Security Considerations

1. **File Upload Validation**:
   - Validate file types by content (magic bytes), not just extension
   - Reject files that don't match declared content type
   - Limit file sizes (10MB images, 100MB videos)

2. **Storage Security**:
   - Use UUID-based filenames to prevent path traversal
   - Validate all paths before file operations
   - Set appropriate file permissions (0644 for files, 0755 for directories)
   - Sanitize user-provided filenames

3. **Video Processing**:
   - Validate video files before processing
   - Set timeout for ffmpeg operations (prevent DOS)
   - Run ffmpeg with limited privileges

## Next Steps

With all technical uncertainties resolved, we can proceed to:
1. **Phase 1**: Generate data model and API contracts
2. Update agent context with new dependencies
3. Begin implementation following TDD principles

