# Requirements Document

## Introduction

This document defines the requirements for unit tests of the `order-service` microservice written in Go. The order-service manages the full lifecycle of shipping orders, exposing five REST API endpoints: Create Order, Get Order, List Orders, Cancel Order, and Update Order. Unit tests must cover handler logic, service/business logic, and repository interactions using mocks, ensuring correctness of each layer in isolation.

## Glossary

- **Order_Service**: The Go microservice under test that manages shipping orders.
- **Handler**: The HTTP handler layer responsible for parsing requests and writing responses.
- **Service**: The business logic layer that enforces domain rules and orchestrates repository calls.
- **Repository**: The data access layer responsible for reading and writing orders to the database.
- **Mock_Repository**: A test double implementing the Repository interface, used to isolate the Service layer from the database.
- **Mock_Service**: A test double implementing the Service interface, used to isolate the Handler layer from business logic.
- **Order**: A shipping order entity containing sender, receiver, parcel, and logistics metadata.
- **Order_Status**: The lifecycle state of an Order; one of `CREATED`, `AWAITING_PICKUP`, `PICKED_UP`, `IN_TRANSIT`, `DELIVERED`, `FAILED`, or `CANCELLED`.
- **Service_Type**: The shipping service tier; one of `REG`, `YES`, `OKE`, or `SAME_DAY`.
- **Tracking_Number**: A unique identifier assigned to an Order at creation time.
- **COD**: Cash on delivery; a payment option where the recipient pays upon delivery.

---

## Requirements

### Requirement 1: Create Order Handler Tests

**User Story:** As a developer, I want unit tests for the Create Order handler, so that I can verify HTTP request parsing, validation, and response serialization are correct.

#### Acceptance Criteria

1. WHEN a valid `POST /orders` request body is received, THE Handler SHALL deserialize the request into a `CreateOrderRequest` struct and pass it to the Service layer.
2. WHEN the Service layer returns a successful result, THE Handler SHALL respond with HTTP 201 and a JSON body containing `order_id`, `tracking_number`, `price`, `estimated_days`, `status`, and `created_at`.
3. WHEN the request body is malformed JSON, THE Handler SHALL respond with HTTP 400 and a descriptive error message without invoking the Service layer.
4. WHEN a required field is missing from the request body, THE Handler SHALL respond with HTTP 422 and a descriptive validation error message without invoking the Service layer.
5. WHEN `service_type` is not one of `REG`, `YES`, `OKE`, or `SAME_DAY`, THE Handler SHALL respond with HTTP 422 and a descriptive validation error message.
6. WHEN `is_cod` is `true` and `cod_amount` is less than or equal to zero, THE Handler SHALL respond with HTTP 422 and a descriptive validation error message.
7. WHEN the Service layer returns an error, THE Handler SHALL respond with HTTP 500 and a descriptive error message.

---

### Requirement 2: Get Order Handler Tests

**User Story:** As a developer, I want unit tests for the Get Order handler, so that I can verify path parameter extraction and full order object serialization.

#### Acceptance Criteria

1. WHEN a valid `GET /orders/{order_id}` request is received and the Service layer returns an Order, THE Handler SHALL respond with HTTP 200 and a JSON body containing all Order fields including sender, receiver, dimension, and all metadata fields.
2. WHEN the Service layer returns a not-found error, THE Handler SHALL respond with HTTP 404 and a descriptive error message.
3. WHEN the Service layer returns an unexpected error, THE Handler SHALL respond with HTTP 500 and a descriptive error message.
4. WHEN `order_id` is an empty string in the path, THE Handler SHALL respond with HTTP 400 and a descriptive error message without invoking the Service layer.

---

### Requirement 3: List Orders Handler Tests

**User Story:** As a developer, I want unit tests for the List Orders handler, so that I can verify query parameter parsing, filtering, and pagination behavior.

#### Acceptance Criteria

1. WHEN a valid `GET /orders` request is received with `user_id`, `status`, `page`, and `limit` query parameters, THE Handler SHALL pass all four parameters correctly to the Service layer.
2. WHEN the Service layer returns a list of Orders, THE Handler SHALL respond with HTTP 200 and a JSON array of Order summaries.
3. WHEN `page` or `limit` query parameters are not positive integers, THE Handler SHALL respond with HTTP 400 and a descriptive error message without invoking the Service layer.
4. WHEN `status` is provided and is not a valid Order_Status value, THE Handler SHALL respond with HTTP 400 and a descriptive error message without invoking the Service layer.
5. WHEN the Service layer returns an empty list, THE Handler SHALL respond with HTTP 200 and an empty JSON array.
6. WHEN the Service layer returns an error, THE Handler SHALL respond with HTTP 500 and a descriptive error message.

