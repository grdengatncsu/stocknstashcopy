"""Capture one synchronized set of views for a scan."""

from pathlib import Path

from edge.cameras.mock_camera import MockCamera


class ThreeCameraCapture:
    """Capture the three fixed views expected by the recognition pipeline."""

    def __init__(self, cameras: list[MockCamera]):
        """Require the exact camera count assumed by downstream components."""
        if len(cameras) != 3:
            raise ValueError("Three cameras are required")
        self.cameras = cameras

    def capture_all(
        self,
        scan_id: str,
        output_root: str = "mock_captures",
    ) -> dict[str, Path]:
        """Save every view under one scan ID and return paths by camera ID."""
        # A folder per scan prevents images from separate items from being
        # mixed together or overwritten.
        scan_directory = f"{output_root}/{scan_id}"
        captured_images: dict[str, Path] = {}

        for camera in self.cameras:
            output_path = camera.capture_image(scan_directory)
            captured_images[camera.camera_id] = output_path
        return captured_images
