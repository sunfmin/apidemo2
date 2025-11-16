# Tasks: Flexible Product Information Management (PIM) System

**Feature Branch**: `001-flexible-pim`  
**Input**: Design documents from `/Users/sunfmin/Developments/apidemo2/specs/001-flexible-pim/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Integration tests are MANDATORY per constitution. All tests use real PostgreSQL database via testcontainers-go (no mocking), follow table-driven patterns, use GORM for fixtures, use protobuf structs (NOT maps), verify OpenTracing instrumentation, and cover comprehensive edge cases. Tests are conducted at HTTP layer only (httptest), which exercises the full stack: HTTP → Service → Repository → Database.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: backend/ directory with internal/ for application code
- **Test organization**: Integration tests in backend/tests/integration/*_test.go
- **Test database**: Use testcontainers-go for automatic PostgreSQL container management
- **Architecture**: Service layer with dependency injection (HTTP → Service → Database via GORM)
- **Database access**: Use GORM for all database operations (no repository pattern per constitution)
- **HTTP framework**: Use standard net/http with http.ServeMux (no external routers)
- **Tracing**: Use OpenTracing for all endpoint instrumentation
- **Media processing**: Use github.com/disintegration/imaging for images, ffmpeg for videos

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and dependency installation

- [x] T001 Initialize Go module with `go mod init github.com/sunfmin/apidemo2/backend`
- [x] T002 [P] Install GORM dependencies: `go get gorm.io/gorm gorm.io/driver/postgres gorm.io/datatypes`
- [x] T003 [P] Install OpenTracing: `go get github.com/opentracing/opentracing-go`
- [x] T004 [P] Install Protocol Buffers: `go get google.golang.org/protobuf google.golang.org/protobuf/testing/protocmp`
- [x] T005 [P] Install testcontainers-go: `go get github.com/testcontainers/testcontainers-go github.com/testcontainers/testcontainers-go/modules/postgres`
- [x] T006 [P] Install google/go-cmp for test assertions: `go get github.com/google/go-cmp`
- [x] T007 [P] Install image processing library: `go get github.com/disintegration/imaging`
- [x] T008 [P] Install UUID library: `go get github.com/google/uuid`
- [x] T009 Copy protobuf files from specs/001-flexible-pim/contracts/ to backend/api/proto/
- [x] T010 Create protobuf generation script at backend/api/generate.sh
- [x] T011 Generate Go code from protobuf files: `cd backend/api && ./generate.sh`
- [x] T012 [P] Create directory structure: backend/internal/{models,services,handlers,middleware}, backend/cmd/server, backend/tests/{integration,testutil}
- [x] T013 [P] Create storage directories: storage/media/{images,videos}/{originals,thumbnails,previews}
- [x] T014 [P] Create configuration file at backend/config.yaml with database, storage, and tracing settings
- [x] T015 Verify ffmpeg is installed: Create startup check in backend/cmd/server/main.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T016 Create testcontainers PostgreSQL setup helper in backend/tests/testutil/db.go (container lifecycle, GORM connection, AutoMigrate)
- [x] T017 [P] Create table truncation helper in backend/tests/testutil/db.go (truncateTables function with CASCADE support)
- [x] T018 [P] Create OpenTracing middleware in backend/internal/middleware/tracing.go (extract/start spans, set tags)
- [x] T019 [P] Create logging middleware in backend/internal/middleware/logging.go
- [x] T020 [P] Create recovery middleware in backend/internal/middleware/recovery.go
- [x] T021 [P] Create CORS middleware in backend/internal/middleware/cors.go
- [x] T022 [P] Create error response types in backend/internal/handlers/errors.go (JSON marshaling for ErrorResponse and FieldError)
- [x] T023 Create HTTP router setup in backend/cmd/server/main.go using http.ServeMux with middleware chain
- [x] T024 [P] Create GORM database connection helper in backend/internal/database/connection.go (connection pool, health check)
- [x] T025 Create MediaStorage interface in backend/internal/services/storage.go (Store, Retrieve, Delete, URL methods)
- [x] T026 Implement LocalStorage for filesystem storage in backend/internal/services/local_storage.go
- [x] T027 [P] Create image processing utilities in backend/internal/services/image_processor.go (thumbnail generation using imaging library)
- [x] T028 [P] Create video processing utilities in backend/internal/services/video_processor.go (preview frame extraction using ffmpeg)
- [x] T029 Write integration test for testcontainers setup in backend/tests/testutil/db_test.go (verify container lifecycle, migrations)
- [x] T030 Write integration test for table truncation in backend/tests/testutil/db_test.go (verify CASCADE behavior)

**Checkpoint**: ✅ Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Define Product Template Structure (Priority: P1) 🎯 MVP

**Goal**: Administrators can create, edit, view, and delete product templates that define flexible attribute structures (text, number, boolean, date, lists, maps, images, videos). Templates serve as schemas for products.

**Independent Test**: Create a template with 10 different attribute types, save it, retrieve it, edit it, and verify all attribute definitions are preserved correctly. Delete the template and verify it's gone. This delivers the foundation for the entire PIM system.

### Integration Tests for User Story 1 (MANDATORY) ⚠️

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T031 [US1] HTTP integration test for POST /api/v1/templates in backend/tests/integration/template_test.go
  - Happy path: Create template with text, number, boolean, date, list, map, image, video attributes
  - Edge case: Empty template name (400)
  - Edge case: Duplicate template name (409 conflict)
  - Edge case: Empty attributes array (400)
  - Edge case: Invalid attribute type (400)
  - Edge case: List attribute with empty options (400)
  - Edge case: Attribute names not unique within template (400)
  - Edge case: Template name with SQL injection attempt (sanitized)
  - Edge case: Very long template name (>255 chars, 400)
  - Edge case: Special characters in attribute names
  - Use httptest.ResponseRecorder with real testcontainers PostgreSQL
  - Use protobuf CreateTemplateRequest/Response structs
  - Use GORM for database verification
  - Verify OpenTracing span creation
  - Table-driven test structure
  - Cleanup with defer truncateTables(db, "product_templates")

- [x] T032 [US1] HTTP integration test for GET /api/v1/templates/{id} in backend/tests/integration/template_test.go
  - Happy path: Retrieve existing template
  - Edge case: Non-existent template ID (404)
  - Edge case: Invalid UUID format (400)
  - Edge case: Empty ID (400)
  - Verify protobuf comparison with protocmp
  - Table-driven test structure

- [x] T033 [US1] HTTP integration test for GET /api/v1/templates in backend/tests/integration/template_test.go
  - Happy path: List all templates with pagination
  - Edge case: Empty database (returns empty array)
  - Edge case: Pagination with page > total pages (empty results)
  - Edge case: Invalid pagination parameters (negative page, page_size > 100)
  - Edge case: Search by name (partial match)
  - Edge case: Sort by name, created_at, updated_at (asc/desc)
  - Verify pagination response metadata
  - Table-driven test structure

- [x] T034 [US1] HTTP integration test for PUT /api/v1/templates/{id} in backend/tests/integration/template_test.go
  - Happy path: Update template name and attributes
  - Edge case: Non-existent template ID (404)
  - Edge case: Update to duplicate name (409)
  - Edge case: Add new attributes
  - Edge case: Remove attributes
  - Edge case: Modify existing attributes
  - Edge case: Empty update (no changes)
  - Verify updated_at timestamp changes
  - Table-driven test structure

- [x] T035 [US1] HTTP integration test for DELETE /api/v1/templates/{id} in backend/tests/integration/template_test.go
  - Happy path: Delete template with no products
  - Edge case: Non-existent template ID (404)
  - Edge case: Delete template with products using it (409 or 400 with clear message)
  - Edge case: Delete already deleted template (404)
  - Verify database row is removed
  - Table-driven test structure

### Implementation for User Story 1

- [x] T036 [P] [US1] Create ProductTemplate GORM model in backend/internal/models/template.go (ID, Name, Attributes JSONB, timestamps)
- [x] T037 [P] [US1] Create AttributeDefinition struct in backend/internal/models/template.go (for JSONB unmarshaling)
- [x] T038 [US1] Define TemplateService interface in backend/internal/services/template_service.go (Create, Get, List, Update, Delete methods)
- [x] T039 [US1] Implement TemplateService with dependency injection in backend/internal/services/template_service.go (inject *gorm.DB)
- [x] T040 [US1] Implement Create method with validation (name required, unique, attributes valid) in backend/internal/services/template_service.go
- [x] T041 [US1] Implement Get method with error handling (not found → error) in backend/internal/services/template_service.go
- [x] T042 [US1] Implement List method with pagination, search, and sorting in backend/internal/services/template_service.go
- [x] T043 [US1] Implement Update method with validation and conflict checking in backend/internal/services/template_service.go
- [x] T044 [US1] Implement Delete method with dependency checking (block if products exist) in backend/internal/services/template_service.go
- [x] T045 [US1] Create conversion helpers (GORM model ↔ protobuf) in backend/internal/services/template_service.go
- [x] T046 [US1] Create TemplateHandler struct in backend/internal/handlers/template_handler.go (inject TemplateService)
- [x] T047 [US1] Implement Create handler (POST /api/v1/templates) as thin wrapper in backend/internal/handlers/template_handler.go
- [x] T048 [US1] Add OpenTracing span for Create handler with tags (http.method, http.url, http.status_code) in backend/internal/handlers/template_handler.go
- [x] T049 [US1] Implement Get handler (GET /api/v1/templates/{id}) in backend/internal/handlers/template_handler.go
- [x] T050 [US1] Implement List handler (GET /api/v1/templates) in backend/internal/handlers/template_handler.go
- [x] T051 [US1] Implement Update handler (PUT /api/v1/templates/{id}) in backend/internal/handlers/template_handler.go
- [x] T052 [US1] Implement Delete handler (DELETE /api/v1/templates/{id}) in backend/internal/handlers/template_handler.go
- [x] T053 [US1] Register template routes in backend/cmd/server/main.go (connect handlers to ServeMux)
- [x] T054 [US1] Create fixture helper for templates in backend/tests/testutil/db.go (CreateTemplateFixture function)
- [x] T055 [US1] Run all US1 integration tests and verify they pass

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently. Administrators can create and manage product templates.

---

## Phase 4: User Story 2 - Create and Manage Products with Templates (Priority: P2)

**Goal**: Administrators can create products based on templates, filling in all attribute values including text, numbers, lists, maps, and media references. Products can be searched, filtered, edited, and deleted. This delivers the core product catalog management capability.

**Independent Test**: Select a template, create a product filling in all attribute types, save it, search for it, edit attribute values, change status, and delete it. Verify all attribute values are preserved and searchable. This works independently of variants and media upload (media references can be UUIDs).

### Integration Tests for User Story 2 (MANDATORY) ⚠️

- [x] T056 [US2] HTTP integration test for POST /api/v1/products in backend/tests/integration/product_test.go (7 test cases)
  - Happy path: Create product with all attribute types filled
  - Edge case: Missing required attributes (400)
  - Edge case: Invalid attribute types (text in number field, etc.) (400)
  - Edge case: Duplicate SKU (409)
  - Edge case: Invalid template reference (404 or 400)
  - Edge case: List values not in predefined options (400)
  - Edge case: Empty product name (400)
  - Edge case: Invalid SKU format (special chars) (400)
  - Edge case: SKU longer than 100 chars (400)
  - Edge case: Number values outside min/max (400)
  - Edge case: SQL injection in text fields (sanitized)
  - Use template fixture from US1
  - Table-driven test structure
  - Cleanup: defer truncateTables(db, "products", "product_templates")

- [x] T057 [US2] HTTP integration test for GET /api/v1/products/{id} in backend/tests/integration/product_test.go (3 test cases)
  - Happy path: Get product with all attributes
  - Edge case: Non-existent product ID (404)
  - Edge case: Include variants flag (variants array empty for now)
  - Edge case: Include media flag (media array empty for now)
  - Verify protobuf comparison for attribute values
  - Table-driven test structure

- [x] T058 [US2] HTTP integration test for GET /api/v1/products in backend/tests/integration/product_test.go (7 test cases)
  - Happy path: List products with pagination
  - Edge case: Filter by template ID
  - Edge case: Filter by status (active, inactive, draft)
  - Edge case: Search by name (partial match, case-insensitive)
  - Edge case: Search by SKU
  - Edge case: Filter by attribute values (JSONB queries)
  - Edge case: Sort by name, sku, created_at, updated_at
  - Edge case: Empty results
  - Edge case: Pagination boundary conditions
  - Table-driven test structure

- [x] T059 [US2] HTTP integration test for PUT /api/v1/products/{id} in backend/tests/integration/product_test.go (5 test cases)
  - Happy path: Update product name, SKU, description, attribute values, status
  - Edge case: Non-existent product ID (404)
  - Edge case: Update SKU to duplicate (409)
  - Edge case: Update with invalid attribute values (400)
  - Edge case: Change status: draft → active → inactive
  - Edge case: Update only some attributes (partial update)
  - Edge case: Clear optional attributes
  - Verify updated_at changes
  - Table-driven test structure

- [x] T060 [US2] HTTP integration test for DELETE /api/v1/products/{id} in backend/tests/integration/product_test.go (3 test cases)
  - Happy path: Delete product with no variants
  - Edge case: Non-existent product ID (404)
  - Edge case: Delete product with variants (cascade or block - verify expected behavior)
  - Edge case: Verify associated media is handled (for future US4)
  - Table-driven test structure

- [x] T061 [US2] HTTP integration test for POST /api/v1/products/bulk/status in backend/tests/integration/product_test.go (4 test cases)
  - Happy path: Update status for multiple products
  - Edge case: Empty product_ids array (400)
  - Edge case: Some products not found (partial success, return failed IDs)
  - Edge case: Invalid status value (400)
  - Table-driven test structure

### Implementation for User Story 2

- [x] T062 [P] [US2] Create Product GORM model in backend/internal/models/product.go (ID, TemplateID, Name, SKU, Description, AttributeValues JSONB, Status, timestamps)
- [x] T063 [P] [US2] Create AttributeValue struct for JSONB in backend/internal/models/product.go (Type, Value interface{})
- [x] T064 [US2] Define ProductService interface in backend/internal/services/product_service.go (Create, Get, List, Update, Delete, BulkUpdateStatus methods)
- [x] T065 [US2] Implement ProductService with dependency injection in backend/internal/services/product_service.go (inject *gorm.DB)
- [x] T066 [US2] Implement Create method with validation (required attributes, type checking, template exists) in backend/internal/services/product_service.go
- [x] T067 [US2] Implement attribute validation logic (check required fields, validate types, check list options) in backend/internal/services/product_service.go
- [x] T068 [US2] Implement Get method with optional includes (variants, media) in backend/internal/services/product_service.go
- [x] T069 [US2] Implement List method with filters (template, status, search, attribute filters), pagination, sorting in backend/internal/services/product_service.go
- [x] T070 [US2] Implement JSONB query builder for attribute filtering in backend/internal/services/product_service.go
- [x] T071 [US2] Implement Update method with validation and conflict checking in backend/internal/services/product_service.go
- [x] T072 [US2] Implement Delete method with cascade handling in backend/internal/services/product_service.go
- [x] T073 [US2] Implement BulkUpdateStatus method with transaction and error tracking in backend/internal/services/product_service.go
- [x] T074 [US2] Create conversion helpers (GORM model ↔ protobuf Product) in backend/internal/services/product_service.go
- [x] T075 [US2] Create ProductHandler struct in backend/internal/handlers/product_handler.go (inject ProductService)
- [x] T076 [US2] Implement Create handler (POST /api/v1/products) with OpenTracing spans in backend/internal/handlers/product_handler.go
- [x] T077 [US2] Implement Get handler (GET /api/v1/products/{id}) in backend/internal/handlers/product_handler.go
- [x] T078 [US2] Implement List handler (GET /api/v1/products) with query parameter parsing in backend/internal/handlers/product_handler.go
- [x] T079 [US2] Implement Update handler (PUT /api/v1/products/{id}) in backend/internal/handlers/product_handler.go
- [x] T080 [US2] Implement Delete handler (DELETE /api/v1/products/{id}) in backend/internal/handlers/product_handler.go
- [x] T081 [US2] Implement BulkUpdateStatus handler (POST /api/v1/products/bulk/status) in backend/internal/handlers/product_handler.go
- [x] T082 [US2] Register product routes in backend/cmd/server/main.go
- [x] T083 [US2] Create fixture helper for products in backend/tests/testutil/db.go (CreateProductFixture function)
- [x] T084 [US2] Run all US2 integration tests and verify they pass

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently. Administrators can define templates and create/manage products.

---

## Phase 5: User Story 3 - Create and Manage Product Variants (Priority: P3)

**Goal**: Administrators can create variants of products with attribute overrides. Variants inherit parent product attributes but can override specific ones (like size, color). Each variant has its own SKU and can have variant-specific media. This delivers the ability to handle product variations.

**Independent Test**: Create a product, add 3 variants with different attribute overrides (e.g., different sizes), verify each variant inherits non-overridden attributes from parent, edit a variant, delete a variant. This works independently and builds on US2.

### Integration Tests for User Story 3 (MANDATORY) ⚠️

- [x] T085 [US3] HTTP integration test for POST /api/v1/products/{product_id}/variants in backend/tests/integration/variant_test.go
  - Happy path: Create variant with some attribute overrides
  - Edge case: Non-existent product ID (404)
  - Edge case: Duplicate variant SKU (409)
  - Edge case: Empty variant name (400)
  - Edge case: Invalid variant SKU format (400)
  - Edge case: Override attribute with invalid type (400)
  - Edge case: Create variant with no overrides (all inherited)
  - Edge case: Create variant overriding all attributes
  - Use product fixture from US2
  - Verify effective_attribute_values merges parent + overrides
  - Table-driven test structure
  - Cleanup: defer truncateTables(db, "product_variants", "products", "product_templates")

- [x] T086 [US3] HTTP integration test for GET /api/v1/variants/{id} in backend/tests/integration/variant_test.go (3 test cases)
  - Happy path: Get variant with effective attributes
  - Edge case: Non-existent variant ID (404)
  - Edge case: Empty ID (400)
  - Verify effective_attribute_values shows merged result
  - Verify attribute_values shows only overrides
  - Table-driven test structure

- [x] T087 [US3] HTTP integration test for GET /api/v1/products/{product_id}/variants in backend/tests/integration/variant_test.go (3 test cases)
  - Happy path: List all variants for a product
  - Edge case: Non-existent product ID (404)
  - Edge case: Product with no variants (empty array)
  - Table-driven test structure

- [x] T088 [US3] HTTP integration test for PUT /api/v1/variants/{id} in backend/tests/integration/variant_test.go (3 test cases)
  - Happy path: Update variant name
  - Happy path: Update variant SKU
  - Edge case: Non-existent variant ID (404)
  - Table-driven test structure

- [x] T089 [US3] HTTP integration test for DELETE /api/v1/variants/{id} in backend/tests/integration/variant_test.go (3 test cases)
  - Happy path: Delete variant
  - Edge case: Non-existent variant ID (404)
  - Edge case: Verify parent product remains
  - Edge case: Verify other variants remain
  - Table-driven test structure

- [x] T090 [US3] HTTP integration test for POST /api/v1/products/{product_id}/variants/bulk in backend/tests/integration/variant_test.go
  - Happy path: Create multiple variants at once
  - Edge case: Empty variants array (400)
  - Edge case: Some variants have duplicate SKUs (partial success, return errors)
  - Edge case: Invalid product ID (404)
  - Edge case: Mix of valid and invalid variants (partial success)
  - Table-driven test structure

### Implementation for User Story 3

- [x] T091 [P] [US3] Create ProductVariant GORM model in backend/internal/models/product.go (ID, ProductID, Name, SKU, AttributeValues JSONB, timestamps)
- [x] T092 [US3] Define VariantService interface in backend/internal/services/variant_service.go (Create, Get, List, Update, Delete, BulkCreate methods)
- [x] T093 [US3] Implement VariantService with dependency injection in backend/internal/services/variant_service.go (inject *gorm.DB)
- [x] T094 [US3] Implement Create method with validation (SKU unique, product exists) in backend/internal/services/variant_service.go
- [x] T095 [US3] Implement attribute merging logic (parent attributes + overrides → effective attributes) in backend/internal/services/variant_service.go
- [x] T096 [US3] Implement Get method with attribute merging in backend/internal/services/variant_service.go
- [x] T097 [US3] Implement List method with filters, pagination, sorting in backend/internal/services/variant_service.go
- [x] T098 [US3] Implement Update method with validation and conflict checking in backend/internal/services/variant_service.go
- [x] T099 [US3] Implement Delete method in backend/internal/services/variant_service.go
- [x] T100 [US3] Implement BulkCreate method with transaction and error tracking in backend/internal/services/variant_service.go
- [x] T101 [US3] Create conversion helpers (GORM model ↔ protobuf ProductVariant) in backend/internal/services/variant_service.go
- [x] T102 [US3] Create VariantHandler struct in backend/internal/handlers/variant_handler.go (inject VariantService)
- [x] T103 [US3] Implement Create handler (POST /api/v1/products/{product_id}/variants) with OpenTracing spans in backend/internal/handlers/variant_handler.go
- [x] T104 [US3] Implement Get handler (GET /api/v1/variants/{id}) in backend/internal/handlers/variant_handler.go
- [x] T105 [US3] Implement List handler (GET /api/v1/products/{product_id}/variants) in backend/internal/handlers/variant_handler.go
- [x] T106 [US3] Implement Update handler (PUT /api/v1/variants/{id}) in backend/internal/handlers/variant_handler.go
- [x] T107 [US3] Implement Delete handler (DELETE /api/v1/variants/{id}) in backend/internal/handlers/variant_handler.go
- [x] T108 [US3] Implement BulkCreate handler (POST /api/v1/products/{product_id}/variants/bulk) in backend/internal/handlers/variant_handler.go
- [x] T109 [US3] Register variant routes in backend/cmd/server/main.go
- [x] T110 [US3] Create fixture helper for variants in backend/tests/testutil/db.go (CreateVariantFixture function)
- [x] T111 [US3] Run all US3 integration tests and verify they pass (20 test cases total)

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently. Administrators can define templates, create products, and manage variants.

---

## Phase 6: User Story 4 - Manage Media Assets (Priority: P3)

**Goal**: Administrators can upload images and videos for products and variants, with automatic thumbnail generation for images and preview frame extraction for videos. Media can be reordered, deleted, and a primary image can be designated. This delivers rich media management capabilities.

**Independent Test**: Upload 5 images to a product, verify thumbnails are generated, reorder them, set one as primary, upload a video and verify preview frame is extracted, delete a media file. This works independently and enhances US2 and US3 with visual content.

### Integration Tests for User Story 4 (MANDATORY) ⚠️

- [ ] T112 [US4] HTTP integration test for POST /api/v1/media/upload in backend/tests/integration/media_test.go
  - Happy path: Upload image (JPEG, PNG, GIF, WebP), verify thumbnail generated
  - Happy path: Upload video (MP4, WebM, MOV), verify preview frame extracted
  - Edge case: Invalid entity type (not "product" or "variant") (400)
  - Edge case: Non-existent entity ID (404)
  - Edge case: Invalid attribute name (attribute doesn't exist in template) (400)
  - Edge case: Attribute type is not image or video (400)
  - Edge case: File size exceeds limit (10MB for images, 100MB for videos) (413)
  - Edge case: Invalid file type (not supported format) (400)
  - Edge case: Corrupted image file (400)
  - Edge case: Empty file (400)
  - Edge case: Validate file by content (magic bytes), not extension
  - Use product/variant fixtures from US2/US3
  - Verify MediaFile record created in database
  - Verify files exist in storage
  - Table-driven test structure
  - Cleanup: defer truncateTables(db, "media_files", "products", "product_templates")

- [ ] T113 [US4] HTTP integration test for POST /api/v1/media/upload/bulk in backend/tests/integration/media_test.go
  - Happy path: Upload multiple images at once
  - Edge case: Empty files array (400)
  - Edge case: Mix of valid and invalid files (partial success, return errors)
  - Edge case: Some files exceed size limit (partial success)
  - Edge case: Duplicate filenames (handled gracefully)
  - Table-driven test structure

- [ ] T114 [US4] HTTP integration test for GET /api/v1/media/{id} in backend/tests/integration/media_test.go
  - Happy path: Get media file metadata
  - Edge case: Non-existent media ID (404)
  - Verify all fields populated (width, height, duration for video, etc.)
  - Table-driven test structure

- [ ] T115 [US4] HTTP integration test for GET /api/v1/media in backend/tests/integration/media_test.go
  - Happy path: List media for product
  - Happy path: List media for variant
  - Edge case: Filter by attribute name
  - Edge case: Filter by file type (image vs video)
  - Edge case: Entity with no media (empty array)
  - Edge case: Invalid entity type (400)
  - Verify display_order is respected
  - Table-driven test structure

- [ ] T116 [US4] HTTP integration test for PUT /api/v1/media/{id} in backend/tests/integration/media_test.go
  - Happy path: Update display_order
  - Happy path: Rename file (display name only)
  - Edge case: Non-existent media ID (404)
  - Edge case: Invalid display_order (negative) (400)
  - Verify updated_at not present (no updated_at in MediaFile per data model)
  - Table-driven test structure

- [ ] T117 [US4] HTTP integration test for DELETE /api/v1/media/{id} in backend/tests/integration/media_test.go
  - Happy path: Delete media file
  - Edge case: Non-existent media ID (404)
  - Verify database record removed
  - Verify files removed from storage
  - Table-driven test structure

- [ ] T118 [US4] HTTP integration test for POST /api/v1/media/reorder in backend/tests/integration/media_test.go
  - Happy path: Reorder media files for an attribute
  - Edge case: Empty media_ids array (400)
  - Edge case: Media IDs don't belong to specified entity (400)
  - Edge case: Media IDs don't belong to specified attribute (400)
  - Edge case: Duplicate IDs in array (400)
  - Verify display_order updated correctly (0, 1, 2, ...)
  - Table-driven test structure

### Implementation for User Story 4

- [x] T119 [P] [US4] Create MediaFile GORM model in backend/internal/models/product.go (ID, EntityType, EntityID, AttributeName, FileType, MimeType, FileName, FilePath, FileSize, Width, Height, Duration, DisplayOrder, timestamps)
- [x] T120 [US4] Define MediaService interface in backend/internal/services/media_service.go (Upload, UploadBulk, Get, List, Update, Delete, Reorder, GetURL, GetThumbnailURL methods)
- [ ] T121 [US4] Implement MediaService with dependency injection in backend/internal/services/media_service.go (inject *gorm.DB, MediaStorage, ImageProcessor, VideoProcessor)
- [ ] T122 [US4] Implement file validation (type by magic bytes, size limits) in backend/internal/services/media_service.go
- [ ] T123 [US4] Implement Upload method for images (validate, store, generate thumbnail, save metadata) in backend/internal/services/media_service.go
- [ ] T124 [US4] Implement Upload method for videos (validate, store, extract preview frame, save metadata) in backend/internal/services/media_service.go
- [ ] T125 [US4] Implement UploadBulk method with error tracking in backend/internal/services/media_service.go
- [ ] T126 [US4] Implement Get method in backend/internal/services/media_service.go
- [ ] T127 [US4] Implement List method with filters (entity, attribute, file type) in backend/internal/services/media_service.go
- [ ] T128 [US4] Implement Update method (display_order, file_name) in backend/internal/services/media_service.go
- [ ] T129 [US4] Implement Delete method (remove from storage and database) in backend/internal/services/media_service.go
- [ ] T130 [US4] Implement Reorder method with validation and transaction in backend/internal/services/media_service.go
- [ ] T131 [US4] Implement GetURL and GetThumbnailURL methods in backend/internal/services/media_service.go
- [ ] T132 [US4] Create conversion helpers (GORM model ↔ protobuf MediaFile) in backend/internal/services/media_service.go
- [ ] T133 [US4] Create MediaHandler struct in backend/internal/handlers/media_handler.go (inject MediaService)
- [ ] T134 [US4] Implement Upload handler (POST /api/v1/media/upload) with multipart form parsing and OpenTracing spans in backend/internal/handlers/media_handler.go
- [ ] T135 [US4] Implement UploadBulk handler (POST /api/v1/media/upload/bulk) in backend/internal/handlers/media_handler.go
- [ ] T136 [US4] Implement Get handler (GET /api/v1/media/{id}) in backend/internal/handlers/media_handler.go
- [ ] T137 [US4] Implement List handler (GET /api/v1/media) with query parameter parsing in backend/internal/handlers/media_handler.go
- [ ] T138 [US4] Implement Update handler (PUT /api/v1/media/{id}) in backend/internal/handlers/media_handler.go
- [ ] T139 [US4] Implement Delete handler (DELETE /api/v1/media/{id}) in backend/internal/handlers/media_handler.go
- [ ] T140 [US4] Implement Reorder handler (POST /api/v1/media/reorder) in backend/internal/handlers/media_handler.go
- [ ] T141 [US4] Implement file serving handler (GET /api/v1/media/file/{id}) with redirect to storage URL in backend/internal/handlers/media_handler.go
- [ ] T142 [US4] Implement thumbnail serving handler (GET /api/v1/media/thumbnail/{id}) in backend/internal/handlers/media_handler.go
- [ ] T143 [US4] Register media routes in backend/cmd/server/main.go
- [ ] T144 [US4] Create fixture helper for media files in backend/tests/testutil/fixtures.go (createMediaFixture function)
- [ ] T145 [US4] Run all US4 integration tests and verify they pass

**Checkpoint**: All user stories should now be independently functional. The PIM system is feature-complete.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories, final quality assurance

- [ ] T146 [P] Add comprehensive error logging across all services
- [ ] T147 [P] Add database query optimization (analyze slow queries, add indexes if needed)
- [ ] T148 [P] Verify all OpenTracing spans are properly nested (parent-child relationships)
- [ ] T149 [P] Add API documentation comments to all handlers
- [ ] T150 [P] Create README.md in backend/ directory with setup and run instructions
- [ ] T151 [P] Create Dockerfile for backend API server
- [ ] T152 [P] Create docker-compose.yml for local development (API + PostgreSQL)
- [ ] T153 Run all integration tests across all user stories to verify no regressions
- [ ] T154 Verify quickstart.md instructions work end-to-end
- [ ] T155 [P] Performance testing: Verify product search meets < 1 second requirement for 10K products
- [ ] T156 [P] Performance testing: Verify media upload meets < 10 second requirement for 10MB files
- [ ] T157 [P] Security audit: Verify SQL injection prevention, XSS prevention, file upload validation
- [ ] T158 Manual testing: Walk through all user stories end-to-end
- [ ] T159 Code cleanup: Remove unused imports, fix linter warnings
- [ ] T160 Final constitution compliance check: Verify all principles followed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Foundational phase completion
  - After Foundational is done, user stories CAN proceed in parallel (if team capacity allows)
  - Or sequentially in priority order: US1 (P1) → US2 (P2) → US3 (P3) → US4 (P3)
  - Each story is independently testable and deliverable
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1) - Templates**: No dependencies on other stories (only Foundational)
- **User Story 2 (P2) - Products**: Depends on US1 for templates, but independently testable
- **User Story 3 (P3) - Variants**: Depends on US2 for products, but independently testable
- **User Story 4 (P3) - Media**: Can attach to products (US2) or variants (US3), but independently testable

**Recommended Order**: US1 → US2 → US3 → US4 (follows dependencies and priorities)

**Parallel Option**: After Foundational, US1 and US4 can be developed in parallel (US4 can use mock/fixture entities initially)

### Within Each User Story

1. **Tests FIRST** (write and ensure they FAIL)
2. **Models** (GORM models)
3. **Service Interface** (business logic contracts)
4. **Service Implementation** (business logic with validation)
5. **Handlers** (thin HTTP wrappers)
6. **Route Registration** (connect to ServeMux)
7. **Fixture Helpers** (test utilities)
8. **Verify Tests PASS**

### Parallel Opportunities

- **Within Setup Phase**: All tasks marked [P] can run in parallel (T002-T008, T012-T014)
- **Within Foundational Phase**: Many tasks marked [P] can run in parallel (T018-T022, T024, T027-T028)
- **Across User Stories**: After Foundational completes, different user stories can be worked on by different developers
- **Within User Story Tests**: All test files for a story can be written in parallel
- **Within User Story Models**: All models for a story (if multiple) can be created in parallel

---

## Parallel Example: User Story 1 (Templates)

After Foundational phase completes, User Story 1 can proceed:

### Step 1: Write ALL tests in parallel (T031-T035)
```bash
# Developer A writes: T031 POST /api/v1/templates test
# Developer B writes: T032 GET /api/v1/templates/{id} test
# Developer C writes: T033 GET /api/v1/templates test
# Developer D writes: T034 PUT /api/v1/templates/{id} test
# Developer E writes: T035 DELETE /api/v1/templates/{id} test
```

### Step 2: Create models in parallel (T036-T037)
```bash
# All model structs can be written simultaneously
```

### Step 3: Implement service (T038-T045 sequential within service)
Service methods depend on each other, so less parallelism here

### Step 4: Implement handlers in parallel (T047-T052)
```bash
# Each endpoint handler can be written independently
# Developer A: Create handler (T047-T048)
# Developer B: Get handler (T049)
# Developer C: List handler (T050)
# Developer D: Update handler (T051)
# Developer E: Delete handler (T052)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only) - Recommended Start

