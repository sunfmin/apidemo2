# Implementation Plan: Flexible Product Information Management (PIM) System

**Branch**: `001-flexible-pim` | **Date**: November 16, 2025 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-flexible-pim/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This implementation plan delivers a flexible Product Information Management (PIM) system that allows administrators to:
1. Define product templates with flexible attribute types (text, number, boolean, date, lists, maps, images, videos)
2. Create and manage products based on templates with all attribute types
3. Create and manage product variants with attribute overrides
4. Upload and manage media assets (images and videos) with thumbnails and previews

The system uses PostgreSQL with JSONB for flexible attribute storage, GORM for database access, Protocol Buffers for API contracts, and OpenTracing for observability. The architecture follows service layer pattern with dependency injection to enable code reuse across multiple applications (HTTP API, CLI, background workers).

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.21+ (latest stable recommended)  
**HTTP Framework**: Standard library `net/http` with `http.ServeMux` (MANDATORY per constitution)  
**Database**: PostgreSQL 15+ with JSONB support for flexible attributes  
**Database Access**: GORM (gorm.io/gorm) with gorm.io/driver/postgres (MANDATORY per constitution)  
**Distributed Tracing**: OpenTracing (github.com/opentracing/opentracing-go) (MANDATORY per constitution)  
**Protocol Buffers**: protoc compiler, protoc-gen-go for API contracts, protoc-gen-validate for validation  
**Testing**: Standard library `testing` with `httptest`, testcontainers-go for PostgreSQL (MANDATORY per constitution)  
**Test Comparison**: google/go-cmp with protocmp for protobuf assertions  
**Media Processing**: NEEDS CLARIFICATION - image processing library for thumbnails (consider: imaging, disintegration/imaging, or standard library image package)  
**Video Processing**: NEEDS CLARIFICATION - video processing for preview frame extraction (consider: ffmpeg wrapper like go-ffmpeg, or external ffmpeg command)  
**File Storage**: NEEDS CLARIFICATION - local filesystem or cloud storage (S3, Azure Blob, GCS)  
**Target Platform**: Linux server, containerized deployment (Docker)  
**Project Type**: Web application (backend API + admin frontend)  
**Performance Goals**: 
- Product search: < 1 second for catalogs up to 10,000 products
- API response time: p95 < 200ms, p99 < 500ms
- Media upload: Complete within 10 seconds for files up to 10MB
- Support 50 concurrent administrators without degradation  
**Constraints**: 
- Maximum file size: 10MB for images, 100MB for videos
- Maximum attributes per template: 100 attributes
- Maximum variants per product: 1,000 variants
- Maximum media files per attribute: 50 files  
**Scale/Scope**: 
- Target: 100,000 products maximum, 500,000 variants maximum
- Concurrent users: 50 administrators
- Daily usage: ~10,000 API requests per day

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Mandatory Requirements (MUST comply)

- ✅ **Integration Testing First**: All tests will use real PostgreSQL database via testcontainers-go, no mocking
- ✅ **Table-Driven Tests**: All tests will follow table-driven pattern with test case structs
- ✅ **Edge Case Coverage**: All endpoints will test input validation, boundary conditions, auth/authz, data state, database errors, HTTP specifics per constitution
- ✅ **Real Database Fixtures**: Test data will be inserted using GORM to real test database
- ✅ **ServeHTTP Testing**: API endpoints will be tested via ServeHTTP with httptest.ResponseRecorder
- ✅ **Protobuf Data Structures**: All API contracts will be defined in .proto files, no map[string]interface{}
- ✅ **Protobuf Comparison**: Tests will use google/go-cmp with protocmp for assertions
- ✅ **Distributed Tracing**: All endpoints will be instrumented with OpenTracing spans at appropriate granularity (service operations, NOT individual SQL queries)
- ✅ **Service Layer Architecture**: Business logic will be in service interfaces with dependency injection, handlers will be thin wrappers
- ✅ **Standard Library HTTP**: Will use net/http with http.ServeMux (no third-party frameworks)
- ✅ **GORM Database Access**: Will use GORM for all database operations
- ✅ **Database Truncation**: Tests will use defer truncation pattern for cleanup

