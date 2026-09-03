import tempfile
import unittest
from pathlib import Path

from edge.cameras.mock_camera import MockCamera
from edge.cameras.three_camera_capture import ThreeCameraCapture


class TestCameraCapture(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)

    def make_camera(self, camera_id: str) -> MockCamera:
        image_path = self.root / f"{camera_id}.jpg"
        image_path.write_bytes(f"image from {camera_id}".encode())

        return MockCamera(camera_id, image_path)

    def test_mock_camera_copies_saved_image(self):
        camera = self.make_camera("camera_1")

        output_path = camera.capture_image(self.root / "output")

        self.assertTrue(output_path.is_file())
        self.assertEqual(
            output_path.read_bytes(),
            b"image from camera_1",
        )

    def test_missing_image_raises_error(self):
        camera = MockCamera(
            "camera_1",
            self.root / "missing.jpg",
        )

        with self.assertRaises(FileNotFoundError):
            camera.capture_image(self.root / "output")

    def test_exactly_three_cameras_are_required(self):
        with self.assertRaises(ValueError):
            ThreeCameraCapture([])

    def test_all_three_cameras_capture(self):
        cameras = [
            self.make_camera("camera_1"),
            self.make_camera("camera_2"),
            self.make_camera("camera_3"),
        ]
        capture = ThreeCameraCapture(cameras)

        results = capture.capture_all(
            scan_id="test_scan",
            output_root=str(self.root / "captures"),
        )

        self.assertEqual(
            set(results),
            {"camera_1", "camera_2", "camera_3"},
        )

        for output_path in results.values():
            self.assertTrue(output_path.is_file())


if __name__ == "__main__":
    unittest.main()