---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Integration tests are MANDATORY per constitution. All tests use real PostgreSQL database (no mocking), follow table-driven patterns, and cover comprehensive edge cases.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: Root level for `main.go`, packages in subdirectories, `*_test.go` files alongside source
- **Test organization**: Integration tests in `*_test.go` files (no separate `tests/` directory per Go convention)
- **Test database**: Use real PostgreSQL, configure via environment variables
- Paths shown below assume Go project structure - adjust based on plan.md

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Initialize Go module with `go mod init`
- [ ] T002 [P] Setup PostgreSQL connection and database configuration
- [ ] T003 [P] Configure environment variables for database URLs
- [ ] T004 [P] Setup database migration framework (golang-migrate, goose, or embedded)
- [ ] T005 [P] Configure linting (golangci-lint) and formatting (gofmt, goimports)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T006 Create initial database migrations for core tables
- [ ] T007 [P] Setup HTTP router (Chi/Echo/Gin or net/http ServeMux)
- [ ] T008 [P] Implement middleware: logging, recovery, CORS
- [ ] T009 [P] Create database connection pool and health check
- [ ] T010 [P] Setup test database helper (create/teardown or transaction rollback)
- [ ] T011 Implement base error response types and JSON marshaling
- [ ] T012 [P] Create fixture helper utilities for test database population

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Integration Tests for User Story 1 (MANDATORY) ⚠️

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**
> **All tests MUST use real PostgreSQL, table-driven pattern, and cover edge cases**

- [ ] T010 [US1] Integration test for [endpoint] in [package]/[handler]_test.go
  - Happy path test cases
  - Edge cases: input validation, boundary conditions, auth errors
  - Edge cases: data state (404, conflicts), database errors, HTTP specifics
  - Use httptest.ResponseRecorder and real database fixtures
  - Table-driven test structure with test case structs

### Implementation for User Story 1

- [ ] T011 [P] [US1] Create [Entity1] model/struct in [package]/[entity1].go
- [ ] T012 [P] [US1] Create [Entity2] model/struct in [package]/[entity2].go
- [ ] T013 [US1] Implement database repository in [package]/[repository].go (depends on T011, T012)
- [ ] T014 [US1] Implement ServeHTTP handler in [package]/[handler].go
- [ ] T015 [US1] Add request validation and error responses
- [ ] T016 [US1] Add logging and error handling
- [ ] T017 [US1] Create fixture helpers in [package]/fixtures_test.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Integration Tests for User Story 2 (MANDATORY) ⚠️

- [ ] T018 [US2] Integration test for [endpoint] in [package]/[handler]_test.go
  - Table-driven tests with comprehensive edge cases per constitution

### Implementation for User Story 2

- [ ] T019 [P] [US2] Create [Entity] model/struct in [package]/[entity].go
- [ ] T020 [US2] Implement database operations in [package]/[repository].go
- [ ] T021 [US2] Implement ServeHTTP handler in [package]/[handler].go
- [ ] T022 [US2] Integrate with User Story 1 components (if needed)
- [ ] T023 [US2] Create fixture helpers for test data

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Integration Tests for User Story 3 (MANDATORY) ⚠️

- [ ] T024 [US3] Integration test for [endpoint] in [package]/[handler]_test.go
  - Table-driven tests with comprehensive edge cases per constitution

### Implementation for User Story 3

- [ ] T025 [P] [US3] Create [Entity] model/struct in [package]/[entity].go
- [ ] T026 [US3] Implement database operations in [package]/[repository].go
- [ ] T027 [US3] Implement ServeHTTP handler in [package]/[handler].go
- [ ] T028 [US3] Create fixture helpers for test data

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in docs/
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX Performance optimization across all stories
- [ ] TXXX Verify all integration tests pass with real database
- [ ] TXXX Security hardening
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (if tests requested):
Task: "Contract test for [endpoint] in tests/contract/test_[name].py"
Task: "Integration test for [user journey] in tests/integration/test_[name].py"

# Launch all models for User Story 1 together:
Task: "Create [Entity1] model in src/models/[entity1].py"
Task: "Create [Entity2] model in src/models/[entity2].py"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
