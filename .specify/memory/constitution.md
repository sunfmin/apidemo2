<!--
Sync Impact Report:
- Version: 1.5.1 → 1.5.2 (PATCH bump - clarify tracing granularity: service operations, not individual SQL)
- Modified principles:
  - VII. Distributed Tracing - clarified NOT to trace individual SQL executions
- Clarifications added:
  - Trace at service operation level (e.g., "ProductService.Create")
  - Do NOT trace individual SQL queries (too much overhead)
  - Use database logging/metrics for SQL query analysis instead
  - Keep tracing focused on request flow and service boundaries
- Rationale:
  - Individual SQL tracing creates excessive overhead (spans, storage, bandwidth)
  - Overwhelming trace data makes debugging harder (too much noise)
  - Most SQL queries are fast and don't need individual spans
  - Database query analysis better served by database metrics/slow query logs
  - Tracing should focus on inter-service communication and operation flow
- Impact:
  - Fewer spans created (less overhead, cleaner traces)
  - Service-level spans still show database operation timing
  - Use pg_stat_statements or database logs for SQL-level analysis
  - Clearer guidance on what to trace
- Templates requiring updates:
  ✅ No template changes needed
-->


# apidemo2 Constitution

## Core Principles

### I. Integration Testing First (No Mocking)

All tests MUST be integration tests that interact with real dependencies:
- Tests MUST use real PostgreSQL database connections
- NO mocking of database calls, HTTP clients, or external services
- Tests MUST prepare fixture data directly in the database
- Test database MUST be isolated per test run
- Each test MUST use database truncation for isolation (truncate tables after test)

**Rationale**: Integration tests catch real-world issues that unit tests with mocks cannot, including database constraint violations, connection pooling issues, transaction handling bugs, and serialization problems.

### II. Table-Driven Test Design

All tests MUST follow table-driven test patterns:
- Tests MUST define test cases as slices of structs with inputs and expected outputs
- Each test case MUST have a descriptive name field
- Test runner MUST iterate over cases using `t.Run(testCase.name, func(t *testing.T) {...})`
- Shared setup/teardown logic MUST be extracted to helper functions
- Test tables MUST be readable and maintainable

**Rationale**: Table-driven tests reduce code duplication, make test cases easy to add/modify, improve test readability, and enable comprehensive scenario coverage with minimal code.

### III. Edge Case Coverage (NON-NEGOTIABLE)

Every API endpoint MUST test comprehensive edge cases:
- **Input validation**: Empty strings, nil values, invalid formats, SQL injection attempts, XSS payloads
- **Boundary conditions**: Zero values, negative numbers, maximum values, empty arrays, nil pointers
- **Authentication/Authorization**: Missing tokens, expired tokens, invalid tokens, insufficient permissions
- **Data state**: Non-existent resources (404), duplicate entries (conflict), concurrent modifications
- **Database errors**: Constraint violations, foreign key failures, transaction conflicts
- **HTTP specifics**: Wrong methods, missing headers, invalid content-types, malformed JSON
- Edge cases MUST be documented in test case names

**Rationale**: Production systems face unexpected inputs and conditions. Comprehensive edge case testing prevents security vulnerabilities, data corruption, and runtime panics.

### IV. Real Database Fixtures

Test data MUST be prepared using real database operations:
- Fixture data MUST be inserted using GORM to real test database
- NO in-memory mocks or fake repositories
- Fixtures MUST represent realistic production data scenarios
- Complex fixtures (with foreign keys, relationships) MUST be created with helper functions
- Fixture helpers MUST return created entities for test verification
- Fixtures MUST handle database constraints correctly

**Rationale**: Real database fixtures ensure tests validate actual database behavior including constraints, triggers, indexes, and query performance.

### V. ServeHTTP Endpoint Testing