---

### Requirement 4: Cancel Order Handler Tests

**User Story:** As a developer, I want unit tests for the Cancel Order handler, so that I can verify that cancellation requests are correctly routed and business rule violations are surfaced.

#### Acceptance Criteria

1. WHEN a valid `POST /orders/{order_id}/cancel` request is received with a non-empty `reason`, THE Handler SHALL pass `order_id` and `reason` to the Service layer.
2. WHEN the Service layer returns success, THE Handler SHALL respond with HTTP 200.
3. WHEN the Service layer returns a status-conflict error (order not in `AWAITING_PICKUP`), THE Handler SHALL respond with HTTP 409 and a descriptive error message.
4. WHEN the Service layer returns a not-found error, THE Handler SHALL respond with HTTP 404 and a descriptive error message.
5. WHEN `reason` is missing or empty in the request body, THE Handler SHALL respond with HTTP 422 and a descriptive validation error message without invoking the Service layer.

---

### Requirement 5: Update Order Handler Tests

**User Story:** As a developer, I want unit tests for the Update Order handler, so that I can verify partial update request handling and status constraint enforcement.

#### Acceptance Criteria

1. WHEN a valid `PATCH /orders/{order_id}` request is received with at least one updatable field (`receiver` or `item_description`), THE Handler SHALL pass `order_id` and the update payload to the Service layer.
2. WHEN the Service layer returns success, THE Handler SHALL respond with HTTP 200 and a JSON body containing `order_id` and `status` set to `"UPDATED"`.
3. WHEN the Service layer returns a status-conflict error (order not in `AWAITING_PICKUP`), THE Handler SHALL respond with HTTP 409 and a descriptive error message.
4. WHEN the Service layer returns a not-found error, THE Handler SHALL respond with HTTP 404 and a descriptive error message.
5. WHEN the request body contains no updatable fields, THE Handler SHALL respond with HTTP 422 and a descriptive validation error message without invoking the Service layer.

---

### Requirement 6: Create Order Service Logic Tests

**User Story:** As a developer, I want unit tests for the Create Order service logic, so that I can verify price calculation, tracking number generation, and repository interactions.

#### Acceptance Criteria

1. WHEN `CreateOrder` is called with a valid request, THE Service SHALL generate a unique `Tracking_Number` and persist the Order via the Mock_Repository.
2. WHEN `CreateOrder` is called with a valid request, THE Service SHALL set the initial `Order_Status` to `AWAITING_PICKUP`.
3. WHEN `CreateOrder` is called with `service_type` set to each of `REG`, `YES`, `OKE`, and `SAME_DAY`, THE Service SHALL calculate a non-zero `price` for each service type.
4. WHEN `CreateOrder` is called with `insurance` set to `true`, THE Service SHALL include an insurance surcharge in the calculated `price`.
5. WHEN `CreateOrder` is called with `is_cod` set to `true`, THE Service SHALL include a COD surcharge in the calculated `price`.
6. WHEN the Mock_Repository returns an error on save, THE Service SHALL return an error to the caller without retrying.
7. FOR ALL valid `CreateOrderRequest` inputs, THE Service SHALL produce an Order whose `price` is greater than zero and whose `Tracking_Number` is non-empty (invariant property).

---

### Requirement 7: Get Order Service Logic Tests

**User Story:** As a developer, I want unit tests for the Get Order service logic, so that I can verify retrieval delegation and error propagation.

#### Acceptance Criteria

1. WHEN `GetOrder` is called with a valid `order_id`, THE Service SHALL delegate to the Mock_Repository and return the retrieved Order.
2. WHEN the Mock_Repository returns a not-found error, THE Service SHALL propagate a not-found error to the caller.
3. WHEN the Mock_Repository returns an unexpected error, THE Service SHALL propagate the error to the caller.

---

### Requirement 8: List Orders Service Logic Tests

**User Story:** As a developer, I want unit tests for the List Orders service logic, so that I can verify filter and pagination delegation.

#### Acceptance Criteria

1. WHEN `ListOrders` is called with `user_id`, `status`, `page`, and `limit`, THE Service SHALL pass all four parameters unchanged to the Mock_Repository.
2. WHEN the Mock_Repository returns an empty slice, THE Service SHALL return an empty slice to the caller without error.
3. WHEN the Mock_Repository returns an error, THE Service SHALL propagate the error to the caller.

---

### Requirement 9: Cancel Order Service Logic Tests

