"""End-to-end coverage for the Python reporter and the real Go API."""

import json
import os
import shutil
import socket
import subprocess
import tempfile
import time
import unittest
from pathlib import Path
from urllib.request import urlopen

from edge.association.associator import AssociatedItem, AssociationReport
from edge.recognition.recognizer import RecognizedItem
from edge.reporting.http_reporter import HttpReporter


RUN_GO_INTEGRATION = os.environ.get("STOCKNSTASH_RUN_GO_INTEGRATION") == "1"


@unittest.skipUnless(
    RUN_GO_INTEGRATION,
    "set STOCKNSTASH_RUN_GO_INTEGRATION=1 to run the Go integration test",
)
class TestHttpReporterGoIntegration(unittest.TestCase):
    """Exercise the reporter, HTTP boundary, SQLite, and duplicate handling."""

    @classmethod
    def setUpClass(cls):
        if shutil.which("go") is None:
            raise unittest.SkipTest("Go is not installed")

        cls.temporary_directory = tempfile.TemporaryDirectory()
        temporary_root = Path(cls.temporary_directory.name)
        cls.database_path = temporary_root / "stocknstash-integration.db"

        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
            listener.bind(("127.0.0.1", 0))
            cls.port = listener.getsockname()[1]

        cls.base_url = f"http://127.0.0.1:{cls.port}"
        repository_root = Path(__file__).resolve().parents[1]
        environment = os.environ.copy()
        environment.update(
            {
                "STOCKNSTASH_ADDR": f"127.0.0.1:{cls.port}",
                "STOCKNSTASH_DB_PATH": str(cls.database_path),
            }
        )
        cls.server = subprocess.Popen(
            ["go", "run", "."],
            cwd=repository_root / "server",
            env=environment,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )

        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            if cls.server.poll() is not None:
                output = cls.server.stdout.read() if cls.server.stdout else ""
                cls.temporary_directory.cleanup()
                raise RuntimeError(f"Go server exited during startup:\n{output}")
            try:
                with urlopen(f"{cls.base_url}/api/status", timeout=0.25):
                    return
            except OSError:
                time.sleep(0.1)

        cls.server.terminate()
        cls.server.wait(timeout=5)
        cls.temporary_directory.cleanup()
        raise RuntimeError("Go server did not become ready within 30 seconds")

    @classmethod
    def tearDownClass(cls):
        cls.server.terminate()
        try:
            cls.server.wait(timeout=5)
        except subprocess.TimeoutExpired:
            cls.server.kill()
            cls.server.wait(timeout=5)
        if cls.server.stdout:
            cls.server.stdout.close()
        cls.temporary_directory.cleanup()

    def get_json(self, path: str) -> dict:
        with urlopen(f"{self.base_url}{path}", timeout=2) as response:
            return json.load(response)

    def test_report_reaches_go_api_and_retry_does_not_duplicate_items(self):
        scan_id = "integration-scan-46"
        observation = RecognizedItem(
            name="milk",
            confidence=0.95,
            source_camera="overhead",
        )
        report = AssociationReport(
            scan_id=scan_id,
            items=[
                AssociatedItem(
                    association_id=f"{scan_id}:item-1",
                    name="milk",
                    confidence=0.95,
                    observations=[observation],
                )
            ],
        )
        reporter = HttpReporter(
            self.base_url,
            review_confidence_threshold=0.80,
            timeout_seconds=2,
        )

        self.assertTrue(reporter.report(report))
        self.assertTrue(reporter.report(report))

        inventory = self.get_json("/api/inventory")
        self.assertEqual(len(inventory["items"]), 1)
        self.assertEqual(inventory["items"][0]["name"], "milk")
        self.assertEqual(inventory["items"][0]["quantity"], 1)


if __name__ == "__main__":
    unittest.main()