API endpoints MUST be tested via ServeHTTP interface:
- Tests MUST create `httptest.ResponseRecorder` to capture responses
- Tests MUST construct `*http.Request` with proper method, URL, headers, and body
- Tests MUST pass requests through actual HTTP handler chains (middleware included)
- Tests MUST verify response status codes, headers, and body content
- JSON responses MUST be parsed and validated structurally
- Tests MUST NOT bypass HTTP layer by calling service functions directly

**Rationale**: Testing through ServeHTTP ensures complete HTTP stack validation including routing, middleware, request parsing, content negotiation, error handling, and response formatting.

### VI. Protobuf Data Structures

All public API data structures MUST be defined in Protocol Buffers:
- API request and response types MUST be defined in `.proto` files
- Tests MUST use protobuf-generated structs, NOT `map[string]interface{}`
- NO use of untyped maps for request/response handling in tests or production code
- Protobuf definitions MUST be the single source of truth for API contracts
- Generated Go structs MUST be used for JSON marshaling/unmarshaling
- Protobuf messages MUST include field validation rules (e.g., `validate.rules`)
- All API changes MUST update the corresponding `.proto` files first
- Tests MUST use proper protobuf comparison packages for assertions (e.g., `protocmp` with `google/go-cmp`)
- Tests MUST NOT use standard `==` or `reflect.DeepEqual` for protobuf message comparison

**Rationale**: Protobuf provides compile-time type safety, eliminates runtime type assertion errors, enables automatic validation, supports multiple language clients, enforces schema-first API design, and prevents the fragile `map[string]interface{}` pattern that loses type information and requires extensive runtime validation. Proper protobuf comparison ensures correct field comparison including unknown fields, extensions, and proto semantics.

**Examples**:
```protobuf
// api/product.proto
message ProductCreateRequest {
  string name = 1 [(validate.rules).string.min_len = 1];
  string sku = 2 [(validate.rules).string.pattern = "^[a-zA-Z0-9_-]+$"];
  string description = 3;
  map<string, AttributeValue> attributes = 4;
}

message Product {
  string id = 1;
  string name = 2;
  string sku = 3;
  string description = 4;
  map<string, AttributeValue> attributes = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}
```

**Test Usage**:
```go
// CORRECT: Use protobuf structs
req := &pb.ProductCreateRequest{
    Name: "Test Product",
    Sku:  "TEST-001",
}

// WRONG: Do not use maps
req := map[string]interface{}{
    "name": "Test Product",
    "sku":  "TEST-001",
}
```

**Test Assertions**:
```go
import (
    "testing"
    "github.com/google/go-cmp/cmp"
    "google.golang.org/protobuf/testing/protocmp"
)

// CORRECT: Use protocmp for protobuf comparison
expected := &pb.Product{Name: "Test", Sku: "TEST-001"}
actual := &pb.Product{Name: "Test", Sku: "TEST-001"}

if diff := cmp.Diff(expected, actual, protocmp.Transform()); diff != "" {
    t.Errorf("Product mismatch (-want +got):\n%s", diff)
}

// WRONG: Do not use == or DeepEqual
if actual != expected { // Incorrect for protobuf
    t.Error("mismatch")
}
```

### VII. Distributed Tracing (OpenTracing)

All API endpoints MUST be instrumented with distributed tracing at appropriate granularity:
- Each HTTP endpoint handler MUST create or continue an OpenTracing span
- Spans MUST include operation name matching the endpoint (e.g., "POST /api/products")
- Service method calls SHOULD create child spans (e.g., "ProductService.Create")
- Database operations SHOULD be traced as a single child span per transaction (NOT per SQL query)
- Individual SQL queries MUST NOT be traced (too much overhead, use database metrics instead)
- External service calls (HTTP, gRPC) MUST propagate trace context and create spans
- Error conditions MUST be logged to the active span with `span.SetTag("error", true)`
- Trace context MUST be extracted from incoming HTTP headers (e.g., `X-B3-TraceId`)
- Trace context MUST be injected into outgoing HTTP requests
- Spans MUST include relevant tags: `http.method`, `http.url`, `http.status_code`, `service.method`
- Tests MUST verify tracing instrumentation (e.g., using mock tracer or test spans)

