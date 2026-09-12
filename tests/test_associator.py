import unittest

from edge.association.associator import AssociationReport
from edge.association.position_associator import PositionAssociator
from edge.recognition.recognizer import RecognizedItem, RecognitionReport


class TestPositionAssociator(unittest.TestCase):
    def make_observation(
        self,
        name: str,
        confidence: float,
        source_camera: str,
        platform_position: tuple[float, float] | None,
        unknown: bool = False,
    ) -> RecognizedItem:
        return RecognizedItem(
            name=name,
            confidence=confidence,
            source_camera=source_camera,
            unknown=unknown,
            platform_position=platform_position,
        )

    def test_invalid_maximum_distance_raises_value_error(self):
        with self.assertRaises(ValueError):
            PositionAssociator(maximum_distance=0)

        with self.assertRaises(ValueError):
            PositionAssociator(maximum_distance=-0.1)

    def test_calculate_distance(self):
        associator = PositionAssociator()

        distance = associator.calculate_distance(
            (0.20, 0.60),
            (0.22, 0.58),
        )

        self.assertAlmostEqual(distance, 0.0282842712)

    def test_same_item_from_three_cameras_is_associated(self):
        associator = PositionAssociator(maximum_distance=0.1)
        report = RecognitionReport(
            scan_id="scan-123",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", (0.20, 0.60)
                ),
                self.make_observation(
                    "apple", 0.8, "left", (0.22, 0.58)
                ),
                self.make_observation(
                    "apple", 0.7, "right", (0.21, 0.61)
                ),
            ],
        )

        result = associator.associate(report)

        self.assertIsInstance(result, AssociationReport)
        self.assertEqual(result.scan_id, "scan-123")
        self.assertEqual(len(result.items), 1)

        associated_item = result.items[0]

        self.assertEqual(associated_item.name, "apple")
        self.assertEqual(len(associated_item.observations), 3)
        self.assertAlmostEqual(associated_item.confidence, 0.8)
        self.assertAlmostEqual(
            associated_item.platform_position[0],
            0.21,
        )
        self.assertAlmostEqual(
            associated_item.platform_position[1],
            0.5966666667,
        )

    def test_different_names_are_not_associated(self):
        associator = PositionAssociator()
        report = RecognitionReport(
            scan_id="scan-different-names",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", (0.20, 0.60)
                ),
                self.make_observation(
                    "orange", 0.9, "left", (0.21, 0.59)
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(len(result.items), 2)

    def test_same_camera_observations_are_not_associated(self):
        associator = PositionAssociator()
        report = RecognitionReport(
            scan_id="scan-same-camera",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", (0.20, 0.60)
                ),
                self.make_observation(
                    "apple", 0.8, "overhead", (0.21, 0.59)
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(len(result.items), 2)

    def test_distant_observations_are_not_associated(self):
        associator = PositionAssociator(maximum_distance=0.1)
        report = RecognitionReport(
            scan_id="scan-distant",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", (0.10, 0.10)
                ),
                self.make_observation(
                    "apple", 0.8, "left", (0.80, 0.80)
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(len(result.items), 2)

    def test_unknown_observations_are_not_associated(self):
        associator = PositionAssociator()
        report = RecognitionReport(
            scan_id="scan-unknown",
            items=[
                self.make_observation(
                    "unknown",
                    0.0,
                    "overhead",
                    (0.20, 0.60),
                    unknown=True,
                ),
                self.make_observation(
                    "unknown",
                    0.0,
                    "left",
                    (0.21, 0.59),
                    unknown=True,
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(len(result.items), 2)

    def test_missing_positions_are_not_associated(self):
        associator = PositionAssociator()
        report = RecognitionReport(
            scan_id="scan-missing-position",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", None
                ),
                self.make_observation(
                    "apple", 0.8, "left", None
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(len(result.items), 2)

    def test_observation_joins_closest_compatible_item(self):
        associator = PositionAssociator(maximum_distance=0.1)
        report = RecognitionReport(
            scan_id="scan-closest",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", (0.10, 0.10)
                ),
                self.make_observation(
                    "apple", 0.8, "overhead", (0.80, 0.80)
                ),
                self.make_observation(
                    "apple", 0.7, "left", (0.78, 0.81)
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(len(result.items), 2)
        self.assertEqual(len(result.items[0].observations), 1)
        self.assertEqual(len(result.items[1].observations), 2)

    def test_association_ids_are_unique_within_scan(self):
        associator = PositionAssociator()
        report = RecognitionReport(
            scan_id="scan-identifiers",
            items=[
                self.make_observation(
                    "apple", 0.9, "overhead", (0.10, 0.10)
                ),
                self.make_observation(
                    "orange", 0.8, "left", (0.80, 0.80)
                ),
            ],
        )

        result = associator.associate(report)

        self.assertEqual(
            [item.association_id for item in result.items],
            [
                "scan-identifiers:item-1",
                "scan-identifiers:item-2",
            ],
        )

    def test_empty_scan_id_raises_value_error(self):
        associator = PositionAssociator()
        report = RecognitionReport(scan_id="", items=[])

        with self.assertRaises(ValueError):
            associator.associate(report)


if __name__ == "__main__":
    unittest.main()