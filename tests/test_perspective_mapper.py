"""Tests for mapping camera bounding boxes into normalized platform positions."""

import unittest

from edge.positioning.perspective_mapper import PerspectiveCoordinateMapper


IDENTITY_MATRIX = (
    (1.0, 0.0, 0.0),
    (0.0, 1.0, 0.0),
    (0.0, 0.0, 1.0),
)


class TestPerspectiveCoordinateMapper(unittest.TestCase):
    def assert_position_almost_equal(self, actual, expected):
        self.assertIsNotNone(actual)
        self.assertAlmostEqual(actual[0], expected[0])
        self.assertAlmostEqual(actual[1], expected[1])

    def test_overhead_camera_uses_bounding_box_center(self):
        mapper = PerspectiveCoordinateMapper(
            {"overhead": IDENTITY_MATRIX}
        )

        position = mapper.map_detection(
            "overhead",
            (0.2, 0.4, 0.6, 0.8),
        )

        self.assert_position_almost_equal(position, (0.4, 0.6))

    def test_angled_camera_uses_bounding_box_bottom_center(self):
        mapper = PerspectiveCoordinateMapper(
            {"rear_left": IDENTITY_MATRIX}
        )

        position = mapper.map_detection(
            "rear_left",
            (0.2, 0.4, 0.6, 0.8),
        )

        self.assert_position_almost_equal(position, (0.4, 0.8))

    def test_homography_transforms_anchor(self):
        matrix = (
            (0.5, 0.0, 0.1),
            (0.0, 0.5, 0.2),
            (0.0, 0.0, 1.0),
        )
        mapper = PerspectiveCoordinateMapper({"overhead": matrix})

        position = mapper.map_detection(
            "overhead",
            (0.2, 0.4, 0.6, 0.8),
        )

        self.assert_position_almost_equal(position, (0.3, 0.5))

    def test_missing_homography_returns_none(self):
        mapper = PerspectiveCoordinateMapper({})

        position = mapper.map_detection(
            "overhead",
            (0.2, 0.4, 0.6, 0.8),
        )

        self.assertIsNone(position)

    def test_zero_scale_returns_none(self):
        zero_scale_matrix = (
            (1.0, 0.0, 0.0),
            (0.0, 1.0, 0.0),
            (0.0, 0.0, 0.0),
        )
        mapper = PerspectiveCoordinateMapper(
            {"overhead": zero_scale_matrix}
        )

        position = mapper.map_detection(
            "overhead",
            (0.2, 0.4, 0.6, 0.8),
        )

        self.assertIsNone(position)

    def test_out_of_bounds_position_returns_none(self):
        outside_matrix = (
            (1.0, 0.0, 1.0),
            (0.0, 1.0, 0.0),
            (0.0, 0.0, 1.0),
        )
        mapper = PerspectiveCoordinateMapper(
            {"overhead": outside_matrix}
        )

        position = mapper.map_detection(
            "overhead",
            (0.2, 0.4, 0.6, 0.8),
        )

        self.assertIsNone(position)


if __name__ == "__main__":
    unittest.main()
