"""Deterministic recognition for tests and hardware-independent demos."""

from pathlib import Path

from edge.recognition.recognizer import RecognitionReport, RecognizedItem


class MockRecognizer:
    """Return configured predictions instead of running an ML model.

    HARDWARE TODO: Use the deployed Hailo-backed recognizer in production.
    Keep this class for repeatable automated tests.
    """

    def __init__(self, predictions: dict[str, tuple[str, float] | tuple[str, float, tuple[float, float]]]):
        """Store ``camera_id: (item name, confidence)`` predictions."""
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

            # Missing fixture data becomes an explicit unknown observation
            # rather than silently removing a camera from the report.

            prediction = self.predictions.get(camera_id, ("unknown", 0.0))
            if len(prediction) == 2:
                name, confidence = prediction
                platform_position = None
            else:
                name, confidence, platform_position = prediction

            recognized_items.append(
                RecognizedItem(
                    name=name,
                    confidence=confidence,
                    source_camera=camera_id,
                    unknown=name == "unknown",
                    platform_position=platform_position,
                )
            )

        return RecognitionReport(scan_id=scan_id, items=recognized_items)
