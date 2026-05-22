# Requirements Document

## Introduction

The tracking-service is a Go microservice responsible for managing packet status history using an event-based, append-only approach. It exposes HTTP APIs to record tracking events, retrieve the full tracking history for a single packet, and bulk-retrieve tracking summaries for multiple packets. The service is stateless, targets ~10,000 requests per second, and persists data in two tables: `tracking_events` (immutable event log) and `tracking_summary` (current state projection).

## Glossary

- **Tracking_Service**: The Go microservice described in this document.
- **Tracking_Event**: An immutable record capturing a status change for a packet at a point in time.
- **Tracking_Summary**: A projection of the latest known state for a given tracking number, derived from its events.
- **Tracking_Number**: A unique string identifier assigned to a packet (e.g., `TRK123`).
- **Status**: An enumerated string representing the lifecycle state of a packet. Valid values: `CREATED`, `PICKED_UP`, `ARRIVED_AT_HUB`, `IN_TRANSIT`, `OUT_FOR_DELIVERY`, `DELIVERED`, `FAILED_DELIVERY`, `RETURNED`.
- **Hub**: A physical logistics facility identified by a `hub_id` string.
- **Event_Store**: The `tracking_events` database table, which is append-only and immutable.
- **Summary_Store**: The `tracking_summary` database table, which holds the latest projection per tracking number.
- **Bulk_Query**: A request to retrieve tracking summaries for multiple tracking numbers in a single HTTP call.
- **Event_Handler**: The internal component that validates, persists, and projects tracking events.
- **Repository**: The internal component that abstracts database access for both the Event_Store and Summary_Store.

---

## Requirements

### Requirement 1: Record a Tracking Event

**User Story:** As a logistics system, I want to post a tracking event for a packet, so that the packet's status history is updated in real time.

#### Acceptance Criteria

1. WHEN a `POST /tracking/events` request is received with a valid body, THE Tracking_Service SHALL persist the event to the Event_Store and return HTTP 201 with the created event's `event_id`.
2. WHEN a `POST /tracking/events` request is received, THE Event_Handler SHALL validate that `tracking_number`, `status`, and `timestamp` are present and non-empty.
3. IF `tracking_number`, `status`, or `timestamp` is missing or empty in the request body, THEN THE Tracking_Service SHALL return HTTP 400 with a descriptive error message identifying the missing field(s).
4. IF the `status` field contains a value not in the defined Status enumeration, THEN THE Tracking_Service SHALL return HTTP 422 with an error message listing the valid status values.
5. IF the `timestamp` field is not a valid RFC 3339 date-time string, THEN THE Tracking_Service SHALL return HTTP 422 with an error message indicating the expected format.
6. WHEN a tracking event is successfully persisted, THE Event_Handler SHALL upsert the corresponding Tracking_Summary record with the new `current_status`, `last_location`, and `updated_at` values.
7. THE Event_Store SHALL store each event with a server-generated `event_id` (UUID), the provided fields (`tracking_number`, `status`, `location`, `hub_id`, `notes`, `timestamp`), and a `created_by_service` field set to `"tracking-service"`.
8. WHEN two concurrent `POST /tracking/events` requests arrive for the same `tracking_number`, THE Event_Handler SHALL persist both events independently without data loss or corruption.

---

### Requirement 2: Retrieve Tracking History for a Single Packet

**User Story:** As a consumer application, I want to retrieve the full tracking history for a packet, so that I can display its current status and event timeline to end users.

#### Acceptance Criteria

1. WHEN a `GET /tracking/{tracking_number}` request is received for an existing tracking number, THE Tracking_Service SHALL return HTTP 200 with a JSON body containing `tracking_number`, `current_status`, `estimated_delivery`, and `history`.
2. THE `history` array in the response SHALL contain one entry per Tracking_Event for the given tracking number, each with `status` and `timestamp` fields, ordered by `timestamp` ascending.
3. IF no events exist for the requested `tracking_number`, THEN THE Tracking_Service SHALL return HTTP 404 with a descriptive error message.
4. WHEN a `GET /tracking/{tracking_number}` request is received, THE Tracking_Service SHALL respond within 100ms at the 99th percentile under normal load.
5. THE `current_status` field in the response SHALL reflect the status from the most recent Tracking_Event for that tracking number.
6. WHERE `estimated_delivery` is available in the Tracking_Summary, THE Tracking_Service SHALL include it in the response; WHERE it is not available, THE Tracking_Service SHALL return `null` for that field.

---

### Requirement 3: Bulk Retrieve Tracking Summaries

**User Story:** As a logistics dashboard, I want to retrieve tracking summaries for multiple packets in a single request, so that I can display a list view without issuing one request per packet.

#### Acceptance Criteria

