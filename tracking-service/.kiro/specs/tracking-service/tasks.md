# Implementation Plan: tracking-service

## Overview

Implement the `tracking-service` Go microservice incrementally, starting from project scaffolding and working up through data models, repository, validation, handlers, middleware, and finally integration tests. Each task builds on the previous one so that no code is left orphaned. Property-based tests using `pgregory.net/rapid` are placed immediately after the component they validate.

## Tasks

- [x] 1. Scaffold project structure and Go module
  - Create the Go module (`go mod init`) with module path `tracking-service`
  - Create the top-level directory layout:
    - `cmd/server/` — main entrypoint
    - `internal/config/` — configuration loading
    - `internal/models/` — domain types and HTTP structs
    - `internal/validator/` — pure validation logic
    - `internal/repository/` — Repository interface, PostgreSQL implementation, mock
    - `internal/handler/` — HTTP handlers
    - `internal/middleware/` — CorrelationID, Logging, Recovery
    - `internal/router/` — route registration
    - `migrations/` — SQL migration files
    - `docker/` — Docker Compose for integration tests
  - Add direct dependencies with pinned versions: `github.com/go-chi/chi/v5`, `github.com/jackc/pgx/v5`, `pgregory.net/rapid`
  - Run `go mod tidy` to populate `go.sum`
  - _Requirements: 7.1_

- [x] 2. Define configuration and environment loading
  - [x] 2.1 Implement `Config` struct and `LoadConfig` function in `internal/config/config.go`
    - Fields: `HTTPPort` (default 8080), `DatabaseDSN`, `DBMaxOpenConns` (default 50), `DBMaxIdleConns` (default 10), `DBQueryTimeout` (default 5s)
    - Read values from environment variables; apply defaults when variables are absent
    - Return an error if `DatabaseDSN` is empty
    - _Requirements: 7.2_

- [x] 3. Define domain models and HTTP structs
  - [x] 3.1 Implement `Status` type, constants, and all domain/HTTP structs in `internal/models/models.go`
    - `Status` string type with all 8 constants: `CREATED`, `PICKED_UP`, `ARRIVED_AT_HUB`, `IN_TRANSIT`, `OUT_FOR_DELIVERY`, `DELIVERED`, `FAILED_DELIVERY`, `RETURNED`
    - `TrackingEvent`, `TrackingSummary`, `CreateEventRequest`, `CreateEventResponse`, `HistoryEntry`, `TrackingHistoryResponse`, `ErrorResponse` — match JSON tags exactly as specified in the design
    - `ValidationError` struct with `Field` and `Message` string fields
    - _Requirements: 1.7, 6.1, 6.2, 6.3_

  - [x] 3.2 Write property test for JSON round-trip (Property 10)
    - **Property 10: JSON serialization round-trip**
    - **Validates: Requirements 6.4**
    - Use `rapid` generators to produce random valid `CreateEventRequest` values; serialize → deserialize → serialize → deserialize and assert field equality

- [x] 4. Implement the validator
  - [x] 4.1 Implement `ValidateCreateEventRequest`, `ValidateStatus`, `ValidateTimestamp`, and `ValidateBulkNumbers` in `internal/validator/validator.go`
    - `ValidateStatus`: return true only for the 8 defined Status constants
    - `ValidateTimestamp`: return true only for strings parseable by `time.Parse(time.RFC3339, s)`
    - `ValidateCreateEventRequest`: collect all field-level errors; check presence of `tracking_number`, `status`, `timestamp`; then check status enum and timestamp format
    - `ValidateBulkNumbers`: reject empty slice, slices > 100, and slices containing empty strings
    - _Requirements: 1.2, 1.3, 1.4, 1.5, 3.2, 3.3, 3.4_

  - [x] 4.2 Write property test for missing required fields → HTTP 400 (Property 2)
    - **Property 2: Missing required fields return HTTP 400 identifying the missing field(s)**
    - **Validates: Requirements 1.2, 1.3**
    - Generate requests with random subsets of `tracking_number`, `status`, `timestamp` absent or empty; assert `ValidateCreateEventRequest` returns errors naming every missing field

  - [x] 4.3 Write property test for invalid status → HTTP 422 (Property 3)
    - **Property 3: Invalid status value returns HTTP 422**
    - **Validates: Requirements 1.4**
    - Generate arbitrary strings not in the Status enum; assert `ValidateStatus` returns false for all of them

  - [x] 4.4 Write property test for invalid timestamp → HTTP 422 (Property 4)
    - **Property 4: Invalid timestamp format returns HTTP 422**
    - **Validates: Requirements 1.5**
    - Generate arbitrary non-RFC-3339 strings; assert `ValidateTimestamp` returns false for all of them

  - [x] 4.5 Write property test for bulk count boundary (Property 9)
    - **Property 9: Bulk query count boundary**
    - **Validates: Requirements 3.2, 3.4**
    - Generate slices of 1–100 non-empty strings; assert `ValidateBulkNumbers` returns no errors
    - Generate slices of 101+ non-empty strings; assert `ValidateBulkNumbers` returns an error

