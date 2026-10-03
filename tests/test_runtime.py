"""Tests for edge runtime composition and stepping."""

import unittest
from unittest.mock import Mock

from edge.presence import PresenceSignals
from edge.runtime import process_runtime_step


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
    from pathlib import Path
    from tempfile import TemporaryDirectory

    from edge.association.position_associator import PositionAssociator
    from edge.recognition.mock_recognizer import MockRecognizer

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

if __name__ == "__main__":
    unittest.main()