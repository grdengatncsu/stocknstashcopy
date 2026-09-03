from pathlib import Path
from shutil import copy2

class MockCamera:
    """Represent one camera using a saved image"""
    def __init__(self, camera_id: str, image_path: str | Path):
        self.camera_id = camera_id
        self.image_path = Path(image_path)

    def capture_image(self, save_path: str | Path):
        """Copy the saved image to simulate taking a new picture"""
        if not self.image_path.exists():
            raise FileNotFoundError(f"Image file not found for {self.camera_id}: {self.image_path}")
        output_directory = Path(save_path)
        output_directory.mkdir(parents=True, exist_ok=True)
        output_path = output_directory / (self.camera_id + self.image_path.suffix)
        copy2(self.image_path, output_path)
        return output_path
    
    def get_image(self):
        """Return the saved image"""
        return self.image_path