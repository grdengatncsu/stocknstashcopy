from pathlib import Path
from edge.recognition.recognizer import RecognitionReport
from edge.association.associator import AssociationReport

class MockReporter:

    def __init__(self, should_acknowledge: bool):
        self.processed_scan_ids: set[str] = set()
        self.should_acknowledge = should_acknowledge
    def report(self, association_report: AssociationReport) -> bool:
        """Simulate reporting the association results to an external system."""
        if not self.should_acknowledge:
            return False  # Simulate a failure to acknowledge the report
        if association_report.scan_id in self.processed_scan_ids:
            return True  # Already processed this scan_id, so we skip reporting again
        self.processed_scan_ids.add(association_report.scan_id)
        return True