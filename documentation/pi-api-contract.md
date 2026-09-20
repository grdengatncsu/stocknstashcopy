```md
# Stock 'n Stash Pi API Contract

## Purpose

This document defines the interface between the Stock 'n Stash Raspberry Pi software and the phone PWA.

The Raspberry Pi owns:

- Inventory persistence
- Pending recognition results
- Duplicate-scan protection
- The local HTTP API
- SQLite storage

The Python edge software performs hardware control, recognition, and multi-camera association, then reports completed scan results to the Pi backend.

The PWA displays inventory and pending results and allows the user to confirm, correct, edit, or remove inventory records.

The PWA does not access SQLite directly.

---

## System Architecture

```text
Cameras / Load Cells / Hailo
            |
            v
      Python Edge Software
            |
            | HTTP / JSON
            v
      Pi Inventory API
            |
            v
          SQLite
            |
            | HTTP / JSON
            v
          Phone PWA
```

---

## General API Rules

Base path:

```text
/api
```

Data format:

```text
application/json
```

Identifiers:

- `scan_id` uniquely identifies one physical scan.
- `association_id` identifies one physical item detected during a scan.
- Inventory and pending-result records receive persistent backend-generated IDs.

Dates and timestamps:

- Timestamps use ISO 8601 format.
- Unknown expiration dates are represented as `null`.
- The vision system does not invent an expiration date when one is not known.

---

# Data Models

## Inventory Item

```json
{
  "id": "item-123",
  "name": "Honey Nut Cheerios",
  "upc": "016000275287",
  "quantity": 1,
  "expiration_date": null,
  "created_at": "2026-09-20T18:00:00Z",
  "updated_at": "2026-09-20T18:00:00Z"
}
```

| Field | Type | Required | Description |
|---|---|---:|---|
| `id` | string | Yes | Persistent backend-generated inventory ID |
| `name` | string | Yes | Product name |
| `upc` | string or null | No | UPC or barcode if known |
| `quantity` | integer | Yes | Current inventory quantity |
| `expiration_date` | string or null | No | Expiration date if known |
| `created_at` | string | Yes | Creation timestamp |
| `updated_at` | string | Yes | Last modification timestamp |

---

## Pending Recognition Result

```json
{
  "id": "pending-456",
  "scan_id": "scan-123",
  "association_id": "scan-123:item-2",
  "suggested_name": "Coca-Cola Zero",
  "suggested_upc": null,
  "confidence": 0.68,
  "quantity": 1,
  "created_at": "2026-09-20T18:01:00Z"
}
```

| Field | Type | Required | Description |
|---|---|---:|---|
| `id` | string | Yes | Backend-generated pending-result ID |
| `scan_id` | string | Yes | Original scan ID |
| `association_id` | string | Yes | Physical item ID from the edge pipeline |
| `suggested_name` | string | Yes | Best current recognition result |
| `suggested_upc` | string or null | No | UPC if known |
| `confidence` | number | Yes | Recognition confidence |
| `quantity` | integer | Yes | Suggested quantity |
| `created_at` | string | Yes | Creation timestamp |

---

## Scan Report

```json
{
  "scan_id": "scan-123",
  "items": [
    {
      "association_id": "scan-123:item-1",
      "name": "Honey Nut Cheerios",
      "upc": "016000275287",
      "confidence": 0.97,
      "quantity": 1,
      "requires_review": false
    },
    {
      "association_id": "scan-123:item-2",
      "name": "Coca-Cola Zero",
      "upc": null,
      "confidence": 0.68,
      "quantity": 1,
      "requires_review": true
    }
  ]
}
```

Items with `requires_review: false` may be added directly to inventory.

Items with `requires_review: true` are stored as pending results until the user confirms or corrects them.

---

# API Endpoints

## Submit Scan Result

```text
POST /api/scans
```

Example request:

```json
{
  "scan_id": "scan-123",
  "items": [
    {
      "association_id": "scan-123:item-1",
      "name": "Honey Nut Cheerios",
      "upc": "016000275287",
      "confidence": 0.97,
      "quantity": 1,
      "requires_review": false
    }
  ]
}
```

Example response:

```json
{
  "scan_id": "scan-123",
  "status": "accepted"
}
```

Behavior:

- Confirmed items are added to inventory.
- Uncertain items are added to the pending-results list.
- Repeated `scan_id` values must not duplicate inventory.

---

## List Inventory

```text
GET /api/inventory
```

Example response:

```json
{
  "items": [
    {
      "id": "item-123",
      "name": "Honey Nut Cheerios",
      "upc": "016000275287",
      "quantity": 1,
      "expiration_date": null,
      "created_at": "2026-09-20T18:00:00Z",
      "updated_at": "2026-09-20T18:00:00Z"
    }
  ]
}
```

---

## List Pending Results

```text
GET /api/pending
```

Example response:

```json
{
  "items": [
    {
      "id": "pending-456",
      "scan_id": "scan-123",
      "association_id": "scan-123:item-2",
      "suggested_name": "Coca-Cola Zero",
      "suggested_upc": null,
      "confidence": 0.68,
      "quantity": 1,
      "created_at": "2026-09-20T18:01:00Z"
    }
  ]
}
```

---

## Resolve Pending Result

```text
POST /api/pending/{id}/resolve
```

Example request:

```json
{
  "name": "Coca-Cola Zero Sugar",
  "upc": "049000051236",
  "quantity": 1,
  "expiration_date": null
}
```

Behavior:

- The pending result is removed from the pending list.
- The confirmed or corrected product is added to inventory.

---

## Edit Inventory Item

```text
PATCH /api/inventory/{id}
```

Example request:

```json
{
  "name": "Honey Nut Cheerios",
  "quantity": 2,
  "expiration_date": null
}
```

The backend updates only the fields provided in the request.

---

## Remove Inventory Item

```text
DELETE /api/inventory/{id}
```

---

## System Status

```text
GET /api/status
```

Example response:

```json
{
  "status": "ok"
}
```

---

# Error Format

```json
{
  "error": {
    "code": "invalid_request",
    "message": "quantity must be greater than zero"
  }
}
```

| HTTP Code | Meaning |
|---|---|
| `200` | Request succeeded |
| `201` | Resource created |
| `400` | Invalid request |
| `404` | Resource not found |
| `409` | Request conflicts with existing state |
| `500` | Internal backend error |
| `503` | Required service temporarily unavailable |

---

# Duplicate Scan Protection

`scan_id` acts as an idempotency key.

If the Python edge software retries a scan report using a `scan_id` that has already been processed, the backend must not add the groceries a second time.

Example response:

```json
{
  "scan_id": "scan-123",
  "status": "already_processed"
}
```

The duplicate request should still receive a successful acknowledgment so the edge state machine can continue.

---

# Temporary Disconnection Behavior

## PWA Cannot Reach the Pi

The PWA should:

- Display that the Pi is unavailable.
- Not claim that an edit or correction was saved.
- Allow the user to retry after reconnection.

## Edge Pipeline Cannot Reach the Backend

The Python edge software should:

- Treat the report as unacknowledged.
- Retain the current `scan_id`.
- Allow the report operation to be retried.
- Avoid repeating successful capture or recognition steps.

Because the backend stores processed `scan_id` values, a retry must not create duplicate inventory.

---

# Responsibility Boundaries

| Component | Responsibility |
|---|---|
| Python edge software | Hardware control, detection, recognition, association, scan reporting |
| Pi backend | HTTP API, inventory logic, pending results, duplicate-scan protection |
| SQLite | Persistent storage |
| Phone PWA | Inventory display, pending-result review, user edits |

---

# Open Design Questions

- What confidence threshold should cause `requires_review` to become `true`?
- Should scanning the same UPC automatically increase the existing inventory quantity?
- How should products without UPCs be matched against existing inventory?
- Should recognition evidence be retained after an uncertain item is confirmed?
- Should manual inventory additions be supported by the PWA?
- Should product-database metadata be stored in the same SQLite database or a separate cache?

---

# Review Checklist

- [ ] Inventory-item fields are agreed on.
- [ ] Pending-result fields are agreed on.
- [ ] Scan-report format is agreed on.
- [ ] Inventory API operations are agreed on.
- [ ] Pending-result confirmation behavior is agreed on.
- [ ] Error responses are documented.
- [ ] Temporary-disconnection behavior is documented.
- [ ] Edge/API owner reviewed the contract.
- [ ] PWA contributor reviewed the contract.
```