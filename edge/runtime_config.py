from dataclasses import dataclass
from pathlib import Path

@dataclass(frozen=True)
class RuntimeConfig:
    api_base_url: str
    capture_directory: Path
    camera_ids: tuple[str, str, str]
    mock_image_directory: Path

    recognition_threshold: float
    association_distance: float

    report_timeout_seconds: float

    model_path: Path | None = None

def development_config() -> RuntimeConfig:
    return RuntimeConfig(
        api_base_url="http://localhost:8080",
        capture_directory=Path("captures"),
        mock_image_directory=Path("fixtures/development"),
        camera_ids=("overhead", "oblique_csi0", "oblique_csi1"),
        recognition_threshold=0.8,
        association_distance=0.10,
        report_timeout_seconds=5.0,
        model_path=None,
    )