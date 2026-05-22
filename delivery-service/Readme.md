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