"""Perspective-based conversion from camera coordinates to platform position."""

from edge.positioning.coordinate_mapper import CoordinateMapper

# A homography is a 3-by-3 calibration matrix that maps points from one plane
# (the camera image) to another (the physical platform).
Matrix3x3 = tuple[
    tuple[float, float, float],
    tuple[float, float, float],
    tuple[float, float, float],
]


class PerspectiveCoordinateMapper(CoordinateMapper):
    """Apply each camera's calibration matrix to detected item locations."""

    def __init__(
        self,
        homographies: dict[str, Matrix3x3],
        overhead_camera_id: str = "overhead",
    ):
        """Store calibration matrices keyed by the camera IDs used at capture."""
        self.homographies = homographies
        self.overhead_camera_id = overhead_camera_id

    def calculate_anchor(
        self,
        camera_id: str,
        bounding_box: tuple[float, float, float, float],
    ) -> tuple[float, float]:
        """Choose the bounding-box point that best represents item position.

        Overhead views use the object's center. Side views use the bottom-center
        because that is the approximate point where the object meets the
        platform plane used during calibration.
        """
        x1, y1, x2, y2 = bounding_box
        anchor_x = (x1 + x2) / 2
        if camera_id == self.overhead_camera_id:
            anchor_y = (y1 + y2) / 2
        else:
            anchor_y = y2
        return (anchor_x, anchor_y)

    def map_detection(
        self,
        camera_id: str,
        bounding_box: tuple[float, float, float, float],
    ) -> tuple[float, float] | None:
        """Return a normalized platform point, or ``None`` when mapping is invalid."""
        homography = self.homographies.get(camera_id)

        if homography is None:
            # An uncalibrated camera cannot produce a trustworthy position.
            return None

        image_x, image_y = self.calculate_anchor(camera_id, bounding_box)

        first_row, second_row, third_row = homography

        # Multiply the anchor point by the homography. The third row produces a
        # scale factor used to convert homogeneous coordinates back to x and y.
        mapped_x_raw = first_row[0] * image_x + first_row[1] * image_y + first_row[2]
        mapped_y_raw = second_row[0] * image_x + second_row[1] * image_y + second_row[2]
        scale = third_row[0] * image_x + third_row[1] * image_y + third_row[2]

        if abs(scale) < 1e-9:
            return None

        platform_x = mapped_x_raw / scale
        platform_y = mapped_y_raw / scale

        if not (0 <= platform_x <= 1 and 0 <= platform_y <= 1):
            # Values outside the calibrated platform should not participate in
            # multi-camera association.
            return None

        return (platform_x, platform_y)
