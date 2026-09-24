"""Capture one set of views from the scanner's three logical cameras."""

from collections.abc import Collection
from pathlib import Path

from edge.cameras.camera import Camera


class ThreeCameraCapture:
    """Capture the three views required by the recognition pipeline."""

    def __init__(
        self,
        cameras: list[Camera],
        expected_camera_ids: Collection[str] | None = None,
    ):
        """Require three uniquely named cameras and, optionally, exact IDs.

        Final physical roles are optional because GitHub issue #33 still owns
        assignment of the two provisional MIPI devices to ``rear_left`` and
        ``front_right``. Uniqueness is enforced now because duplicate keys
        would silently overwrite a captured image in the result dictionary.
        """
        if len(cameras) != 3:
            raise ValueError("Three cameras are required")

        camera_ids = [camera.camera_id.strip() for camera in cameras]
        if any(not camera_id for camera_id in camera_ids):
            raise ValueError("Every camera must have a non-empty camera_id")
        if len(set(camera_ids)) != len(camera_ids):
            raise ValueError("Camera IDs must be unique")

        if expected_camera_ids is not None:
            expected = set(expected_camera_ids)
            if set(camera_ids) != expected:
                raise ValueError(
                    "Camera IDs do not match the configured logical camera set"
                )
        self.cameras = cameras

    def capture_all(
        self,
        scan_id: str,
        output_root: str = "mock_captures",
    ) -> dict[str, Path]:
        """Save all views under one scan ID and return paths by camera ID.

        HARDWARE TODO: Configure a durable production image directory instead
        of the ``mock_captures`` default after CM5 storage and retention rules
        are decided.
        """
        # A separate folder keeps images from different items from being mixed
        # together or overwritten.
        scan_directory = f"{output_root}/{scan_id}"
        captured_images: dict[str, Path] = {}

        for camera in self.cameras:
            output_path = camera.capture_image(scan_directory)
            captured_images[camera.camera_id] = output_path
        return captured_images