1. ✅ Complete Phase 1: Setup (T001-T015)
2. ✅ Complete Phase 2: Foundational (T016-T030) **CRITICAL BLOCKER**
3. ✅ Complete Phase 3: User Story 1 (T031-T055)
4. 🛑 **STOP and VALIDATE**: Test US1 independently
5. ✅ Deploy/demo template management if ready

**Outcome**: Administrators can create and manage product templates (foundation for entire system)

### Incremental Delivery (Recommended Full Approach)

1. **Foundation Ready**: Setup + Foundational → Infrastructure complete
2. **MVP Delivery**: + User Story 1 → Template management (deployable!)
3. **Core Product Catalog**: + User Story 2 → Product management (deployable!)
4. **Product Variations**: + User Story 3 → Variant management (deployable!)
5. **Rich Media**: + User Story 4 → Media management (feature-complete!)
6. **Production Ready**: + Polish → Optimized and documented

**Each step delivers independent value and can be deployed**

### Parallel Team Strategy (If Multiple Developers)

**Phase 1-2**: Everyone works together on Setup and Foundational

**Once Foundational is complete, split into parallel tracks**:

- **Track A (Priority 1)**: Developer(s) implement US1 (Templates)
  - T031-T055
  - First to complete, critical blocker for US2

- **Track B (Can start simultaneously)**: Developer(s) implement US4 (Media) infrastructure
  - T119-T121, T122-T132 (service layer and utilities)
  - Can use mock entities initially, integrate with US2/US3 later
  - OR wait for US1 completion

