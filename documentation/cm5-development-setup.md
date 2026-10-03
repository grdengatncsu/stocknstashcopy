# CM5 Development Environment

## System

- Device: Raspberry Pi Compute Module 5
- Hostname: `stock-n-stash-cm5`
- Architecture: `aarch64`
- Operating system: Debian 13 Trixie, Raspberry Pi build
- Kernel: `6.18.39+rpt-rpi-2712`
- Python: `3.13.5`

The Go backend module currently declares Go `1.27.1` in `server/go.mod`.
Backend execution on the CM5 has not yet been recorded as verified in this
document.

## Remote Access

The CM5 is reachable from Jaelynn's Mac over SSH:

```bash
ssh stocknstash@stock-n-stash-cm5.local
```

## Repository

The repository is located at:

```text
/home/stocknstash/Projects/stock-n-stash
```

It was cloned using GitHub CLI:

```bash
gh repo clone jae-white/stock-n-stash
```

GitHub authentication and repository access have been verified.

## Python Environment

The virtual environment uses system packages so the Raspberry Pi and Hailo Python bindings remain available:

```bash
python3 -m venv --system-site-packages .venv
source .venv/bin/activate
```

Desktop Ultralytics and PyTorch packages are not installed on the CM5. The deployed recognition pipeline will use the Hailo accelerator.

## Go Backend

Verify the installed Go toolchain before running the backend:

```bash
go version
```

The reported toolchain must satisfy the version declared in `server/go.mod`.
From the repository root, download dependencies and run the tests:

```bash
cd server
go mod download
go test ./...
```

Start the backend from the `server` directory:

```bash
go run .
```

The server listens on port `8080` by default. From the CM5 itself, its status
endpoint is:

```text
http://localhost:8080/api/status
```

Because the database path is relative to the process working directory,
starting the backend from `server/` creates or opens:

```text
/home/stocknstash/Projects/stock-n-stash/server/stocknstash.db
```

Deployment-specific values belong in environment configuration rather than
source code. For example:

```bash
STOCKNSTASH_ADDR=127.0.0.1:8080 \
STOCKNSTASH_DB_PATH=/var/lib/stock-n-stash/inventory.db \
go run .
```

The schema is initialized automatically. The current server has no
authentication or TLS and should not be exposed directly to the public
internet.

## Installed Hardware Support

- HailoRT CLI `4.23.0`
- Hailo Python bindings
- Raspberry Pi camera applications `1.13.0`
- libcamera `0.7.2+rpt20260817`
- Picamera2
- Video4Linux utilities
- I2C utilities
- `cedargrove-nau7802`
- GitHub CLI `2.46.0`

I2C is enabled for the NAU7802 load-cell amplifier.

## Verification

Hailo hardware detection was verified using:

```bash
hailortcli fw-control identify
```

The full Python test suite passes on the CM5:

```bash
python -m unittest discover -s tests -v
```

After Go is installed or confirmed, record CM5 backend verification by running:

```bash
cd /home/stocknstash/Projects/stock-n-stash/server
go test ./...
go run .
```

In a second SSH session, verify the live process:

```bash
curl -i http://localhost:8080/api/status
```

The expected body is:

```json
{"status":"ok"}
```

Physical camera configuration, load-cell calibration, and final port assignments
are handled during hardware bring-up. Record every verified value and remaining
gap in [`hardware-integration-checklist.md`](hardware-integration-checklist.md).

## Edge Runtime

The edge runtime composes the scan controller, camera capture, recognition,
association, presence sensing, and HTTP reporting components.

The current development runtime uses simulated presence signals, mock cameras,
and the mock recognizer while reporting scan results through the real Go Pi API.

### Run with development defaults

Start the Go API first:

```bash
cd server
go run .
```

Then, from the repository root in a second terminal:

```bash
python -m edge.runtime
```

The runtime will perform the simulated scan flow and then continue running,
waiting for additional presence events.

Press `Ctrl+C` to request a clean shutdown.

### Runtime configuration

Runtime settings can be overridden with environment variables instead of
editing the Python source code.

Supported environment variables:

- `STOCKNSTASH_API_BASE_URL`
- `STOCKNSTASH_CAPTURE_DIRECTORY`
- `STOCKNSTASH_MOCK_IMAGE_DIRECTORY`
- `STOCKNSTASH_CAMERA_IDS`
- `STOCKNSTASH_RECOGNITION_THRESHOLD`
- `STOCKNSTASH_ASSOCIATION_DISTANCE`
- `STOCKNSTASH_REPORT_TIMEOUT_SECONDS`
- `STOCKNSTASH_POLL_INTERVAL_SECONDS`
- `STOCKNSTASH_MODEL_PATH`

