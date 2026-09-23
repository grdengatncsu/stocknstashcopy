"""Shared types for consolidating observations from multiple cameras."""

from dataclasses import dataclass
from typing import Protocol

from edge.recognition.recognizer import RecognitionReport, RecognizedItem

@dataclass
class AssociatedItem:
    """Represent one physical item observed by one or more cameras."""

    association_id: str
    name: str
    confidence: float
    observations: list[RecognizedItem]
    # Position is the average normalized (x, y) location of valid observations.
    platform_position: tuple[float, float] | None = None

@dataclass
class AssociationReport:
    """Collect the physical items inferred from all views in one scan."""

    scan_id: str
    items: list[AssociatedItem]

class Associator(Protocol):
    """Contract for grouping camera observations into physical items."""

    def associate(self, recognition_report: RecognitionReport) -> AssociationReport:
        """Consolidate a recognition report into unique physical items."""
        ...
