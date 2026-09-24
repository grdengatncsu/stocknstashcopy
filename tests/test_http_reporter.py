import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from edge.association.associator import AssociatedItem, AssociationReport
from edge.errors import ReportingError
from edge.recognition.recognizer import RecognizedItem
from edge.reporting.http_reporter import HttpReporter


class RecordingHandler(BaseHTTPRequestHandler):
    """Small local endpoint used to test the real HTTP serialization path."""

    def do_POST(self):
        length = int(self.headers["Content-Length"])
        self.server.request_path = self.path
        self.server.request_json = json.loads(self.rfile.read(length))
        self.send_response(self.server.response_status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(self.server.response_json).encode("utf-8"))

    def log_message(self, format, *args):
        # Test output should contain assertion failures, not access logs.
        return


class TestHttpReporter(unittest.TestCase):
    def setUp(self):
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), RecordingHandler)
        self.server.response_status = 200
        self.server.response_json = {
            "scan_id": "scan-123",
            "status": "accepted",
        }
        self.server.request_json = None
        self.server.request_path = None
        self.thread = threading.Thread(
            target=self.server.serve_forever,
            daemon=True,
        )
        self.thread.start()
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.server.shutdown)

        self.reporter = HttpReporter(
            f"http://127.0.0.1:{self.server.server_port}",
            review_confidence_threshold=0.80,
            timeout_seconds=1.0,
        )

    def make_report(
        self,
        confidence: float = 0.95,
        unknown: bool = False,
    ) -> AssociationReport:
        observation = RecognizedItem(
            name="milk",
            confidence=confidence,
            source_camera="overhead",
            unknown=unknown,
        )
        return AssociationReport(
            scan_id="scan-123",
            items=[
                AssociatedItem(
                    association_id="scan-123:item-1",
                    name="milk",
                    confidence=confidence,
                    observations=[observation],
                )
            ],
        )

    def test_posts_contract_payload_and_accepts_acknowledgement(self):
        self.assertTrue(self.reporter.report(self.make_report()))

        self.assertEqual(self.server.request_path, "/api/scans")
        self.assertEqual(
            self.server.request_json,
            {
                "scan_id": "scan-123",
                "items": [
                    {
                        "association_id": "scan-123:item-1",
                        "name": "milk",
                        "upc": None,
                        "confidence": 0.95,
                        "quantity": 1,
                        "requires_review": False,
                    }
                ],
            },
        )

    def test_low_confidence_and_unknown_observations_require_review(self):
        low_confidence = self.reporter.build_payload(
            self.make_report(confidence=0.79)
        )
        unknown = self.reporter.build_payload(self.make_report(unknown=True))

        self.assertTrue(low_confidence["items"][0]["requires_review"])
        self.assertTrue(unknown["items"][0]["requires_review"])

    def test_duplicate_acknowledgement_is_success(self):
        self.server.response_json["status"] = "already_processed"

        self.assertTrue(self.reporter.report(self.make_report()))

    def test_server_failure_remains_retryable(self):
        self.server.response_status = 503

        self.assertFalse(self.reporter.report(self.make_report()))

    def test_client_rejection_is_not_silently_retried(self):
        self.server.response_status = 400

        with self.assertRaises(ReportingError):
            self.reporter.report(self.make_report())

    def test_wrong_scan_acknowledgement_raises_error(self):
        self.server.response_json["scan_id"] = "different-scan"

        with self.assertRaises(ReportingError):
            self.reporter.report(self.make_report())


if __name__ == "__main__":
    unittest.main()