### Architecture Compliance

- ✅ **Service Interfaces**: ProductTemplateService, ProductService, VariantService, MediaService with injected dependencies
- ✅ **HTTP Handlers**: Thin wrappers that delegate to services
- ✅ **Repository Pattern**: NOT USED - Direct GORM access from services per constitution (no repository abstraction)
- ✅ **Project Structure**: Web application structure (backend/ for API, frontend/ for admin UI)

### Technology Stack Compliance

- ✅ **Go 1.21+**: Latest stable Go version
- ✅ **PostgreSQL 15+**: With JSONB support for flexible attributes
- ✅ **GORM**: gorm.io/gorm with gorm.io/driver/postgres
- ✅ **OpenTracing**: github.com/opentracing/opentracing-go
- ✅ **Testcontainers**: testcontainers-go with PostgreSQL module
- ✅ **Protobuf**: protoc, protoc-gen-go, protoc-gen-validate

### Outstanding Clarifications (from Technical Context)

The following items are marked as NEEDS CLARIFICATION and will be resolved in Phase 0 research:

1. **Media Processing**: Image processing library for thumbnail generation
2. **Video Processing**: Video processing for preview frame extraction
3. **File Storage**: Local filesystem vs cloud storage decision

### Gate Status: ✅ PASS

All mandatory constitution requirements are met. Project structure, architecture, and technology stack comply with constitution principles. Outstanding clarifications will be resolved in Phase 0 research before proceeding to implementation.

---

### Post-Phase 1 Re-evaluation

**Date**: November 16, 2025  
**Status**: ✅ PASS

After completing Phase 0 (Research) and Phase 1 (Design & Contracts), all constitution requirements remain satisfied:

**Research Decisions**:
- ✅ Image processing: `github.com/disintegration/imaging` (pure Go, no CGO, constitution compliant)
- ✅ Video processing: `ffmpeg` via `os/exec` (system dependency, no CGO, constitution compliant)
- ✅ File storage: Local filesystem with abstraction for future migration (simple, testable, constitution compliant)

**Design Artifacts**:
- ✅ Data model uses GORM models per constitution
- ✅ API contracts defined in Protocol Buffers (.proto files)
- ✅ All entities have clear relationships and validation rules
- ✅ Service layer architecture with dependency injection
- ✅ No repository pattern abstraction (direct GORM access per constitution)

**Implementation Readiness**:
- ✅ All NEEDS CLARIFICATION items resolved in research.md
- ✅ Data model documented with GORM models and indexes
- ✅ API contracts complete in contracts/ directory
- ✅ Quick start guide created with TDD workflow
- ✅ Agent context updated with new dependencies

**No Constitution Violations**: All technical decisions align with constitution principles. Ready to proceed to task breakdown and implementation.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── api/
│   └── proto/
│       ├── template.proto        # Product template API contracts
│       ├── product.proto          # Product API contracts
│       ├── variant.proto          # Variant API contracts
│       ├── media.proto            # Media API contracts
│       └── common.proto           # Shared types (AttributeValue, etc.)
├── internal/
│   ├── models/
│   │   ├── template.go            # ProductTemplate GORM model
│   │   ├── product.go             # Product GORM model
│   │   ├── variant.go             # ProductVariant GORM model
│   │   └── media.go               # MediaFile GORM model
│   ├── services/
│   │   ├── template_service.go    # Template business logic
│   │   ├── product_service.go     # Product business logic
│   │   ├── variant_service.go     # Variant business logic
│   │   └── media_service.go       # Media processing & storage
│   ├── handlers/
│   │   ├── template_handler.go    # HTTP handlers for templates
│   │   ├── product_handler.go     # HTTP handlers for products
│   │   ├── variant_handler.go     # HTTP handlers for variants
│   │   └── media_handler.go       # HTTP handlers for media
│   └── middleware/
│       ├── tracing.go             # OpenTracing middleware
│       ├── logging.go             # Request logging
│       └── cors.go                # CORS for frontend
├── cmd/
│   └── server/
│       └── main.go                # HTTP server entry point
└── tests/
    ├── integration/
    │   ├── template_test.go       # Template endpoint tests
    │   ├── product_test.go        # Product endpoint tests
    │   ├── variant_test.go        # Variant endpoint tests
    │   └── media_test.go          # Media endpoint tests
    └── testutil/
        ├── db.go                  # testcontainers setup
        ├── fixtures.go            # Test data helpers
        └── assertions.go          # Custom test assertions

