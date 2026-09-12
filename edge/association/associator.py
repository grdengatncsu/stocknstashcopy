from typing import Protocol
from dataclasses import dataclass
from edge.recognition.recognizer import RecognitionReport, RecognizedItem

@dataclass
class AssociatedItem:
    """Represents one physical item observed by one or more cameras"""

    association_id: str
    name: str
    confidence: float
    observations: list[RecognizedItem]
    platform_position: tuple[float, float] | None = None  # (x, y) position on the platform in normalized coordinates

@dataclass
class AssociationReport:
    """Represents a report of associated items from multiple cameras"""
    scan_id: str
    items: list[AssociatedItem]

class Associator(Protocol):
    """Associates recognized items from multiple cameras into a single representation of physical items."""

    def associate(self, recognition_report: RecognitionReport) -> AssociationReport:
        """Associate recognized items from multiple cameras into a single representation of physical items."""
        ...

