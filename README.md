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
| `pricing-service` | ✅ `internal/service/` | ✅ `tests/functional/` | In-memory (no DB); PostgreSQL with `-tags postgres` (Docker) |
| `warehouse-service` | ✅ `internal/service/`, `internal/delivery/http/` | ✅ `functional/` | In-memory (no DB); PostgreSQL with `-tags postgres` (Docker) |
| `api-gateway` | ✅ `gateway_test.go` | ✅ `gateway_functional_test.go` | No DB or Docker needed — real `httptest.Server` stubs |

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

#### In-memory (no DB required)

Functional tests in `tests/functional/pricing_api_test.go` wire the real `PricingHandler`, real `PricingService`, and the built-in `memoryRepository` together end-to-end — **no database or Docker required**. The memory repo has one hardcoded zone (`CGK→BDO/Z1`) and one rate (`Z1/REG`: 10 000/kg, max dims 100×100×100, max weight 50).

**What's covered:**
- Normal package — actual weight dominates, correct total price, all response fields present
- Volumetric weight dominates — `(L×W×H)/6000 > actual weight`
- Minimum weight applied — both actual and volumetric below `min_weight`
- Oversize by length, width, height, and weight — each dimension independently triggers the 50 000 surcharge
- Exactly at limits — no surcharge when package is exactly at max dimensions
- Zone not found — unknown origin/destination returns HTTP 500
- Invalid JSON body — returns HTTP 400
- Wrong HTTP method (GET) — returns HTTP 405
- Full lifecycle — normal, volumetric, and oversize cases in one table-driven test

```bash
cd pricing-service
go test -v ./tests/functional/...
```

---

#### PostgreSQL (requires Docker)

`tests/functional/postgres_functional_test.go` uses **Testcontainers** to spin up a `postgres:15-alpine` container and exercises the real `PostgresRepository` end-to-end. Uses the `postgres` build tag.

**Prerequisites:** Docker running — no environment variables needed.

**What's covered:**
- Normal package, volumetric dominates, oversize by length and weight — same scenarios as in-memory but against a real DB
- Different zone (`CGK→SBY/Z2`) — verifies correct rate lookup per zone
- YES service type — tighter limits (MaxLength 80) and higher rate (20 000/kg)
- YES oversize — 90 cm package exceeds YES MaxLength of 80
- Zone not found and rate not found — HTTP 500 error paths
- Full lifecycle — 5 scenarios in one table-driven test

```bash
cd pricing-service
go test -v -tags postgres ./tests/functional/...
```

> The first run may take longer while Docker pulls the `postgres:15-alpine` image.

---

## Warehouse Service

### Unit Tests

Unit tests are split across the service and handler layers and use `go.uber.org/mock` — no database required.

**What's covered:**
- `internal/service/warehouse_service_test.go` — ReceiveItem (new item, quantity merge, location update, save error), DispatchItem (valid, exact stock, out-of-stock, not found, save error), CheckStock (found, not found, repository error)
- `internal/delivery/http/warehouse_handler_test.go` — all three HTTP handlers: valid requests, invalid JSON, wrong HTTP method, service errors, domain errors (ErrOutOfStock, ErrItemNotFound)

```bash
cd warehouse-service
go test -v ./internal/service/... ./internal/delivery/...
```

---

### Functional Tests

#### In-memory (no DB required)

Functional tests in `functional/functional_test.go` wire the real handler, real service, and the built-in in-memory repository together end-to-end — **no database or Docker required**.

**What's covered:**
- `POST /receive` — new item, stock accumulation, location overwrite, invalid JSON
- `POST /dispatch` — valid dispatch, exact stock, out-of-stock (stock unchanged), item not found, invalid JSON
- `GET /check-stock` — found (correct JSON fields), not found, missing `id` param
- Full lifecycle: receive → check → receive more → dispatch → check → failed dispatch → check
- Multiple independent items tracked separately

```bash
cd warehouse-service
go test -v ./functional/...
```

---

#### PostgreSQL (requires Docker)

`functional/postgres_functional_test.go` uses **Testcontainers** to spin up a `postgres:15-alpine` container and exercises the real `PostgresRepository` end-to-end. Uses the `postgres` build tag.

**Prerequisites:** Docker running — no environment variables needed.

**What's covered:**
- `POST /receive` persists row in DB, upsert accumulates quantity
- `POST /dispatch` updates row in DB, out-of-stock leaves row unchanged
- `GET /check-stock` returns data from DB, not-found returns 404
- Full lifecycle against PostgreSQL

```bash
cd warehouse-service
go test -v -tags postgres ./functional/...
```

> The first run may take longer while Docker pulls the `postgres:15-alpine` image.

---

## API Gateway

The routing logic was extracted from `main()` into a testable `NewGateway(pricingURL, warehouseURL *url.URL) http.Handler` function. `main()` remains a thin entry point that parses the upstream URLs and calls `NewGateway`.

### Unit Tests

Unit tests live in `gateway_test.go` and use `httptest.Server` stub backends — no real services required.

**What's covered:**
- `/pricing/*` prefix routing — path stripping, POST, deep paths, `/pricing` root, isolation from warehouse backend
- `/warehouse/*` prefix routing — receive, check-stock (query string preserved), dispatch, `/warehouse` root, isolation from pricing backend
- Unknown paths — 404 for unrecognised routes, bare `/`, response body message
- Lookalike prefix behaviour — documents that `/pricingextra` matches `/pricing` (HasPrefix semantics)
- Upstream error propagation — 500 and 503 from backends passed through; unreachable backend returns 502
- Response passthrough — body and status code forwarded unchanged
- `NewGateway` independence — two instances with different backends don't interfere

```bash
cd api-gateway
go test -v -run TestRouting ./...
go test -v -run TestUpstream ./...
go test -v -run TestResponse ./...
go test -v -run TestNewGateway ./...
```

Or run all at once:

```bash
cd api-gateway
go test -v ./...
```

---

### Functional Tests

Functional tests live in `gateway_functional_test.go` and spin up **three real `httptest.Server` instances** — one for the gateway, one for each upstream backend. No mocks, no Docker, no external dependencies.

**What's covered:**
- `POST /pricing/calculate-price` — full round-trip, response body and headers passed through
- `POST /pricing/calculate-price` (POST method) — method forwarded correctly
- `POST /warehouse/receive` — request body forwarded intact, 201 returned
- `GET /warehouse/check-stock?id=BRG-001` — query string preserved, JSON response decoded
- `POST /warehouse/dispatch` — path stripped correctly
- Unknown routes — 404 with message body, backends not called
- Upstream 500 and 400 propagated to client
- Unreachable backend returns 502
- Full lifecycle — price calculation → stock receive → stock check → unknown route

```bash
cd api-gateway
go test -v -run TestFunctional ./...
```

Or run everything together:

```bash
cd api-gateway
go test -v ./...
```

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

# Pricing — functional (no DB needed)
cd pricing-service && go test -v ./tests/functional/...

# Pricing — functional with PostgreSQL (requires Docker)
cd pricing-service && go test -v -tags postgres ./tests/functional/...

# API Gateway — unit + functional (no DB or Docker needed)
cd api-gateway && go test -v ./...

# Warehouse — unit only
cd warehouse-service && go test ./internal/service/... ./internal/delivery/...

# Warehouse — functional (no DB needed)
cd warehouse-service && go test -v ./functional/...

# Warehouse — functional with PostgreSQL (requires Docker)
cd warehouse-service && go test -v -tags postgres ./functional/...
```
