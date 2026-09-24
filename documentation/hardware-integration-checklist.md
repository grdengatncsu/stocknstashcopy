# Hardware Integration Checklist

This document separates verified bench facts from values that still require the
physical Stock 'n Stash platform. Contributors must not replace a blocked value
with a guess just to make a software checklist appear complete.

Current platform status: [issue #31](https://github.com/jae-white/stock-n-stash/issues/31)
completed independent NAU7802/load-cell bring-up. The assembled fixture is
still open in [issue #33](https://github.com/jae-white/stock-n-stash/issues/33),
so final camera roles, geometry, mounting, and system-wide sign-off are blocked.

Search the code for the matching handoff markers with:

```bash
rg "HARDWARE TODO|PLATFORM BLOCKED|INTEGRATION TODO|DEPLOYMENT TODO" edge server
```

`HARDWARE TODO` identifies a mock, fixed development value, missing driver, or
calibration value that depends on the final electronics and mechanical build.
`PLATFORM BLOCKED` includes the GitHub issue that must supply physical evidence.
`INTEGRATION TODO` identifies software that exists but is not yet composed into
the production process.
`DEPLOYMENT TODO` identifies installation configuration that is not a physical
port but still must change before the Pi service is deployed.

Mocks should remain available for tests after physical implementations are
added. Production composition should choose physical adapters; automated tests
should continue choosing mocks.

## Port and Interface Record

Fill this table during hardware bring-up and update the configuration and code
references listed in the final column.

| Function | Planned hardware | Value that must be recorded | Current development substitute | Code/configuration location |
| --- | --- | --- | --- | --- |
| Overhead camera | USB IMX415 | Stable USB identity and video-device mapping; resolution and format | Saved `camera_1.jpg` | `edge/cameras/camera.py`, `edge/capture_demo.py` |
| Rear-left camera | MIPI IMX415 | **Blocked by #33:** assign mounted `oblique_csi0` or `oblique_csi1`; record resolution and format | Saved `camera_2.jpg` | `edge/cameras/camera.py`, `edge/capture_demo.py` |
| Front-right camera | MIPI IMX415 | **Blocked by #33:** assign the other mounted MIPI device; record resolution and format | Saved `camera_3.jpg` | `edge/cameras/camera.py`, `edge/capture_demo.py` |
| Weight input | Two 20 kg load cells in parallel on NAU7802 channel A | **Verified:** `/dev/i2c-1`, address `0x2A`, CM5 physical pins 1/3/5/9; **blocked by #33/#54:** final stability thresholds | Boolean arguments passed to `ScanController`; provisional occupied `baseline + 2000`, empty `baseline + 1000` | Future load-sensor adapter and production loop (#55) |
| Platform lighting | Final LED driver | GPIO or driver channel, active level, brightness control, and settling time | No implementation; capture proceeds immediately | Future lighting adapter called before capture |
| Actuators | Final motor/actuator driver | GPIO/PWM/driver channels, direction logic, safe positions, and timeouts | No implementation | Future actuator adapter and state-machine stages if required |
| Limit switches | Final switches | GPIO pins, pull-up/pull-down configuration, active level, and debounce time | No implementation | Future safety/input adapter |
| Recognition accelerator | Hailo-8L | Runtime device identifier, approved HEF path, input dimensions, label map, and confidence threshold | Ultralytics `.pt` baseline or `MockRecognizer` | Future Hailo recognizer; `edge/recognition/yolo_recognizer.py` |
| Camera geometry | Final mounted cameras and platform | One homography per logical camera and association-distance threshold | Caller-supplied development matrices and default `0.1` threshold | `edge/positioning/perspective_mapper.py`, `edge/association/position_associator.py` |
| Edge-to-Pi reporting | Go service on the Pi | Production bind address, TCP port, base URL, authentication, and retry limits | `HttpReporter`; server defaults to `:8080` with explicit timeouts | `edge/reporting/http_reporter.py`; `STOCKNSTASH_ADDR` in `server/main.go` |
| Captured-image storage | CM5 storage | Persistent directory, available-space limit, retention period, and cleanup policy | Relative `mock_captures/` directory | `edge/cameras/three_camera_capture.py` |
| Backend database | CM5 storage | Persistent database directory, ownership, backup, and recovery policy | Relative `stocknstash.db` | `STOCKNSTASH_DB_PATH` in `server/main.go` |

## Required Physical Implementations

### Camera adapter

Implement the `Camera` protocol in `edge/cameras/camera.py`. The adapter must:

- Expose one stable logical ID: `overhead`, `rear_left`, or `front_right`.
- Open the verified physical camera rather than assuming that `/dev/video0` or
  another OS number will always refer to the same device.
- Configure the resolution and pixel format used for calibration and inference.
- Save a complete image before returning its path.
- Translate vendor failures to `CaptureError` when capture cannot complete.
  Legacy `OSError` and `FileNotFoundError` remain accepted during migration.
- Release device handles cleanly during shutdown and recovery.

The three logical IDs are part of stored results and calibration lookup. Do not
rename them to transient Linux device names.

### Load-cell adapter

The production loop needs readings that produce:

- `weight_detected`: an item has newly crossed the presence threshold.
- `weight_present`: an item remains on the platform.
- `weight_stable`: recent readings remain inside the approved tolerance for
  the approved duration.
- `platform_empty`: the reading has returned to the calibrated empty range.

Document zeroing, tare, per-cell calibration, combined-weight calculation,
filtering, stable-window duration, and failure detection. The adapter should not
hide disconnected or saturated sensors as a zero-weight reading.

### Lighting and actuator adapters

Lighting must turn on before capture and remain stable long enough for exposure
and white balance. Record the safe startup and shutdown output states. Actuator
movement must have explicit timeouts and must incorporate the verified limit
switch states; software must not run a motor indefinitely while waiting for a
switch.

If lighting or motion requires additional workflow stages, update
`edge/state_machine.py`, `edge/scan_controller.py`, the tests, and
`documentation/state-machine.md` together.

### Hailo recognizer

The production recognizer must implement the existing `Recognizer` protocol so
the controller does not depend on Hailo-specific details. It must convert Hailo
output into the same normalized `RecognizedItem` fields used by the baseline:
name, confidence, logical camera ID, bounding box, and mapped platform position.

Keep `MockRecognizer` and `YoloRecognizer` for deterministic tests and portable
development. The deployed composition is responsible for selecting the Hailo
implementation.

### HTTP reporter

`edge/reporting/http_reporter.py` now implements the `Reporter` protocol using
`POST /api/scans`. It:

- Serializes the association result to the agreed API contract.
- Uses a configurable base URL rather than a hard-coded development address.
- Sets explicit connection and response timeouts.
- Treats `accepted` and `already_processed` as acknowledgements.
- Preserves the report for retry when no acknowledgement is received.
- Reuses the same `scan_id` during retries so the server remains idempotent.
- Leaves authentication as a deployment task because the local API does not yet
  define a device credential.

Run its unit and loopback HTTP tests with:

```bash
python -m unittest tests.test_http_reporter -v
```

For a full local edge-to-Go check, start the API in one terminal:

```bash
cd server
STOCKNSTASH_ADDR=127.0.0.1:8080 \
STOCKNSTASH_DB_PATH=/tmp/stocknstash-e2e.db \
go run .
```

Configure `HttpReporter` with `base_url="http://127.0.0.1:8080"` and an
explicit review threshold. Re-send the same report and verify that the first
response is `accepted`, the second is `already_processed`, and only one item is
stored. The repeated `scan_id` is intentional: it verifies idempotency rather
than creating a second test scan.

## Physical Platform Sign-Off

Do not sign this section until issue #33 has an assembled fixture. The table is
deliberately evidence-based so someone can distinguish “software implemented”
from “works on the real machine.”

| Check | Blocking issue | Evidence link or file | Verified by | Date | Status |
| --- | --- | --- | --- | --- | --- |
| Platform level, clear, and supported only by load cells | #33 | — | — | — | Blocked |
| `oblique_csi0/1` assigned to rear-left/front-right | #33 | — | — | — | Blocked |
| All three camera fields cover usable platform | #33 | — | — | — | Blocked |
| Presence works at left, center, and right after mounting | #33, #54 | — | — | — | Blocked |
| Final hysteresis and recovery behavior approved | #54, #55 | — | — | — | Blocked |
| Full scan reaches the local API without duplicate inventory | #46 | Automated coverage exists; physical run pending | — | — | Partially complete |

## Bring-Up Evidence to Record

Before replacing a blocked or unverified entry, record enough information for
another team member to reproduce it:

- Hardware revision and wiring diagram version
- CM5 operating-system image and kernel version
- Command or utility used to discover the port/device
- Observed identifier, GPIO number, bus, address, or network value
- Configuration file or code location that consumes the value
- A successful test and the expected failure behavior
- Date and contributor who verified it

Do not mark a hardware item complete because a mock test passes. Each physical
adapter needs a disconnected-device test and an on-device success test.

## Hardware Pull Request Checklist

- [ ] Exact ports, buses, addresses, and GPIO assignments are documented.
- [ ] Values are loaded from configuration where they may change by deployment.
- [ ] Stable logical camera names are preserved.
- [ ] Mocks remain usable by automated tests.
- [ ] Hardware failures reach the state machine's error path.
- [ ] Safe startup, recovery, timeout, and shutdown behavior is tested.
- [ ] Calibration data identifies the hardware arrangement that produced it.
- [ ] No device credentials or network secrets are committed.
- [ ] The relevant `HARDWARE TODO` or `PLATFORM BLOCKED` marker is removed or
      updated only after physical verification.
