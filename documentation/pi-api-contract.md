# Stock 'n Stash Pi API Contract

## Purpose

This document defines the local interface owned by the Stock 'n Stash Raspberry Pi and how that interface fits into the product's remote-access architecture.

A core product use case is checking home inventory while grocery shopping. Therefore, the production-target architecture must not require the phone to be on the same network as the Raspberry Pi.

The Raspberry Pi owns local edge responsibilities:

- Local inventory persistence for device operation
- Pending recognition results
- Duplicate-scan protection
- The local HTTP API
- SQLite storage
- Local operation when internet/cloud connectivity is unavailable

The Python edge software performs hardware control, recognition, and multi-camera association, then reports completed scan results to the Pi backend.

For the production-target architecture, inventory changes are synchronized between the Pi and a cloud backend. The phone PWA uses the cloud-backed inventory service for remote access away from home. During local prototype/demo development, the PWA may still connect directly to the Pi API on the same network.

The PWA does not access SQLite directly.

---

## Product Architecture

### Production target

```text
Cameras / Load Cells / Hailo
            |
            v
      Python Edge Software
            |
            | local HTTP / JSON
            v
      Pi Inventory API
            |
            v
          SQLite
            |
            | synchronized changes
            v
     Cloud Inventory Service
            ^
            | Internet / JSON
            |
          Phone PWA
```

The Pi remains able to process scans and persist local state when internet access is unavailable. When connectivity returns, unsynchronized inventory changes are sent to the cloud.

The phone does not need direct reachability to the Pi for normal remote inventory viewing while the user is away from home.

### Prototype/local-demo path

```text
Phone PWA
    |
    | local Wi-Fi / HTTP
    v
Pi Inventory API
    |
    v
  SQLite
```

This local path is useful for integration testing and live demonstrations. It is not the complete production access model.

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

# Local Pi API Endpoints

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

## Add Inventory Item Manually

```text
POST /api/inventory
```

Manual inventory addition is supported as a fallback when an item cannot be recognized automatically.

Example request:

```json
{
  "name": "Pop-Tarts",
  "upc": null,
  "quantity": 1,
  "expiration_date": null
}
```

Behavior:

- The backend generates the persistent inventory ID and timestamps.
- The new item is stored in inventory.
- In the production-target architecture, a manual addition made remotely through the phone is written to the cloud inventory service and synchronized back to the Pi.

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

# Connectivity and Synchronization Behavior

## Pi loses internet/cloud connectivity

The Pi should:

- Continue local recognition and inventory operations.
- Persist changes in SQLite.
- Retain unsynchronized changes for later cloud synchronization.
- Resume synchronization when connectivity returns.

## Phone is away from home

The phone should:

- Use the cloud inventory service over the internet.
- Be able to list inventory without direct access to the Pi or home network.
- Send manual additions, edits, removals, and pending-result decisions through the cloud-backed service when those capabilities are exposed remotely.

## Local prototype/demo connection

When the PWA is connected directly to the Pi:

- Display when the Pi is unavailable.
- Do not claim that an edit or correction was saved unless acknowledged.
- Allow retry after reconnection.

## Edge pipeline cannot reach the Pi backend

The Python edge software should:

- Treat the report as unacknowledged.
- Retain the current `scan_id`.
- Allow the report operation to be retried.
- Avoid repeating successful capture or recognition steps.

Because the Pi backend stores processed `scan_id` values, a retry must not create duplicate inventory.

---

# Responsibility Boundaries

| Component | Responsibility |
|---|---|
| Python edge software | Hardware control, detection, recognition, association, scan reporting |
| Pi backend | Local HTTP API, local inventory logic, pending results, duplicate-scan protection, local persistence |
| SQLite | Local operational storage and offline resilience |
| Cloud inventory service | Remotely accessible synchronized inventory/account state |
| Cloud sync component | Transfers acknowledged changes between Pi local state and cloud state, retries after outages, applies conflict rules |
| Phone PWA | Inventory display, remote inventory access, manual additions, pending-result review, user edits |

The phone does not own the authoritative inventory database and does not write directly to SQLite.

---

# Cloud Sync Requirements

The exact cloud provider and synchronization protocol are intentionally separate from the local Pi API implementation, but the product architecture requires:

- Remote inventory reads while the user is away from home.
- Pi-to-cloud synchronization of device-generated inventory changes.
- Cloud-to-Pi synchronization of remote user edits and manual additions.
- Retry after temporary internet outages.
- A defined conflict-resolution strategy.
- Authentication/authorization before exposing a user's inventory remotely.

The Pi should not be exposed directly to the public internet as the normal remote-access mechanism.

---

# Open Design Questions

- What confidence threshold should cause `requires_review` to become `true`?
- Should scanning the same UPC automatically increase the existing inventory quantity?
- How should products without UPCs be matched against existing inventory?
- Should recognition evidence be retained after an uncertain item is confirmed?
- Which cloud provider/service should host synchronized inventory for the prototype and production target?
- What is the minimum authentication model required for the prototype?
- What conflict-resolution rule should apply if the Pi and phone change the same inventory item while disconnected?
- Should product-database metadata be stored in the same SQLite database or a separate cache?

---

# Resolved Design Decisions

- Manual inventory additions are supported as a fallback for items that cannot be recognized.
- The production-target phone experience must support inventory access away from the home network.
- The Pi remains the local edge backend and offline-capable persistence layer.
- Remote phone access uses a cloud-backed synchronized inventory service rather than requiring direct public access to the Pi.

---

# Review Checklist

- [ ] Inventory-item fields are agreed on.
- [ ] Pending-result fields are agreed on.
- [ ] Scan-report format is agreed on.
- [ ] Local inventory API operations are agreed on.
- [ ] Manual-add behavior is agreed on.
- [ ] Pending-result confirmation behavior is agreed on.
- [ ] Error responses are documented.
- [ ] Local disconnection behavior is documented.
- [ ] Remote/cloud synchronization responsibilities are documented.
- [ ] Edge/API owner reviewed the contract.
- [ ] PWA contributor reviewed the contract.
