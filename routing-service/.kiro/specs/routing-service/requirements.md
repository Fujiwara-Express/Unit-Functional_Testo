# Requirements Document

## Introduction

Routing Service adalah microservice dalam ekosistem aplikasi pengiriman ekspres yang bertanggung jawab menentukan rute optimal pengiriman paket antar hub (inter-hub routing) dan rute pengantaran kurir harian (last-mile delivery routing). Layanan ini menggunakan konsep Graph Theory dengan algoritma Dijkstra/A* untuk inter-hub routing dan nearest neighbor heuristic sebagai TSP approximation untuk courier delivery routing. Layanan berjalan stateless dengan dukungan Redis cache untuk performa tinggi pada beban ~1000 req/sec.

## Glossary

- **Routing Service**: Microservice yang mengelola dan menghitung rute pengiriman paket.
- **Route Node**: Representasi hub fisik dalam graf rute, menyimpan koordinat geografis dan kode kota.
- **Route Edge**: Koneksi terarah antara dua Route Node yang merepresentasikan jalur pengiriman beserta atribut jarak, waktu tempuh, dan moda transportasi.
- **Hub**: Fasilitas fisik pusat distribusi paket (contoh: HUB_BDG, HUB_JKT).
- **Inter-Hub Route**: Rute optimal antar hub yang dihitung menggunakan algoritma Dijkstra/A*.
- **Courier Delivery Route**: Rute optimal harian kurir untuk semua titik pengantaran, dihitung menggunakan nearest neighbor heuristic.
- **TSP (Travelling Salesman Problem)**: Masalah optimasi untuk menemukan rute terpendek yang mengunjungi semua titik tepat satu kali.
- **Nearest Neighbor Heuristic**: Algoritma aproksimasi TSP yang secara iteratif memilih titik terdekat berikutnya yang belum dikunjungi.
- **Dijkstra**: Algoritma shortest path pada graf berbobot non-negatif.
- **Redis Cache**: Penyimpanan in-memory untuk menyimpan hasil kalkulasi rute sementara.
- **TTL (Time To Live)**: Durasi validitas data dalam cache sebelum kedaluwarsa.
- **Service Type**: Jenis layanan pengiriman (contoh: REG = Reguler, EXP = Express).
- **Transport Mode**: Moda transportasi pada sebuah edge: DARAT, UDARA, atau LAUT.
- **Delivery Point**: Titik tujuan pengantaran paket oleh kurir, memiliki koordinat lat/lng.
- **Admin**: Pengguna dengan hak akses untuk mengelola data node dan edge pada graf rute.
- **Courier**: Kurir yang bertanggung jawab mengantarkan paket ke delivery point.
- **Warehouse Service**: Microservice eksternal yang mengelola proses sorting paket.
- **Pricing Service**: Microservice eksternal yang menghitung estimasi biaya dan hari pengiriman.
- **Delivery Service**: Microservice eksternal yang mengelola proses pengantaran dan assign kurir.

## Requirements

### Requirement 1

**User Story:** As a logistics system, I want to calculate the optimal inter-hub route between an origin and destination city, so that packages can be routed through the most efficient path across distribution hubs.

#### Acceptance Criteria

1. WHEN the Routing Service receives a GET /routing/route request with valid `origin`, `destination`, and `service_type` query parameters, THE Routing Service SHALL return a route containing an ordered sequence of hubs with `hub_id`, `city`, and `sequence` fields, along with `total_distance_km` and `estimated_transit_hours`.
2. WHEN the Routing Service calculates an inter-hub route, THE Routing Service SHALL apply Dijkstra or A* algorithm on the active route edges graph with a combined weight of distance and transit time.
3. WHEN the Routing Service receives a route request for an origin-destination-service_type combination that exists in cache, THE Routing Service SHALL return the cached result without recalculating the route.
4. WHEN a cached inter-hub route reaches its TTL of 24 hours, THE Routing Service SHALL invalidate the cache entry and recalculate the route on the next request.
5. IF no valid path exists between the requested origin and destination using only active edges, THEN THE Routing Service SHALL return an error response with a descriptive message indicating no route is available.

---

### Requirement 2

**User Story:** As an admin, I want to manage route nodes (hubs) in the routing graph, so that I can keep the hub network up to date as the logistics network expands.

#### Acceptance Criteria

1. WHEN an admin sends a GET /routing/nodes request, THE Routing Service SHALL return a list of all route nodes, each containing `node_id`, `hub_id`, `city_code`, `latitude`, and `longitude`.
2. WHEN an admin sends a POST /routing/nodes request with valid `hub_id`, `city_code`, `latitude`, and `longitude` fields, THE Routing Service SHALL create a new route node, persist it to the database, and return the generated `node_id` with status `CREATED`.
3. IF an admin sends a POST /routing/nodes request with a `hub_id` that already exists, THEN THE Routing Service SHALL reject the request and return an error response indicating a duplicate hub.
4. IF an admin sends a POST /routing/nodes request with missing or invalid required fields, THEN THE Routing Service SHALL return a validation error response specifying the invalid fields.

