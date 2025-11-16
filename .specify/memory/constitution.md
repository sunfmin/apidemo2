<!--
Sync Impact Report:
- Version: NEW → 1.0.0 (initial constitution)
- New sections: All sections created from template
- Principles established:
  1. Integration Testing First (No Mocking)
  2. Table-Driven Test Design
  3. Edge Case Coverage (NON-NEGOTIABLE)
  4. Real Database Fixtures
  5. ServeHTTP Endpoint Testing
- Templates requiring updates: ✅ Will validate after constitution update
- Rationale: Initial constitution for Go API backend with PostgreSQL integration testing philosophy
-->

# apidemo1 Constitution

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

## Technology Stack

- **Language**: Go (version TBD - recommend Go 1.21+)
- **Database**: PostgreSQL (version TBD - recommend PostgreSQL 15+)
- **HTTP Framework**: Standard library `net/http` or framework TBD (e.g., Chi, Echo, Gin)
- **Database Access**: Library TBD (e.g., `database/sql` + `pgx`, GORM, sqlc)
- **Testing**: Standard library `testing` package with `httptest`
- **Test Database**: Docker PostgreSQL container or dedicated test instance
- **Migration Tool**: TBD (e.g., golang-migrate, goose, or embedded migrations)

## Development Workflow

### Test-First Development (TDD)

1. **Design Phase**: Design API contract (HTTP endpoint, request/response schemas)
2. **Write Tests**: Create table-driven integration tests that cover happy path + edge cases
3. **Verify Failure**: Run tests to confirm they fail (red phase)
4. **Review Tests**: Review test design with team/lead before implementation
5. **Implement**: Write minimal code to make tests pass (green phase)
6. **Refactor**: Improve code quality while keeping tests green
7. **No Implementation Before Tests**: Code written before test approval MUST be discarded

### Test Database Management

- Each developer MUST have local PostgreSQL instance or Docker container
- Test suite MUST create/drop test database or use transactions with rollback
- CI/CD MUST provision ephemeral test databases
- Test database schema MUST match production schema via migrations
- Database connection strings MUST be configurable via environment variables

### Code Review Requirements

- Pull requests MUST include integration tests for all new endpoints
- Tests MUST demonstrate edge case coverage
- Reviewers MUST verify table-driven test structure
- Reviewers MUST verify no mocking is used for database or HTTP layers
- Tests MUST be reviewed before implementation code

## Governance

### Amendment Process

1. Constitution changes MUST be proposed in writing with rationale
2. Changes MUST be reviewed by project lead or team
3. Version MUST be incremented per semantic versioning:
   - **MAJOR**: Backward incompatible principle changes (e.g., removing no-mocking rule)
   - **MINOR**: New principles added or major expansions
   - **PATCH**: Clarifications, examples, typo fixes
4. All dependent templates and documentation MUST be updated to reflect changes

### Compliance

- All pull requests MUST comply with these principles
- Constitution violations MUST be justified in PR description
- Complexity that violates simplicity principles MUST document "why needed" and "simpler alternatives rejected"
- When in doubt, integration test over unit test, real database over mock, table-driven over individual tests

### Version Control

This constitution is version-controlled alongside code and follows the same review process as code changes.

**Version**: 1.0.0 | **Ratified**: 2025-11-14 | **Last Amended**: 2025-11-14
