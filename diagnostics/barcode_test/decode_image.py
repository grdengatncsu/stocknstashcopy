"""Decode UPC/EAN barcodes from a saved image.

This is a standalone diagnostic for validating grocery barcode readability on
the Raspberry Pi. It is intentionally separate from the production edge
recognition pipeline.
"""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image
from pyzbar.pyzbar import ZBarSymbol, decode


SUPPORTED_SYMBOLS = [
    ZBarSymbol.UPCA,
    ZBarSymbol.EAN13,
    ZBarSymbol.EAN8,
]


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Decode UPC/EAN barcodes from a saved camera image."
    )
    parser.add_argument("image", type=Path, help="Path to the image to scan.")
    args = parser.parse_args()

    if not args.image.is_file():
        raise SystemExit(f"Image does not exist: {args.image}")

    with Image.open(args.image) as image:
        results = decode(image, symbols=SUPPORTED_SYMBOLS)

    if not results:
        print("No barcode found.")
        raise SystemExit(1)

    for result in results:
        value = result.data.decode("utf-8", errors="replace")
        rect = result.rect

        print(f"Type: {result.type}")
        print(f"Value: {value}")
        print(
            "Location: "
            f"x={rect.left}, y={rect.top}, "
            f"width={rect.width}, height={rect.height}"
        )


if __name__ == "__main__":
    main()
