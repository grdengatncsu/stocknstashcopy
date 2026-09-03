from edge.cameras.mock_camera import MockCamera

class ThreeCameraCapture:
    """Capture images from three cameras"""
    def __init__(self, cameras: list[MockCamera]):
        if len(cameras) != 3:
            raise ValueError("Three cameras are required")
        self.cameras = cameras
    def capture_all(self, scan_id: str, output_root: str = "mock_captures") -> dict:
        """Capture images from all three cameras and save them to the output directory"""
        scan_directory = f"{output_root}/{scan_id}"
        captured_images = {}

        for camera in self.cameras:
            output_path = camera.capture_image(scan_directory)
            captured_images[camera.camera_id] = output_path
        return captured_images
