from edge.positioning.coordinate_mapper import CoordinateMapper

Matrix3x3 = tuple[tuple[float, float, float], tuple[float, float, float], tuple[float, float, float]]

class PerspectiveCoordinateMapper(CoordinateMapper):
    """Convert camera detections into shared platform coordinates using perspective mapping."""
    def __init__(self, homographies: dict[str, Matrix3x3], overhead_camera_id: str = "overhead"):
        self.homographies = homographies
        self.overhead_camera_id = overhead_camera_id

    def calculate_anchor(self, camera_id: str, bounding_box: tuple[float, float, float, float]) -> tuple[float, float]:
        """Calculate the anchor point for a detection based on its bounding box."""
        x1, y1, x2, y2 = bounding_box
        anchor_x = (x1 + x2) / 2
        if camera_id == self.overhead_camera_id:
            anchor_y = (y1 + y2) / 2  # Use the center of the bounding box as the anchor point for overhead camera
        else:
            anchor_y = y2  # Use the bottom of the bounding box as the anchor point
        return (anchor_x, anchor_y)

    def map_detection(self, camera_id: str, bounding_box: tuple[float, float, float, float]) -> tuple[float, float] | None:
        """Map one detection into normalized platform coordinates using perspective mapping."""
        homography = self.homographies.get(camera_id)

        if homography is None:
            return None  # No homography available for this camera

        image_x, image_y = self.calculate_anchor(camera_id, bounding_box)

        first_row, second_row, third_row = homography

        mapped_x_raw = (first_row[0] * image_x + first_row[1] * image_y + first_row[2])
        mapped_y_raw = (second_row[0] * image_x + second_row[1] * image_y + second_row[2])
        scale = (third_row[0] * image_x + third_row[1] * image_y + third_row[2])

        if abs(scale) < 1e-9: 
            return None  # Avoid division by zero

        platform_x = mapped_x_raw / scale
        platform_y = mapped_y_raw / scale

        if not (0 <= platform_x <= 1 and 0 <= platform_y <= 1):
            return None  # Coordinates are out of bounds

        return (platform_x, platform_y)
        