# Stock 'n Stash System Architecture

## Purpose

This document defines the production-target architecture agreed for GitHub
Issue #37. It explains how the edge pipeline, Pi service, SQLite database,
Supabase cloud layer, and React PWA divide responsibility while supporting
offline operation and remote inventory access.

The matching data and HTTP contract is documented in
[`pi-api-contract.md`](pi-api-contract.md).

## Production Target

```text
┌──────────────────────── Stock 'n Stash device ────────────────────────┐
│                                                                       │
│  Cameras / load cells / lighting / Hailo                              │
│                         │                                             │
│                         v                                             │
│                Python edge pipeline                                   │
│                         │ local HTTP / JSON                           │
│                         v                                             │
│                   Go Pi service                                       │
│                         │                                             │
│                         v                                             │
│             SQLite state + synchronization outbox                     │
│                         │                                             │
└─────────────────────────┼─────────────────────────────────────────────┘
                          │ authenticated HTTPS
                          v
               Supabase / PostgreSQL cloud
              Auth + RLS + REST + Realtime
                          ^
                          │ Internet
                          v
                 React / TypeScript PWA
               IndexedDB offline cache/queue
```

The phone does not need direct network access to the Pi in production. The PWA
reads and writes shared household state through Supabase. Same-network direct
Pi access remains available for local integration and demonstrations.

## Technology Stack

| Layer | Technology |
|---|---|
| Edge and hardware control | Python |
| Accelerated inference | Hailo-8L |
| Pi HTTP API and sync worker | Go |
| Pi persistence and outbox | SQLite |
| Cloud database and APIs | Supabase / PostgreSQL |
| Cloud authentication and authorization | Supabase Auth and PostgreSQL row-level security |
| Phone application | React, TypeScript/TSX, Vite, CSS, semantic HTML |
| PWA capability | Web app manifest, service worker, HTTPS, IndexedDB |
| Source control and CI | GitHub and GitHub Actions |

Supabase Free is sufficient for the prototype's expected data and user volume.
Free projects may pause after inactivity, so the project must be opened and
verified before demonstrations.

## Component Responsibilities

### Python Edge Pipeline

- Reads load-cell, camera, lighting, and other hardware interfaces.
- Controls the scan state machine.
- Runs recognition and multi-camera association.
- Produces one idempotent scan report.
- Retries an unacknowledged report without repeating capture or recognition.

The edge pipeline does not write SQLite or Supabase directly.

### Go Pi Service

- Owns the local HTTP/JSON API.
- Validates scan reports and inventory operations.
- Stores confirmed inventory, pending results, scan IDs, and tombstones.
- Writes local changes to SQLite before acknowledging them.
- Maintains an idempotent synchronization outbox.
- Pushes local changes to Supabase and pulls remote changes incrementally.
- Exposes synchronization health through the status endpoint.

The Pi continues accepting local scan reports while the internet is down.

### SQLite

- Stores the Pi's offline-capable copy of household state.
- Stores processed scan IDs for duplicate protection.
- Stores queued synchronization operations and retry state.
- Retains cloud synchronization cursors and last-success timestamps.

SQLite is the device's local durable state, not the production remote-access
endpoint.

### Supabase / PostgreSQL

- Stores the shared cloud representation of household inventory.
- Stores households, profiles, paired devices, pending results, tombstones, and
  record versions.
- Authenticates the household owner and joined phone sessions.
- Applies row-level security by household.
- Issues or validates the trusted flow that exchanges a one-time product code
  for a revocable Pi credential.
- Provides REST access and Realtime change delivery to the PWA.
- Rejects stale writes through optimistic concurrency rules.

### React PWA

- Displays inventory, pending results, connection state, and sync status.
- Supports manual inventory creation, edits, resolution, and deletion.
- Caches the application shell and most recently synchronized household data.
- Queues offline writes in IndexedDB.
- Applies optimistic UI changes while visibly marking pending operations.
- Sends queued operations after reconnection.
- Receives Supabase Realtime updates while online.

## Household Model

One household owns one shared inventory. The prototype does not create separate
private inventories for individual family members.

```text
Permanent household-owner account
                │
                v
            Household
          /     |      \
         v      v       v
   Pi device  Phone   Phone
              profile profile
```

The household owner uses email/password authentication for recovery and
administrative actions. Other family members may join with a temporary or
rotatable household join code and provide only a preferred display name.
Joined phones receive distinct, revocable sessions even though the underlying
inventory is shared.

## Product Pairing

The product pairing code connects hardware to a household. It is separate from
the household join code used by family phones.

```text
1. Owner signs in or creates a household.
2. Owner scans the product QR code or enters its printed code.
3. Trusted cloud logic verifies that the code is valid and unclaimed.
4. The product is assigned to the owner's household.
5. The Pi receives a revocable device credential.
6. The one-time pairing code is permanently invalidated.
```

Requirements:

- QR codes include a human-readable fallback code.
- Pairing codes are single-use and are never permanent API credentials.
- The owner can revoke or replace a paired Pi.
- Ownership transfer requires revoking the old household assignment before the
  product can be paired again.
- Privileged Supabase credentials are never embedded in the PWA.

## Family Phone Joining

