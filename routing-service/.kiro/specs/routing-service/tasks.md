# Implementation Plan

- [x] 1. Set up project structure and core interfaces





  - Initialize Node.js/TypeScript project with Express, Jest, fast-check, ioredis, pg, and zod
  - Create directory structure: `src/routes`, `src/services`, `src/repositories`, `src/calculators`, `src/clients`, `src/models`, `src/middleware`
  - Define all TypeScript interfaces: `RouteNode`, `RouteEdge`, `InterHubRoute`, `CourierRoute`, `CourierRouteStop`, `DeliveryPoint`, `HubOrigin`
  - Configure Jest and fast-check in `jest.config.ts`
  - _Requirements: 1.1, 2.1, 3.1, 4.3_

- [x] 2. Implement database schema and Graph Repository





  - [x] 2.1 Create PostgreSQL migration scripts for `route_nodes` and `route_edges` tables


    - Define schema with all columns, constraints, FK relationships, and ENUM for transport_mode
    - _Requirements: 2.2, 3.2_
  - [x] 2.2 Implement `GraphRepository` with `getAllNodes`, `createNode`, `getAllEdges`, `createEdge`, `updateEdge`, `getActiveGraph`


    - Use parameterized queries to prevent SQL injection
    - _Requirements: 2.1, 2.2, 3.1, 3.2, 3.3_

- [x] 3. Implement Cache Service





  - [x] 3.1 Implement `CacheService` wrapping ioredis with `get`, `set`, and `invalidatePattern` methods


    - Serialize/deserialize using `JSON.stringify` / `JSON.parse`
    - Implement fallback: if Redis is unavailable, log warning and return null (caller handles fallback)
    - Cache keys: `route:{origin}:{destination}:{service_type}` (TTL 86400s), `courier_route:{courier_id}:{date}` (TTL 600s)
    - _Requirements: 1.3, 1.4, 4.4, 4.5, 5.3, 6.2_
  - [x] 3.2 Write property test for cache round-trip (Property 2)


    - **Property 2: Route cache round-trip consistency**
    - **Validates: Requirements 5.3**

- [x] 4. Implement Inter-Hub Route Calculator





  - [x] 4.1 Implement Haversine distance utility function


    - _Requirements: 4.2_
  - [x] 4.2 Implement Dijkstra algorithm on `Graph` (adjacency list of active edges) with combined weight `alpha * distance_km + beta * avg_transit_hours`


    - Return ordered hub sequence from origin to destination
    - Return null if no path exists
    - _Requirements: 1.1, 1.2, 1.5_
  - [x] 4.3 Write property test for Dijkstra optimality (Property 8)


    - **Property 8: Dijkstra optimality — no shorter path exists**
    - **Validates: Requirements 1.2**
  - [x] 4.4 Write property test for inter-hub route sequence integrity (Property 1)

    - **Property 1: Inter-hub route sequence integrity**
    - **Validates: Requirements 1.1, 1.2**

- [x] 5. Implement Courier Route Calculator





  - [x] 5.1 Implement nearest neighbor heuristic using Haversine distances on lat/lng coordinates


    - Fixed start: hub origin
    - Return `CourierRoute` with all required fields including `estimated_arrival` (based on avg speed assumption) and `distance_from_prev_km`
    - Handle empty delivery points: return zero totals and empty array
    - _Requirements: 4.2, 4.3, 4.6_
  - [x] 5.2 Write property test for nearest neighbor starts at hub (Property 3)


    - **Property 3: Nearest neighbor route starts at hub**
    - **Validates: Requirements 4.2**
  - [x] 5.3 Write property test for courier route covers all delivery points (Property 4)


    - **Property 4: Courier route covers all delivery points**
    - **Validates: Requirements 4.2, 4.3**
  - [x] 5.4 Write property test for total distance consistency (Property 5)


    - **Property 5: Total distance consistency**
    - **Validates: Requirements 4.3**
  - [x] 5.5 Write property test for empty delivery points returns zero totals (Property 7)


    - **Property 7: Empty delivery points returns zero totals**
    - **Validates: Requirements 4.6**

- [x] 6. Implement Delivery Service Client





  - Implement `DeliveryServiceClient` with `getCourierDeliveryPoints` and `getCourierHub` using HTTP (axios or node-fetch)
  - Handle 503/timeout: throw `UpstreamUnavailableError`
  - _Requirements: 4.1, 4.7_

- [x] 7. Implement Route Node and Edge API handlers





  - [x] 7.1 Implement GET /routing/nodes handler using `GraphRepository.getAllNodes`


    - _Requirements: 2.1_
  - [x] 7.2 Implement POST /routing/nodes handler with zod validation (hub_id, city_code, latitude, longitude)


    - Handle duplicate hub_id → 409, missing fields → 400
    - _Requirements: 2.2, 2.3, 2.4_
  - [x] 7.3 Implement GET /routing/edges handler with optional `from_node_id` filter


    - _Requirements: 3.1_
  - [x] 7.4 Implement POST /routing/edges handler with zod validation


    - Default `is_active = true`, validate transport_mode enum, check node existence → 404
    - _Requirements: 3.2, 3.5, 3.7_
  - [x] 7.5 Implement PATCH /routing/edges/:edge_id handler


    - Partial update, invalidate cache for affected routes after update
    - Handle edge not found → 404
    - _Requirements: 3.3, 3.4, 3.6_
  - [x] 7.6 Write property test for edge update invalidates cache (Property 6)



    - **Property 6: Edge update invalidates affected cache**
    - **Validates: Requirements 3.4**

- [x] 8. Implement Inter-Hub Route API handler





  - Implement GET /routing/route handler
    - Check cache first; on miss, load active graph from DB, run Dijkstra, store result in cache
    - On Redis unavailable: skip cache, compute from DB
    - On no path found: return 404
    - Validate query params with zod → 400 on invalid
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.2, 6.3_

- [x] 9. Implement Courier Route API handler





  - Implement GET /routing/courier-route/:courier_id handler
    - Check cache first; on miss, fetch delivery points and hub from Delivery Service, run nearest neighbor, store in cache
    - On Delivery Service unavailable: return 503
    - On empty delivery points: return zero-total response
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7_

- [x] 10. Implement global error handling middleware





  - Map `UpstreamUnavailableError` → 503, `ValidationError` → 400, `NotFoundError` → 404, `DuplicateError` → 409
  - Ensure unhandled errors return 500 without leaking stack traces
  - _Requirements: 1.5, 2.3, 2.4, 3.5, 3.6, 3.7, 4.7, 6.3_

- [x] 11. Checkpoint — Ensure all tests pass, ask the user if questions arise.






- [x] 12. Wire up Express app and validate end-to-end





  - Register all routers, middleware, and error handler in `src/app.ts`
  - Create `src/server.ts` entry point with graceful shutdown
  - Add Docker configuration (`Dockerfile`, `docker-compose.yml` with PostgreSQL and Redis)
  - _Requirements: 6.1, 6.2_

- [x] 13. Final Checkpoint — Ensure all tests pass, ask the user if questions arise.