**Rationale**: Distributed tracing provides critical observability for debugging latency issues, understanding request flows across services, identifying bottlenecks, and correlating logs across distributed systems. OpenTracing offers a vendor-neutral API compatible with Jaeger, Zipkin, and other tracing backends. Tracing at service operation level (not individual SQL queries) keeps overhead low while providing actionable insights. For SQL query analysis, use database-specific tools like `pg_stat_statements`, slow query logs, or APM database profiling.

**Example - Correct Tracing Granularity**:
```go
import (
    "net/http"
    "github.com/opentracing/opentracing-go"
    "github.com/opentracing/opentracing-go/ext"
)

// HTTP Handler - creates root span
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Extract or start HTTP span
    spanCtx, _ := opentracing.GlobalTracer().Extract(
        opentracing.HTTPHeaders,
        opentracing.HTTPHeadersCarrier(r.Header),
    )
    span := opentracing.StartSpan("POST /api/products", ext.RPCServerOption(spanCtx))
    defer span.Finish()
    
    span.SetTag("http.method", r.Method)
    span.SetTag("http.url", r.URL.String())
    
    // Parse request
    var req pb.ProductCreateRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Service call creates child span (one span for entire operation)
    ctx := opentracing.ContextWithSpan(r.Context(), span)
    product, err := h.service.Create(ctx, &req)
    
    if err != nil {
        span.SetTag("error", true)
        span.SetTag("error.message", err.Error())
        http.Error(w, err.Error(), 500)
        return
    }
    
    span.SetTag("http.status_code", http.StatusCreated)
    json.NewEncoder(w).Encode(product)
}

// Service - creates child span for business logic
func (s *productService) Create(ctx context.Context, req *pb.ProductCreateRequest) (*pb.Product, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "ProductService.Create")
    defer span.Finish()
    
    span.SetTag("product.name", req.Name)
    
    // Entire database transaction in this span
    // DO NOT create spans for individual SQL queries
    tx := s.db.WithContext(ctx).Begin()
    defer tx.Rollback()
    
    product := &Product{Name: req.Name, SKU: req.Sku}
    if err := tx.Create(product).Error; err != nil {
        span.SetTag("error", true)
        return nil, err
    }
    
    if err := tx.Commit().Error; err != nil {
        span.SetTag("error", true)
        return nil, err
    }
    
    return toProto(product), nil
}

// Result: Clean trace hierarchy
// POST /api/products (100ms)
//   └─ ProductService.Create (95ms)  ← Database work included here
//
// NOT this (too noisy):
// POST /api/products (100ms)
//   └─ ProductService.Create (95ms)
//       ├─ db.begin (1ms)
//       ├─ db.insert (50ms)
//       ├─ db.select (30ms)
//       └─ db.commit (10ms)  ← Too much detail!
```

**For SQL Query Analysis**: Use PostgreSQL's `pg_stat_statements` extension, slow query logs, or database monitoring tools (not distributed tracing).

### VIII. Service Layer Architecture (Dependency Injection)

Business logic MUST be separated from HTTP transport using service interfaces:
- Business logic MUST be implemented as Go interfaces (service layer)
- Services MUST NOT depend on HTTP types (`http.Request`, `http.ResponseWriter`, `context.Context` is allowed)
- HTTP handlers MUST be thin wrappers that call service methods
- Services MUST accept all inputs as method parameters (no HTTP request parsing in services)
- External dependencies (database, logger, cache, etc.) MUST be injected via constructor
- Services MUST be exported and usable as normal Go packages
- Service interfaces MUST be defined in the same package as implementation
- Multiple applications MUST be able to share the same service instances

