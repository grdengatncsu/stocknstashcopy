"""Interface shared by simulated and physical camera implementations."""

from pathlib import Path
from typing import Protocol


class Camera(Protocol):
    """Capture interface required by ``ThreeCameraCapture``.

    PLATFORM BLOCKED (#33): Add a physical implementation after provisional
    MIPI devices ``oblique_csi0`` and ``oblique_csi1`` are mounted and assigned
    to ``rear_left`` and ``front_right``. The overhead identity is verified.
    Preserve logical IDs regardless of transient Linux device numbering.
    """

    camera_id: str

    def capture_image(self, save_path: str | Path) -> Path:
        """Capture one image, save it below ``save_path``, and return its path."""
        ...
