# Quick Start Guide: Flexible PIM System

**Feature**: Flexible Product Information Management (PIM) System  
**Branch**: `001-flexible-pim`  
**Created**: November 16, 2025

## Overview

This quick start guide helps developers set up and begin implementing the Flexible PIM system. Follow these steps to get the development environment ready and understand the implementation workflow.

## Prerequisites

### Required Software

1. **Go 1.21+**
   ```bash
   go version  # Should show 1.21 or higher
   ```

2. **Docker** (for testcontainers)
   ```bash
   docker --version
   docker ps  # Verify Docker daemon is running
   ```

3. **PostgreSQL Client Tools** (optional, for manual database inspection)
   ```bash
   psql --version
   ```

4. **FFmpeg** (for video processing)
   ```bash
   # Mac
   brew install ffmpeg
   
   # Ubuntu/Debian
   sudo apt-get install ffmpeg
   
   # Verify installation
   ffmpeg -version
   ```

5. **Protocol Buffers Compiler**
   ```bash
   # Mac
   brew install protobuf
   
   # Ubuntu/Debian
   sudo apt-get install protobuf-compiler
   
   # Verify installation
   protoc --version  # Should show libprotoc 3.x or higher
   ```

6. **Go Protobuf Plugins**
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

### Recommended IDE Setup

- **VS Code** with Go extension
- **GoLand** by JetBrains
- Or any editor with Go language server (gopls) support

## Project Setup

### 1. Clone and Navigate to Project

```bash
cd /Users/sunfmin/Developments/apidemo2
git checkout 001-flexible-pim
```

### 2. Initialize Go Module (if not already done)

```bash
cd backend
go mod init github.com/sunfmin/apidemo2/backend
```

### 3. Install Go Dependencies

```bash
# GORM and PostgreSQL driver
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get gorm.io/datatypes

# OpenTracing
go get github.com/opentracing/opentracing-go

# Protocol Buffers
go get google.golang.org/protobuf

# Testing
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/google/go-cmp
go get google.golang.org/protobuf/testing/protocmp

# Image processing
go get github.com/disintegration/imaging

# UUID generation
go get github.com/google/uuid
```

### 4. Create Directory Structure

```bash
# From project root
mkdir -p backend/api/proto
mkdir -p backend/internal/{models,services,handlers,middleware}
mkdir -p backend/cmd/server
mkdir -p backend/tests/{integration,testutil}
mkdir -p storage/media/{images,videos}/{originals,thumbnails,previews}

# Copy proto files from spec
cp specs/001-flexible-pim/contracts/*.proto backend/api/proto/
```

### 5. Generate Protocol Buffer Code

Create a generation script:

```bash
# backend/api/generate.sh
#!/bin/bash
set -e

PROTO_DIR="./proto"
OUT_DIR="./gen/pim/v1"

mkdir -p "$OUT_DIR"

protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$OUT_DIR" \
  --go_opt=paths=source_relative \
  "$PROTO_DIR"/*.proto

echo "✅ Proto generation complete"
```

Make it executable and run:

```bash
chmod +x backend/api/generate.sh
cd backend/api
./generate.sh
```

### 6. Create Initial Configuration File

```bash
# backend/config.yaml
server:
  port: 8080
  host: localhost

database:
  # Will be overridden by testcontainers in tests
  # For local development, use local PostgreSQL instance
  host: localhost
  port: 5432
  database: pim_dev
  username: postgres
  password: postgres
  ssl_mode: disable

storage:
  type: local  # "local" or "s3"
  base_path: ./storage/media
  base_url: http://localhost:8080/media

media:
  max_image_size: 10485760  # 10MB in bytes
  max_video_size: 104857600  # 100MB in bytes
  thumbnail_width: 300
  thumbnail_height: 300

tracing:
  enabled: false  # Set to true with proper backend in production
  service_name: pim-api
```

## Implementation Workflow (TDD)

Follow Test-Driven Development per constitution:

### Step 1: Start with Tests

**Example**: Implementing Template Creation

1. **Write test first** (backend/tests/integration/template_test.go):