1. WHEN a `GET /tracking/bulk?numbers=TRK1,TRK2,TRK3` request is received, THE Tracking_Service SHALL return HTTP 200 with a JSON array containing one summary object per requested tracking number that exists.
2. THE Tracking_Service SHALL accept between 1 and 100 tracking numbers per Bulk_Query request.
3. IF the `numbers` query parameter is missing or empty, THEN THE Tracking_Service SHALL return HTTP 400 with a descriptive error message.
4. IF more than 100 tracking numbers are provided in a single Bulk_Query, THEN THE Tracking_Service SHALL return HTTP 422 with an error message stating the maximum allowed count.
5. WHEN some requested tracking numbers do not exist, THE Tracking_Service SHALL return summaries only for the tracking numbers that exist, without returning an error for the missing ones.
6. WHEN a `GET /tracking/bulk` request is received, THE Tracking_Service SHALL fetch all requested summaries in a single database query rather than issuing one query per tracking number.
7. WHEN a `GET /tracking/bulk` request is received, THE Tracking_Service SHALL respond within 200ms at the 99th percentile under normal load.

---

### Requirement 4: Data Immutability and Append-Only Event Store

**User Story:** As a system architect, I want all tracking events to be immutable and append-only, so that the audit trail is tamper-proof and the system can be replayed or re-projected at any time.

#### Acceptance Criteria

1. THE Event_Store SHALL be append-only; THE Repository SHALL never issue `UPDATE` or `DELETE` statements against the `tracking_events` table.
2. WHEN an event is written to the Event_Store, THE Repository SHALL assign a unique `event_id` (UUID v4) before insertion.
3. THE `tracking_number` column in the Event_Store SHALL be indexed to support efficient history queries.
4. WHEN the Event_Store is queried for a tracking number's history, THE Repository SHALL return events ordered by `timestamp` ascending.

---

### Requirement 5: Tracking Summary Projection

**User Story:** As a system operator, I want the tracking summary to always reflect the latest event, so that single-field lookups are fast without scanning the full event history.

#### Acceptance Criteria

1. WHEN a new Tracking_Event is persisted, THE Event_Handler SHALL upsert the Summary_Store record for the same `tracking_number` within the same database transaction.
2. THE Summary_Store record SHALL contain `tracking_number` (PK), `current_status`, `last_location`, `estimated_delivery`, and `updated_at`.
3. IF an upsert to the Summary_Store fails, THEN THE Event_Handler SHALL roll back the entire transaction and return HTTP 500 to the caller.
4. WHILE a database transaction is in progress, THE Repository SHALL hold a row-level lock on the Summary_Store record for the affected `tracking_number` to prevent concurrent summary corruption.

---

### Requirement 6: Input Serialization and Deserialization

**User Story:** As a developer integrating with the service, I want the API to consistently parse and serialize JSON payloads, so that I can rely on a stable contract.

#### Acceptance Criteria

1. WHEN a request body is received, THE Tracking_Service SHALL deserialize it from JSON into the corresponding Go struct.
2. THE Tracking_Service SHALL serialize all response bodies to JSON with `Content-Type: application/json`.
3. THE JSON_Serializer SHALL format `timestamp` fields as RFC 3339 strings in both requests and responses.
4. FOR ALL valid request payloads, deserializing then serializing then deserializing SHALL produce an equivalent struct (round-trip property).
5. IF the request body is not valid JSON, THEN THE Tracking_Service SHALL return HTTP 400 with an error message indicating a malformed request body.

---

### Requirement 7: Performance and Scalability

**User Story:** As a platform engineer, I want the service to handle high throughput, so that it can sustain ~10,000 requests per second without degradation.

#### Acceptance Criteria

1. THE Tracking_Service SHALL be stateless; no in-process session or request state SHALL be shared between goroutines handling different requests.
2. THE Tracking_Service SHALL use a database connection pool with a configurable maximum connection count.
3. WHEN the database connection pool is exhausted, THE Tracking_Service SHALL return HTTP 503 with an error message rather than blocking indefinitely.
4. THE Tracking_Service SHALL expose a `GET /health` endpoint that returns HTTP 200 when the service and its database connection are healthy.
5. IF the database is unreachable during a health check, THEN THE Tracking_Service SHALL return HTTP 503 with a message indicating the database is unavailable.

---

### Requirement 8: Error Handling and Observability

**User Story:** As an on-call engineer, I want all errors to be logged with context, so that I can diagnose issues quickly in production.

#### Acceptance Criteria

1. WHEN an unhandled error occurs during request processing, THE Tracking_Service SHALL log the error with the request method, path, and a correlation ID, then return HTTP 500 with a generic error message.
2. THE Tracking_Service SHALL assign a unique correlation ID to every incoming request and include it in all log entries and error responses for that request.
3. WHEN a request completes, THE Tracking_Service SHALL emit a structured log entry containing the HTTP method, path, status code, latency in milliseconds, and correlation ID.
4. IF a database operation exceeds 5 seconds, THEN THE Tracking_Service SHALL cancel the operation via context cancellation and return HTTP 504 to the caller.