frontend/
├── src/
│   ├── components/
│   │   ├── templates/
│   │   │   ├── TemplateList.tsx
│   │   │   ├── TemplateForm.tsx
│   │   │   └── AttributeBuilder.tsx
│   │   ├── products/
│   │   │   ├── ProductList.tsx
│   │   │   ├── ProductForm.tsx
│   │   │   └── AttributeFields.tsx
│   │   ├── variants/
│   │   │   ├── VariantList.tsx
│   │   │   └── VariantForm.tsx
│   │   └── media/
│   │       ├── MediaUpload.tsx
│   │       ├── MediaGallery.tsx
│   │       └── MediaPreview.tsx
│   ├── pages/
│   │   ├── templates/
│   │   │   ├── index.tsx          # Template list page
│   │   │   ├── create.tsx         # Template creation page
│   │   │   └── [id]/edit.tsx      # Template edit page
│   │   ├── products/
│   │   │   ├── index.tsx          # Product list page
│   │   │   ├── create.tsx         # Product creation page
│   │   │   └── [id]/edit.tsx      # Product edit page
│   │   └── variants/
│   │       └── [productId]/[id]/edit.tsx
│   ├── services/
│   │   ├── api.ts                 # API client base
│   │   ├── templateService.ts     # Template API calls
│   │   ├── productService.ts      # Product API calls
│   │   ├── variantService.ts      # Variant API calls
│   │   └── mediaService.ts        # Media upload/management
│   └── types/
│       └── generated/             # Generated TypeScript types from protobuf
└── tests/
    └── e2e/
        ├── templates.spec.ts
        ├── products.spec.ts
        └── media.spec.ts

storage/                            # Local file storage (if not using cloud)
└── media/
    ├── images/
    │   ├── originals/
    │   └── thumbnails/
    └── videos/
        ├── originals/
        └── previews/
```

**Structure Decision**: 

This implementation uses a web application structure with backend API and admin frontend:

- **backend/**: Go API server following constitution principles
  - `api/proto/`: Protocol Buffer definitions (source of truth for API contracts)
  - `internal/`: Application code (not importable by other packages)
    - `models/`: GORM database models
    - `services/`: Business logic interfaces with dependency injection
    - `handlers/`: Thin HTTP handlers that delegate to services
    - `middleware/`: HTTP middleware (tracing, logging, CORS)
  - `cmd/server/`: HTTP server entry point
  - `tests/`: Integration tests using testcontainers-go

- **frontend/**: React/TypeScript admin UI
  - `src/components/`: Reusable UI components organized by feature
  - `src/pages/`: Page components (routing)
  - `src/services/`: API client wrappers
  - `src/types/generated/`: TypeScript types generated from protobuf

- **storage/**: Local file storage directory (if not using cloud storage)
  - Organized by media type with originals and processed versions

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations in this implementation. All architecture decisions comply with constitution principles:
- Using standard library net/http (no third-party frameworks)
- Using GORM for database access (no repository pattern abstraction)
- Using service layer with dependency injection (per constitution)
- Using integration tests with real database (no mocking)
- Using Protocol Buffers for API contracts (no map[string]interface{})

This section is not applicable as there are no violations to track.
