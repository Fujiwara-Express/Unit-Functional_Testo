# Unit & Functional Tests — Fujiwara Express

This repo contains multiple independently runnable Go services. Each service has its own test suite split into **unit tests** (no external dependencies) and **functional tests** (require a real database or Docker).

---

## Services Overview

| Service | Unit Tests | Functional Tests | Notes |
|---|---|---|---|
| `delivery-service` | ✅ `service/` | ✅ `functional/` | Requires PostgreSQL |
| `notification-service` | ✅ `test/unit/` | ✅ `test/functional/` | Uses Testcontainers (Docker) |
| `payment-service` | ✅ `service/` | ✅ `functional/` | Requires PostgreSQL |
| `order-service` | ✅ `service/`, `handler/`, `repository/`, `tracking/` | ✅ `functional/` | No DB needed (in-memory fake) |
| `pickup-service` | ✅ `test/unit/` | ✅ `test/functional/` | Uses Testcontainers (Docker) |
| `report-and-analytics` | ✅ `test/unit/` | ✅ `test/functional/` | Uses Testcontainers (Docker) |
| `tracking-service` | ✅ scattered in `internal/` | ✅ `internal/handler/functional_test.go` | No DB needed (in-memory mock); integration tests need `INTEGRATION=true` |
| `routing-service` | ✅ scattered in `internal/` | ✅ `scripts/functional_tests.sh` | Shell script against a live server |
| `user-service` | ✅ scattered in `internal/` | ✅ `scripts/functional_test.sh` | Shell script against a live server |
| `pricing-service` | ✅ `internal/service/` | ⚠️ `tests/functional/` | Functional test is a stub/placeholder |
| `warehouse-service` | ❌ none | ❌ none | No tests found |
| `api-gateway` | ❌ none | ❌ none | No tests found |

---

## Delivery Service

### Unit Tests

Unit tests live in `delivery-service/service/` and use `go.uber.org/mock` — no database or external services required.

**What's covered:**
- `assign_test.go` — courier assignment, routing client errors, missing fields, repository errors
- `courier_test.go` — register/update/list couriers, validation, not-found, repository errors
- `query_test.go` — get delivery detail, get courier jobs
- `status_test.go` — update delivery status, invalid transitions

```bash
cd delivery-service
go test ./service/...
```

Verbose:

```bash
go test -v ./service/...
```

---

### Functional Tests

Functional tests live in `delivery-service/functional/` and require a running PostgreSQL instance. They use the `functional` build tag and are skipped unless the tag is provided.

**Prerequisites:**
- PostgreSQL running with a `delivery_test` database
- `couriers` and `delivery_jobs` tables created (see `database.sql`)

**Environment variables:**

| Variable | Default |
|---|---|
| `TEST_DB_HOST` | `localhost` |
| `TEST_DB_PORT` | `5432` |
| `TEST_DB_USER` | `postgres` |
| `TEST_DB_PASSWORD` | *(empty)* |
| `TEST_DB_NAME` | `delivery_test` |

**Run (Linux/macOS):**

```bash
cd delivery-service
go test -v -tags functional ./functional/...
```

With custom credentials:

```bash
TEST_DB_USER=postgres TEST_DB_PASSWORD=secret go test -v -tags functional ./functional/...
```

**Run (Windows CMD):**

```cmd
cd delivery-service
set TEST_DB_HOST=localhost
set TEST_DB_PORT=5432
set TEST_DB_USER=postgres
set TEST_DB_PASSWORD=secret
set TEST_DB_NAME=delivery_test
go test -v -tags functional ./functional/...
```

**Run (Windows PowerShell):**

```powershell
cd delivery-service
$env:TEST_DB_HOST = "localhost"
$env:TEST_DB_PORT = "5432"
$env:TEST_DB_USER = "postgres"
$env:TEST_DB_PASSWORD = "secret"
$env:TEST_DB_NAME = "delivery_test"
go test -v -tags functional ./functional/...
```

---

### Run All Delivery Service Tests

```bash
cd delivery-service
go test -v -tags functional ./...
```

> Requires PostgreSQL to be available for functional tests to pass.

---

## Notification Service

### Unit Tests

Unit tests live in `notification-service/test/unit/` and use `go.uber.org/mock` and `go-sqlmock` — no database or external services required.

**What's covered:**
- `test/unit/service/` — SendNotification, CreateTemplate, UpdateTemplate, ListTemplates
- `test/unit/handler/` — HTTP handler request/response validation
- `test/unit/repository/` — repository layer with `go-sqlmock`
- `test/unit/client/` — Firebase, SendGrid, WhatsApp client logic
- `test/unit/domain/` — domain model and template rendering

```bash
cd notification-service
go test ./test/unit/... -count=1 -timeout 30s
```

