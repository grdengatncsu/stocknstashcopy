"""Unit tests for coordination between scan stages and injected components."""

import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock

from edge.association.associator import AssociationReport
from edge.association.position_associator import PositionAssociator
from edge.cameras.three_camera_capture import ThreeCameraCapture
from edge.errors import ReportingError
from edge.recognition.mock_recognizer import MockRecognizer
from edge.recognition.recognizer import RecognizedItem, RecognitionReport
from edge.reporting.mock_reporter import MockReporter
from edge.scan_controller import ScanController
from edge.state_machine import State, StateMachine


class TestScanController(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)

        self.captured_images = {
            "camera_1": self.create_image("camera_1"),
            "camera_2": self.create_image("camera_2"),
            "camera_3": self.create_image("camera_3"),
        }

        self.capture = Mock(spec=ThreeCameraCapture)
        self.capture.capture_all.return_value = self.captured_images

        self.recognizer = MockRecognizer(
            predictions={
                "camera_1": ("milk", 0.95),
                "camera_2": ("milk", 0.91),
                "camera_3": ("milk", 0.89),
            }
        )

        self.associator = PositionAssociator(
            maximum_distance=0.1
        )
        self.reporter = MockReporter(should_acknowledge=True)
        self.state_machine = StateMachine()

        self.controller = ScanController(
            state_machine=self.state_machine,
            capture=self.capture,
            recognizer=self.recognizer,
            reporter=self.reporter,
            associator=self.associator,
        )

    def create_image(self, camera_id: str) -> Path:
        image_path = self.root / f"{camera_id}.jpg"
        image_path.write_bytes(
            f"image from {camera_id}".encode()
        )
        return image_path

    def advance_to_capture(self):
        self.controller.process_current_state(
            weight_detected=True
        )
        self.controller.process_current_state(
            weight_present=True,
            weight_stable=True,
        )

    def advance_to_report(self):
        self.advance_to_capture()
        self.controller.process_current_state()
        self.controller.process_current_state()

    def test_successful_scan_advances_one_state_per_call(self):
        self.assertEqual(
            self.state_machine.state,
            State.IDLE,
        )

        self.controller.process_current_state(
            weight_detected=True
        )
        self.assertEqual(
            self.state_machine.state,
            State.STABILIZING,
        )

        scan_id = self.state_machine.scan_id

        self.controller.process_current_state(
            weight_present=True,
            weight_stable=True,
        )
        self.assertEqual(
            self.state_machine.state,
            State.CAPTURE,
        )

        self.controller.process_current_state()
        self.assertEqual(
            self.state_machine.state,
            State.RECOGNIZE,
        )

        self.controller.process_current_state()
        self.assertEqual(
            self.state_machine.state,
            State.REPORT,
        )
        self.assertIsNotNone(
            self.controller.current_association_report
        )

        self.controller.process_current_state()
        self.assertEqual(
            self.state_machine.state,
            State.RESET,
        )

        self.controller.process_current_state(
            platform_empty=True
        )
        self.assertEqual(
            self.state_machine.state,
            State.IDLE,
        )

        self.assertIsNone(self.controller.captured_images)
        self.assertIsNone(self.controller.current_report)
        self.assertIsNone(
            self.controller.current_association_report
        )
        self.assertIn(
            scan_id,
            self.reporter.processed_scan_ids,
        )

        self.capture.capture_all.assert_called_once_with(
            scan_id=scan_id
        )

    def test_recognition_results_are_associated_before_reporting(
        self,
    ):
        def recognize_with_positions(
            scan_id,
            captured_images,
        ):
            return RecognitionReport(
                scan_id=scan_id,
                items=[
                    RecognizedItem(
                        name="apple",
                        confidence=0.95,
                        source_camera="camera_1",
                        platform_position=(0.50, 0.50),
                    ),
                    RecognizedItem(
                        name="apple",
                        confidence=0.91,
                        source_camera="camera_2",
                        platform_position=(0.53, 0.52),
                    ),
                    RecognizedItem(
                        name="apple",
                        confidence=0.89,
                        source_camera="camera_3",
                        platform_position=(0.49, 0.51),
                    ),
                ],
            )

        positioned_recognizer = Mock()
        positioned_recognizer.recognize.side_effect = (
            recognize_with_positions
        )
        self.controller.recognizer = positioned_recognizer

        recording_reporter = Mock()
        recording_reporter.report.return_value = True
        self.controller.reporter = recording_reporter

        self.advance_to_report()

        association_report = (
            self.controller.current_association_report
        )

        self.assertIsInstance(
            association_report,
            AssociationReport,
        )
        self.assertEqual(len(association_report.items), 1)

        associated_item = association_report.items[0]
        self.assertEqual(associated_item.name, "apple")
        self.assertEqual(
            len(associated_item.observations),
            3,
        )

        self.controller.process_current_state()

        sent_report = (
            recording_reporter.report.call_args.args[0]
        )
        self.assertIs(sent_report, association_report)
        self.assertEqual(
            self.state_machine.state,
            State.RESET,
        )

    def test_unacknowledged_report_remains_available_for_retry(
        self,
    ):
        self.advance_to_report()

        original_recognition_report = (
            self.controller.current_report
        )
        original_association_report = (
            self.controller.current_association_report
        )
        scan_id = self.state_machine.scan_id

        self.reporter.should_acknowledge = False
        self.controller.process_current_state()

        self.assertEqual(
            self.state_machine.state,
            State.REPORT,
        )
        self.assertIs(
            self.controller.current_report,
            original_recognition_report,
        )
        self.assertIs(
            self.controller.current_association_report,
            original_association_report,
        )
        self.assertNotIn(
            scan_id,
            self.reporter.processed_scan_ids,
        )

        self.reporter.should_acknowledge = True
        self.controller.process_current_state()

        self.assertEqual(
            self.state_machine.state,
            State.RESET,
        )
        self.assertIn(
            scan_id,
            self.reporter.processed_scan_ids,
        )

    def test_occupied_platform_preserves_current_scan_data(
        self,
    ):
        self.advance_to_report()
        self.controller.process_current_state()

        captured_images = self.controller.captured_images
        recognition_report = self.controller.current_report
        association_report = (
            self.controller.current_association_report
        )

        self.controller.process_current_state(
            platform_empty=False
        )

        self.assertEqual(
            self.state_machine.state,
            State.RESET,
        )
        self.assertIs(
            self.controller.captured_images,
            captured_images,
        )
        self.assertIs(
            self.controller.current_report,
            recognition_report,
        )
        self.assertIs(
            self.controller.current_association_report,
            association_report,
        )

        self.controller.process_current_state(
            platform_empty=True
        )

        self.assertEqual(
            self.state_machine.state,
            State.IDLE,
        )
        self.assertIsNone(self.controller.captured_images)
        self.assertIsNone(self.controller.current_report)
        self.assertIsNone(
            self.controller.current_association_report
        )

    def test_capture_failure_enters_error_and_can_recover(
        self,
    ):
        self.capture.capture_all.side_effect = (
            FileNotFoundError("camera image missing")
        )
        self.advance_to_capture()

        self.controller.process_current_state()

        self.assertEqual(
            self.state_machine.state,
            State.ERROR,
        )
        self.assertEqual(
            self.state_machine.error_source,
            State.CAPTURE,
        )
        self.assertFalse(self.state_machine.cap_valid)

        self.capture.capture_all.side_effect = None
        self.controller.process_current_state(
            recovered=True
        )

        self.assertEqual(
            self.state_machine.state,
            State.CAPTURE,
        )

        self.controller.process_current_state()

        self.assertEqual(
            self.state_machine.state,
            State.RECOGNIZE,
        )
        self.assertTrue(self.state_machine.cap_valid)

    def test_recognition_failure_enters_error(self):
        self.advance_to_capture()
        self.controller.process_current_state()

        self.captured_images["camera_3"].unlink()
        self.controller.process_current_state()

        self.assertEqual(
            self.state_machine.state,
            State.ERROR,
        )
        self.assertEqual(
            self.state_machine.error_source,
            State.RECOGNIZE,
        )
        self.assertTrue(self.state_machine.cap_valid)
        self.assertFalse(self.state_machine.rec_valid)
        self.assertIsNone(self.controller.current_report)
        self.assertIsNone(
            self.controller.current_association_report
        )

    def test_reporting_exception_enters_error_and_preserves_report(
        self,
    ):
        self.advance_to_report()

        original_recognition_report = (
            self.controller.current_report
        )
        original_association_report = (
            self.controller.current_association_report
        )

        self.reporter.report = Mock(
            side_effect=ReportingError("API rejected the report")
        )

        self.controller.process_current_state()

        self.assertEqual(
            self.state_machine.state,
            State.ERROR,
        )
        self.assertEqual(
            self.state_machine.error_source,
            State.REPORT,
        )
        self.assertFalse(self.state_machine.report_ack)
        self.assertIs(
            self.controller.current_report,
            original_recognition_report,
        )
        self.assertIs(
            self.controller.current_association_report,
            original_association_report,
        )


if __name__ == "__main__":
    unittest.main()
