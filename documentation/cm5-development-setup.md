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

The server listens on port `8080`. From the CM5 itself, its status endpoint is:

```text
http://localhost:8080/api/status
```

Because the database path is relative to the process working directory,
starting the backend from `server/` creates or opens:

```text
/home/stocknstash/Projects/stock-n-stash/server/stocknstash.db
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

Physical camera configuration, load-cell calibration, and final port assignments are handled separately during hardware bring-up.