Verbose with coverage:

```bash
go test -v ./test/unit/... -count=1 -timeout 30s -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Via Makefile:

```bash
make test-unit
```

---

### Functional Tests

Functional tests live in `notification-service/test/functional/` and use **Testcontainers** to spin up a PostgreSQL container automatically — no manual database setup required. Docker must be running.

**Prerequisites:**
- Docker running (Testcontainers pulls `postgres:15-alpine` automatically)
- No environment variables needed — the container is configured in `main_test.go`

**What's covered:**
- `send_notification_test.go` — PUSH/EMAIL/WHATSAPP round-trips, provider failure, auth, missing fields
- `create_template_test.go` — template creation, validation, unauthorized access
- `list_templates_test.go` — list templates
- `lifecycle_test.go` — full create → send → verify DB log flows for all channels
- `kafka_handler_test.go` — Kafka handler with valid events, malformed JSON, template not found

**Run:**

```bash
cd notification-service
go test ./test/functional/... -count=1 -timeout 120s -v
```

Via Makefile:

```bash
make test-functional
```

> The first run may take longer while Docker pulls the `postgres:15-alpine` image.

---

### Run All Notification Service Tests

```bash
cd notification-service
go test ./test/... -count=1 -timeout 120s -v
```

---

## Payment Service

### Unit Tests

Unit tests live in `payment-service/service/` and use `go.uber.org/mock` — no database or external services required.

**What's covered:**
- `charge_test.go` — TRANSFER/VIRTUAL_ACCOUNT/QRIS/COD charge flows, gateway errors, duplicate order, unsupported method, missing fields, validation
- `callback_test.go` — SUCCESS/FAILED/EXPIRED callback handling, invalid signature, idempotency, missing fields
- `refund_test.go` — refund flow, payment not found, already-refunded, repository errors
- `cod_confirm_test.go` — COD confirmation, duplicate confirmation, repository errors
- `query_test.go` — get payment by ID, get payment by order ID, not found
- `signature_test.go` — HMAC-SHA512 signature compute, validate, determinism, round-trip

```bash
cd payment-service
go test ./service/...
```

Verbose:

```bash
go test -v ./service/...
```

Run a specific test by name:

```bash
go test -v ./service/... -run TestCharge
```

---

### Functional Tests

Functional tests live in `payment-service/functional/` and require a running PostgreSQL instance. They use the `functional` build tag and are skipped unless the tag is provided.

**Prerequisites:**
- PostgreSQL running with a `payment_test` database
- `payments` and `cod_collections` tables created — run `database.sql` first:

```bash
psql -U postgres -f payment-service/database.sql
```

**Environment variables:**

| Variable | Default |
|---|---|
| `TEST_DB_HOST` | `localhost` |
| `TEST_DB_PORT` | `5432` |
| `TEST_DB_USER` | `postgres` |
| `TEST_DB_PASSWORD` | *(empty)* |
| `TEST_DB_NAME` | `payment_test` |

**What's covered:**
- `charge_functional_test.go` — TRANSFER/COD charge persists correct row, duplicate order_id returns 409
- `callback_functional_test.go` — SUCCESS/FAILED callbacks update payment status, idempotent duplicate callback
- `refund_functional_test.go` — refund updates status to REFUNDED
- `cod_confirm_functional_test.go` — COD confirmation persists cod_collections row

**Run (Linux/macOS):**

```bash
cd payment-service
go test -v -tags functional ./functional/...
```

With custom credentials:

```bash
TEST_DB_USER=postgres TEST_DB_PASSWORD=secret go test -v -tags functional ./functional/...
```

**Run (Windows CMD):**

```cmd
cd payment-service
set TEST_DB_HOST=localhost
set TEST_DB_PORT=5432
set TEST_DB_USER=postgres
set TEST_DB_PASSWORD=secret
set TEST_DB_NAME=payment_test
go test -v -tags functional ./functional/...
```

**Run (Windows PowerShell):**

```powershell
cd payment-service
$env:TEST_DB_HOST = "localhost"
$env:TEST_DB_PORT = "5432"
$env:TEST_DB_USER = "postgres"
$env:TEST_DB_PASSWORD = "secret"
$env:TEST_DB_NAME = "payment_test"
go test -v -tags functional ./functional/...
```

---

### Run All Payment Service Tests

```bash
cd payment-service
go test ./service/... && go test -v -tags functional ./functional/...
```

> Requires PostgreSQL to be available for functional tests to pass.

---

## Order Service

### Unit Tests

Unit tests are spread across `handler/`, `service/`, `repository/`, and `tracking/` packages. They use `go-sqlmock` and `pgregory.net/rapid` for property-based tests — no database required.

**What's covered:**
- `service/` — create/get/list/update/cancel order (table-driven + property-based)
- `handler/` — HTTP handler request/response for all five endpoints
- `repository/order_repository_test.go` — repository layer with `go-sqlmock`
- `tracking/generator_test.go` — tracking number generation

```bash
cd order-service
go test ./service/... ./handler/... ./repository/... ./tracking/...
```

Verbose:

```bash
go test -v ./service/... ./handler/... ./repository/... ./tracking/...
```

---

### Functional Tests

Functional tests live in `order-service/functional/` and use an **in-memory fake repository** — no database or Docker required. They wire the real handler and real service together end-to-end.

**What's covered:**
- `functional_test.go` — full lifecycle (create → get → list → update → cancel), all service types, insurance surcharge, COD validation, error paths

```bash
cd order-service
go test -v ./functional/...
```

---

## Pickup Service

### Unit Tests

Unit tests live in `pickup-service/test/unit/` and use `go.uber.org/mock` and `go-sqlmock` — no database or external services required.

**What's covered:**
- `test/unit/service/` — pickup service business logic
- `test/unit/handler/http/` — HTTP handler and middleware
- `test/unit/handler/kafka/` — Kafka handler
- `test/unit/repository/` — repository layer with `go-sqlmock`
- `test/unit/client/` — delivery, notification, and tracking client logic
- `test/unit/domain/` — domain model validation

```bash
cd pickup-service
go test ./test/unit/... -count=1 -timeout 30s
```

Via Makefile:

```bash
make test-unit
```

---

### Functional Tests

Functional tests live in `pickup-service/test/functional/` and use **Testcontainers** to spin up a PostgreSQL container automatically. Docker must be running.

**What's covered:**
- `request_pickup_test.go`, `assign_courier_test.go`, `update_status_test.go`, `cancel_pickup_test.go`, `get_pickup_test.go`, `list_pickups_test.go` — full CRUD flows
- `kafka_handler_test.go` — Kafka order-created handler
- `lifecycle_test.go` — end-to-end pickup lifecycle

```bash
cd pickup-service
go test ./test/functional/... -count=1 -timeout 120s -v
```

Via Makefile:

```bash
make test-functional
```

---

## Report & Analytics Service

### Unit Tests

Unit tests live in `report-and-analytics/test/unit/` and use `go.uber.org/mock` — no database required.

**What's covered:**
- `test/unit/service/report_service_test.go` — report service logic
- `test/unit/handler/http/report_handler_test.go` — HTTP handler

```bash
cd report-and-analytics
go test ./test/unit/... -count=1 -timeout 30s
```

Via Makefile:

```bash
make test-unit
```

---

### Functional Tests

Functional tests live in `report-and-analytics/test/functional/` and use **Testcontainers** to spin up a PostgreSQL container automatically. Docker must be running.

**What's covered:**
- `order_report_test.go`, `revenue_report_test.go`, `delivery_performance_test.go`, `hub_performance_test.go` — all four report endpoints with seeded data

```bash
cd report-and-analytics
go test ./test/functional/... -count=1 -timeout 120s -v
```

Via Makefile:

```bash
make test-functional
```

---

## Tracking Service

### Unit Tests

Unit tests are scattered throughout `internal/` alongside the source files. They use an in-memory `MockRepository` — no database required.

**What's covered:**
- `internal/handler/` — event, history, bulk, health handlers + property-based tests
- `internal/middleware/` — correlation ID, logging, recovery middleware
- `internal/models/` — model validation
- `internal/validator/` — request validator
- `internal/config/` — config loading
- `internal/repository/integration_test.go` — skipped by default (see below)

```bash
cd tracking-service
go test ./...
```

---

### Functional Tests

Functional tests live in `internal/handler/functional_test.go` and use the in-memory `MockRepository` — **no database or Docker required**. They exercise the full HTTP stack (all middleware + all routes) end-to-end.

**What's covered:**
- POST/GET events, history, bulk lookup, health endpoint
- Middleware: correlation ID propagation, panic recovery
- Error paths: 400/404/422/500/503

```bash
cd tracking-service
go test -v -run TestFunctional ./internal/handler/...
```

Or run everything together:

```bash
go test -v ./...
```

---

### Integration Tests (optional)

Repository integration tests in `internal/repository/` require a real PostgreSQL instance and are **skipped by default**. Enable them with the `INTEGRATION=true` env var.

```bash
cd tracking-service
INTEGRATION=true TEST_DATABASE_DSN="postgres://tracking:tracking_secret@localhost:5432/tracking_test" go test ./internal/repository/...
```

---

## Routing Service

### Unit Tests

Unit tests are scattered throughout `internal/` alongside the source files. They use `pgregory.net/rapid` for property-based tests — no database or Redis required.

**What's covered:**
- `internal/calculators/` — Dijkstra, Haversine, courier route, inter-hub calculations
- `internal/handlers/` — route, nodes, edges handlers
- `internal/middleware/` — error handler
- `internal/services/` — cache key generation and cache logic
- `internal/apperrors/` — error types

```bash
cd routing-service
go test ./...
```

---

### Functional Tests

Functional tests are a **shell script** (`scripts/functional_tests.sh`) that runs HTTP requests against a live server instance. The server must be started manually first.

**Prerequisites:**
- Routing service running (with PostgreSQL and Redis configured)
- `curl` available in PATH

**Run:**

```bash
cd routing-service
go run ./cmd/server   # in a separate terminal

