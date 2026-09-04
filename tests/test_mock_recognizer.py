import tempfile
import unittest
from pathlib import Path

from edge.recognition.mock_recognizer import MockRecognizer
from edge.recognition.recognizer import RecognitionReport


class TestMockRecognizer(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)

    def create_image(self, camera_id: str) -> Path:
        image_path = self.root / f"{camera_id}.jpg"
        image_path.write_bytes(f"image from {camera_id}".encode())
        return image_path

    def test_three_valid_images_return_recognition_report(self):
        predictions = {
            "camera_1": ("apple", 0.97),
            "camera_2": ("banana", 0.89),
            "camera_3": ("carrot", 0.76),
        }
        recognizer = MockRecognizer(predictions=predictions)
        scan_id = "scan-123"
        captured_images = {
            "camera_1": self.create_image("camera_1"),
            "camera_2": self.create_image("camera_2"),
            "camera_3": self.create_image("camera_3"),
        }

        report = recognizer.recognize(scan_id=scan_id, captured_images=captured_images)

        self.assertIsInstance(report, RecognitionReport)
        self.assertEqual(report.scan_id, scan_id)

        items_by_camera = {item.source_camera: item for item in report.items}
        self.assertEqual(len(items_by_camera), 3)
        self.assertEqual(items_by_camera["camera_1"].name, "apple")
        self.assertEqual(items_by_camera["camera_1"].confidence, 0.97)
        self.assertEqual(items_by_camera["camera_2"].name, "banana")
        self.assertEqual(items_by_camera["camera_2"].confidence, 0.89)
        self.assertEqual(items_by_camera["camera_3"].name, "carrot")
        self.assertEqual(items_by_camera["camera_3"].confidence, 0.76)

    def test_camera_without_prediction_returns_unknown(self):
        recognizer = MockRecognizer(predictions={"camera_1": ("apple", 0.92)})
        report = recognizer.recognize(
            scan_id="scan-unknown",
            captured_images={
                "camera_1": self.create_image("camera_1"),
                "camera_2": self.create_image("camera_2"),
                "camera_3": self.create_image("camera_3"),
            },
        )

        items_by_camera = {item.source_camera: item for item in report.items}
        self.assertFalse(items_by_camera["camera_1"].unknown)
        self.assertEqual(items_by_camera["camera_2"].name, "unknown")
        self.assertEqual(items_by_camera["camera_2"].confidence, 0.0)
        self.assertTrue(items_by_camera["camera_2"].unknown)

    def test_missing_scan_id_raises_value_error(self):
        recognizer = MockRecognizer(predictions={})

        with self.assertRaises(ValueError):
            recognizer.recognize(
                scan_id="",
                captured_images={
                    "camera_1": self.create_image("camera_1"),
                    "camera_2": self.create_image("camera_2"),
                    "camera_3": self.create_image("camera_3"),
                },
            )

    def test_fewer_than_three_images_raises_value_error(self):
        recognizer = MockRecognizer(predictions={})

        with self.assertRaises(ValueError):
            recognizer.recognize(
                scan_id="scan-too-few",
                captured_images={
                    "camera_1": self.create_image("camera_1"),
                    "camera_2": self.create_image("camera_2"),
                },
            )

    def test_missing_image_raises_file_not_found_error(self):
        recognizer = MockRecognizer(predictions={})

        with self.assertRaises(FileNotFoundError):
            recognizer.recognize(
                scan_id="scan-missing-file",
                captured_images={
                    "camera_1": self.create_image("camera_1"),
                    "camera_2": self.create_image("camera_2"),
                    "camera_3": self.root / "does-not-exist.jpg",
                },
            )


if __name__ == "__main__":
    unittest.main()
