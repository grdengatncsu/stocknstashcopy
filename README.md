# Stock ’n Stash

Stock ’n Stash is a countertop grocery-scanning system that automatically identifies grocery items and adds them to a household inventory.

## Prototype 1 Goal

Prototype 1 will demonstrate the following process:

1. Detect groceries placed on the platform.
2. Wait until the measured weight is stable.
3. Capture images from three cameras under ambient lighting, with controlled lighting evaluated if needed.
4. Identify the grocery items.
5. Send the results to the inventory application.
6. Ask the user to confirm uncertain results.

## Hardware

The current prototype hardware includes:

- Raspberry Pi Compute Module 5 development kit
- Hailo-8L AI accelerator
- Two MIPI side cameras
- One overhead USB camera
- Four load cells
- NAU7802 load-cell ADC
- Controlled lighting under evaluation for consistent image capture
- Actuators and limit switches
- Network communication with the application

Controlled lighting is not required for initial hardware bring-up, but remains an active design investigation and may be added if ambient lighting causes unreliable exposure or recognition.

## System Operating States

The embedded system follows this sequence:

```text
Idle
→ Weight Detected
→ Stable
→ Capture
→ Recognize
→ Report
→ Reset
```

## Camera Names

The three cameras use permanent position-based names:

- `overhead`
- `rear_left`
- `front_right`

Images captured during the same scan will share a scan ID and timestamp.

## Software Components

- Python edge pipeline for camera capture, scan control, recognition, and
  multi-camera association
- Go HTTP/JSON backend for inventory and pending-result workflows
- SQLite persistence and duplicate-scan protection
- Phone application integration (in progress)
- Load-cell, lighting, barcode/OCR, and hardware integration (in progress)

## Project Structure

```text
stock-n-stash/
├── edge/
│   ├── association/
│   ├── cameras/
│   ├── positioning/
│   ├── recognition/
│   ├── reporting/
│   └── state_machine.py
├── server/
│   ├── api/
│   ├── database/
│   └── models/
├── documentation/
├── models/
├── mock_camera_inputs/
├── mock_captures/
└── tests/
```

## Development Status

Implemented in the current repository:

- Mock three-camera capture and saved-image recognition paths
- Scan state machine and multi-camera association
- Scan-report data model and mock reporting
- Go backend with inventory, pending-review, and scan-report endpoints
- SQLite persistence and duplicate-scan protection
- Python and Go automated tests

Remaining integration work includes the physical sensors and lighting, the
deployed Hailo recognition path, phone/cloud synchronization, and complete
end-to-end hardware testing.

## Running and Testing the Project

### Python edge software

Create a virtual environment and install the Python dependencies:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt
```

Run the Python test suite:

```bash
python -m unittest discover -s tests -v
```

The edge pipeline is currently exercised through component demos and tests;
there is not yet a single production startup command.

### Go backend

The backend module currently declares Go `1.27.1`. From the repository root:

```bash
cd server
go mod download
go test ./...
go run .
```

The server listens on `http://localhost:8080`. When it is started from the
`server` directory, it creates or opens `server/stocknstash.db` and initializes
the SQLite schema automatically.

Implemented routes:

```text
GET    /api/status
GET    /api/inventory
POST   /api/inventory
PATCH  /api/inventory/{id}
DELETE /api/inventory/{id}
GET    /api/pending
POST   /api/pending/{id}/resolve
POST   /api/scans
```

The current backend exposes a local Pi API. Phone access outside the Pi's local
network will require the planned synchronization layer; it is not implemented
by this server yet.

## Recognition Result Format

Example output:

```json
{
  "scan_id": "scan-001",
  "items": [
    {
      "association_id": "scan-001:item-1",
      "name": "Cheerios",
      "upc": "016000275287",
      "confidence": 0.93,
      "quantity": 1,
      "requires_review": false
    },
    {
      "association_id": "scan-001:item-2",
      "name": "Unknown",
      "upc": null,
      "confidence": 0.41,
      "quantity": 1,
      "requires_review": true
    }
  ]
}
```

`POST /api/scans` stores items with `requires_review: false` in inventory and
stores items with `requires_review: true` in the pending-results list.

## Documentation

Setup, camera configuration, model deployment, calibration, testing, and recovery procedures will be maintained in the `documentation` folder.
