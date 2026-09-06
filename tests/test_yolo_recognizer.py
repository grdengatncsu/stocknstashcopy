import tempfile
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch

from edge.recognition.yolo_recognizer import YoloRecognizer


class TestYoloRecognizer(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)

        self.captured_images = {
            "overhead": self.root / "overhead.jpg",
            "rear_left": self.root / "rear_left.jpg",
            "front_right": self.root / "front_right.jpg",
        }

        for image_path in self.captured_images.values():
            image_path.write_bytes(b"test image")

        self.yolo_patcher = patch(
            "edge.recognition.yolo_recognizer.YOLO"
        )
        self.mock_yolo_class = self.yolo_patcher.start()
        self.addCleanup(self.yolo_patcher.stop)

        self.mock_model = MagicMock()
        self.mock_yolo_class.return_value = self.mock_model

    def make_result(
        self,
        name: str,
        confidence: float,
        bounding_box: tuple[float, float, float, float],
    ):
        box = MagicMock()
        box.cls.item.return_value = 0
        box.conf.item.return_value = confidence
        box.xyxyn.__getitem__.return_value.tolist.return_value = list(
            bounding_box
        )

        result = MagicMock()
        result.names = {0: name}
        result.boxes = [box]
        return result

    def make_empty_result(self):
        result = MagicMock()
        result.names = {}
        result.boxes = []
        return result

    def test_model_is_loaded_once(self):
        YoloRecognizer("models/yolov8n.pt")

        self.mock_yolo_class.assert_called_once_with(
            "models/yolov8n.pt"
        )

    def test_invalid_confidence_threshold_raises_value_error(self):
        for threshold in (-0.01, 1.01):
            with self.subTest(threshold=threshold):
                with self.assertRaises(ValueError):
                    YoloRecognizer(
                        "models/yolov8n.pt",
                        confidence_threshold=threshold,
                    )

    def test_missing_scan_id_raises_value_error(self):
        recognizer = YoloRecognizer("models/yolov8n.pt")

        with self.assertRaises(ValueError):
            recognizer.recognize(
                scan_id="",
                captured_images=self.captured_images,
            )

    def test_exactly_three_images_are_required(self):
        recognizer = YoloRecognizer("models/yolov8n.pt")

        with self.assertRaises(ValueError):
            recognizer.recognize(
                scan_id="scan-123",
                captured_images={
                    "overhead": self.captured_images["overhead"],
                    "rear_left": self.captured_images["rear_left"],
                },
            )

    def test_missing_image_raises_file_not_found_error(self):
        recognizer = YoloRecognizer("models/yolov8n.pt")
        images = dict(self.captured_images)
        images["front_right"] = self.root / "missing.jpg"

        with self.assertRaises(FileNotFoundError):
            recognizer.recognize(
                scan_id="scan-123",
                captured_images=images,
            )

    def test_detection_becomes_recognized_item(self):
        self.mock_model.predict.side_effect = [
            [
                self.make_result(
                    "apple",
                    0.91,
                    (0.10, 0.20, 0.70, 0.80),
                )
            ],
            [self.make_empty_result()],
            [self.make_empty_result()],
        ]

        recognizer = YoloRecognizer(
            "models/yolov8n.pt",
            confidence_threshold=0.50,
        )

        report = recognizer.recognize(
            scan_id="scan-123",
            captured_images=self.captured_images,
        )

        self.assertEqual(report.scan_id, "scan-123")
        self.assertEqual(len(report.items), 1)

        item = report.items[0]
        self.assertEqual(item.name, "apple")
        self.assertEqual(item.confidence, 0.91)
        self.assertEqual(item.source_camera, "overhead")
        self.assertFalse(item.unknown)
        self.assertEqual(
            item.bounding_box,
            (0.10, 0.20, 0.70, 0.80),
        )

        self.assertEqual(self.mock_model.predict.call_count, 3)
        self.mock_model.predict.assert_any_call(
            source=str(self.captured_images["overhead"]),
            conf=0.50,
            verbose=False,
        )


if __name__ == "__main__":
    unittest.main()