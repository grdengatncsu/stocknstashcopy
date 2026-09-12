from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

@dataclass
class RecognizedItem:
    """Represents a recognized item with its name and confidence score."""
    name: str
    confidence: float
    source_camera: str
    unknown: bool = False
    bounding_box: tuple[float, float, float, float] | None = None  # (xmin, y_min, x_max, y_max) in normalized coordinates
    platform_position: tuple[float, float] | None = None  # (x, y) position on the platform in normalized coordinates

@dataclass
class RecognitionReport:
    """Represents a recognition report containing recognized items and the scan ID."""
    scan_id: str
    items: list[RecognizedItem]

class Recognizer(Protocol):
    """Protocol for a recognizer that can recognize items from images."""
    def recognize(self, scan_id: str, captured_images: dict[str, Path]) -> RecognitionReport:
        """Recognize items from the given images and return a recognition report."""
        ...