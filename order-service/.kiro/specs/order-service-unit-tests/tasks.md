# Implementation Plan: order-service Unit Tests

## Overview

Implement a comprehensive unit test suite for the `order-service` Go microservice covering all three layers (Handler, Service, Repository) plus utilities. Tests use table-driven patterns, interface-based mocks, `sqlmock`/`pgxmock` for the repository layer, and `rapid` for property-based tests. Target: ≥ 80% line coverage.

## Tasks

- [x] 1. Set up test infrastructure and Go module dependencies
  - Add `testify/assert`, `testify/require`, `testify/mock` to `go.mod` / `go.sum`
  - Add `pgxmock` or `sqlmock` for repository-level DB driver mocking
  - Add `rapid` (`pgregory.net/rapid`) for property-based tests
  - Verify `go test ./...` runs without errors on an empty test suite
  - _Requirements: 13.1, 13.2, 13.3_

- [x] 2. Define domain types, interfaces, and enums
  - [x] 2.1 Create `Order` struct, `OrderStatus` enum, and `ServiceType` enum
    - Define all fields listed in the design (`OrderID`, `TrackingNumber`, `SenderUserID`, etc.)
    - Define status constants: `CREATED`, `AWAITING_PICKUP`, `PICKED_UP`, `IN_TRANSIT`, `DELIVERED`, `FAILED`, `CANCELLED`
    - Define service type constants: `REG`, `YES`, `OKE`, `SAME_DAY`
    - _Requirements: 13.1_
  - [x] 2.2 Define `OrderService` and `OrderRepository` interfaces
    - `OrderService`: `CreateOrder`, `GetOrder`, `ListOrders`, `CancelOrder`, `UpdateOrder`
    - `OrderRepository`: `SaveOrder`, `FindOrderByID`, `FindOrders`, `UpdateOrder`
    - Define `CreateOrderRequest`, `UpdateOrderRequest`, `ListOrdersParams`, `CreateOrderResponse` structs
    - _Requirements: 13.2_

- [x] 3. Implement mock types
  - [x] 3.1 Implement `MockOrderService` in `mocks/mock_service.go`
    - Embed `testify/mock.Mock`; implement all five `OrderService` methods recording calls and returning configured values
    - _Requirements: 13.2_
  - [x] 3.2 Implement `MockOrderRepository` in `mocks/mock_repository.go`
    - Embed `testify/mock.Mock`; implement all four `OrderRepository` methods recording calls and returning configured values
    - _Requirements: 13.2_

- [x] 4. Implement Create Order handler tests
  - [x] 4.1 Write table-driven tests in `handler/create_order_handler_test.go`
    - Cases: valid request → 201 with `order_id`, `tracking_number`, `price`, `estimated_days`, `status`, `created_at`; malformed JSON → 400; missing required field → 422; invalid `service_type` → 422; `is_cod=true` with `cod_amount ≤ 0` → 422; service error → 500
    - Assert mock service called / not called per case; use `mockSvc.AssertExpectations(t)`
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 13.6_

- [x] 5. Implement Get Order handler tests
  - [x] 5.1 Write table-driven tests in `handler/get_order_handler_test.go`
    - Cases: valid `order_id` → 200 with full Order JSON; service not-found → 404; service unexpected error → 500; empty `order_id` path param → 400 (no service call)
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 13.6_

- [x] 6. Implement List Orders handler tests
  - [x] 6.1 Write table-driven tests in `handler/list_orders_handler_test.go`
    - Cases: valid params → 200 with order array; non-positive `page`/`limit` → 400; invalid `status` enum → 400; empty list → 200 with `[]`; service error → 500
    - Assert all four params passed to mock service
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 13.6_

- [x] 7. Implement Cancel Order handler tests
  - [x] 7.1 Write table-driven tests in `handler/cancel_order_handler_test.go`
    - Cases: valid request → 200; service conflict error → 409; service not-found → 404; missing/empty `reason` → 422 (no service call)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 13.6_

