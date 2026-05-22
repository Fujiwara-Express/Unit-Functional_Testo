#### Data Model

```
payments
- payment_id (PK)
- order_id (FK, unique)
- user_id
- amount
- method
- status: PENDING | SUCCESS | FAILED | REFUNDED
- external_ref
- created_at, updated_at

cod_collections
- collection_id (PK)
- order_id
- courier_id
- amount_collected
- collected_at
- remittance_status: PENDING | REMITTED
```

#### Endpoint API

##### 1. Charge Payment

```
POST /payments/charge
```

**Request**

```json
{
  "order_id": "ORD-20260322-001",
  "amount": 25000,
  "method": "TRANSFER | VIRTUAL_ACCOUNT | QRIS | COD",
  "user_id": "USR123"
}
```

**Response**

```json
{
  "payment_id": "PAY123",
  "order_id": "ORD-20260322-001",
  "status": "SUCCESS | PENDING | FAILED",
  "method": "VIRTUAL_ACCOUNT",
  "va_number": "8808123456789",
  "expired_at": "2026-03-22T20:00:00Z"
}
```

---

##### 2. Confirm COD

```
POST /payments/cod/confirm
```

**Request**

```json
{
  "order_id": "ORD-20260322-001",
  "courier_id": "CR123",
  "amount_collected": 150000
}
```

---

##### 3. Refund

```
POST /payments/refund
```

**Request**

```json
{
  "order_id": "ORD-20260322-001",
  "reason": "ORDER_CANCELLED"
}
```

---

##### 4. Get Payment

```
GET /payments/{payment_id}
```

**Response**

```json
{
  "payment_id": "PAY123",
  "order_id": "ORD-20260322-001",
  "user_id": "USR123",
  "amount": 25000,
  "method": "VIRTUAL_ACCOUNT",
  "status": "PENDING",
  "va_number": "8808123456789",
  "expired_at": "2026-03-22T20:00:00Z",
  "created_at": "2026-03-22T08:00:00Z",
  "updated_at": "2026-03-22T08:00:00Z"
}
```

---

##### 5. Get Payment by Order

```
GET /payments?order_id=ORD-20260322-001
```

**Response**

```json
{
  "payment_id": "PAY123",
  "order_id": "ORD-20260322-001",
  "status": "SUCCESS",
  "method": "VIRTUAL_ACCOUNT",
  "amount": 25000
}
```

---

##### 6. Payment Gateway Callback

Endpoint ini dipanggil oleh payment gateway eksternal (Midtrans / Xendit) untuk memberitahu hasil transaksi. Tidak diekspos ke publik — hanya dapat diakses dari IP whitelist gateway.

```
POST /payments/callback
```

**Request**

```json
{
  "external_ref": "MIDTRANS-TXN-98765",
  "order_id": "ORD-20260322-001",
  "status": "SUCCESS | FAILED | EXPIRED",
  "method": "VIRTUAL_ACCOUNT | QRIS | TRANSFER",
  "amount": 25000,
  "paid_at": "2026-03-22T09:45:00Z",
  "signature": "sha512-hash-string"
}
```

> `signature` wajib divalidasi menggunakan HMAC-SHA512 dengan secret key yang disepakati bersama gateway sebelum memproses update status.

**Response**

```json
{
  "payment_id": "PAY123",
  "order_id": "ORD-20260322-001",
  "status": "SUCCESS",
  "updated_at": "2026-03-22T09:45:00Z"
}
```

**Behavior setelah callback:**
- Status `SUCCESS` → update `payments.status` ke `SUCCESS`, publish event `PAYMENT_SUCCESS` ke Kafka
- Status `FAILED` / `EXPIRED` → update `payments.status` ke `FAILED`, publish event `PAYMENT_FAILED` ke Kafka
- Event `PAYMENT_SUCCESS` dikonsumsi Order Service untuk melanjutkan proses order
- Idempotent: jika `external_ref` sudah pernah diproses, return 200 tanpa update ulang

---