- [x] 5. Define the Repository interface and sentinel errors
  - Create `internal/repository/repository.go` with the `Repository` interface, `ErrNotFound`, and `ErrPoolExhausted` sentinel errors
  - Interface methods: `InsertEventAndUpsertSummary`, `GetEventsByTrackingNumber`, `GetSummaryByTrackingNumber`, `GetSummariesByTrackingNumbers`, `Ping`
  - _Requirements: 2.3, 4.1, 5.3, 7.3_

- [x] 6. Implement the in-memory mock repository
  - Create `internal/repository/mock.go` implementing `Repository` with an in-memory map protected by a `sync.RWMutex`
  - `InsertEventAndUpsertSummary`: append event to slice, upsert summary (keep entry with latest timestamp as current)
  - `GetEventsByTrackingNumber`: return events sorted by `Timestamp` ascending; return `ErrNotFound` if none
  - `GetSummaryByTrackingNumber`: return summary or `ErrNotFound`
  - `GetSummariesByTrackingNumbers`: return only summaries that exist, silently omit missing ones
  - `Ping`: always return nil
  - Support an injectable error field to simulate failures for error-path tests
  - _Requirements: 2.3, 3.5, 5.3_

- [x] 7. Write the database migration
  - Create `migrations/001_initial_schema.sql` with the exact DDL from the design:
    - `tracking_events` table with all columns and `gen_random_uuid()` default for `event_id`
    - `idx_tracking_events_tracking_number` index
    - `tracking_summary` table
  - _Requirements: 4.3_

- [x] 8. Implement the PostgreSQL repository
  - [x] 8.1 Create `internal/repository/postgres.go` with `PostgresRepository` struct holding a `*sql.DB`
    - Constructor `NewPostgresRepository(db *sql.DB) *PostgresRepository`
    - `InsertEventAndUpsertSummary`: open transaction, `SELECT ... FOR UPDATE` on `tracking_summary`, insert into `tracking_events`, upsert `tracking_summary`, commit; wrap errors with context; map `context.DeadlineExceeded` to `ErrPoolExhausted` where appropriate
    - `GetEventsByTrackingNumber`: `SELECT` ordered by `timestamp ASC`; return `ErrNotFound` on empty result
    - `GetSummaryByTrackingNumber`: single-row `SELECT`; return `ErrNotFound` on `sql.ErrNoRows`
    - `GetSummariesByTrackingNumbers`: single `SELECT ... WHERE tracking_number = ANY($1)` using `pgx` array binding
    - `Ping`: delegate to `db.PingContext`
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 5.1, 5.2, 5.4, 7.2_

- [x] 9. Implement middleware
  - [x] 9.1 Implement `CorrelationIDMiddleware` in `internal/middleware/correlation.go`
    - Read `X-Correlation-ID` header; if absent or empty, generate a UUID v4
    - Store the correlation ID in `context.Context` under a typed key
    - Set `X-Correlation-ID` on the response header
    - _Requirements: 8.2_

  - [x] 9.2 Implement `LoggingMiddleware` in `internal/middleware/logging.go`
    - Wrap `http.ResponseWriter` to capture the status code
    - At request completion emit one structured log line (JSON or key=value) with: method, path, status code, latency ms, correlation ID
    - _Requirements: 8.3_

  - [x] 9.3 Implement `RecoveryMiddleware` in `internal/middleware/recovery.go`
    - Recover from panics, log the stack trace with the correlation ID, return HTTP 500 with `ErrorResponse`
    - _Requirements: 8.1_

  - [x] 9.4 Write property test for correlation ID uniqueness (Property 13)
    - **Property 13: Every request receives a unique correlation ID**
    - **Validates: Requirements 8.2**
    - Use `httptest.NewServer` backed by the mock repository; generate random valid requests; assert each response has a non-empty, unique `X-Correlation-ID` header and that error responses include `correlation_id` in the body

  - [x] 9.5 Write property test for structured log entry per request (Property 14)
    - **Property 14: Structured log entry emitted for every request**
    - **Validates: Requirements 8.3**
    - Inject a `bytes.Buffer` as the log writer; generate random valid requests; assert exactly one log entry per request containing method, path, status code, latency, and correlation ID

