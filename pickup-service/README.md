
# 3. Pickup Service
## Deskripsi
Mengelola penjadwalan dan eksekusi pickup paket dari lokasi pengirim oleh kurir.
## Penanggung Jawab
Daffa Dhiyaa Candra
## Endpoint API
### 1. Request Pickup
```
POST /pickups
```
**Request**
```json
{
  "order_id": "ORD-20260322-001",
  "user_id": "USR123",
  "pickup_address": "string",
  "pickup_city_code": "BDG",
  "contact_name": "string",
  "contact_phone": "string",
  "notes": "string"
}
```
**Response**
```json
{
  "pickup_id": "PKP123",
  "order_id": "ORD-20260322-001",
  "status": "SCHEDULED",
  "estimated_pickup_time": "2026-03-22T10:00:00Z",
  "created_at": "2026-03-22T08:00:00Z"
}
```
---
### 2. Assign Pickup Courier
```
POST /pickups/{pickup_id}/assign
```
**Request**
```json
{
  "courier_id": "CR123"
}
```
**Response**
```json
{
  "pickup_id": "PKP123",
  "courier_id": "CR123",
  "status": "ASSIGNED"
}
```
---
### 3. Update Pickup Status
```
POST /pickups/{pickup_id}/status
```
**Request**
```json
{
  "courier_id": "CR123",
  "status": "PICKED_UP | FAILED_ATTEMPT | CANCELLED",
  "notes": "string",
  "timestamp": "2026-03-22T10:30:00Z"
}
```
**Response**
```json
{
  "pickup_id": "PKP123",
  "status": "PICKED_UP",
  "timestamp": "2026-03-22T10:30:00Z"
}
```
---
### 4. Get Pickup
```
GET /pickups/{pickup_id}
```
**Response**

```json
{
  "pickup_id": "PKP123",
  "order_id": "ORD-20260322-001",
  "user_id": "USR123",
  "courier_id": "CR123",
  "pickup_address": "string",
  "pickup_city_code": "BDG",
  "contact_name": "string",
  "contact_phone": "string",
  "status": "PICKED_UP",
  "estimated_pickup_time": "2026-03-22T10:00:00Z",
  "notes": "string",
  "created_at": "2026-03-22T08:00:00Z",
  "updated_at": "2026-03-22T10:30:00Z"
}
```
---
### 5. List Pickups
```
GET /pickups?courier_id=CR123&status=SCHEDULED&date=2026-03-22
```
**Response**
```json
[
  {
    "pickup_id": "PKP123",
    "order_id": "ORD-20260322-001",
    "pickup_address": "string",
    "contact_name": "string",
    "contact_phone": "string",
    "status": "SCHEDULED",
    "estimated_pickup_time": "2026-03-22T10:00:00Z"
  }
]
```
---
### 6. Cancel Pickup
```
POST /pickups/{pickup_id}/cancel
```
**Request**
```json
{
  "reason": "string"
}
```
**Response**
```json
{
  "pickup_id": "PKP123",
  "status": "CANCELLED"
}
```
---
## Data Model
```
pickups
- pickup_id (PK)
- order_id (FK, unique)
- user_id (FK)
- courier_id (FK, nullable)
- pickup_address
- pickup_city_code
- contact_name
- contact_phone
- estimated_pickup_time
- status: SCHEDULED | ASSIGNED | PICKED_UP | FAILED_ATTEMPT | CANCELLED
- attempt_count
- notes
- created_at, updated_at
pickup_attempts
- attempt_id (PK)
- pickup_id (FK)
- courier_id
- status: PICKED_UP | FAILED_ATTEMPT
- notes
- timestamp
```
## State
Stateless
## Integrasi
- Order Service (event ORDER_CREATED → jadwalkan pickup)
- Delivery Service (GET /delivery/couriers untuk cari kurir tersedia)
- Tracking Service (event PICKED_UP saat status berhasil)
- Warehouse Service (paket diserahkan ke hub setelah pickup)
- Notification Service (notif ke pengirim saat kurir dalam perjalanan / pickup berhasil)
## Load
~2000 req/sec
---