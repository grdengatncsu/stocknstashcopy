#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_PATH="${1:-/tmp/stocknstash-barcode-csi0.jpg}"

echo "Capturing CSI0/libcamera camera index 0 to: ${IMAGE_PATH}"
rpicam-still --camera 0 -t 1500 -o "${IMAGE_PATH}"

echo "Decoding barcode..."
python "${SCRIPT_DIR}/decode_image.py" "${IMAGE_PATH}"