```text
1. Owner displays or shares a temporary household join code.
2. Family member opens the PWA and enters or scans the code.
3. Family member chooses a preferred display name.
4. Supabase creates a distinct phone session/profile in the household.
5. Row-level security grants access only to that household's records.
```

Join codes are temporary or rotatable. The owner can remove a phone/profile and
rotate the join code. Preferred names provide attribution and device management
but are not security identities by themselves.

## Local Write and Upload Flow

```text
Local scan or Pi API operation
              │
              v
Validate request and expected version
              │
              v
SQLite transaction:
  update local record
  insert outbox operation UUID
              │
              v
Acknowledge local request
              │
              v
Upload immediately when online
              │
      ┌───────┴────────┐
      v                v
  accepted          unavailable
      │                │
      v                v
remove/complete    retain and retry
outbox entry       with backoff
```

Writing the data and outbox entry in one SQLite transaction prevents a local
change from being acknowledged without being queued for synchronization.

## Cloud-to-Pi Flow

- The Pi synchronizes at startup and after network reconnection.
- The Pi requests records changed after its last cloud cursor approximately
  every two seconds.
- A slower recovery pass detects missed incremental updates.
- Pulled records, versions, and tombstones are applied atomically to SQLite.
- The Pi records `last_successful_sync_at` only after a complete accepted batch.

The two-second poll provides near-real-time phone-to-Pi updates without making
the Go service depend on a custom permanent Realtime WebSocket client.

## PWA Offline Flow

```text
User performs write while offline
              │
              v
Store operation in IndexedDB
              │
              v
Apply optimistic UI update
and show "pending"
              │
              v
Browser reconnects
              │
              v
Submit queued operations in order
              │
      ┌───────┴─────────┐
      v                 v
  accepted          409 conflict
      │                 │
      v                 v
mark synced       refresh record and
                  ask user to retry
```

Queued operations contain an operation UUID, record UUID, expected version,
requested change, creation time, and sync state. The queue removes an operation
only after acknowledgment.

## Identity and Idempotency

- Permanent database records use UUIDs.
- Synchronization operations use UUIDs.
- `scan_id` remains the idempotency key for physical scan reports.
- `association_id` remains the physical-item traceability key within a scan.
- Retried operation UUIDs and scan IDs do not apply changes twice.

UUID record identity is independent of recognition traceability.

## Versioning and Conflicts

Every synchronized record includes an integer `version`. Clients submit the
version they last observed. Supabase accepts the change only if that version is
still current, then increments it.

Stale changes return `409 Conflict`. Clients refresh the record before retrying
instead of silently overwriting newer state. Timestamps support display and
incremental queries but do not determine conflict winners because device clocks
may differ.

## Deletion

Inventory and pending-result deletion uses `deleted_at` tombstones:

- Normal reads exclude tombstones.
- Synchronization includes tombstones.
- Deletes increment the record version.
- An accepted tombstone prevents an older offline copy from restoring a record.
- Prototype 1 retains tombstones indefinitely because the records are small and
  retention simplifies recovery.

## Synchronization Health

The Pi and PWA expose these states:

| State | Meaning |
|---|---|
| `offline` | Network or cloud is unavailable |
| `pending` | Local changes are waiting to upload |
| `syncing` | A synchronization attempt is active |
| `synced` | No known pending changes remain |
| `failed` | A non-transient error needs attention |

The UI displays the last successful synchronization time. The cloud stores
`last_seen_at` for the Pi and joined phones so the owner can identify stale or
lost devices.

## Security Boundaries

- PWA: publishable Supabase key plus a user session; access restricted by RLS.
- Pi: separately issued device credential scoped to one household.
- Trusted cloud logic: privileged operations such as product claiming,
  credential issuance, ownership transfer, and code rotation.
- Pairing code: single-use claim secret, never an ongoing credential.
- Household join code: temporary/rotatable invitation, never a privileged key.
- Service-role or other privileged secrets: trusted server/cloud code only.

Every exposed Supabase table must enable row-level security and include tests
for allowed and denied household access.

## Development and Demonstration Mode

For local integration, the PWA may call the Pi API on the same network. This
mode does not replace the production cloud path and must be clearly labeled in
configuration.

The production PWA uses HTTPS. Installability requires a web app manifest and
secure hosting. Offline behavior must be tested independently from the browser
being merely unable to reach the Pi.

## Delivery Sequence

1. Commit the revised contract, architecture document, and JSON fixtures.
2. Reconcile local Go list envelopes and validation behavior.
3. Replace temporary IDs with UUIDs.
4. Add three-state nullable PATCH fields.
5. Add versions, conflict responses, and tombstones.
6. Add household scope and synchronization metadata.
7. Create the Supabase schema, Auth configuration, and RLS tests.
8. Implement the SQLite outbox and Pi synchronization worker.
9. Scaffold the React/TypeScript/Vite PWA.
10. Add IndexedDB offline reads/writes and Realtime updates.
11. Implement product pairing, phone joining, revocation, and ownership transfer.
12. Run offline, retry, duplicate, conflict, and end-to-end tests.

This sequence keeps each change testable and avoids coupling the existing local
backend to an unfinished cloud implementation all at once.