- [x] 8. Implement Update Order handler tests
  - [x] 8.1 Write table-driven tests in `handler/update_order_handler_test.go`
    - Cases: valid partial update → 200 with `{"order_id": ..., "status": "UPDATED"}`; service conflict → 409; service not-found → 404; no updatable fields → 422 (no service call)
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 13.6_

- [x] 9. Checkpoint — handler layer
  - Ensure all handler tests pass and `go test ./handler/...` reports no failures. Ask the user if questions arise.

- [x] 10. Implement Create Order service logic tests
  - [x] 10.1 Write table-driven tests in `service/create_order_service_test.go`
    - Cases: valid request → tracking number non-empty, status `AWAITING_PICKUP`, price > 0; each of `REG`/`YES`/`OKE`/`SAME_DAY` → non-zero price; `insurance=true` → higher price; `is_cod=true` → higher price; repo save error → error returned
    - Inject `MockOrderRepository` via constructor; call `mockRepo.AssertExpectations(t)`
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 13.6_
  - [x] 10.2 Write property test — Property 1: Price and Tracking Number Invariants
    - **Property 1: For any valid `CreateOrderRequest`, `price > 0` and `TrackingNumber` non-empty**
    - Use `rapid.Check` with a generator for valid requests (positive weight, valid service type, valid dimensions)
    - **Validates: Requirements 6.7, 6.1**
  - [x] 10.3 Write property test — Property 2: Initial Status Invariant
    - **Property 2: For any valid `CreateOrderRequest`, persisted Order has `Status == AWAITING_PICKUP`**
    - **Validates: Requirements 6.2**
  - [x] 10.4 Write property test — Property 3: Insurance Surcharge Monotonicity
    - **Property 3: price with `insurance=true` > price with `insurance=false`, all other fields equal**
    - **Validates: Requirements 6.4**
  - [x] 10.5 Write property test — Property 4: COD Surcharge Monotonicity
    - **Property 4: price with `is_cod=true, cod_amount>0` > price without COD, all other fields equal**
    - **Validates: Requirements 6.5**

- [x] 11. Implement Get Order service logic tests
  - [x] 11.1 Write table-driven tests in `service/get_order_service_test.go`
    - Cases: valid `order_id` → Order returned; repo not-found → not-found error propagated; repo unexpected error → error propagated
    - _Requirements: 7.1, 7.2, 7.3, 13.6_

- [x] 12. Implement List Orders service logic tests
  - [x] 12.1 Write table-driven tests in `service/list_orders_service_test.go`
    - Cases: all four params passed unchanged to repo; repo returns empty slice → empty slice returned; repo error → error propagated
    - _Requirements: 8.1, 8.2, 8.3, 13.6_
  - [x] 12.2 Write property test — Property 8: List Orders Parameter Passthrough
    - **Property 8: For any combination of `user_id`, `status`, `page`, `limit`, all four values reach the repository unchanged**
    - Use `rapid` generators for each parameter type
    - **Validates: Requirements 8.1**

- [x] 13. Implement Cancel Order service logic tests
  - [x] 13.1 Write table-driven tests in `service/cancel_order_service_test.go`
    - Cases: order in `AWAITING_PICKUP` → status set to `CANCELLED`, repo update called; order not in `AWAITING_PICKUP` → conflict error, repo update NOT called; repo fetch not-found → not-found propagated; repo update error → error propagated
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 13.6_
  - [x] 13.2 Write property test — Property 5: Cancel Status Guard
    - **Property 5: For any Order with `Status != AWAITING_PICKUP`, `CancelOrder` returns conflict error and repo update is never called**
    - Generate random non-`AWAITING_PICKUP` statuses via `rapid`
    - **Validates: Requirements 9.2**

- [x] 14. Implement Update Order service logic tests
  - [x] 14.1 Write table-driven tests in `service/update_order_service_test.go`
    - Cases: order in `AWAITING_PICKUP` → fields applied, repo update called; order not in `AWAITING_PICKUP` → conflict error, repo update NOT called; only `receiver` fields provided → only receiver fields changed; only `item_description` provided → only that field changed; repo update error → error propagated
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 13.6_
  - [x] 14.2 Write property test — Property 6: Update Status Guard
    - **Property 6: For any Order with `Status != AWAITING_PICKUP`, `UpdateOrder` returns conflict error and repo update is never called**
    - **Validates: Requirements 10.2**
  - [x] 14.3 Write property test — Property 7: Partial Update Field Isolation
    - **Property 7: For any `UpdateOrderRequest` setting only a subset of fields, exactly those fields are changed and all others remain equal to the original**
    - **Validates: Requirements 10.3, 10.4**