Example:

```bash
STOCKNSTASH_API_BASE_URL=http://localhost:8080 \
STOCKNSTASH_POLL_INTERVAL_SECONDS=0.1 \
python -m edge.runtime
```

Camera IDs are provided as a comma-separated list:

```bash
STOCKNSTASH_CAMERA_IDS=overhead,oblique_csi0,oblique_csi1 \
python -m edge.runtime
```

Invalid configuration values fail during startup with an error describing the
invalid setting.

For example:

```bash
STOCKNSTASH_REPORT_TIMEOUT_SECONDS=banana \
python -m edge.runtime
```

fails because the report timeout must be numeric.

### Runtime lifecycle

The long-running runtime repeatedly reads the configured presence source and
passes those signals into the existing `ScanController`.

The normal scan lifecycle is:

```text
IDLE
  -> STABILIZING
  -> CAPTURE
  -> RECOGNIZE
  -> REPORT
  -> RESET
  -> IDLE
```

The same scan ID is retained throughout the scan lifecycle.

### Report retries

If the Pi API is unavailable or returns a server-side error, the runtime
remains in the `REPORT` state and retries the same association report.

The retry uses the same scan ID rather than creating a new scan. This preserves
the existing idempotent reporting behavior and prevents a temporary API failure
from creating duplicate scan records.

Retryable reporting failures are logged as warnings.

### Graceful shutdown

The runtime handles both `SIGINT` and `SIGTERM`.

`SIGINT` is generated when `Ctrl+C` is pressed in a terminal.

`SIGTERM` allows the runtime to be stopped cleanly by a Linux process or service
manager.

When either signal is received, the runtime requests shutdown through its stop
event, exits the long-running loop, and logs that the edge runtime has stopped.

### Current development limitations

The development runtime still uses simulated presence sensing and mock
recognition components.

Physical NAU7802 presence sensing is handled separately by issue #55.

The production Hailo-backed recognizer is handled separately by issue #59.

### CM5 deployment entry point

The long-running runtime is also the process entry point intended for the CM5.

After the repository and Python virtual environment are installed on the CM5,
the runtime can be started from the repository root with:

```bash
cd /opt/stock-n-stash

STOCKNSTASH_API_BASE_URL=http://localhost:8080 \
STOCKNSTASH_CAPTURE_DIRECTORY=/var/lib/stocknstash/captures \
STOCKNSTASH_CAMERA_IDS=overhead,oblique_csi0,oblique_csi1 \
STOCKNSTASH_RECOGNITION_THRESHOLD=0.80 \
STOCKNSTASH_ASSOCIATION_DISTANCE=0.10 \
STOCKNSTASH_REPORT_TIMEOUT_SECONDS=5.0 \
STOCKNSTASH_POLL_INTERVAL_SECONDS=0.1 \
/opt/stock-n-stash/.venv/bin/python -m edge.runtime
```

Deployment-specific settings may instead be placed in the environment by the
service manager so that deployment values do not need to be stored in source
code.

A systemd service can use the same runtime entry point:

```ini
[Unit]
Description=Stock n Stash Edge Runtime
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/stock-n-stash
EnvironmentFile=-/etc/stocknstash/edge.env
ExecStart=/opt/stock-n-stash/.venv/bin/python -m edge.runtime
Restart=on-failure
RestartSec=2
KillSignal=SIGTERM
TimeoutStopSec=10

[Install]
WantedBy=multi-user.target
```

The environment file can contain the deployment-specific runtime configuration:

```text
STOCKNSTASH_API_BASE_URL=http://localhost:8080
STOCKNSTASH_CAPTURE_DIRECTORY=/var/lib/stocknstash/captures
STOCKNSTASH_CAMERA_IDS=overhead,oblique_csi0,oblique_csi1
STOCKNSTASH_RECOGNITION_THRESHOLD=0.80
STOCKNSTASH_ASSOCIATION_DISTANCE=0.10
STOCKNSTASH_REPORT_TIMEOUT_SECONDS=5.0
STOCKNSTASH_POLL_INTERVAL_SECONDS=0.1
```

`SIGTERM` from systemd uses the runtime's normal graceful-shutdown path.

The runtime currently composes the simulated presence and recognition adapters
for hardware-independent development. Issue #55 supplies the NAU7802-backed
presence implementation and issue #59 supplies the Hailo-backed production
recognizer. The runtime process lifecycle and deployment entry point remain the
same when those implementations are substituted.