- [x] 10. Implement the HealthHandler
  - Create `internal/handler/health.go` with `HealthHandler` struct
  - `ServeHTTP`: call `repo.Ping` with a 5-second deadline context; return HTTP 200 `{"status":"ok"}` on success; return HTTP 503 `ErrorResponse` on failure
  - _Requirements: 7.4, 7.5_

- [x] 11. Implement the EventHandler
  - [x] 11.1 Create `internal/handler/event.go` with `EventHandler` struct
    - Parse JSON body; return HTTP 400 on malformed JSON
    - Call `ValidateCreateEventRequest`; return HTTP 400 for missing fields, HTTP 422 for invalid status or timestamp
    - Build `TrackingEvent` with a new UUID v4 `event_id`, `created_by_service = "tracking-service"`, and parsed `time.Time` timestamp
    - Call `repo.InsertEventAndUpsertSummary` with a context carrying the configured query timeout
    - Map `ErrPoolExhausted` → 503, `context.DeadlineExceeded` → 504, other errors → 500
    - Return HTTP 201 with `CreateEventResponse{EventID: event.EventID}`
    - Set `Content-Type: application/json` on all responses
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 5.3, 6.2, 7.3, 8.4_

  - [x] 11.2 Write property test for valid event creation round-trip (Property 1)
    - **Property 1: Valid event creation round-trip**
    - **Validates: Requirements 1.1, 1.7, 4.2**
    - Generate random valid `CreateEventRequest` values; POST to test server; assert HTTP 201, non-empty UUID `event_id`; retrieve via mock and assert all fields preserved and `created_by_service = "tracking-service"`

  - [x] 11.3 Write property test for Content-Type on all responses (Property 11)
    - **Property 11: All responses carry Content-Type: application/json**
    - **Validates: Requirements 6.2**
    - Generate random requests (valid and invalid) to all endpoints; assert every response has `Content-Type: application/json` or `application/json; charset=utf-8`

- [x] 12. Implement the HistoryHandler
  - [x] 12.1 Create `internal/handler/history.go` with `HistoryHandler` struct
    - Extract `{tracking_number}` from the URL path parameter
    - Call `repo.GetEventsByTrackingNumber` and `repo.GetSummaryByTrackingNumber` with query timeout context
    - Map `ErrNotFound` → 404 with message `"tracking number '{n}' not found"`
    - Build `TrackingHistoryResponse`: populate `history` from events (sorted ascending), `current_status` and `estimated_delivery` from summary
    - Return HTTP 200 with JSON body; set `Content-Type: application/json`
    - _Requirements: 2.1, 2.2, 2.3, 2.5, 2.6, 6.2, 6.3_

  - [x] 12.2 Write property test for summary projection correctness (Property 5)
    - **Property 5: Summary projection reflects the most recent event**
    - **Validates: Requirements 1.6, 2.1, 2.2, 2.5, 5.2**
    - Generate random sequences of events for the same tracking number with varying timestamps; POST all events; GET history; assert `current_status` equals the status of the event with the latest timestamp and `history` length equals event count ordered ascending

  - [x] 12.3 Write property test for non-existent tracking number → 404 (Property 6)
    - **Property 6: Non-existent tracking number returns HTTP 404**
    - **Validates: Requirements 2.3**
    - Generate random tracking number strings never inserted into the mock; assert GET returns HTTP 404

  - [x] 12.4 Write property test for estimated_delivery serialization (Property 7)
    - **Property 7: estimated_delivery serialization**
    - **Validates: Requirements 2.6**
    - Generate summaries with and without `estimated_delivery` set; assert response includes a valid RFC 3339 string when set and `null` when not set

  - [x] 12.5 Write property test for RFC 3339 timestamps in responses (Property 12)
    - **Property 12: Timestamp fields in responses are valid RFC 3339 strings**
    - **Validates: Requirements 6.3**
    - Generate random events; POST them; GET history; parse all `timestamp` and `updated_at` fields in the response and assert each is a valid RFC 3339 string