BASE_URL=http://localhost:8080 bash scripts/functional_tests.sh
```

**What's covered:** 12 test cases — GET nodes/edges, POST nodes (valid, duplicate, missing fields), GET route (valid, no path, missing params), POST edges, PATCH edges, cache hit latency.

---

## User Service

### Unit Tests

Unit tests are scattered throughout `internal/` alongside the source files. They use `pgregory.net/rapid` for property-based tests — no database required.

**What's covered:**
- `internal/service/auth/auth_service_test.go` — login, refresh, logout
- `internal/service/user/user_service_test.go` — register, update, get profile
- `internal/token/jwt_test.go` — JWT generation and validation
- `internal/domain/validation_test.go` — domain validation rules
- `internal/transport/http/handlers_test.go` — HTTP handler layer

```bash
cd user-service
go test ./...
```

---

### Functional Tests

Functional tests are a **shell script** (`scripts/functional_test.sh`) that runs HTTP requests against a live server instance. The server must be started manually first.

**Prerequisites:**
- User service running (with PostgreSQL configured)
- `curl` available in PATH

**Run:**

```bash
cd user-service
go run ./cmd/main.go   # in a separate terminal

BASE_URL=http://localhost:8080 bash scripts/functional_test.sh
```

**What's covered:** 12 test cases — register (valid, duplicate, invalid email), login (valid, wrong password), get/update profile, token refresh, logout, admin endpoint authorization.

---

## Pricing Service

### Unit Tests

Unit tests live in `internal/service/pricing_service_test.go` and use `go.uber.org/mock` — no database required.

```bash
cd pricing-service
go test ./internal/service/...
```

---

### Functional Tests

> ⚠️ The functional test in `tests/functional/pricing_api_test.go` is a **placeholder/stub** — the router and database wiring are commented out. It will always fail. No runnable functional tests exist yet for this service.

---

## Warehouse Service

> ❌ No tests found. The service has no `*_test.go` files.

---

## API Gateway

> ❌ No tests found. The gateway has no `*_test.go` files.

---

## Quick Reference

```bash
# Delivery — unit only
cd delivery-service && go test ./service/...