- [x] 15. Checkpoint — service layer
  - Ensure all service tests pass and `go test ./service/...` reports no failures. Ask the user if questions arise.

- [x] 16. Implement repository layer tests
  - [x] 16.1 Write `SaveOrder` tests in `repository/order_repository_test.go`
    - Set up `sqlmock`; expect `INSERT INTO orders` with all field args in column order; assert no error; assert all mock expectations met
    - _Requirements: 11.1, 13.6_
  - [x] 16.2 Write `FindOrderByID` tests
    - Cases: valid `order_id` → all columns mapped back to correct `Order` struct fields; DB returns no rows → sentinel not-found error returned; DB error → error propagated
    - _Requirements: 11.2, 11.6, 11.5, 13.6_
  - [x] 16.3 Write `FindOrders` tests
    - Cases: each non-empty filter param (`user_id`, `status`, `page`, `limit`) → corresponding `WHERE` clause present in SQL; all params empty → no extra `WHERE` clauses; DB error → error propagated
    - _Requirements: 11.3, 11.5, 13.6_
  - [x] 16.4 Write `UpdateOrder` tests
    - Cases: valid Order → `UPDATE` statement sets correct columns and filters by `order_id`; DB error → error propagated without re-wrapping
    - _Requirements: 11.4, 11.5, 13.6_
  - [x] 16.5 Write property test — Property 9: Repository Round-Trip Fidelity
    - **Property 9: For any valid Order, `SaveOrder` followed by `FindOrderByID` returns an Order with all fields equal to the original**
    - Use `sqlmock` to simulate the round-trip; generate random valid Orders via `rapid`
    - **Validates: Requirements 11.1, 11.2**
  - [x] 16.6 Write property test — Property 10: Dynamic Filter Query Construction
    - **Property 10: For any combination of non-empty filter parameters, the constructed SQL includes a `WHERE` clause condition for each non-empty parameter**
    - **Validates: Requirements 11.3**

- [x] 17. Implement tracking number generator tests
  - [x] 17.1 Write unit tests in `tracking/generator_test.go`
    - Cases: generator returns non-empty string; two consecutive calls return distinct values; output matches alphanumeric format regex `^[A-Z0-9]+$`
    - _Requirements: 12.1, 12.2, 12.3, 13.6_
  - [x] 17.2 Write property test — Property 11: Tracking Number Uniqueness and Format
    - **Property 11: For any set of independently generated tracking numbers, all values are distinct, non-empty, and match `^[A-Z0-9]+$`**
    - Use `rapid.Check` with a `seen` map to detect duplicates across iterations
    - **Validates: Requirements 12.1, 12.2, 12.3**

- [x] 18. Checkpoint — repository and tracking layers
  - Ensure all repository and tracking tests pass and `go test ./repository/... ./tracking/...` reports no failures. Ask the user if questions arise.

- [x] 19. Verify coverage target
  - Run `go test -cover ./handler/... ./service/... ./repository/... ./tracking/...` and confirm line coverage ≥ 80% for each package
  - If any package is below 80%, add targeted test cases to cover the missing lines
  - _Requirements: 13.5_

- [x] 20. Final checkpoint — full suite
  - Run `go test ./...` and confirm all tests pass with no race conditions (`go test -race ./...`). Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for a faster MVP
- Each task references specific requirements for traceability
- Property tests use `rapid` (min 100 iterations per property by default)
- Unit tests and property tests are complementary — both are needed for full confidence
- `mockObj.AssertExpectations(t)` must be called at the end of every mock-based test case
- State-modifying tests must use `t.Cleanup` or `defer` to restore shared state (Requirement 13.4)
