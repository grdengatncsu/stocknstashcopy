"""Deterministic recognizer used when an ML model is unavailable or unnecessary."""

from pathlib import Path

from edge.recognition.recognizer import RecognitionReport, RecognizedItem


class MockRecognizer:
    """Return configured predictions instead of running an ML model."""

    def __init__(self, predictions: dict[str, tuple[str, float]]):
        """Store ``camera_id: (item name, confidence)`` test predictions."""
        self.predictions = predictions

    def recognize(self, scan_id: str, captured_images: dict[str, Path]) -> RecognitionReport:
        """Create one predictable observation for each captured camera image."""
        if not scan_id:
            raise ValueError("scan_id must be provided")
        if len(captured_images) != 3:
            raise ValueError("Three captured images are required")
        recognized_items = []

        for camera_id, image_path in captured_images.items():
            if not image_path.exists():
                raise FileNotFoundError(
                    f"Captured image for camera {camera_id} does not exist at {image_path}"
                )

            # Missing test data becomes an explicit unknown result instead of
            # silently omitting a camera from the report.
            name, confidence = self.predictions.get(camera_id, ("unknown", 0.0))
            recognized_items.append(
                RecognizedItem(
                    name=name,
                    confidence=confidence,
                    source_camera=camera_id,
                    unknown=name == "unknown",
                )
            )

        return RecognitionReport(scan_id=scan_id, items=recognized_items)
