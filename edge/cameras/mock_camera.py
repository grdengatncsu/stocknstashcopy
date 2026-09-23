"""Saved-image camera substitute for development without physical hardware."""

from pathlib import Path
from shutil import copy2


class MockCamera:
    """Represent one camera using a repeatable image from disk."""

    def __init__(self, camera_id: str, image_path: str | Path):
        """Identify the camera and the source image it should reuse."""
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

        # Keeping the camera ID in the filename makes each view easy to trace
        # after all three files are placed in the same scan directory.
        output_path = output_directory / (self.camera_id + self.image_path.suffix)
        copy2(self.image_path, output_path)
        return output_path

    def get_image(self) -> Path:
        """Return the original fixture path without copying the image."""
        return self.image_path
