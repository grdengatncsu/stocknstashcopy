# Stock 'n Stash API Contract

## Status

This document is the revised target contract for GitHub Issue #37. It defines
the shared data model and API behavior for the Python edge pipeline, the Go Pi
service, synchronized Supabase state, and the React PWA.

The target contract intentionally includes behavior that the current local Go
backend does not implement yet. Those differences are listed in
[Implementation Gaps](#implementation-gaps) so contributors can change the
code deliberately rather than treating the target behavior as already shipped.

Example payloads are stored as machine-readable JSON in
[`documentation/fixtures/pi-api`](fixtures/pi-api/README.md).

## Architecture Summary

```text
Python edge pipeline
        |
        | local HTTP / JSON
        v
Go Pi inventory service
        |
        +---- SQLite local state and sync outbox
        |
        | authenticated synchronization
        v
Supabase / PostgreSQL
        ^
        | HTTPS, Auth, RLS, and Realtime
        |
React / TypeScript PWA
```

The Pi continues operating without internet access. Supabase provides the
remotely accessible shared household inventory. The PWA communicates with
Supabase in production and may communicate directly with the Pi during
same-network development and demonstrations.

The complete responsibility and synchronization design is documented in
[`system-architecture.md`](system-architecture.md).

## General Rules

- Local Pi API base path: `/api`.
- Request and response bodies use `application/json` unless the response is
  `204 No Content`.
- Permanent record and synchronization-operation IDs are UUID strings.
- `scan_id` identifies one physical scan and acts as an idempotency key.
- `association_id` identifies one physical item associated across camera views.
- Timestamps use RFC 3339/ISO 8601 UTC strings, such as
  `2026-09-23T14:30:00Z`.
- Unknown optional values are represented as JSON `null`.
- The vision pipeline never invents an expiration date.
- Synchronized records include an integer `version` for optimistic concurrency.
- Deleted synchronized records retain a `deleted_at` tombstone.

## Household and Access Model

One household owns one shared inventory. A permanent household-owner account
provides recovery and administrative control. Family members may join with a
temporary or rotatable household join code and a preferred display name; they
do not receive private inventories.

The one-time product pairing code links a physical Pi device to a household.
After pairing, the Pi uses a separate revocable device credential. Product
pairing codes, household join codes, PWA sessions, and Pi device credentials
are separate credentials and must not be reused for one another.

## Data Models

### Inventory Item

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "household_id": "7681bc26-b22e-4a9a-a4b7-7b3cf0f80a61",
  "name": "Honey Nut Cheerios",
  "upc": "016000275287",
  "quantity": 1,
  "expiration_date": null,
  "version": 4,
  "created_at": "2026-09-23T14:00:00Z",
  "updated_at": "2026-09-23T14:30:00Z",
  "deleted_at": null
}
```

| Field | Type | Required | Description |
|---|---|---:|---|
| `id` | UUID string | Yes | Permanent inventory record ID |
| `household_id` | UUID string | Yes | Household that owns the item |
| `name` | string | Yes | Nonblank product name |
| `upc` | string or null | No | UPC/barcode when known |
| `quantity` | positive integer | Yes | Current inventory quantity |
| `expiration_date` | string or null | No | Known expiration date |
| `version` | positive integer | Yes | Cloud-controlled concurrency version |
| `created_at` | timestamp | Yes | Creation time |
| `updated_at` | timestamp | Yes | Most recent accepted change |
| `deleted_at` | timestamp or null | Yes | Soft-delete time; `null` when active |

Normal inventory lists exclude records whose `deleted_at` is not `null`.
Synchronization includes those tombstones so offline clients do not restore
deleted records.

### Pending Recognition Result

```json
{
  "id": "166092cb-38ce-46ba-851f-a9bd7d885e5c",
  "household_id": "7681bc26-b22e-4a9a-a4b7-7b3cf0f80a61",
  "scan_id": "scan-20260923-001",
  "association_id": "scan-20260923-001:item-2",
  "suggested_name": "Coca-Cola Zero",
  "suggested_upc": null,
  "confidence": 0.68,
  "quantity": 1,
  "version": 1,
  "created_at": "2026-09-23T14:01:00Z",
  "updated_at": "2026-09-23T14:01:00Z",
  "deleted_at": null
}
```

| Field | Type | Required | Description |
|---|---|---:|---|
| `id` | UUID string | Yes | Permanent pending-result ID |
| `household_id` | UUID string | Yes | Household that owns the result |
| `scan_id` | string | Yes | Original scan ID |
| `association_id` | string | Yes | Associated physical item ID |
| `suggested_name` | string | Yes | Best current recognition result |
| `suggested_upc` | string or null | No | Suggested UPC when known |
| `confidence` | number from 0 through 1 | Yes | Recognition confidence |
| `quantity` | positive integer | Yes | Suggested quantity |
| `version` | positive integer | Yes | Cloud-controlled concurrency version |
| `created_at` | timestamp | Yes | Creation time |
| `updated_at` | timestamp | Yes | Most recent accepted change |
| `deleted_at` | timestamp or null | Yes | Resolution/deletion tombstone |

### Scan Report

```json
{
  "scan_id": "scan-20260923-001",
  "items": [
    {
      "association_id": "scan-20260923-001:item-1",
      "name": "Honey Nut Cheerios",
      "upc": "016000275287",
      "confidence": 0.97,
      "quantity": 1,
      "requires_review": false
    },
    {
      "association_id": "scan-20260923-001:item-2",
      "name": "Coca-Cola Zero",
      "upc": null,
      "confidence": 0.68,
      "quantity": 1,
      "requires_review": true
    }
  ]
}
```

Items with `requires_review: false` become inventory records. Items with
`requires_review: true` become pending results. Repeating a successfully
processed `scan_id` must not create additional records.

### Synchronization Operation

Offline Pi and PWA changes are represented by idempotent operations:

```json
{
  "operation_id": "f1efec2b-b3d8-48ab-931f-61fa34c19d0a",
  "household_id": "7681bc26-b22e-4a9a-a4b7-7b3cf0f80a61",
  "record_type": "inventory_item",
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "action": "update",
  "expected_version": 4,
  "created_at": "2026-09-23T14:35:00Z"
}
```

`operation_id` prevents a retry from applying the same change twice.
`expected_version` prevents stale clients from silently overwriting newer data.

## API Endpoints

### System Status

```text
GET /api/status
```

Successful response: `200 OK`.

```json
{
  "status": "ok",
  "sync": {
    "state": "synced",
    "pending_operations": 0,
    "last_successful_sync_at": "2026-09-23T14:35:10Z"
  }
}
```

### Submit Scan Result

```text
POST /api/scans
```

Successful first submission: `200 OK`.

```json
{
  "scan_id": "scan-20260923-001",
  "status": "accepted"
}
```

Successful duplicate submission: `200 OK`.

```json
{
  "scan_id": "scan-20260923-001",
  "status": "already_processed"
}
```

A scan is fully processed or not processed at all. Confirmed items, pending
results, and the processed-scan record are stored atomically.

### List Inventory

```text
GET /api/inventory
```

Successful response: `200 OK`.

```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "household_id": "7681bc26-b22e-4a9a-a4b7-7b3cf0f80a61",
      "name": "Honey Nut Cheerios",
      "upc": "016000275287",
      "quantity": 1,
      "expiration_date": null,
      "version": 4,
      "created_at": "2026-09-23T14:00:00Z",
      "updated_at": "2026-09-23T14:30:00Z",
      "deleted_at": null
    }
  ],
  "sync_cursor": "2026-09-23T14:30:00Z"
}
```

An empty inventory returns `"items": []`.

### Add Inventory Item

```text
POST /api/inventory
```

```json
{
  "name": "Whole Milk",
  "upc": "012345678901",
  "quantity": 1,
  "expiration_date": "2026-10-01"
}
```

Successful response: `201 Created` with the created inventory item.

### Edit Inventory Item

```text
PATCH /api/inventory/{id}
```

```json
{
  "expected_version": 4,
  "quantity": 2,
  "expiration_date": null
}
```

Successful response: `200 OK` with the updated item at version `5`.

PATCH fields use three-state behavior:

- Omitted field: keep the current value.
- Supplied non-null value: replace the current value.
- Supplied `null`: clear a nullable field.

`upc` and `expiration_date` may be cleared with `null`. `name` and `quantity`
may not be `null`; supplied names must be nonblank and supplied quantities must
be positive.

If `expected_version` is stale, the response is `409 Conflict`.

### Remove Inventory Item

```text
DELETE /api/inventory/{id}
If-Match: "4"
```

Successful response: `204 No Content`. The record receives `deleted_at` and a
new version instead of being immediately erased. A stale `If-Match` value
returns `409 Conflict`.

### List Pending Results

```text
GET /api/pending
```

Successful response: `200 OK`.

```json
{
  "items": [
    {
      "id": "166092cb-38ce-46ba-851f-a9bd7d885e5c",
      "household_id": "7681bc26-b22e-4a9a-a4b7-7b3cf0f80a61",
      "scan_id": "scan-20260923-001",
      "association_id": "scan-20260923-001:item-2",
      "suggested_name": "Coca-Cola Zero",
      "suggested_upc": null,
      "confidence": 0.68,
      "quantity": 1,
      "version": 1,
      "created_at": "2026-09-23T14:01:00Z",
      "updated_at": "2026-09-23T14:01:00Z",
      "deleted_at": null
    }
  ],
  "sync_cursor": "2026-09-23T14:01:00Z"
}
```

No pending results returns `"items": []`.

### Resolve Pending Result

```text
POST /api/pending/{id}/resolve
```

```json
{
  "expected_version": 1,
  "name": "Coca-Cola Zero Sugar",
  "upc": "049000051236",
  "quantity": 1,
  "expiration_date": null
}
```

Successful response: `200 OK` with the created inventory item. Omitted
correction fields use the pending result's suggested values. The pending result
receives a tombstone atomically with inventory creation. A stale
`expected_version` returns `409 Conflict`.

## Success Status Summary

| Endpoint | Status |
|---|---:|
| `GET /api/status` | `200 OK` |
| `GET /api/inventory` | `200 OK` |
| `POST /api/inventory` | `201 Created` |
| `PATCH /api/inventory/{id}` | `200 OK` |
| `DELETE /api/inventory/{id}` | `204 No Content` |
| `GET /api/pending` | `200 OK` |
| `POST /api/pending/{id}/resolve` | `200 OK` |
| `POST /api/scans` | `200 OK` |

Scan submission remains `200 OK` because it is an idempotent processing command
that may create multiple records or acknowledge an already-processed scan.

## Error Format

```json
{
  "error": {
    "code": "invalid_request",
    "message": "quantity must be greater than zero"
  }
}
```

| HTTP status | Error code | Meaning |
|---:|---|---|
| `400` | `invalid_request` | Malformed JSON or invalid fields |
| `401` | `unauthorized` | Missing or invalid authentication |
| `403` | `forbidden` | Authenticated identity cannot access the household or operation |
| `404` | `not_found` | Requested record does not exist or is not visible |
| `409` | `version_conflict` | `expected_version` or `If-Match` is stale |
| `500` | `internal_error` | Unexpected backend failure |
| `503` | `service_unavailable` | Required dependency is temporarily unavailable |

### Implemented Invalid-Request Messages

The current Go handlers and tests use these messages. Contract changes should
update implementation, tests, fixtures, and this table together.

| Endpoint | Invalid condition | `error.message` |
|---|---|---|
| `POST /api/inventory` | Malformed JSON | `failed to parse request body` |
| `POST /api/inventory` | Missing or blank name | `name is required` |
| `POST /api/inventory` | Missing, zero, or negative quantity | `quantity must be greater than zero` |
| `PATCH /api/inventory/{id}` | Malformed JSON | `invalid JSON request body` |
| `PATCH /api/inventory/{id}` | Supplied name is blank | `name is required` |
| `PATCH /api/inventory/{id}` | Supplied quantity is zero or negative | `quantity must be greater than zero` |
| `POST /api/pending/{id}/resolve` | Malformed JSON | `failed to parse request body` |
| `POST /api/pending/{id}/resolve` | Supplied name is blank | `name is required` |
| `POST /api/pending/{id}/resolve` | Supplied quantity is zero or negative | `quantity must be greater than zero` |
| `POST /api/scans` | Malformed JSON | `failed to parse scan report` |
| `POST /api/scans` | Missing or blank scan ID | `scan_id is required` |
| `POST /api/scans` | Missing or empty items | `items are required` |
| `POST /api/scans` | Missing or blank association ID | `association_id cannot be blank` |
| `POST /api/scans` | Blank item name | `name cannot be blank` |
| `POST /api/scans` | Confidence outside 0 through 1 | `confidence must be between 0 and 1` |
| `POST /api/scans` | Zero or negative item quantity | `item quantity must be greater than zero` |
| `POST /api/scans` | Repeated association ID | `item association_id must be unique` |

## Offline and Synchronization Behavior

### Pi

- Writes local SQLite state before acknowledging a local operation.
- Adds synchronized changes to an outbox identified by operation UUID.
- Uploads immediately when online and retries unacknowledged operations.
- Synchronizes at startup and after reconnecting.
- Polls incrementally for cloud changes approximately every two seconds, with a
  slower recovery check as a fallback.
- Never duplicates an operation solely because a response was lost.

### PWA

- Caches the application shell and latest household state.
- Stores offline additions, edits, resolutions, and deletions in IndexedDB.
- Updates the interface optimistically and labels unsynchronized changes.
- Sends queued operations after reconnecting and removes them only after
  acknowledgment.
- Receives Supabase Realtime updates while online.
- Displays `offline`, `pending`, `syncing`, `synced`, and `failed` states plus
  the last successful synchronization time.

### Conflict Handling

- Accepted changes increment `version`.
- Stale versions return `409 Conflict`.
- The client refreshes the current record before retrying.
- Device timestamps are not used to silently choose a winner.
- A newer accepted tombstone prevents an older offline copy from restoring a
  deleted record.

## Responsibility Boundaries

| Component | Responsibility |
|---|---|
| Python edge pipeline | Hardware control, recognition, association, scan reporting |
| Go Pi service | Local API, SQLite persistence, pending results, scan idempotency, outbox, cloud synchronization |
| SQLite | Offline-capable local state and queued operations |
| Supabase/PostgreSQL | Shared household state, record versions, tombstones, authentication, authorization, Realtime |
| React PWA | Inventory UI, review/correction, offline cache and queue, synchronization status |

Privileged Supabase credentials must never be shipped in the PWA. The PWA uses
a publishable key with authenticated sessions and row-level security. The Pi
uses a separate revocable device credential.

## Implementation Status and Gaps

The local API now implements the core offline contract. The remaining rows keep
cloud and household work visible without making completed local work look open:

| Area | Current local Go backend | Remaining work |
|---|---|---|
| Permanent IDs | Inventory and pending records use UUIDs; scan and association IDs remain separate | Cloud operation IDs are still needed |
| List responses | `{ "items": [...], "sync_cursor": ... }` envelopes implemented | Cursor-filtered incremental reads are not implemented |
| Nullable PATCH fields | Omitted versus explicit `null` implemented with `Optional` | None for the local endpoints |
| Deletion | Versioned inventory tombstones implemented | Tombstone upload and retention policy remain |
| Concurrency | `expected_version`/`If-Match` and `409` responses implemented | Reconcile versions with cloud writes |
| Household scope | No household fields | Every synchronized record belongs to a household |
| Cloud state | Not implemented | Supabase/PostgreSQL with Auth, RLS, and Realtime |
| Synchronization | Not implemented | SQLite outbox, retries, incremental pull, operation UUIDs |
| Pairing and profiles | Not implemented | One-time product pairing and rotatable household join codes |
| Status endpoint | `{ "status": "ok" }` | Includes synchronization health metadata |

Issue #37's local implementation format has been reconciled and tested. Cloud
completion remains separate work rather than a reason to hold the local API
open indefinitely.

## Open Product Questions Outside This Contract

- What confidence threshold sets `requires_review`?
- Should repeated scans of the same UPC automatically merge quantities?
- How should products without UPCs be matched against existing inventory?
- How long should recognition evidence be retained?
- Should product-catalog metadata share the inventory database or use a cache?

These questions do not change the transport, authentication, offline, or
synchronization decisions in this contract.

## Review Checklist

- [x] Inventory-item fields are agreed on.
- [x] Pending-result fields are agreed on.
- [x] Scan-report format is agreed on.
- [x] Inventory API operations are agreed on.
- [x] Pending-result confirmation behavior is agreed on.
- [x] Error response shape and validation behavior are documented.
- [x] Offline and synchronization behavior are documented.
- [x] Production remote-access responsibilities are documented.
- [ ] Edge/API owner reviewed the revised contract and fixtures.
- [ ] PWA contributor reviewed the revised contract and fixtures.
