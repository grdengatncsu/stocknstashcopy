"""Shared recognition data types and the recognizer interface."""

from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

@dataclass
class RecognizedItem:
    """One item observation produced from one camera image.

    Multiple observations may later be combined into one physical item by an
    associator. Bounding boxes and platform positions use normalized values in
    the range 0.0 to 1.0 so they do not depend on image resolution.
    """

    name: str
    confidence: float
    source_camera: str
    unknown: bool = False
    # Bounding-box order is (left, top, right, bottom).
    bounding_box: tuple[float, float, float, float] | None = None
    platform_position: tuple[float, float] | None = None

@dataclass
class RecognitionReport:
    """Collect all per-camera observations generated for one scan."""

    scan_id: str
    items: list[RecognizedItem]

class Recognizer(Protocol):
    """Contract shared by real and mock recognition implementations."""

    def recognize(
        self,
        scan_id: str,
        captured_images: dict[str, Path],
    ) -> RecognitionReport:
        """Convert a set of camera images into item observations."""
        ...
