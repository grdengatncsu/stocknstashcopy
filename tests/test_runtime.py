"""Tests for edge runtime composition and stepping."""
import unittest
import signal
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import TestCase
from unittest.mock import Mock, call, patch

from edge.association.position_associator import PositionAssociator
from edge.presence import (
    PresenceSignals,
    SimulatedPresenceSource,
)
from edge.recognition.mock_recognizer import MockRecognizer
from edge.reporting.mock_reporter import MockReporter
from edge.runtime import (
    process_runtime_step,
    run_forever,
    run_simulated_scan,
    install_signal_handlers,
)
from edge.scan_controller import ScanController
from edge.state_machine import State, StateMachine


class TestRuntime(unittest.TestCase):
    def test_runtime_step_passes_presence_signals_to_controller(self):
        controller = Mock()

        presence_source = Mock()
        presence_source.read.return_value = PresenceSignals(
            weight_detected=True,
            weight_present=True,
            weight_stable=False,
            platform_empty=False,
        )

        process_runtime_step(
            controller=controller,
            presence_source=presence_source,
        )

        presence_source.read.assert_called_once_with()

        controller.process_current_state.assert_called_once_with(
            weight_detected=True,
            weight_present=True,
            weight_stable=False,
            platform_empty=False,
        )
    def test_positioned_mock_observations_merge_into_one_item(self):
        with TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            captured_images = {}
            for camera_id in (
                "overhead",
                "oblique_csi0",
                "oblique_csi1",
            ):
                image_path = root / f"{camera_id}.jpg"
                image_path.write_bytes(b"fake image")
                captured_images[camera_id] = image_path

            recognizer = MockRecognizer(
                predictions={
                    "overhead": ("milk", 0.95, (0.50, 0.50)),
                    "oblique_csi0": ("milk", 0.92, (0.53, 0.51)),
                    "oblique_csi1": ("milk", 0.90, (0.49, 0.52)),
                }
            )

            recognition_report = recognizer.recognize(
                scan_id="test-scan",
                captured_images=captured_images,
            )

            association_report = PositionAssociator(
                maximum_distance=0.10
            ).associate(recognition_report)

            self.assertEqual(len(association_report.items), 1)

            item = association_report.items[0]

            self.assertEqual(item.name, "milk")
            self.assertEqual(len(item.observations), 3)

    def test_run_forever_stops_when_requested(self):
        controller = Mock()
        controller.state_machine.state = State.IDLE
        controller.state_machine.scan_id = None

        presence_source = Mock()
        presence_source.read.return_value = PresenceSignals()

        stop_event = Mock()

        # First check: keep running.
        # Second check: stop.
        stop_event.is_set.side_effect = [False, True]

        run_forever(
            controller=controller,
            presence_source=presence_source,
            stop_event=stop_event,
            poll_interval_seconds=0.1,
        )

        presence_source.read.assert_called_once()

        controller.process_current_state.assert_called_once_with(
            weight_detected=False,
            weight_present=False,
            weight_stable=False,
            platform_empty=False,
        )

        stop_event.wait.assert_called_once_with(0.1)

    def test_signal_handlers_request_shutdown(self):
        stop_event = Mock()

        with patch("edge.runtime.signal.signal") as mock_signal:
            install_signal_handlers(stop_event)

        mock_signal.assert_has_calls(
            [
                call(signal.SIGINT, mock_signal.call_args_list[0].args[1]),
                call(signal.SIGTERM, mock_signal.call_args_list[1].args[1]),
            ]
        )

        sigint_handler = mock_signal.call_args_list[0].args[1]

        sigint_handler(signal.SIGINT, None)

        stop_event.set.assert_called_once_with()

    def test_report_retry_keeps_same_scan_id(self):
        reporter = Mock()
        reporter.report.side_effect = [False, True]

        state_machine = StateMachine()
        state_machine.state = State.REPORT
        state_machine.scan_id = "scan-retry-123"

        association_report = Mock()
        association_report.scan_id = "scan-retry-123"

        controller = ScanController(
            state_machine=state_machine,
            capture=Mock(),
            recognizer=Mock(),
            reporter=reporter,
            associator=Mock(),
        )

        controller.current_association_report = association_report

        controller.process_report()

        self.assertEqual(
            controller.state_machine.state,
            State.REPORT,
        )
        self.assertEqual(
            controller.state_machine.scan_id,
            "scan-retry-123",
        )

        controller.process_report()

        self.assertEqual(
            controller.state_machine.state,
            State.RESET,
        )
        self.assertEqual(
            controller.state_machine.scan_id,
            "scan-retry-123",
        )

        self.assertEqual(reporter.report.call_count, 2)

    def test_simulated_scan_completes_full_lifecycle(self):
        capture = Mock()

        with TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)

            captured_images = {}

            for camera_id in (
                "overhead",
                "oblique_csi0",
                "oblique_csi1",
            ):
                image_path = root / f"{camera_id}.jpg"
                image_path.write_bytes(b"fake image")
                captured_images[camera_id] = image_path

            capture.capture_all.return_value = captured_images

            recognizer = MockRecognizer(
                predictions={
                    "overhead": ("milk", 0.95, (0.50, 0.50)),
                    "oblique_csi0": ("milk", 0.92, (0.53, 0.51)),
                    "oblique_csi1": ("milk", 0.90, (0.49, 0.52)),
                }
            )

            reporter = MockReporter(
                should_acknowledge=True,
            )

            controller = ScanController(
                state_machine=StateMachine(),
                capture=capture,
                recognizer=recognizer,
                associator=PositionAssociator(
                    maximum_distance=0.10,
                ),
                reporter=reporter,
            )

            presence_source = SimulatedPresenceSource()

            run_simulated_scan(
                controller=controller,
                presence_source=presence_source,
            )

            self.assertEqual(
                controller.state_machine.state,
                State.IDLE,
            )

            self.assertIsNone(
                controller.state_machine.scan_id,
            )

            self.assertIsNone(
                controller.current_association_report,
            )

            self.assertEqual(
                len(reporter.processed_scan_ids),
                1,
            )

            capture.capture_all.assert_called_once()

if __name__ == "__main__":
    unittest.main()