**Rationale**: Separating business logic from HTTP transport enables code reuse across multiple contexts (HTTP APIs, gRPC services, CLI tools, background workers, embedded usage in other Go apps). Dependency injection allows applications to share expensive resources like database connection pools and caches. This architecture makes services testable without HTTP layer overhead and allows the package to be imported and used as a library in other Go applications.

**Architecture Layers**:
```
HTTP Handler (thin) → Service Interface (business logic) → Repository (data access)
```

**Example Service Interface**:
```go
// Service interface - pure business logic, no HTTP dependencies
type ProductService interface {
    Create(ctx context.Context, req *pb.ProductCreateRequest) (*pb.Product, error)
    Get(ctx context.Context, id string) (*pb.Product, error)
    List(ctx context.Context, limit, offset int) ([]*pb.Product, error)
    Update(ctx context.Context, id string, req *pb.ProductUpdateRequest) (*pb.Product, error)
    Delete(ctx context.Context, id string) error
}

// Service implementation with dependency injection
type productService struct {
    db     *gorm.DB
    logger Logger
    cache  Cache
}

// Constructor with dependency injection
func NewProductService(db *gorm.DB, logger Logger, cache Cache) ProductService {
    return &productService{
        db:     db,
        logger: logger,
        cache:  cache,
    }
}

// Service method - pure business logic, no HTTP types
func (s *productService) Create(ctx context.Context, req *pb.ProductCreateRequest) (*pb.Product, error) {
    // Validate business rules
    if req.Name == "" {
        return nil, errors.New("product name required")
    }
    
    // Create entity
    product := &Product{
        Name: req.Name,
        SKU:  req.Sku,
        Description: req.Description,
    }
    
    // Persist with transaction
    tx := s.db.WithContext(ctx).Begin()
    defer tx.Rollback()
    
    if err := tx.Create(product).Error; err != nil {
        s.logger.Error("failed to create product", err)
        return nil, err
    }
    
    if err := tx.Commit().Error; err != nil {
        return nil, err
    }
    
    // Convert to protobuf response
    return &pb.Product{
        Id:          product.ID,
        Name:        product.Name,
        Sku:         product.SKU,
        Description: product.Description,
    }, nil
}
```

**HTTP Handler (Thin Wrapper)**:
```go
// HTTP handler delegates to service
type ProductHandler struct {
    service ProductService // Injected dependency
}

func NewProductHandler(service ProductService) *ProductHandler {
    return &ProductHandler{service: service}
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Extract trace span (HTTP concern)
    span, ctx := opentracing.StartSpanFromContext(r.Context(), "POST /api/products")
    defer span.Finish()
    
    // Parse HTTP request (HTTP concern)
    var req pb.ProductCreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", 400)
        return
    }
    
    // Delegate to service (business logic)
    product, err := h.service.Create(ctx, &req)
    if err != nil {
        span.SetTag("error", true)
        http.Error(w, err.Error(), 500)
        return
    }
    
    // Format HTTP response (HTTP concern)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(product)
}
```

**Usage in Other Go Applications**:
```go
// Application 1: HTTP API server
func main() {
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    logger := NewLogger()
    cache := NewCache()
    
    // Create service with shared dependencies
    productService := NewProductService(db, logger, cache)
    
    // Use in HTTP handlers
    handler := NewProductHandler(productService)
    http.HandleFunc("/api/products", handler.Create)
    http.ListenAndServe(":8080", nil)
}

// Application 2: Background worker (shares same db/logger)
func main() {
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    logger := NewLogger()
    cache := NewCache()
    
    // Share same service instances!
    productService := NewProductService(db, logger, cache)
    
    // Use directly without HTTP layer
    ctx := context.Background()
    products, _ := productService.List(ctx, 100, 0)
    for _, p := range products {
        // Process products...
    }
}

// Application 3: Embedded in larger application
import "github.com/yourorg/apidemo2/services"

func processOrders() {
    // Import and use services directly
    productSvc := services.NewProductService(sharedDB, sharedLogger, sharedCache)
    
    product, err := productSvc.Get(ctx, productID)
    // Use product in your business logic
}
```

