# 9. Notification Service

## Deskripsi

Mengirim notifikasi ke user berdasarkan event.

## Penanggung Jawab

Daffa Dhiyaa Candra

## Endpoint API

### 1. Send Notification (manual/admin)

```
POST /notifications/send
```

**Request**

```json
{
  "user_id": "USR123",
  "channel": "PUSH |  EMAIL | WHATSAPP",
  "template_id": "PACKAGE_OUT_FOR_DELIVERY",
  "variables": { "tracking_number": "TRK123" }
}
```

**Response**

```json
{
  "notification_id": "NOTIF123",
  "status": "SENT",
  "channel": "PUSH"
}
```

> Notifikasi otomatis dikonsumsi via Kafka topic `tracking.status_updated`

---

### 2. List Templates

```
GET /notifications/templates
```

**Response**

```json
[
  {
    "template_id": "PACKAGE_OUT_FOR_DELIVERY",
    "event_type": "OUT_FOR_DELIVERY",
    "channel": "PUSH",
    "subject": "Paket Sedang Diantar",
    "body_template": "Paket {{tracking_number}} sedang dalam perjalanan ke Anda."
  }
]
```

---

### 3. Create Template

```
POST /notifications/templates
```

**Request**

```json
{
  "event_type": "OUT_FOR_DELIVERY",
  "channel": "PUSH",
  "subject": "Paket Sedang Diantar",
  "body_template": "Paket {{tracking_number}} sedang dalam perjalanan ke Anda."
}
```

**Response**

```json
{
  "template_id": "PACKAGE_OUT_FOR_DELIVERY",
  "status": "CREATED"
}
```

---

### 4. Update Template

```
PUT /notifications/templates/{template_id}
```

**Request**

```json
{
  "subject": "string",
  "body_template": "string"
}
```

**Response**

```json
{
  "template_id": "PACKAGE_OUT_FOR_DELIVERY",
  "status": "UPDATED"
}
```

---

## Data Model

```
notification_logs
- notif_id (PK)
- user_id
- tracking_number
- channel
- template_id
- message
- status: SENT | FAILED | PENDING
- sent_at

notification_templates
- template_id (PK)
- event_type
- channel
- subject
- body_template
```

## State

Stateless (fire-and-forget via queue)

## Integrasi

- Tracking Service (Kafka event STATUS_UPDATED)
- External: Firebase (push), SendGrid (email), WhatsApp Business API

## Load

~5000 event/sec

---