- [x] 13. Implement the BulkHandler
  - [x] 13.1 Create `internal/handler/bulk.go` with `BulkHandler` struct
    - Parse `numbers` query parameter (comma-separated); return HTTP 400 if absent or empty
    - Call `ValidateBulkNumbers`; return HTTP 422 if > 100 numbers
    - Call `repo.GetSummariesByTrackingNumbers` with query timeout context
    - Return HTTP 200 with JSON array of `TrackingSummary` objects (only existing ones); set `Content-Type: application/json`
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 6.2_

  - [x] 13.2 Write property test for bulk returns only existing numbers (Property 8)
    - **Property 8: Bulk query returns exactly the existing tracking numbers**
    - **Validates: Requirements 3.1, 3.5**
    - Generate mixed sets of existing and non-existing tracking numbers; assert response array contains exactly one entry per existing number and no entries for non-existing ones

- [x] 14. Set up the router and wire all components
  - Create `internal/router/router.go`
  - Register routes using `chi`:
    - `POST /tracking/events` → `EventHandler`
    - `GET /tracking/{tracking_number}` → `HistoryHandler`
    - `GET /tracking/bulk` → `BulkHandler`
    - `GET /health` → `HealthHandler`
  - Apply middleware chain in order: `CorrelationIDMiddleware`, `LoggingMiddleware`, `RecoveryMiddleware`
  - _Requirements: 7.1_

- [x] 15. Implement the main entrypoint
  - Create `cmd/server/main.go`
  - Call `config.LoadConfig()`; exit with error if config is invalid
  - Open `*sql.DB` using `pgx` stdlib driver with `DBMaxOpenConns` and `DBMaxIdleConns` applied
  - Run database migrations from `migrations/` on startup (or log a warning if migration runner is not wired)
  - Construct `PostgresRepository`, all handlers, and the router
  - Start `http.Server` on the configured port with graceful shutdown on `SIGINT`/`SIGTERM`
  - _Requirements: 7.1, 7.2_

- [x] 16. Checkpoint — unit tests passing
  - Ensure all unit and property-based tests pass with `go test ./...`
  - Fix any compilation errors or test failures before proceeding
  - Ask the user if any questions arise.

- [x] 17. Set up Docker Compose for integration tests
  - Create `docker/docker-compose.yml` with a `postgres:16-alpine` service, health check, and environment variables for `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`
  - Create `internal/repository/integration_test.go` with a `TestMain` that:
    - Skips all tests if `INTEGRATION=true` env var is not set
    - Connects to the PostgreSQL container, applies `migrations/001_initial_schema.sql`, and tears down after the suite
  - _Requirements: 4.3_

- [x] 18. Write integration tests against PostgreSQL
  - [x] 18.1 Full `POST → GET` flow for a single tracking number
    - POST a valid event; GET history; assert HTTP 200, correct `current_status`, and `history` length 1
    - _Requirements: 1.1, 2.1, 2.2_

  - [x] 18.2 Full `POST → GET /bulk` flow for multiple tracking numbers
    - POST events for 3 tracking numbers; GET bulk; assert all 3 appear in the response
    - _Requirements: 3.1, 3.6_

  - [x] 18.3 Concurrent POST requests for the same tracking number
    - Launch 10 goroutines each posting a distinct event for the same tracking number; assert all 10 events are persisted and `current_status` reflects the latest timestamp
    - _Requirements: 1.8, 5.4_

  - [x] 18.4 Transaction rollback on injected summary failure
    - Inject a fault that causes the summary upsert to fail; assert the event is not persisted (transaction rolled back) and HTTP 500 is returned
    - _Requirements: 5.3_

  - [x] 18.5 Connection pool exhaustion returns HTTP 503
    - Set `DBMaxOpenConns=1`, hold a long-running transaction, fire a second request; assert HTTP 503
    - _Requirements: 7.3_

- [x] 19. Final checkpoint — all tests passing
  - Run `go test ./...` (unit + property tests) and the integration suite with `INTEGRATION=true go test ./internal/repository/...`
  - Ensure `go vet ./...` and `go build ./...` succeed with no errors
  - Ask the user if any questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for a faster MVP
- Each task references specific requirements for traceability
- Property tests use `pgregory.net/rapid` and are tagged with `Feature: tracking-service, Property N: ...`
- Unit tests use the in-memory mock repository; integration tests require the Docker Compose PostgreSQL container (`INTEGRATION=true`)
- Checkpoints at tasks 16 and 19 ensure incremental validation before moving to the next phase