**Testing Through HTTP Layer (Full Stack)**:
```go
// HTTP integration tests cover full stack: HTTP → Service → Repository
func TestProductHandler_Create(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    defer truncateTables(db, "products")
    
    // Setup service with real dependencies
    logger := NewLogger()
    cache := NewCache()
    service := NewProductService(db, logger, cache)
    
    // Setup HTTP handler
    handler := NewProductHandler(service)
    
    // Create HTTP request
    reqBody := &pb.ProductCreateRequest{
        Name: "Test Product",
        Sku:  "TEST-001",
    }
    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest("POST", "/api/products", bytes.NewReader(body))
    rec := httptest.NewRecorder()
    
    // Test through HTTP layer (exercises Service → Repository)
    handler.Create(rec, req)
    
    // Assert HTTP response
    if rec.Code != 200 {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
    
    var product pb.Product
    json.NewDecoder(rec.Body).Decode(&product)
    if product.Name != reqBody.Name {
        t.Errorf("Expected name %s, got %s", reqBody.Name, product.Name)
    }
    
    // Note: This test covers HTTP parsing, service business logic, 
    // and repository database operations - full integration!
}
```

**Note**: We do NOT test services separately. HTTP integration tests already cover the full stack (HTTP → Service → Repository → Database). The service layer exists for code reusability and clean architecture, not for separate testing.

## Technology Stack

- **Language**: Go 1.21+ (recommend latest stable)
- **Database**: PostgreSQL 15+ (with JSONB support)
- **HTTP Framework**: Standard library `net/http` using `http.ServeMux`
- **Database Access**: GORM (gorm.io/gorm with gorm.io/driver/postgres)
- **Distributed Tracing**: OpenTracing (github.com/opentracing/opentracing-go)
- **Protocol Buffers**: protoc compiler, protoc-gen-go, protoc-gen-go-grpc
- **Validation**: protoc-gen-validate for protobuf field validation
- **Testing**: Standard library `testing` package with `httptest`
- **Test Comparison**: google/go-cmp with protocmp for protobuf message assertions
- **Test Database**: testcontainers-go with PostgreSQL module (automatic Docker container management)
- **Migration Tool**: GORM AutoMigrate (for development and testing)

## Development Workflow

### Test-First Development (TDD)

1. **Design Phase**: Define API contract in `.proto` files (request/response messages)
2. **Generate Code**: Run `protoc` to generate Go structs from protobuf definitions
3. **Write Tests**: Create table-driven integration tests using protobuf structs (NOT maps)
4. **Verify Failure**: Run tests to confirm they fail (red phase)
5. **Review Tests**: Review test design with team/lead before implementation
6. **Implement**: Write minimal code to make tests pass (green phase)
7. **Refactor**: Improve code quality while keeping tests green
8. **No Implementation Before Tests**: Code written before test approval MUST be discarded

### Protobuf Workflow

1. **Define Schema**: Create or update `.proto` files in `api/` directory
2. **Generate Code**: Run `go generate` to create Go structs (add `//go:generate` directives)
3. **Use in Code**: Import generated packages, use typed structs throughout
4. **Validate**: Use protoc-gen-validate for automatic field validation
5. **Version**: Use protobuf field numbers consistently (never reuse deleted field numbers)

### Test Database Management

- Each developer MUST have Docker Engine (Linux) or Docker Desktop (Mac/Windows) installed
- Test suite MUST use testcontainers-go library for PostgreSQL container management
- Test suite MUST use `testcontainers.PostgresContainer` for automatic lifecycle management
- Container startup, port allocation, and cleanup are handled automatically by testcontainers
- Test database schema MUST match production schema via GORM AutoMigrate
- Container cleanup MUST use `defer container.Terminate(ctx)` pattern
- CI/CD environments MUST have Docker daemon available for testcontainers

