import os
import tempfile
import unittest
from pathlib import Path

from edge.cameras.mock_camera import MockCamera
from edge.cameras.three_camera_capture import ThreeCameraCapture
from edge.recognition.mock_recognizer import MockRecognizer
from edge.reporting.mock_reporter import MockReporter
from edge.scan_controller import ScanController
from edge.state_machine import State, StateMachine


class TestScanIntegration(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)

        self.original_directory = Path.cwd()
        os.chdir(self.root)
        self.addCleanup(os.chdir, self.original_directory)

        self.input_directory = self.root / "inputs"
        self.input_directory.mkdir()

        self.source_images = {
            "camera_1": self.create_image("camera_1"),
            "camera_2": self.create_image("camera_2"),
            "camera_3": self.create_image("camera_3"),
        }

        cameras = [
            MockCamera("camera_1", self.source_images["camera_1"]),
            MockCamera("camera_2", self.source_images["camera_2"]),
            MockCamera("camera_3", self.source_images["camera_3"]),
        ]

        self.capture = ThreeCameraCapture(cameras)
        self.recognizer = MockRecognizer(
            predictions={
                "camera_1": ("milk", 0.95),
                "camera_2": ("apple", 0.91),
                "camera_3": ("bread", 0.89),
            }
        )
        self.reporter = MockReporter(should_acknowledge=True)
        self.state_machine = StateMachine()

        self.controller = ScanController(
            state_machine=self.state_machine,
            capture=self.capture,
            recognizer=self.recognizer,
            reporter=self.reporter,
        )

    def create_image(self, camera_id: str) -> Path:
        image_path = self.input_directory / f"{camera_id}.jpg"
        image_path.write_bytes(f"image from {camera_id}".encode())
        return image_path

    def advance_to_capture(self):
        self.controller.process_current_state(weight_detected=True)
        self.controller.process_current_state(
            weight_present=True,
            weight_stable=True,
        )

    def advance_to_recognize(self):
        self.advance_to_capture()
        self.controller.process_current_state()

    def advance_to_report(self):
        self.advance_to_recognize()
        self.controller.process_current_state()

    def advance_to_reset(self):
        self.advance_to_report()
        self.controller.process_current_state()

    def test_complete_scan_returns_to_idle(self):
        self.assertEqual(self.state_machine.state, State.IDLE)

        self.controller.process_current_state(weight_detected=True)
        self.assertEqual(self.state_machine.state, State.STABILIZING)

        scan_id = self.state_machine.scan_id
        self.assertIsNotNone(scan_id)

        self.controller.process_current_state(
            weight_present=True,
            weight_stable=True,
        )
        self.assertEqual(self.state_machine.state, State.CAPTURE)

        self.controller.process_current_state()
        self.assertEqual(self.state_machine.state, State.RECOGNIZE)
        self.assertEqual(len(self.controller.captured_images), 3)

        self.controller.process_current_state()
        self.assertEqual(self.state_machine.state, State.REPORT)
        self.assertEqual(len(self.controller.current_report.items), 3)

        self.controller.process_current_state()
        self.assertEqual(self.state_machine.state, State.RESET)
        self.assertIn(scan_id, self.reporter.processed_scan_ids)

        self.controller.process_current_state(platform_empty=True)
        self.assertEqual(self.state_machine.state, State.IDLE)
        self.assertIsNone(self.state_machine.scan_id)
        self.assertIsNone(self.controller.captured_images)
        self.assertIsNone(self.controller.current_report)

        capture_directory = self.root / "mock_captures" / scan_id
        self.assertTrue((capture_directory / "camera_1.jpg").is_file())
        self.assertTrue((capture_directory / "camera_2.jpg").is_file())
        self.assertTrue((capture_directory / "camera_3.jpg").is_file())

    def test_capture_failure_recovers_to_capture(self):
        self.source_images["camera_3"].unlink()

        self.advance_to_capture()
        scan_id = self.state_machine.scan_id

        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.ERROR)
        self.assertEqual(
            self.state_machine.error_source,
            State.CAPTURE,
        )
        self.assertFalse(self.state_machine.cap_valid)

        self.source_images["camera_3"].write_bytes(b"restored image")

        self.controller.process_current_state(recovered=True)

        self.assertEqual(self.state_machine.state, State.CAPTURE)
        self.assertEqual(self.state_machine.scan_id, scan_id)

        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.RECOGNIZE)
        self.assertTrue(self.state_machine.cap_valid)

    def test_recognition_failure_recovers_without_recapturing(self):
        self.advance_to_recognize()
        scan_id = self.state_machine.scan_id

        missing_image = self.controller.captured_images["camera_3"]
        missing_image.unlink()

        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.ERROR)
        self.assertEqual(
            self.state_machine.error_source,
            State.RECOGNIZE,
        )
        self.assertTrue(self.state_machine.cap_valid)
        self.assertFalse(self.state_machine.rec_valid)

        missing_image.write_bytes(b"restored captured image")

        self.controller.process_current_state(recovered=True)
        self.assertEqual(self.state_machine.state, State.RECOGNIZE)

        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.REPORT)
        self.assertTrue(self.state_machine.rec_valid)
        self.assertEqual(self.state_machine.scan_id, scan_id)

    def test_unacknowledged_report_can_be_retried(self):
        self.advance_to_report()
        scan_id = self.state_machine.scan_id
        original_report = self.controller.current_report

        self.reporter.should_acknowledge = False
        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.REPORT)
        self.assertFalse(self.state_machine.report_ack)
        self.assertIs(self.controller.current_report, original_report)
        self.assertNotIn(scan_id, self.reporter.processed_scan_ids)

        self.reporter.should_acknowledge = True
        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.RESET)
        self.assertTrue(self.state_machine.report_ack)
        self.assertIn(scan_id, self.reporter.processed_scan_ids)

    def test_duplicate_report_is_not_processed_twice(self):
        self.advance_to_report()
        scan_id = self.state_machine.scan_id
        recognition_report = self.controller.current_report

        first_acknowledgment = self.reporter.report(
            recognition_report
        )
        self.assertTrue(first_acknowledgment)

        self.controller.process_current_state()

        self.assertEqual(self.state_machine.state, State.RESET)
        self.assertEqual(
            self.reporter.processed_scan_ids,
            {scan_id},
        )

    def test_occupied_platform_preserves_scan_until_empty(self):
        self.advance_to_reset()
        scan_id = self.state_machine.scan_id
        captured_images = self.controller.captured_images
        recognition_report = self.controller.current_report

        self.controller.process_current_state(platform_empty=False)

        self.assertEqual(self.state_machine.state, State.RESET)
        self.assertEqual(self.state_machine.scan_id, scan_id)
        self.assertIs(self.controller.captured_images, captured_images)
        self.assertIs(self.controller.current_report, recognition_report)

        self.controller.process_current_state(platform_empty=True)

        self.assertEqual(self.state_machine.state, State.IDLE)
        self.assertIsNone(self.state_machine.scan_id)
        self.assertIsNone(self.controller.captured_images)
        self.assertIsNone(self.controller.current_report)


if __name__ == "__main__":
    unittest.main()