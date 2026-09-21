# Stock 'n Stash System Architecture

## Core Product Use Case

A primary Stock 'n Stash use case is checking what food is already at home while grocery shopping so users can avoid unnecessary duplicate purchases.

That means the phone application must be able to access inventory when it is not on the home network and cannot directly reach the Raspberry Pi.

## Architecture Decision

Stock 'n Stash uses a hybrid edge + cloud architecture.

```text
                    Internet
                       |
              Cloud Inventory Service
                 /             \
                /               \
        synchronized state      remote app access
              /                   \
             v                     v
      Raspberry Pi              Phone PWA
      ----------------          ---------
      Python edge pipeline      inventory UI
      Go inventory API          manual add/edit
      SQLite local state        pending review
```

## Raspberry Pi / Edge Responsibilities

The Pi remains responsible for:

- Camera/load-cell/Hailo processing through the Python edge pipeline
- Local Go inventory API
- SQLite persistence
- Pending recognition results
- Duplicate `scan_id` protection
- Continued operation during an internet outage
- Queueing/retrying cloud synchronization when needed

The device must not depend on cloud availability to perform a physical scan.

## Cloud Responsibilities

The cloud layer is responsible for:

- Making synchronized inventory available while the user is away from home
- Receiving device-generated inventory changes
- Receiving phone-originated manual additions, edits, and removals
- Making account inventory available to the PWA over the internet
- Authentication/authorization for remote access
- Supporting synchronization/retry semantics between cloud and Pi

The Raspberry Pi should not be directly exposed to the public internet as the normal remote-access mechanism.

## Phone PWA Responsibilities

The PWA is a client, not the authoritative database.

It should:

- Read inventory remotely from the cloud-backed inventory service
- Add an item manually when recognition fails
- Edit/remove inventory
- Confirm or correct uncertain recognition results
- Indicate whether a remote write has actually been acknowledged
- Optionally cache the most recently fetched inventory for display when temporarily offline

## Local Demo / Prototype Mode

For Senior Design integration and live demonstrations, the phone may connect directly to the Pi's local HTTP API while both devices are on the same network.

```text
Phone PWA -- local Wi-Fi --> Pi Go API --> SQLite
```

This local path is still useful for validating the hardware-to-backend-to-app pipeline, but it is not the full production access model.

## Data Flow Examples

### Confident automatic detection

```text
Product placed on platform
        |
        v
Python recognition
        |
        v
Pi Go API
        |
        v
SQLite
        |
        v
Cloud sync
        |
        v
Phone can view inventory anywhere
```

### Recognition failure / manual addition

```text
Phone PWA
   |
   v
Cloud inventory service
   |
   v
Synchronized inventory state
   |
   v
Pi receives change when connected
```

### Internet outage at home

```text
Recognition
   |
   v
Pi + SQLite continue normally
   |
   v
unsynchronized changes retained
   |
   v
internet restored
   |
   v
cloud synchronization resumes
```

## Prototype vs. Production Target

| Concern | Prototype/local demo | Production target |
|---|---|---|
| Device recognition | Pi/edge | Pi/edge |
| Local persistence | SQLite on Pi | SQLite on Pi |
| Phone access at home | Direct Pi API allowed | Cloud preferred; local path may remain useful |
| Phone access away from home | Not guaranteed by local-only demo path | Required through cloud service |
| Internet required for scanning | No | No |
| Cloud sync | May be minimal/prototyped | Required |
| Publicly expose Pi | No | No |

## Design Follow-Ups

- Select a cloud backend/database provider.
- Define authentication.
- Define inventory synchronization payloads.
- Define conflict resolution for offline edits.
- Decide whether remote pending-result review is included in the first cloud prototype.
- Define how the Pi queues unacknowledged cloud writes.
