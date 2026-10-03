import os

from dataclasses import dataclass
from pathlib import Path
from urllib.parse import urlparse

@dataclass(frozen=True)
class RuntimeConfig:
    api_base_url: str
    capture_directory: Path
    camera_ids: tuple[str, ...]
    mock_image_directory: Path

    recognition_threshold: float
    association_distance: float

    report_timeout_seconds: float
    poll_interval_seconds: float

    model_path: Path | None = None

    def __post_init__(self):
        parsed_url = urlparse(self.api_base_url)

        if parsed_url.scheme not in ("http", "https") or not parsed_url.netloc:
            raise ValueError("api_base_url must be a valid HTTP or HTTPS URL")

        if len(self.camera_ids) != 3:
            raise ValueError("camera_ids must contain exactly three unique camera IDs")
        if any(not camera_id.strip() for camera_id in self.camera_ids):
            raise ValueError("camera_ids must not contain empty or whitespace-only strings")

        if len(set(self.camera_ids)) != 3:
            raise ValueError("camera_ids must contain unique camera IDs")

        if not 0.0 <= self.recognition_threshold <= 1.0:
            raise ValueError("recognition_threshold must be between 0.0 and 1.0")

        if self.association_distance <= 0.0:
            raise ValueError("association_distance must be a positive number")
        if self.report_timeout_seconds <= 0.0:
            raise ValueError("report_timeout_seconds must be a positive number")
        if self.poll_interval_seconds <= 0.0:
            raise ValueError("poll_interval_seconds must be a positive number")
    
def development_config() -> RuntimeConfig:
    return RuntimeConfig(
        api_base_url="http://localhost:8080",
        capture_directory=Path("captures"),
        mock_image_directory=Path("fixtures/development"),
        camera_ids=("overhead", "oblique_csi0", "oblique_csi1"),
        recognition_threshold=0.8,
        association_distance=0.10,
        report_timeout_seconds=5.0,
        poll_interval_seconds=0.1,
        model_path=None,
    )

def load_runtime_config() -> RuntimeConfig:
    defaults = development_config()

    camera_ids_value = os.getenv(
        "STOCKNSTASH_CAMERA_IDS",
        ",".join(defaults.camera_ids),
    )

    camera_ids = tuple(
        camera_id.strip()
        for camera_id in camera_ids_value.split(",")
    )

    model_path_value = os.getenv("STOCKNSTASH_MODEL_PATH")

    return RuntimeConfig(
        api_base_url=os.getenv(
            "STOCKNSTASH_API_BASE_URL",
            defaults.api_base_url,
        ),
        capture_directory=Path(
            os.getenv(
                "STOCKNSTASH_CAPTURE_DIRECTORY",
                str(defaults.capture_directory),
            )
        ),
        mock_image_directory=Path(
            os.getenv(
                "STOCKNSTASH_MOCK_IMAGE_DIRECTORY",
                str(defaults.mock_image_directory),
            )
        ),
        camera_ids=camera_ids,
        recognition_threshold=_float_from_env(
            "STOCKNSTASH_RECOGNITION_THRESHOLD",
            defaults.recognition_threshold,
        ),
        association_distance=_float_from_env(
            "STOCKNSTASH_ASSOCIATION_DISTANCE",
            defaults.association_distance,
        ),
        report_timeout_seconds=_float_from_env(
            "STOCKNSTASH_REPORT_TIMEOUT_SECONDS",
            defaults.report_timeout_seconds,
        ),
        poll_interval_seconds=_float_from_env(
            "STOCKNSTASH_POLL_INTERVAL_SECONDS",
            defaults.poll_interval_seconds,
        ),
        model_path=(
            Path(model_path_value)
            if model_path_value
            else defaults.model_path
        ),
    )

def _float_from_env(name: str, default: float) -> float:
    value = os.getenv(name)
    if value is None:
        return default
    try:
        return float(value)
    except ValueError as error:
        raise ValueError(f"{name} must be a number, got {value!r}") from error

