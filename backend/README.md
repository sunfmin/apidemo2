# Flexible PIM System - Backend API

A flexible Product Information Management (PIM) system built with Go, PostgreSQL, and Protocol Buffers.

## Features

- **Product Template Management**: Define custom product schemas with 8 flexible attribute types
- **Product Catalog**: Create and manage products with template-based validation
- **Variant Management**: Handle product variations with attribute inheritance and overrides
- **Media Management**: Upload and manage images and videos with automatic processing
- **Distributed Tracing**: OpenTracing instrumentation throughout
- **Type-Safe APIs**: Protocol Buffer contracts for all endpoints

## Architecture

- **Language**: Go 1.21+
- **Database**: PostgreSQL 15+ with JSONB support
- **ORM**: GORM
- **API Format**: Protocol Buffers
- **HTTP Framework**: Standard library `net/http` with `http.ServeMux`
- **Testing**: Integration tests with testcontainers-go
- **Image Processing**: github.com/disintegration/imaging
- **Video Processing**: ffmpeg

## Prerequisites

### Required
- Go 1.21 or later
- PostgreSQL 15+ (or use Docker)
- Protocol Buffers compiler (`protoc`)
- ffmpeg (for video processing)

### Installation

**macOS:**
```bash
brew install go postgresql ffmpeg protobuf
```

**Ubuntu/Debian:**
```bash
sudo apt-get install golang postgresql ffmpeg protobuf-compiler
```

## Quick Start

### 1. Clone and Setup

```bash
cd backend
go mod download
```

### 2. Setup Database

**Option A: Using Docker**
```bash
docker run -d \
  --name pim-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=pim_dev \
  -p 5432:5432 \
  postgres:15-alpine
```

**Option B: Local PostgreSQL**
```bash
createdb pim_dev
```

### 3. Configure Environment

Create a `.env` file or set environment variables:

```bash
export DATABASE_URL="host=localhost user=postgres password=postgres dbname=pim_dev port=5432 sslmode=disable"
export STORAGE_PATH="./storage/media"
export BASE_URL="http://localhost:8080/media"
```

### 4. Run Database Migrations

Migrations run automatically on server startup using GORM AutoMigrate.

### 5. Start the Server

```bash
go run ./cmd/server/main.go
```

The server will start on `http://localhost:8080`

### 6. Verify Installation

```bash
curl http://localhost:8080/health
# Should return: OK
```

## API Endpoints

### Templates
- `POST /api/v1/templates` - Create template
- `GET /api/v1/templates` - List templates
- `GET /api/v1/templates/{id}` - Get template
- `PUT /api/v1/templates/{id}` - Update template
- `DELETE /api/v1/templates/{id}` - Delete template

### Products
- `POST /api/v1/products` - Create product
- `GET /api/v1/products` - List products
- `GET /api/v1/products/{id}` - Get product
- `PUT /api/v1/products/{id}` - Update product
- `DELETE /api/v1/products/{id}` - Delete product
- `POST /api/v1/products/bulk/status` - Bulk update status

### Variants
- `POST /api/v1/products/{product_id}/variants` - Create variant
- `POST /api/v1/products/{product_id}/variants/bulk` - Bulk create variants
- `GET /api/v1/products/{product_id}/variants` - List variants
- `GET /api/v1/variants/{id}` - Get variant
- `PUT /api/v1/variants/{id}` - Update variant
- `DELETE /api/v1/variants/{id}` - Delete variant

### Media
- `POST /api/v1/media/upload` - Upload media file (multipart/form-data)
- `POST /api/v1/media/upload/bulk` - Upload multiple files
- `GET /api/v1/media` - List media files (query params: entity_type, entity_id, attribute_name, file_type)
- `GET /api/v1/media/{id}` - Get media metadata
- `PUT /api/v1/media/{id}` - Update media metadata
- `DELETE /api/v1/media/{id}` - Delete media file
- `POST /api/v1/media/reorder` - Reorder media files
- `GET /api/v1/media/file/{id}` - Serve original file
- `GET /api/v1/media/thumbnail/{id}` - Serve thumbnail/preview

## Development

### Running Tests

**All integration tests:**
```bash
go test -v ./tests/integration/... -count=1
```

**Specific test suite:**
```bash
go test -v ./tests/integration/template_test.go -count=1
go test -v ./tests/integration/product_test.go -count=1
go test -v ./tests/integration/variant_test.go -count=1
go test -v ./tests/integration/media_test.go -count=1
```

**Test infrastructure:**
```bash
go test -v ./tests/testutil/... -count=1
```

### Regenerating Protocol Buffers

```bash
cd api
./generate.sh
```

### Project Structure

```
backend/
├── api/
│   ├── proto/           # Protocol Buffer definitions
│   └── gen/             # Generated Go code
├── cmd/
│   └── server/          # Server entry point
├── internal/
│   ├── models/          # GORM database models
│   ├── services/        # Business logic
│   ├── handlers/        # HTTP handlers
│   └── middleware/      # HTTP middleware
└── tests/
    ├── integration/     # Integration tests
    └── testutil/        # Test utilities
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `host=localhost user=postgres password=postgres dbname=pim_dev port=5432 sslmode=disable` | PostgreSQL connection string |
| `STORAGE_PATH` | `./storage/media` | Base path for media file storage |
| `BASE_URL` | `http://localhost:8080/media` | Base URL for serving media files |
| `PORT` | `8080` | Server port |

### File Storage

Media files are stored locally in the following structure:

```
storage/media/
├── images/
│   ├── originals/       # Original uploaded images
│   └── thumbnails/      # Generated thumbnails (300x300)
└── videos/
    ├── originals/       # Original uploaded videos
    └── previews/        # Extracted preview frames
```

### Limits

- Maximum image size: 10 MB
- Maximum video size: 100 MB
- Thumbnail size: 300x300 pixels
- Supported image formats: JPEG, PNG, GIF, WebP
- Supported video formats: MP4, WebM, MOV
- Maximum attributes per template: 100
- Maximum variants per product: 1,000
- Maximum media files per attribute: 50

## Production Deployment

### Docker

```bash
# Build image
docker build -t pim-backend .

# Run container
docker run -d \
  --name pim-api \
  -p 8080:8080 \
  -e DATABASE_URL="host=postgres user=postgres password=secret dbname=pim_prod" \
  -v /path/to/storage:/app/storage \
  pim-backend
```

### Docker Compose

```bash
docker-compose up -d
```

This starts both the API server and PostgreSQL database.

## Troubleshooting

### Database Connection Issues

```bash
# Test database connection
psql -h localhost -U postgres -d pim_dev

# Check if server can connect
go run ./cmd/server/main.go
# Look for: "✅ Database connected"
```

### FFmpeg Not Found

```bash
# Verify ffmpeg is installed
ffmpeg -version

# macOS: Install via Homebrew
brew install ffmpeg

# Ubuntu/Debian
sudo apt-get install ffmpeg
```

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use a different port
PORT=8081 go run ./cmd/server/main.go
```

### Storage Permission Issues

```bash
# Ensure storage directory is writable
chmod -R 755 ./storage/media
```

## License

Copyright © 2025. All rights reserved.

## Support

For issues and questions, please open an issue on the repository.

