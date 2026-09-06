from pathlib import Path
from edge.recognition.recognizer import RecognitionReport

class MockReporter:

    def __init__(self, should_acknowledge: bool):
        self.processed_scan_ids: set[str] = set()
        self.should_acknowledge = should_acknowledge
    def report(self, recognition_report: RecognitionReport) -> bool:
        """Simulate reporting the recognition results to an external system."""
        if not self.should_acknowledge:
            return False  # Simulate a failure to acknowledge the report
        if recognition_report.scan_id in self.processed_scan_ids:
            return True  # Already processed this scan_id, so we skip reporting again
        self.processed_scan_ids.add(recognition_report.scan_id)
        return True