# Delivery — functional (requires PostgreSQL)
cd delivery-service && go test -v -tags functional ./functional/...

# Notification — unit only
cd notification-service && go test ./test/unit/... -count=1 -timeout 30s

# Notification — functional (requires Docker)
cd notification-service && go test ./test/functional/... -count=1 -timeout 120s -v

# Payment — unit only
cd payment-service && go test ./service/...

# Payment — functional (requires PostgreSQL)
cd payment-service && go test -v -tags functional ./functional/...

# Order — unit only
cd order-service && go test ./service/... ./handler/... ./repository/... ./tracking/...

# Order — functional (no DB needed)
cd order-service && go test -v ./functional/...

# Pickup — unit only
cd pickup-service && go test ./test/unit/... -count=1 -timeout 30s

# Pickup — functional (requires Docker)
cd pickup-service && go test ./test/functional/... -count=1 -timeout 120s -v

# Report & Analytics — unit only
cd report-and-analytics && go test ./test/unit/... -count=1 -timeout 30s

# Report & Analytics — functional (requires Docker)
cd report-and-analytics && go test ./test/functional/... -count=1 -timeout 120s -v

# Tracking — unit + functional (no DB needed)
cd tracking-service && go test ./...

# Tracking — integration tests (requires PostgreSQL)
cd tracking-service && INTEGRATION=true TEST_DATABASE_DSN="postgres://tracking:tracking_secret@localhost:5432/tracking_test" go test ./internal/repository/...

# Routing — unit only
cd routing-service && go test ./...

# Routing — functional (requires running server)
cd routing-service && BASE_URL=http://localhost:8080 bash scripts/functional_tests.sh

# User — unit only
cd user-service && go test ./...

# User — functional (requires running server)
cd user-service && BASE_URL=http://localhost:8080 bash scripts/functional_test.sh

# Pricing — unit only
cd pricing-service && go test ./internal/service/...
```