**User Story:** As a developer, I want unit tests for the Cancel Order service logic, so that I can verify the status guard and state transition.

#### Acceptance Criteria

1. WHEN `CancelOrder` is called and the Mock_Repository returns an Order with `Order_Status` equal to `AWAITING_PICKUP`, THE Service SHALL update the Order's `Order_Status` to `CANCELLED` and persist the change via the Mock_Repository.
2. WHEN `CancelOrder` is called and the Mock_Repository returns an Order whose `Order_Status` is not `AWAITING_PICKUP`, THE Service SHALL return a status-conflict error without calling the repository update method.
3. WHEN `CancelOrder` is called and the Mock_Repository returns a not-found error on fetch, THE Service SHALL propagate a not-found error to the caller.
4. WHEN `CancelOrder` is called and the Mock_Repository returns an error on update, THE Service SHALL propagate the error to the caller.

---

### Requirement 10: Update Order Service Logic Tests

**User Story:** As a developer, I want unit tests for the Update Order service logic, so that I can verify the status guard and partial field update behavior.

#### Acceptance Criteria

1. WHEN `UpdateOrder` is called and the Mock_Repository returns an Order with `Order_Status` equal to `AWAITING_PICKUP`, THE Service SHALL apply the provided field updates and persist the updated Order via the Mock_Repository.
2. WHEN `UpdateOrder` is called and the Mock_Repository returns an Order whose `Order_Status` is not `AWAITING_PICKUP`, THE Service SHALL return a status-conflict error without calling the repository update method.
3. WHEN `UpdateOrder` is called with only `receiver` fields provided, THE Service SHALL update only the receiver fields and leave all other Order fields unchanged.
4. WHEN `UpdateOrder` is called with only `item_description` provided, THE Service SHALL update only `item_description` and leave all other Order fields unchanged.
5. WHEN the Mock_Repository returns an error on update, THE Service SHALL propagate the error to the caller.

---

### Requirement 11: Repository Layer Tests

**User Story:** As a developer, I want unit tests for the Repository layer, so that I can verify SQL query construction and result mapping using a mock database driver.

#### Acceptance Criteria

1. WHEN `SaveOrder` is called with a valid Order, THE Repository SHALL execute an `INSERT` statement mapping all Order fields to the correct database columns.
2. WHEN `FindOrderByID` is called with a valid `order_id`, THE Repository SHALL execute a `SELECT` statement and map all result columns back to the correct Order struct fields.
3. WHEN `FindOrders` is called with filter parameters, THE Repository SHALL construct a `SELECT` statement that includes `WHERE` clauses for each non-empty filter parameter.
4. WHEN `UpdateOrder` is called with an updated Order, THE Repository SHALL execute an `UPDATE` statement that sets the correct columns and filters by `order_id`.
5. WHEN the database driver returns an error, THE Repository SHALL propagate the error to the caller without wrapping it in a new error type that loses the original context.
6. WHEN `FindOrderByID` is called and the database returns no rows, THE Repository SHALL return a not-found sentinel error.

---

### Requirement 12: Tracking Number Generation Tests

**User Story:** As a developer, I want unit tests for the tracking number generator, so that I can verify uniqueness and format properties.

#### Acceptance Criteria

1. THE Tracking_Number_Generator SHALL produce a non-empty string for every invocation.
2. FOR ALL pairs of independently generated tracking numbers, THE Tracking_Number_Generator SHALL produce distinct values (uniqueness property).
3. THE Tracking_Number_Generator SHALL produce tracking numbers that conform to the defined format pattern (alphanumeric, fixed or bounded length).

---

### Requirement 13: Test Infrastructure Requirements

**User Story:** As a developer, I want a consistent test infrastructure, so that all unit tests are isolated, repeatable, and maintainable.

#### Acceptance Criteria

1. THE Test_Suite SHALL use Go's standard `testing` package as the test runner.
2. THE Test_Suite SHALL use interface-based Mock_Repository and Mock_Service doubles generated or written by hand to isolate each layer.
3. THE Test_Suite SHALL use `testify/assert` or `testify/require` for assertions to produce clear failure messages.
4. WHEN any test modifies shared state, THE Test_Suite SHALL reset that state in a `t.Cleanup` or `defer` block so subsequent tests are not affected.
5. THE Test_Suite SHALL achieve a minimum of 80% line coverage across handler, service, and repository packages as measured by `go test -cover`.
6. WHEN a table-driven test is applicable (multiple input/output pairs for the same function), THE Test_Suite SHALL use Go's table-driven test pattern with a `[]struct` test case slice.
