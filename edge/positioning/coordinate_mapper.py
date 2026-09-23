"""Interface for translating camera pixels into a shared platform space."""

from typing import Protocol


class CoordinateMapper(Protocol):
    """Contract implemented by camera-calibration mapping strategies."""

    def map_detection(
        self,
        camera_id: str,
        bounding_box: tuple[float, float, float, float],
    ) -> tuple[float, float] | None:
        """Map one bounding box to normalized ``(x, y)`` platform coordinates."""
        ...
