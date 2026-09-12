from typing import Protocol

class CoordinateMapper(Protocol):
    """Convert camera detections into shared platform coordinates."""
    def map_detection(self, camera_id: str, bounding_box: tuple[float, float, float, float]) -> tuple[float, float] | None:
        """ Map one detection into normalized platform coordinates."""
        ...