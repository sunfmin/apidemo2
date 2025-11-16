<!--
Sync Impact Report:
- Version: 1.1.1 → 1.2.0 (MINOR bump - new principle added + technology stack restrictions)
- Project name: apidemo1 → apidemo2
- Modified principles:
  - None renamed
- Added principles:
  - VII. Distributed Tracing (OpenTracing) - NEW observability requirement
- Technology Stack changes:
  - Database Access: Now MANDATES GORM (previously allowed database/sql+pgx, sqlx, GORM, or sqlc)
  - HTTP Framework: Now MANDATES standard net/http (previously allowed Chi, Echo, Gin, or net/http)
  - Test Database: Now MANDATES Docker PostgreSQL container (previously allowed Docker or dedicated instance)
- Removed principles: None
- Templates requiring updates:
  ✅ .specify/templates/tasks-template.md (Add OpenTracing setup, GORM setup, Docker test DB tasks)
  ✅ .specify/templates/plan-template.md (Update Technical Context to reflect GORM, net/http, Docker)
  ✅ .specify/templates/spec-template.md (Add tracing validation to edge cases)
- Rationale:
  - OpenTracing provides distributed tracing for debugging and monitoring API calls across services
  - GORM simplifies database operations while maintaining type safety and migration support
  - Standard net/http reduces external dependencies and leverages Go's robust standard library
  - Docker containers ensure consistent, isolated test environments across all development machines and CI/CD
- Impact:
  - All new API endpoints MUST instrument OpenTracing spans
  - All database access MUST use GORM
  - All HTTP routing MUST use standard net/http (no external routers)
  - All test suites MUST use Docker PostgreSQL containers
- Dependencies:
  - Requires github.com/opentracing/opentracing-go
  - Requires gorm.io/gorm and gorm.io/driver/postgres
  - Docker Engine or Docker Desktop for test database
-->

# apidemo2 Constitution

## Core Principles

### I. Integration Testing First (No Mocking)

All tests MUST be integration tests that interact with real dependencies:
- Tests MUST use real PostgreSQL database connections
- NO mocking of database calls, HTTP clients, or external services
- Tests MUST prepare fixture data directly in the database
- Test database MUST be isolated per test run
- Each test MUST clean up its own data or use transactions that rollback

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
- Fixture data MUST be inserted using SQL or ORM calls to real test database
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

All API endpoints MUST be instrumented with distributed tracing:
- Each HTTP endpoint handler MUST create or continue an OpenTracing span
- Spans MUST include operation name matching the endpoint (e.g., "POST /api/products")
- Database operations MUST be traced as child spans with operation details
- External service calls MUST propagate trace context
- Error conditions MUST be logged to the active span with `span.SetTag("error", true)`
- Trace context MUST be extracted from incoming HTTP headers (e.g., `X-B3-TraceId`)
- Trace context MUST be injected into outgoing HTTP requests
- Spans MUST include relevant tags: `http.method`, `http.url`, `http.status_code`
- Tests MUST verify tracing instrumentation (e.g., using mock tracer or test spans)

**Rationale**: Distributed tracing provides critical observability for debugging latency issues, understanding request flows across services, identifying bottlenecks, and correlating logs across distributed systems. OpenTracing offers a vendor-neutral API compatible with Jaeger, Zipkin, and other tracing backends.

**Example**:
```go
import (
    "net/http"
    "github.com/opentracing/opentracing-go"
    "github.com/opentracing/opentracing-go/ext"
)

func ProductCreateHandler(w http.ResponseWriter, r *http.Request) {
    // Extract or start trace span
    spanCtx, _ := opentracing.GlobalTracer().Extract(
        opentracing.HTTPHeaders,
        opentracing.HTTPHeadersCarrier(r.Header),
    )
    span := opentracing.StartSpan("POST /api/products", ext.RPCServerOption(spanCtx))
    defer span.Finish()
    
    // Add tags
    span.SetTag("http.method", r.Method)
    span.SetTag("http.url", r.URL.String())
    
    // Database operation as child span
    dbSpan := opentracing.StartSpan("db.insert_product", opentracing.ChildOf(span.Context()))
    // ... database work ...
    dbSpan.Finish()
    
    // Set response status
    span.SetTag("http.status_code", http.StatusCreated)
}
```

## Technology Stack

- **Language**: Go 1.21+ (recommend latest stable)
- **Database**: PostgreSQL 15+ (with JSONB support)
- **HTTP Framework**: Standard library `net/http` (using `http.ServeMux` or custom routing)
- **Database Access**: GORM (gorm.io/gorm with gorm.io/driver/postgres)
- **Distributed Tracing**: OpenTracing (github.com/opentracing/opentracing-go)
- **Protocol Buffers**: protoc compiler, protoc-gen-go, protoc-gen-go-grpc
- **Validation**: protoc-gen-validate for protobuf field validation
- **Testing**: Standard library `testing` package with `httptest`
- **Test Comparison**: google/go-cmp with protocmp for protobuf message assertions
- **Test Database**: Docker PostgreSQL container (official postgres:15+ image)
- **Migration Tool**: golang-migrate, goose, or GORM AutoMigrate

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

1. **Define Schema**: Create or update `.proto` files in `api/` or `proto/` directory
2. **Generate Code**: Run `make proto` or `go generate` to create Go structs
3. **Use in Code**: Import generated packages, use typed structs throughout
4. **Validate**: Use protoc-gen-validate for automatic field validation
5. **Version**: Use protobuf field numbers consistently (never reuse deleted field numbers)

### Test Database Management

- Each developer MUST have Docker Engine or Docker Desktop installed
- Test suite MUST use Docker PostgreSQL container (official postgres:15+ image)
- Test suite MUST start container before tests and clean up after tests
- Container MUST use unique port or container name per test run to avoid conflicts
- Test database schema MUST match production schema via GORM migrations or SQL migrations
- Database connection strings MUST be configurable via environment variables
- CI/CD MUST provision ephemeral Docker PostgreSQL containers

**Example Docker Test Setup**:
```go
// Start PostgreSQL container for tests
func setupTestDB(t *testing.T) *gorm.DB {
    // docker run -d -p 5433:5432 -e POSTGRES_PASSWORD=test postgres:15
    // Connect using GORM
    dsn := "host=localhost user=postgres password=test dbname=postgres port=5433 sslmode=disable"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }
    
    // Run migrations
    db.AutoMigrate(&Product{}, &Order{})
    
    return db
}
```

### Tracing Setup

- Development environment MUST configure OpenTracing global tracer (Jaeger, Zipkin, or NoopTracer)
- Tests MUST use `opentracing.NoopTracer{}` or mock tracer to avoid external dependencies
- Production MUST configure real tracing backend (Jaeger, Zipkin, Datadog, etc.)
- Tracing configuration MUST be loaded from environment variables

### Code Review Requirements

- Pull requests MUST include integration tests for all new endpoints
- Tests MUST use protobuf-generated structs (no `map[string]interface{}`)
- Tests MUST demonstrate edge case coverage
- Reviewers MUST verify table-driven test structure
- Reviewers MUST verify no mocking is used for database or HTTP layers
- Reviewers MUST verify `.proto` files are updated for API changes
- Reviewers MUST verify OpenTracing spans are created for new endpoints
- Reviewers MUST verify GORM is used for database access (no raw SQL unless justified)
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

**Version**: 1.2.0 | **Ratified**: 2025-11-14 | **Last Amended**: 2025-11-16