### Test Isolation Strategy

**Database Truncation** - The single, simple approach for all tests.

**How it works:**
- Tests run with real database and commit transactions normally
- After each test completes, truncate all modified tables to clean up
- Use `defer` pattern to ensure cleanup happens even on test failure

**Requirements:**
- Tests MUST truncate all relevant tables after each test using `defer`
- Truncation MUST use `CASCADE` to handle foreign key constraints
- Truncate in reverse dependency order (children before parents) for safety
- Use helper function to centralize truncation logic

**Benefits:**
- ✅ Works with any code structure (no special patterns needed)
- ✅ Tests actual production behavior (with real commits)
- ✅ Simple and reliable
- ✅ No limitations or gotchas
- ✅ Write production code naturally
- ✅ Fast enough (~1-5ms overhead per test)

**Example Testcontainers Setup**:
```go
import (
    "context"
    "fmt"
    "testing"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

// Setup PostgreSQL test container using testcontainers
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
    ctx := context.Background()
    
    // Create PostgreSQL container
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("postgres"),
        postgres.WithPassword("postgres"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2),
        ),
    )
    if err != nil {
        t.Fatalf("Failed to start PostgreSQL container: %v", err)
    }
    
    // Cleanup function
    cleanup := func() {
        if err := pgContainer.Terminate(ctx); err != nil {
            t.Logf("Failed to terminate container: %v", err)
        }
    }
    
    // Get connection string
    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        cleanup()
        t.Fatalf("Failed to get connection string: %v", err)
    }
    
    // Connect using GORM
    db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
    if err != nil {
        cleanup()
        t.Fatalf("Failed to connect to database: %v", err)
    }
    
    // Run GORM AutoMigrate
    if err := db.AutoMigrate(&Product{}, &Order{}); err != nil {
        cleanup()
        t.Fatalf("Failed to run migrations: %v", err)
    }
    
    return db, cleanup
}

// Create helper function for truncation (reusable across all tests)
func truncateTables(db *gorm.DB, tables ...string) {
    // Truncate in reverse order (children before parents)
    for i := len(tables) - 1; i >= 0; i-- {
        db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tables[i]))
    }
}

// Example test with inline truncation
func TestProductCreate(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Truncate tables after test (in reverse dependency order)
    defer func() {
        db.Exec("TRUNCATE TABLE order_items CASCADE")
        db.Exec("TRUNCATE TABLE orders CASCADE")
        db.Exec("TRUNCATE TABLE products CASCADE")
    }()
    
    // Handler uses normal code with real db and commits
    handler := NewProductHandler(db)
    product := &Product{Name: "Test Product", SKU: "TEST-001"}
    
    // Handler commits internally (normal production behavior)
    err := handler.Create(product)
    if err != nil {
        t.Fatalf("Failed to create product: %v", err)
    }
    
    // Verify committed data (tests actual behavior)
    var found Product
    if err := db.First(&found, product.ID).Error; err != nil {
        t.Fatalf("Product not found: %v", err)
    }
    
    // Truncate cleans up at end (even on test failure)
}

// Cleaner approach using helper function
func TestProductCreateWithHelper(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    defer truncateTables(db, "products", "orders", "order_items")
    
    handler := NewProductHandler(db)
    product := &Product{Name: "Test Product", SKU: "TEST-001"}
    
    if err := handler.Create(product); err != nil {
        t.Fatalf("Failed: %v", err)
    }
    
    // Verify product was created
    var found Product
    if err := db.First(&found, product.ID).Error; err != nil {
        t.Fatalf("Product not found: %v", err)
    }
    
    if found.Name != product.Name {
        t.Errorf("Expected name %s, got %s", product.Name, found.Name)
    }
}
```

