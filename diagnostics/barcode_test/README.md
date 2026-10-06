# Barcode Test Diagnostic

Quick proof-of-concept for decoding grocery UPC/EAN barcodes from Stock 'n
Stash camera images on the Raspberry Pi.

This is a diagnostic utility only. It is not yet part of the production
recognition pipeline.

## Setup on the Pi

Install the ZBar system library:

```bash
sudo apt update
sudo apt install -y libzbar0
```

Activate the project virtual environment, then install the Python wrappers:

```bash
python -m pip install -r diagnostics/barcode_test/requirements.txt
```

## Verify the CSI camera index

The current test assumes the camera physically connected to CSI0 appears as
libcamera camera index 0. Verify before testing:

```bash
rpicam-hello --list-cameras
```

## Fast CSI0 test

From the repository root:

```bash
bash diagnostics/barcode_test/capture_and_decode_csi0.sh
```

The script captures one still image from camera index 0 and immediately runs
the barcode decoder on that image.

To choose a different temporary output image:

```bash
bash diagnostics/barcode_test/capture_and_decode_csi0.sh /tmp/my-barcode.jpg
```

## Decode an existing image

```bash
python diagnostics/barcode_test/decode_image.py /path/to/image.jpg
```

A successful read should look similar to:

```text
Type: EAN13
Value: 0016000275287
Location: x=412, y=268, width=315, height=141
```

If the decoder prints:

```text
No barcode found.
```

check that the barcode is sharp, large enough in the frame, not heavily
foreshortened, and not covered by glare. Try changing product distance and
orientation before treating the result as a decoder failure.
