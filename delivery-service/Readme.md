#### Data Model

```
couriers
- courier_id (PK)
- name
- phone
- hub_id
- vehicle_type
- is_available
- current_lat, current_lng

delivery_jobs
- job_id (PK)
- tracking_number
- courier_id
- hub_id
- status: ASSIGNED | OUT_FOR_DELIVERY | DELIVERED | FAILED | RETURNED
- attempt_count
- proof_photo_url
- recipient_name
- notes
- assigned_at
- completed_at
```

#### Endpoint API

##### 1. Assign Courier

```
POST /delivery/assign
```

**Request**

```json
{
  "tracking_number": "TRK123",
  "hub_id": "HUB_JKT",
  "courier_id": "CR123"
}
```

**Response**

```json
{
  "delivery_id": "DLV123",
  "tracking_number": "TRK123",
  "courier_id": "CR123",
  "status": "OUT_FOR_DELIVERY",
  "delivery_route": {
    "total_stops": 3,
    "total_distance_km": 12.4,
    "estimated_total_duration_minutes": 140,
    "optimized_route": [
      {
        "sequence": 1,
        "tracking_number": "TRK123",
        "delivery_id": "DLV123",
        "recipient_name": "Budi Santoso",
        "address": "Jl. Kebon Jeruk No. 12, Jakarta Barat",
        "lat": -6.1944,
        "lng": 106.8229,
        "estimated_arrival": "2026-03-24T09:30:00Z",
        "distance_from_prev_km": 3.2
      },
      {
        "sequence": 2,
        "tracking_number": "TRK456",
        "delivery_id": "DLV456",
        "recipient_name": "Siti Rahayu",
        "address": "Jl. Puri Indah No. 5, Jakarta Barat",
        "lat": -6.2146,
        "lng": 106.7651,
        "estimated_arrival": "2026-03-24T10:15:00Z",
        "distance_from_prev_km": 4.7
      },
      {
        "sequence": 3,
        "tracking_number": "TRK789",
        "delivery_id": "DLV789",
        "recipient_name": "Ahmad Fauzi",
        "address": "Jl. Raya Pos No. 88, Jakarta Barat",
        "lat": -6.2301,
        "lng": 106.7890,
        "estimated_arrival": "2026-03-24T11:00:00Z",
        "distance_from_prev_km": 4.5
      }
    ]
  }
}
```

> Setelah assign, Delivery Service memanggil `GET /routing/courier-route/{courier_id}` untuk mendapatkan rute optimal seluruh delivery point kurir tersebut, lalu menyertakan hasilnya di response.

---

##### 2. Update Delivery Status

```
POST /delivery/status
```

**Request**

```json
{
  "tracking_number": "TRK123",
  "courier_id": "CR123",
  "status": "DELIVERED | FAILED_ATTEMPT | RETURNED",
  "notes": "Penerima tidak ada di tempat",
  "proof_photo_url": "https://...",
  "recipient_name": "string",
  "timestamp": "2026-03-24T14:00:00Z"
}
```

---

##### 3. Get Courier Jobs

```
GET /delivery/courier/{courier_id}/jobs
```

---

##### 4. Get Delivery Detail

```
GET /delivery/{delivery_id}
```

**Response**

```json
{
  "delivery_id": "DLV123",
  "tracking_number": "TRK123",
  "courier_id": "CR123",
  "hub_id": "HUB_JKT",
  "status": "DELIVERED",
  "attempt_count": 1,
  "proof_photo_url": "https://...",
  "recipient_name": "string",
  "notes": "string",
  "assigned_at": "2026-03-24T08:00:00Z",
  "completed_at": "2026-03-24T14:00:00Z"
}
```

---

##### 5. List Couriers

```
GET /delivery/couriers?hub_id=HUB_JKT&is_available=true&city_code=BDG
```

**Response**

```json
[
  {
    "courier_id": "CR123",
    "name": "string",
    "phone": "string",
    "hub_id": "HUB_JKT",
    "vehicle_type": "MOTOR",
    "is_available": true,
    "current_lat": -6.2088,
    "current_lng": 106.8456
  }
]
```

---

##### 6. Register Courier

```
POST /delivery/couriers
```

**Request**

```json
{
  "name": "string",
  "phone": "string",
  "hub_id": "HUB_JKT",
  "vehicle_type": "MOTOR"
}
```

**Response**

```json
{
  "courier_id": "CR123",
  "status": "CREATED"
}
```

---

##### 7. Update Courier

```
PATCH /delivery/couriers/{courier_id}
```

**Request**

```json
{
  "is_available": true,
  "current_lat": -6.2088,
  "current_lng": 106.8456
}
```

**Response**

```json
{
  "courier_id": "CR123",
  "status": "UPDATED"
}
```

---

#### Running Tests

##### Unit Tests

Unit tests live in `service/` and use mocks — no database required.

```bash
go test ./service/...
```

Verbose output:

```bash
go test -v ./service/...
```

---

##### Functional Tests

Functional tests live in `functional/` and require a running PostgreSQL instance. They use the `functional` build tag and will be skipped unless the tag is provided.

**Prerequisites:**
- PostgreSQL running with a `delivery_test` database
- The `couriers` and `delivery_jobs` tables must exist

**Environment Variables:**

| Variable | Default |
|---|---|
| `TEST_DB_HOST` | `localhost` |
| `TEST_DB_PORT` | `5432` |
| `TEST_DB_USER` | `postgres` |
| `TEST_DB_PASSWORD` | *(empty)* |
| `TEST_DB_NAME` | `delivery_test` |

**Run:**

```bash
go test -v -tags functional ./functional/...
```

With custom DB credentials:

```bash
TEST_DB_USER=postgres TEST_DB_PASSWORD=yourpassword go test -v -tags functional ./functional/...
```

On Windows (CMD):

```cmd
set TEST_DB_HOST=localhost
set TEST_DB_PORT=5432
set TEST_DB_USER=postgres
set TEST_DB_PASSWORD=yourpassword
set TEST_DB_NAME=delivery_test
go test -v -tags functional ./functional/...
```

On Windows (PowerShell):

```powershell
Set-Item Env:TEST_DB_HOST "localhost"
Set-Item Env:TEST_DB_PORT "5432"
Set-Item Env:TEST_DB_USER "postgres"
Set-Item Env:TEST_DB_PASSWORD "yourpassword"
Set-Item Env:TEST_DB_NAME "delivery_test"
go test -v -tags functional ./functional/...
```

---

##### Run All Tests

```bash
go test -v -tags functional ./...
```

> Note: This requires the PostgreSQL database to be available for functional tests to pass.