```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/google/go-cmp/cmp"
    "google.golang.org/protobuf/testing/protocmp"
    pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
)

func TestTemplateHandler_Create(t *testing.T) {
    // Setup test database using testcontainers
    db, cleanup := setupTestDB(t)
    defer cleanup()
    defer truncateTables(db, "product_templates")
    
    // Create service and handler
    templateService := services.NewTemplateService(db)
    handler := handlers.NewTemplateHandler(templateService)
    
    // Table-driven test cases
    testCases := []struct {
        name           string
        request        *pb.CreateTemplateRequest
        expectedStatus int
        expectError    bool
    }{
        {
            name: "successful_creation",
            request: &pb.CreateTemplateRequest{
                Name: "Clothing",
                Attributes: []*pb.AttributeDefinition{
                    {Name: "Size", Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, Required: true, Options: []string{"S", "M", "L"}},
                    {Name: "Material", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
                },
            },
            expectedStatus: http.StatusCreated,
            expectError:    false,
        },
        {
            name: "missing_name",
            request: &pb.CreateTemplateRequest{
                Attributes: []*pb.AttributeDefinition{
                    {Name: "Size", Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, Required: true},
                },
            },
            expectedStatus: http.StatusBadRequest,
            expectError:    true,
        },
        // Add more test cases: duplicate name, empty attributes, invalid attribute type, etc.
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create HTTP request
            body, _ := json.Marshal(tc.request)
            req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            rec := httptest.NewRecorder()
            
            // Execute handler
            handler.Create(rec, req)
            
            // Assert response status
            if rec.Code != tc.expectedStatus {
                t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
            }
            
            if !tc.expectError {
                // Assert response body
                var response pb.CreateTemplateResponse
                json.NewDecoder(rec.Body).Decode(&response)
                
                if response.Template.Name != tc.request.Name {
                    t.Errorf("Expected name %s, got %s", tc.request.Name, response.Template.Name)
                }
                
                // Use protocmp for protobuf comparison
                if diff := cmp.Diff(tc.request.Attributes, response.Template.Attributes, protocmp.Transform()); diff != "" {
                    t.Errorf("Attributes mismatch (-want +got):\n%s", diff)
                }
            }
        })
    }
}
```

2. **Verify test fails** (red):
```bash
cd backend
go test ./tests/integration -v -run TestTemplateHandler_Create
# Should fail because implementation doesn't exist yet
```

3. **Review test with team/lead** - Get approval on test design before implementing

### Step 2: Implement to Pass Tests

4. **Create GORM model** (backend/internal/models/template.go):

```go
package models

import (
    "time"
    "gorm.io/datatypes"
)

type ProductTemplate struct {
    ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name       string         `gorm:"type:varchar(255);not null;uniqueIndex"`
    Attributes datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
    CreatedAt  time.Time      `gorm:"not null"`
    UpdatedAt  time.Time      `gorm:"not null"`
}

func (ProductTemplate) TableName() string {
    return "product_templates"
}
```

5. **Create service interface** (backend/internal/services/template_service.go):

```go
package services

import (
    "context"
    pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
)

type TemplateService interface {
    Create(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.ProductTemplate, error)
    Get(ctx context.Context, id string) (*pb.ProductTemplate, error)
    List(ctx context.Context, req *pb.ListTemplatesRequest) ([]*pb.ProductTemplate, *pb.PaginationResponse, error)
    Update(ctx context.Context, req *pb.UpdateTemplateRequest) (*pb.ProductTemplate, error)
    Delete(ctx context.Context, id string) error
}

type templateService struct {
    db *gorm.DB
}

func NewTemplateService(db *gorm.DB) TemplateService {
    return &templateService{db: db}
}

func (s *templateService) Create(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.ProductTemplate, error) {
    // Validate request
    if req.Name == "" {
        return nil, errors.New("template name is required")
    }
    
    // Convert attributes to JSON
    attrs, _ := json.Marshal(req.Attributes)
    
    // Create model
    template := &models.ProductTemplate{
        Name:       req.Name,
        Attributes: attrs,
    }
    
    // Save to database
    if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
        return nil, err
    }
    
    // Convert to protobuf response
    return toProtoTemplate(template), nil
}

func toProtoTemplate(m *models.ProductTemplate) *pb.ProductTemplate {
    var attrs []*pb.AttributeDefinition
    json.Unmarshal(m.Attributes, &attrs)
    
    return &pb.ProductTemplate{
        Id:         m.ID,
        Name:       m.Name,
        Attributes: attrs,
        CreatedAt:  timestamppb.New(m.CreatedAt),
        UpdatedAt:  timestamppb.New(m.UpdatedAt),
    }
}
```

6. **Create HTTP handler** (backend/internal/handlers/template_handler.go):

