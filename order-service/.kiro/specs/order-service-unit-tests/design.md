# Design Document: order-service Unit Tests

## Overview

This document describes the design for a comprehensive unit test suite for the `order-service` Go microservice. The service manages shipping order lifecycle through five REST endpoints and follows a three-layer architecture: **Handler → Service → Repository**.

The test suite isolates each layer using interface-based mocks:
- `Mock_Service` isolates the Handler layer from business logic
- `Mock_Repository` isolates the Service layer from the database

Testing tools:
- `testing` package (standard Go test runner)
- `testify/assert` and `testify/require` for assertions
- `testify/mock` for mock generation (or hand-written mocks)
- `pgxmock` or `sqlmock` for repository-level database driver mocking
- `pgx/v5` property-based testing via `rapid` or `gopter` for invariant properties

Coverage target: **≥ 80% line coverage** across handler, service, and repository packages.

---

## Architecture

### Three-Layer Architecture Under Test

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Request                          │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                   Handler Layer                          │
│  - Parse & validate HTTP request                        │
│  - Call Service interface                               │
│  - Serialize HTTP response                              │
│  Tested with: Mock_Service                              │
└──────────────────────┬──────────────────────────────────┘
                       │ Service interface
┌──────────────────────▼──────────────────────────────────┐
│                   Service Layer                          │
│  - Business logic & domain rules                        │
│  - Price calculation                                    │
│  - Status guard enforcement                             │
│  - Tracking number generation                           │
│  Tested with: Mock_Repository                           │
└──────────────────────┬──────────────────────────────────┘
                       │ Repository interface
