"""Tests for edge runtime configuration validation."""

import unittest
from pathlib import Path

from edge.runtime_config import RuntimeConfig, development_config, load_runtime_config
from unittest.mock import patch

class TestRuntimeConfig(unittest.TestCase):
    def make_config(self, **overrides) -> RuntimeConfig:
        values = {
            "api_base_url": "http://localhost:8080",
            "capture_directory": Path("captures"),
            "mock_image_directory": Path("fixtures/development"),
            "camera_ids": (
                "overhead",
                "oblique_csi0",
                "oblique_csi1",
            ),
            "recognition_threshold": 0.80,
            "association_distance": 0.10,
            "report_timeout_seconds": 5.0,
            "poll_interval_seconds": 0.1,
            "model_path": None,
        }

        values.update(overrides)

        return RuntimeConfig(**values)

    def test_development_config_is_valid(self):
        config = development_config()

        self.assertEqual(
            config.api_base_url,
            "http://localhost:8080",
        )

    def test_invalid_api_url_is_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "api_base_url",
        ):
            self.make_config(
                api_base_url="localhost:8080",
            )

    def test_requires_exactly_three_camera_ids(self):
        with self.assertRaisesRegex(
            ValueError,
            "exactly three",
        ):
            self.make_config(
                camera_ids=("overhead", "oblique_csi0"),
            )

    def test_empty_camera_id_is_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "empty",
        ):
            self.make_config(
                camera_ids=(
                    "overhead",
                    "",
                    "oblique_csi1",
                ),
            )

    def test_duplicate_camera_ids_are_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "unique",
        ):
            self.make_config(
                camera_ids=(
                    "overhead",
                    "overhead",
                    "oblique_csi1",
                ),
            )

    def test_invalid_recognition_threshold_is_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "recognition_threshold",
        ):
            self.make_config(
                recognition_threshold=1.1,
            )

    def test_nonpositive_association_distance_is_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "association_distance",
        ):
            self.make_config(
                association_distance=0.0,
            )

    def test_nonpositive_report_timeout_is_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "report_timeout_seconds",
        ):
            self.make_config(
                report_timeout_seconds=0.0,
            )

    def test_nonpositive_poll_interval_is_rejected(self):
        with self.assertRaisesRegex(
            ValueError,
            "poll_interval_seconds",
        ):
            self.make_config(
                poll_interval_seconds=0.0,
            )
    def test_environment_overrides_runtime_config(self):
        with patch.dict(
            "os.environ",
            {
                "STOCKNSTASH_API_BASE_URL": "http://192.168.1.50:8080",
                "STOCKNSTASH_CAPTURE_DIRECTORY": "custom-captures",
                "STOCKNSTASH_MOCK_IMAGE_DIRECTORY": "custom-fixtures",
                "STOCKNSTASH_CAMERA_IDS": "cam-a,cam-b,cam-c",
                "STOCKNSTASH_RECOGNITION_THRESHOLD": "0.75",
                "STOCKNSTASH_ASSOCIATION_DISTANCE": "0.20",
                "STOCKNSTASH_REPORT_TIMEOUT_SECONDS": "3.5",
                "STOCKNSTASH_POLL_INTERVAL_SECONDS": "0.25",
                "STOCKNSTASH_MODEL_PATH": "models/test.hef",
            },
            clear=True,
    ):
            config = load_runtime_config()

        self.assertEqual(
            config.api_base_url,
            "http://192.168.1.50:8080",
        )
        
        self.assertEqual(
            config.capture_directory,
            Path("custom-captures"),
        )
    
        self.assertEqual(
            config.mock_image_directory,
            Path("custom-fixtures"),
        )
    
        self.assertEqual(
            config.camera_ids,
            ("cam-a", "cam-b", "cam-c"),
        )
    
        self.assertEqual(
            config.recognition_threshold,
            0.75,
        )
    
        self.assertEqual(
            config.association_distance,
            0.20,
        )
    
        self.assertEqual(
            config.report_timeout_seconds,
            3.5,
        )
        self.assertEqual(
            config.poll_interval_seconds,
            0.25,
        )
    
        self.assertEqual(
            config.model_path,
            Path("models/test.hef"),
        )
    def test_invalid_environment_float_has_actionable_error(self):
        with patch.dict(
            "os.environ",
            {
                "STOCKNSTASH_REPORT_TIMEOUT_SECONDS": "banana",
            },
            clear=True,
        ):
            with self.assertRaisesRegex(
                ValueError,
                "STOCKNSTASH_REPORT_TIMEOUT_SECONDS must be a number",
            ):
                load_runtime_config()

if __name__ == "__main__":
    unittest.main()