```go
package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/opentracing/opentracing-go"
    pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
    "github.com/sunfmin/apidemo2/backend/internal/services"
)

type TemplateHandler struct {
    service services.TemplateService
}

func NewTemplateHandler(service services.TemplateService) *TemplateHandler {
    return &TemplateHandler{service: service}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Create OpenTracing span
    span, ctx := opentracing.StartSpanFromContext(r.Context(), "POST /api/v1/templates")
    defer span.Finish()
    
    span.SetTag("http.method", r.Method)
    span.SetTag("http.url", r.URL.String())
    
    // Parse request
    var req pb.CreateTemplateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        span.SetTag("error", true)
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Call service
    template, err := h.service.Create(ctx, &req)
    if err != nil {
        span.SetTag("error", true)
        span.SetTag("error.message", err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return response
    span.SetTag("http.status_code", http.StatusCreated)
    w.Header().Set("Content-Type", "application/json")
    w.WriteStatus(http.StatusCreated)
    json.NewEncoder(w).Encode(&pb.CreateTemplateResponse{Template: template})
}
```

7. **Run tests again** (green):
```bash
go test ./tests/integration -v -run TestTemplateHandler_Create
# Should pass now
```

8. **Refactor** if needed while keeping tests green

### Step 3: Repeat for Each Endpoint

Continue this TDD cycle for:
- Template List, Get, Update, Delete
- Product Create, List, Get, Update, Delete
- Variant Create, List, Get, Update, Delete
- Media Upload, List, Get, Update, Delete, Reorder

## Running Tests

### Run All Tests

```bash
cd backend
go test ./tests/integration/... -v
```

### Run Specific Test

```bash
go test ./tests/integration -v -run TestTemplateHandler_Create
```

### Run with Coverage

```bash
go test ./tests/integration/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Running the Server

### Development Mode

```bash
cd backend/cmd/server
go run main.go
```

### With Air (hot reload)

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
cd backend
air
```

## Database Management

### Local Development Database

```bash
# Start PostgreSQL with Docker
docker run --name pim-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=pim_dev \
  -p 5432:5432 \
  -d postgres:15-alpine

# Run migrations (via application startup)
go run cmd/server/main.go migrate
```

### Test Database

Tests automatically manage database via testcontainers:
- Each test run creates fresh PostgreSQL container
- Container is cleaned up after tests complete
- No manual database setup needed for tests

## Common Commands

### Generate Proto Files

```bash
cd backend/api
./generate.sh
```

### Format Code

```bash
cd backend
go fmt ./...
```

### Lint Code

```bash
cd backend
golangci-lint run
```

### Run All Tests

```bash
cd backend
go test ./... -v
```

## Troubleshooting

### Docker not running

```bash
# Mac
open -a Docker

# Check status
docker ps
```

### FFmpeg not found

```bash
# Mac
brew install ffmpeg

# Ubuntu/Debian
sudo apt-get install ffmpeg

# Verify
ffmpeg -version
```

### Protoc not found

```bash
# Mac
brew install protobuf

# Ubuntu/Debian
sudo apt-get install protobuf-compiler

# Verify
protoc --version
```

### Test database connection issues

- Ensure Docker daemon is running
- Check no port conflicts (5432)
- Testcontainers will handle database lifecycle

## Next Steps

1. **Read the Constitution**: `/Users/sunfmin/Developments/apidemo2/.specify/memory/constitution.md`
2. **Review Data Model**: `specs/001-flexible-pim/data-model.md`
3. **Review API Contracts**: `specs/001-flexible-pim/contracts/*.proto`
4. **Start with Template Tests**: Follow TDD workflow above
5. **Implement Products**: After templates are working
6. **Implement Variants**: After products are working
7. **Implement Media**: After core entities are working

## Resources

- **Constitution**: `.specify/memory/constitution.md`
- **Feature Spec**: `specs/001-flexible-pim/spec.md`
- **Implementation Plan**: `specs/001-flexible-pim/plan.md`
- **Research Decisions**: `specs/001-flexible-pim/research.md`
- **Data Model**: `specs/001-flexible-pim/data-model.md`
- **API Contracts**: `specs/001-flexible-pim/contracts/`

## Support

For questions or clarifications:
1. Review constitution principles
2. Check research decisions
3. Review data model and contracts
4. Consult with tech lead

---

**Remember**: Always write tests first, get them reviewed, then implement. Follow the constitution principles strictly.