┌──────────────────────▼──────────────────────────────────┐
│                 Repository Layer                         │
│  - SQL query construction                               │
│  - Result mapping to/from Order struct                  │
│  Tested with: sqlmock / pgxmock                         │
└─────────────────────────────────────────────────────────┘
```

### Test Isolation Strategy

Each layer is tested independently:

| Layer      | System Under Test | Test Double Used       | Isolation Mechanism         |
|------------|-------------------|------------------------|-----------------------------|
| Handler    | HTTP handlers     | `Mock_Service`         | Interface injection         |
| Service    | Business logic    | `Mock_Repository`      | Interface injection         |
| Repository | SQL queries       | `sqlmock` / `pgxmock`  | DB driver mock              |
| Utilities  | Tracking number   | None (pure function)   | Direct invocation           |

---

## Components and Interfaces

### Service Interface

```go
type OrderService interface {
    CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
    GetOrder(ctx context.Context, orderID string) (*Order, error)
    ListOrders(ctx context.Context, params ListOrdersParams) ([]*Order, error)
    CancelOrder(ctx context.Context, orderID string, reason string) error
    UpdateOrder(ctx context.Context, orderID string, req UpdateOrderRequest) error
}
```

### Repository Interface

```go
type OrderRepository interface {
    SaveOrder(ctx context.Context, order *Order) error
    FindOrderByID(ctx context.Context, orderID string) (*Order, error)
    FindOrders(ctx context.Context, params ListOrdersParams) ([]*Order, error)
    UpdateOrder(ctx context.Context, order *Order) error
}
```

### Key Data Structures

```go
type Order struct {
    OrderID         string
    TrackingNumber  string
    SenderUserID    string
    SenderName      string
    SenderAddress   string
    SenderPhone     string
    SenderCityCode  string
    ReceiverName    string
    ReceiverAddress string
    ReceiverPhone   string
    ReceiverCityCode string
    Weight          float64
    Length          float64
    Width           float64
    Height          float64
    ServiceType     ServiceType  // REG | YES | OKE | SAME_DAY
    Price           float64
    IsCOD           bool
    CODAmount       float64
    Insurance       bool
    ItemDescription string
    Status          OrderStatus  // CREATED | AWAITING_PICKUP | PICKED_UP | IN_TRANSIT | DELIVERED | FAILED | CANCELLED
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type CreateOrderRequest struct {
    SenderUserID    string
    SenderName      string
    SenderAddress   string
    SenderPhone     string
    SenderCityCode  string
    ReceiverName    string
    ReceiverAddress string
    ReceiverPhone   string
    ReceiverCityCode string
    Weight          float64
    Length          float64
    Width           float64
    Height          float64
    ServiceType     ServiceType
    IsCOD           bool
    CODAmount       float64
    Insurance       bool
    ItemDescription string
}

type UpdateOrderRequest struct {
    ReceiverName    *string
    ReceiverAddress *string
    ReceiverPhone   *string
    ReceiverCityCode *string
    ItemDescription *string
}

type ListOrdersParams struct {
    UserID  string
    Status  OrderStatus
    Page    int
    Limit   int
}
```

### Mock Implementations

**Mock_Service** (used in handler tests):
```go
type MockOrderService struct {
    mock.Mock
}
// Implements all OrderService methods, recording calls and returning configured values
```

**Mock_Repository** (used in service tests):
```go
type MockOrderRepository struct {
    mock.Mock
}
// Implements all OrderRepository methods, recording calls and returning configured values
```

---

## Data Models

### Order Status Transitions

```
CREATED → AWAITING_PICKUP → PICKED_UP → IN_TRANSIT → DELIVERED
                         ↘                          ↘
                          CANCELLED                  FAILED
```

Only orders in `AWAITING_PICKUP` status can be cancelled or updated.

### Price Calculation Model

Price is computed from base rate × weight + dimensional weight factor, with optional surcharges:

| Component       | Condition              |
|-----------------|------------------------|
| Base rate       | Always (varies by service type) |
| Insurance surcharge | `insurance == true` |
| COD surcharge   | `is_cod == true`       |

All valid inputs must produce `price > 0`.

### Test File Organization

```
order-service/
├── handler/
│   ├── create_order_handler.go
│   ├── create_order_handler_test.go
│   ├── get_order_handler_test.go
│   ├── list_orders_handler_test.go
│   ├── cancel_order_handler_test.go
│   └── update_order_handler_test.go
├── service/
│   ├── order_service.go
│   ├── create_order_service_test.go
│   ├── get_order_service_test.go
│   ├── list_orders_service_test.go
│   ├── cancel_order_service_test.go
│   └── update_order_service_test.go
├── repository/
│   ├── order_repository.go
│   └── order_repository_test.go
├── tracking/
│   ├── generator.go
│   └── generator_test.go
└── mocks/
    ├── mock_service.go
    └── mock_repository.go
```

---

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Price and Tracking Number Invariants

*For any* valid `CreateOrderRequest` (positive weight, valid service type, valid dimensions), the service SHALL produce an Order whose `price` is strictly greater than zero and whose `TrackingNumber` is non-empty.

**Validates: Requirements 6.7, 6.1**

### Property 2: Initial Status Invariant

*For any* valid `CreateOrderRequest`, the Order persisted by the service SHALL have `Order_Status` equal to `AWAITING_PICKUP`.

**Validates: Requirements 6.2**

### Property 3: Insurance Surcharge Monotonicity

*For any* valid `CreateOrderRequest`, the price computed with `insurance = true` SHALL be strictly greater than the price computed with `insurance = false` and all other fields equal.

**Validates: Requirements 6.4**

### Property 4: COD Surcharge Monotonicity

*For any* valid `CreateOrderRequest` with `is_cod = true` and `cod_amount > 0`, the price computed with COD SHALL be strictly greater than the price computed without COD and all other fields equal.

**Validates: Requirements 6.5**

### Property 5: Cancel Status Guard

*For any* Order whose `Order_Status` is not `AWAITING_PICKUP`, calling `CancelOrder` SHALL return a status-conflict error and the repository update method SHALL NOT be called.

**Validates: Requirements 9.2**

### Property 6: Update Status Guard

*For any* Order whose `Order_Status` is not `AWAITING_PICKUP`, calling `UpdateOrder` SHALL return a status-conflict error and the repository update method SHALL NOT be called.

**Validates: Requirements 10.2**

### Property 7: Partial Update Field Isolation

*For any* Order in `AWAITING_PICKUP` status and any `UpdateOrderRequest` that sets only a subset of fields (receiver fields only, or `item_description` only), the service SHALL update exactly the specified fields and leave all other Order fields unchanged.

**Validates: Requirements 10.3, 10.4**

### Property 8: List Orders Parameter Passthrough

*For any* combination of `user_id`, `status`, `page`, and `limit` parameters, the service SHALL pass all four values to the repository unchanged.

**Validates: Requirements 8.1**

### Property 9: Repository Round-Trip Fidelity

*For any* valid Order, saving it via `SaveOrder` and then retrieving it via `FindOrderByID` SHALL return an Order with all fields equal to the original.

**Validates: Requirements 11.1, 11.2**

### Property 10: Dynamic Filter Query Construction

*For any* combination of non-empty filter parameters passed to `FindOrders`, the repository SHALL include a `WHERE` clause condition for each non-empty parameter in the constructed SQL.

**Validates: Requirements 11.3**

### Property 11: Tracking Number Uniqueness and Format

*For any* set of independently generated tracking numbers, all values SHALL be distinct, non-empty, and conform to the defined alphanumeric format pattern.

**Validates: Requirements 12.1, 12.2, 12.3**

---

## Error Handling

### Handler Layer Error Mapping

| Error Source              | HTTP Status | Notes                                      |
|---------------------------|-------------|--------------------------------------------|
| Malformed JSON body       | 400         | No service call made                       |
| Missing required field    | 422         | No service call made                       |
| Invalid enum value        | 422         | `service_type`, `status`                   |
| COD validation failure    | 422         | `is_cod=true` with `cod_amount <= 0`       |
| Empty path parameter      | 400         | `order_id` empty                           |
| Invalid pagination params | 400         | `page` or `limit` not positive integers    |
| Service not-found error   | 404         |                                            |
| Service conflict error    | 409         | Status not `AWAITING_PICKUP`               |
| Service unexpected error  | 500         |                                            |

### Service Layer Error Propagation

- Not-found errors from repository → propagated as not-found to handler
- Conflict errors (status guard) → returned as conflict error, no repository update called
- Repository save/update errors → propagated as-is to handler

### Repository Layer Error Handling

- Database driver errors → propagated without wrapping (original context preserved)
- No rows returned by `FindOrderByID` → return sentinel not-found error
- All other errors → propagated directly

---

## Testing Strategy

### Dual Testing Approach

Unit tests and property-based tests are complementary:
- **Unit tests** (table-driven): verify specific examples, edge cases, and error conditions
- **Property-based tests**: verify universal invariants across randomly generated inputs

### Handler Tests

Each handler test file uses a table-driven pattern:

```go
func TestCreateOrderHandler(t *testing.T) {
    tests := []struct {
        name           string
        body           string
        mockSetup      func(*MockOrderService)
        expectedStatus int
        expectedBody   string
    }{
        // happy path, malformed JSON, missing fields, invalid enum, COD validation, service error
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            mockSvc := new(MockOrderService)
            tc.mockSetup(mockSvc)
            // build request, call handler, assert response
            mockSvc.AssertExpectations(t)
        })
    }
}
```

Key assertions per handler:
- Response status code
- Response body JSON fields
- Whether the mock service method was called (or not called for validation failures)

### Service Tests

Each service test uses `Mock_Repository` injected via constructor:

```go
func TestCreateOrderService(t *testing.T) {
    tests := []struct {
        name        string
        req         CreateOrderRequest
        repoSetup   func(*MockOrderRepository)
        expectErr   bool
        checkResult func(*testing.T, *CreateOrderResponse)
    }{
        // valid request, repo error, each service type, insurance, COD
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            mockRepo := new(MockOrderRepository)
            tc.repoSetup(mockRepo)
            svc := NewOrderService(mockRepo)
            result, err := svc.CreateOrder(context.Background(), tc.req)
            // assertions
            mockRepo.AssertExpectations(t)
        })
    }
}
```

### Repository Tests

Repository tests use `sqlmock` (or `pgxmock` for pgx driver) to intercept SQL:

```go
func TestSaveOrder(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    mock.ExpectExec("INSERT INTO orders").
        WithArgs(/* all fields in column order */).
        WillReturnResult(sqlmock.NewResult(1, 1))

    repo := NewOrderRepository(db)
    err = repo.SaveOrder(context.Background(), &testOrder)
    require.NoError(t, err)
    require.NoError(t, mock.ExpectationsMet())
}
```

### Property-Based Tests

Property-based tests use the [`rapid`](https://github.com/flyingmutant/rapid) library (pure Go, no external dependencies):

```go
// Property 1: Price and tracking number invariants
func TestPriceAndTrackingInvariants(t *testing.T) {
    // Feature: order-service-unit-tests, Property 1: price > 0 and tracking number non-empty
    rapid.Check(t, func(t *rapid.T) {
        req := genValidCreateOrderRequest(t)
        mockRepo := new(MockOrderRepository)
        mockRepo.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
        svc := NewOrderService(mockRepo)
        resp, err := svc.CreateOrder(context.Background(), req)
        require.NoError(t, err)
        require.Greater(t, resp.Price, 0.0)
        require.NotEmpty(t, resp.TrackingNumber)
    })
}