---

### Requirement 3

**User Story:** As an admin, I want to manage route edges (connections between hubs) in the routing graph, so that I can configure and update the logistics network topology and transit parameters.

#### Acceptance Criteria

1. WHEN an admin sends a GET /routing/edges request with an optional `from_node_id` query parameter, THE Routing Service SHALL return a list of route edges filtered by `from_node_id` when provided, each containing `edge_id`, `from_node_id`, `to_node_id`, `distance_km`, `avg_transit_hours`, `transport_mode`, and `is_active`.
2. WHEN an admin sends a POST /routing/edges request with valid `from_node_id`, `to_node_id`, `distance_km`, `avg_transit_hours`, and `transport_mode` fields, THE Routing Service SHALL create a new route edge with `is_active` defaulting to `true`, persist it to the database, and return the generated `edge_id` with status `CREATED`.
3. WHEN an admin sends a PATCH /routing/edges/{edge_id} request with valid fields, THE Routing Service SHALL update only the provided fields on the specified edge and return the `edge_id` with status `UPDATED`.
4. WHEN an admin updates a route edge, THE Routing Service SHALL invalidate all cached inter-hub routes that traverse the updated edge.
5. IF an admin sends a POST /routing/edges request referencing a `from_node_id` or `to_node_id` that does not exist, THEN THE Routing Service SHALL return an error response indicating the referenced node does not exist.
6. IF an admin sends a PATCH /routing/edges/{edge_id} request for an `edge_id` that does not exist, THEN THE Routing Service SHALL return a 404 error response.
7. IF an admin sends a POST /routing/edges request with a `transport_mode` value other than `DARAT`, `UDARA`, or `LAUT`, THEN THE Routing Service SHALL return a validation error response.

---

### Requirement 4

**User Story:** As a delivery system, I want to retrieve the optimized daily delivery route for a courier, so that the courier can complete all assigned deliveries in the most efficient order.

#### Acceptance Criteria

1. WHEN the Routing Service receives a GET /routing/courier-route/{courier_id} request, THE Routing Service SHALL retrieve all active delivery points assigned to that courier for the current date from the Delivery Service.
2. WHEN the Routing Service calculates a courier delivery route, THE Routing Service SHALL use the courier's assigned hub as the fixed starting point and apply nearest neighbor heuristic to order all delivery points by minimizing total travel distance using lat/lng coordinates.
3. WHEN the Routing Service calculates a courier delivery route, THE Routing Service SHALL return a response containing `courier_id`, `hub_id`, `origin` coordinates, an `optimized_route` array with `sequence`, `tracking_number`, `delivery_id`, `recipient_name`, `address`, `lat`, `lng`, `estimated_arrival`, and `distance_from_prev_km` for each stop, along with `total_stops`, `total_distance_km`, and `estimated_total_duration_minutes`.
4. WHEN the Routing Service receives a courier-route request for a `courier_id` and current date combination that exists in cache, THE Routing Service SHALL return the cached result without recalculating the route.
5. WHEN a cached courier delivery route reaches its TTL of 10 minutes, THE Routing Service SHALL invalidate the cache entry and recalculate the route on the next request.
6. IF a courier has no delivery points assigned for the current date, THEN THE Routing Service SHALL return a response with an empty `optimized_route` array and zero values for totals.
7. IF the Delivery Service is unavailable when fetching delivery points, THEN THE Routing Service SHALL return an error response with a service unavailability message.

---

### Requirement 5

**User Story:** As a system integrator, I want the Routing Service to integrate with Warehouse Service, Pricing Service, and Delivery Service, so that routing data can be consumed by other microservices in the delivery platform.

#### Acceptance Criteria

1. WHEN the Pricing Service requests route information, THE Routing Service SHALL provide `total_distance_km` and `estimated_transit_hours` for use in delivery cost and time estimation.
2. WHEN the Delivery Service assigns a courier, THE Routing Service SHALL serve the GET /routing/courier-route/{courier_id} endpoint with a response time under 500 milliseconds for cached results.
3. WHEN the Routing Service serializes or deserializes route data to or from Redis cache, THE Routing Service SHALL produce output that round-trips to an equivalent data structure without data loss.

---

### Requirement 6

**User Story:** As a system operator, I want the Routing Service to handle high request volumes reliably, so that the service remains available and performant under production load.

#### Acceptance Criteria

1. WHILE the Routing Service is under load of up to 1000 requests per second, THE Routing Service SHALL serve cached route responses within 500 milliseconds at the 95th percentile.
2. WHEN the Redis cache is unavailable, THE Routing Service SHALL fall back to computing routes directly from the database and return a valid response.
3. IF a route calculation request contains malformed query parameters, THEN THE Routing Service SHALL return a 400 error response with a descriptive validation message without crashing the service.
