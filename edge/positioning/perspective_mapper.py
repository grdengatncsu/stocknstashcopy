"""Perspective mapping from camera coordinates to platform position."""

from edge.positioning.coordinate_mapper import CoordinateMapper

# A homography is a 3-by-3 calibration matrix mapping image points onto the
# physical platform plane.
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
        """Store calibration matrices under their permanent logical camera IDs.

        HARDWARE TODO: Replace development matrices with values measured after
        the final cameras and platform are physically mounted. Recalibrate if a
        camera is moved, its resolution changes, or the platform geometry
        changes.
        """
        self.homographies = homographies
        self.overhead_camera_id = overhead_camera_id

    def calculate_anchor(
        self,
        camera_id: str,
        bounding_box: tuple[float, float, float, float],
    ) -> tuple[float, float]:
        """Choose the box point that best represents position on the platform.

        The overhead view uses the object's center. Side views use the
        bottom-center, which approximates where the object touches the platform
        plane used for calibration.
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
        """Return a normalized platform point or ``None`` if mapping is invalid."""
        homography = self.homographies.get(camera_id)

        if homography is None:
            return None

        image_x, image_y = self.calculate_anchor(camera_id, bounding_box)

        first_row, second_row, third_row = homography

        # Multiply the anchor point by the homography. The third row gives the
        # scale used to convert homogeneous coordinates back to x and y.
        mapped_x_raw = first_row[0] * image_x + first_row[1] * image_y + first_row[2]
        mapped_y_raw = second_row[0] * image_x + second_row[1] * image_y + second_row[2]
        scale = third_row[0] * image_x + third_row[1] * image_y + third_row[2]

        if abs(scale) < 1e-9:
            return None

        platform_x = mapped_x_raw / scale
        platform_y = mapped_y_raw / scale

        if not (0 <= platform_x <= 1 and 0 <= platform_y <= 1):
            # Out-of-bounds detections cannot be safely associated across views.
            return None

        return (platform_x, platform_y)