// Property 11: Tracking number uniqueness and format
func TestTrackingNumberUniquenessAndFormat(t *testing.T) {
    // Feature: order-service-unit-tests, Property 11: tracking numbers are unique and alphanumeric
    seen := make(map[string]struct{})
    rapid.Check(t, func(t *rapid.T) {
        n := GenerateTrackingNumber()
        require.NotEmpty(t, n)
        require.Regexp(t, `^[A-Z0-9]+$`, n)
        _, exists := seen[n]
        require.False(t, exists, "duplicate tracking number: %s", n)
        seen[n] = struct{}{}
    })
}
```

Each property test runs a minimum of **100 iterations** (rapid default is 100; configurable via `rapid.Settings`).

### Coverage Strategy

To reach ≥ 80% line coverage:

| Package     | Key Coverage Areas                                                  |
|-------------|---------------------------------------------------------------------|
| handler     | All 5 handlers × happy path + all error branches                   |
| service     | All 5 service methods × happy path + status guard + repo errors    |
| repository  | SaveOrder, FindOrderByID, FindOrders, UpdateOrder + error paths     |
| tracking    | Generator happy path + format validation                            |

### Test Cleanup

All tests that modify shared state use `t.Cleanup` or `defer`:

```go
func TestWithSharedState(t *testing.T) {
    original := globalState
    t.Cleanup(func() { globalState = original })
    // test body
}
```

Mock expectations are verified with `mockObj.AssertExpectations(t)` at the end of each test case.
