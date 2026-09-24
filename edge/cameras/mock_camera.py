"""Saved-image substitute for a camera during hardware-independent testing."""

from pathlib import Path
from shutil import copy2


class MockCamera:
    """Simulate one camera by repeatedly copying a known image.

    HARDWARE TODO: Use the future physical ``Camera`` implementation in the
    deployed pipeline. Keep this mock for automated tests and laptop demos.
    """

    def __init__(self, camera_id: str, image_path: str | Path):
        """Store the logical camera name and reusable source-image path."""
        self.camera_id = camera_id
        self.image_path = Path(image_path)

    def capture_image(self, save_path: str | Path) -> Path:
        """Copy the source image into a scan folder and return its new path."""
        if not self.image_path.exists():
            raise FileNotFoundError(
                f"Image file not found for {self.camera_id}: {self.image_path}"
            )
        output_directory = Path(save_path)
        output_directory.mkdir(parents=True, exist_ok=True)

        # Use the logical camera ID in the filename so each view remains
        # traceable after all images are copied into the same scan folder.
        output_path = output_directory / (self.camera_id + self.image_path.suffix)
        copy2(self.image_path, output_path)
        return output_path

    def get_image(self) -> Path:
        """Return the original fixture path without copying it."""
        return self.image_path