**After US1 completes**:

- **Track A continues**: Implement US2 (Products)
  - T056-T084

**After US2 completes**:

- **Track A continues**: Implement US3 (Variants)
  - T085-T111

**After US3 completes**:

- **Track A+B merge**: Complete US4 handler integration if needed
  - T133-T145

**Finally, everyone**: Polish (T146-T160)

---

## Task Summary

### Total Tasks: 160

**By Phase**:
- Phase 1 (Setup): 15 tasks
- Phase 2 (Foundational): 15 tasks
- Phase 3 (US1 - Templates): 25 tasks (5 test files + 20 implementation)
- Phase 4 (US2 - Products): 29 tasks (6 test files + 23 implementation)
- Phase 5 (US3 - Variants): 27 tasks (6 test files + 21 implementation)
- Phase 6 (US4 - Media): 34 tasks (7 test files + 27 implementation)
- Phase 7 (Polish): 15 tasks

**By User Story**:
- US1 (P1): 25 tasks
- US2 (P2): 29 tasks
- US3 (P3): 27 tasks
- US4 (P3): 34 tasks
- Infrastructure: 45 tasks (Setup + Foundational + Polish)

**Parallel Opportunities**: 58 tasks marked [P] can run in parallel when dependencies are met

**Independent Test Criteria**:
- ✅ US1: Create/manage templates independently
- ✅ US2: Create/manage products independently (requires US1 templates)
- ✅ US3: Create/manage variants independently (requires US2 products)
- ✅ US4: Upload/manage media independently (requires US2 products OR US3 variants)

**Suggested MVP Scope**: Phases 1, 2, 3 (User Story 1 only) = 55 tasks
- Delivers template management (foundation)
- Fully functional and deployable
- Enables product creation in next iteration

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] labels**: Map task to specific user story for traceability
- **TDD Workflow**: ALL tests must be written FIRST and FAIL before implementation
- **Constitution Compliance**: All tests use real PostgreSQL (testcontainers), table-driven patterns, GORM fixtures, protobuf structs, comprehensive edge cases
- **Each user story is independently testable**: Can deploy US1 alone, then add US2, etc.
- **Stop at any checkpoint**: Validate story works independently before proceeding
- **Commit frequently**: After each task or logical group
- **File paths are exact**: Follow the paths specified in each task description
- **OpenTracing**: ALL endpoints must create spans with proper tags
- **Error handling**: ALL errors must be logged and returned with appropriate HTTP status codes