**Parallel Test Support**:
```go
// Safe with testcontainers - each test gets own container
func TestProductCreateParallel(t *testing.T) {
    t.Parallel() // Each parallel test gets isolated container
    
    db, cleanup := setupTestDB(t)
    defer cleanup()
    defer truncateTables(db, "products", "orders", "order_items")
    
    handler := NewProductHandler(db)
    product := &Product{Name: "Test", SKU: "TEST"}
    
    if err := handler.Create(product); err != nil {
        t.Fatalf("Failed: %v", err)
    }
    
    // Verify product exists
    var found Product
    db.First(&found, product.ID)
    // Assertions...
}
```

**Handler Implementation (Simple Production Code)**:
```go
// Normal handler with db dependency - no special test structure
type ProductHandler struct {
    db *gorm.DB
}

func NewProductHandler(db *gorm.DB) *ProductHandler {
    return &ProductHandler{db: db}
}

func (h *ProductHandler) Create(product *Product) error {
    // Normal production code - manages its own transaction
    tx := h.db.Begin()
    defer tx.Rollback()
    
    if err := tx.Create(product).Error; err != nil {
        return err
    }
    
    return tx.Commit().Error // Commits normally
}

// HTTP handler is straightforward
func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var req pb.ProductCreateRequest
    if err := json.NewDecoder(r.Body).Unmarshal(&req); err != nil {
        http.Error(w, "Invalid request", 400)
        return
    }
    
    product := &Product{Name: req.Name, SKU: req.Sku}
    if err := h.Create(product); err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    
    json.NewEncoder(w).Encode(product)
}
```

### Tracing Setup

- Development environment MUST use `opentracing.NoopTracer{}` (no external dependencies)
- Tests MUST use `opentracing.NoopTracer{}` by default
- Tests that verify tracing instrumentation MAY use mock tracer to validate spans
- Production MUST configure real tracing backend (deployment-specific: Jaeger, Zipkin, Datadog, etc.)
- Production tracing configuration MUST be loaded from environment variables

### Code Review Requirements

- Pull requests MUST include integration tests for all new endpoints
- Tests MUST use protobuf-generated structs (no `map[string]interface{}`)
- Tests MUST demonstrate edge case coverage
- Reviewers MUST verify table-driven test structure
- Reviewers MUST verify no mocking is used for database or HTTP layers
- Reviewers MUST verify `.proto` files are updated for API changes
- Reviewers MUST verify OpenTracing spans are created for new endpoints
- Reviewers MUST verify GORM is used for database access (no raw SQL unless justified)
- Reviewers MUST verify tests use database truncation for cleanup (defer pattern)
- Reviewers MUST verify truncation handles all tables modified by test
- Reviewers MUST verify truncation uses CASCADE for foreign key dependencies
- Tests MUST be reviewed before implementation code

## Governance

### Amendment Process

1. Constitution changes MUST be proposed in writing with rationale
2. Changes MUST be reviewed by project lead or team
3. Version MUST be incremented per semantic versioning:
   - **MAJOR**: Backward incompatible principle changes (e.g., removing no-mocking rule, allowing map[string]interface{})
   - **MINOR**: New principles added or major expansions (e.g., adding protobuf requirement, adding tracing requirement)
   - **PATCH**: Clarifications, examples, typo fixes
4. All dependent templates and documentation MUST be updated to reflect changes

### Compliance

- All pull requests MUST comply with these principles
- Constitution violations MUST be justified in PR description
- Complexity that violates simplicity principles MUST document "why needed" and "simpler alternatives rejected"
- When in doubt: integration test over unit test, real database over mock, table-driven over individual tests, protobuf structs over maps, Docker test database over local installation

### Version Control

This constitution is version-controlled alongside code and follows the same review process as code changes.

**Version**: 1.5.2 | **Ratified**: 2025-11-14 | **Last Amended**: 2025-11-16
