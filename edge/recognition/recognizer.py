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