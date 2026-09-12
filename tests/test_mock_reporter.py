import unittest

from edge.association.associator import (
    AssociatedItem,
    AssociationReport,
)
from edge.recognition.recognizer import RecognizedItem
from edge.reporting.mock_reporter import MockReporter


class TestMockReporter(unittest.TestCase):
    def make_report(
        self,
        scan_id: str = "scan-123",
    ) -> AssociationReport:
        observation = RecognizedItem(
            name="milk",
            confidence=0.95,
            source_camera="camera_1",
        )

        item = AssociatedItem(
            association_id=f"{scan_id}:item-1",
            name="milk",
            confidence=0.95,
            observations=[observation],
        )

        return AssociationReport(
            scan_id=scan_id,
            items=[item],
        )

    def test_successful_report_returns_true(self):
        reporter = MockReporter(
            should_acknowledge=True
        )
        association_report = self.make_report()

        acknowledged = reporter.report(
            association_report
        )

        self.assertTrue(acknowledged)
        self.assertIn(
            association_report.scan_id,
            reporter.processed_scan_ids,
        )
        self.assertEqual(
            len(reporter.processed_scan_ids),
            1,
        )

    def test_duplicate_scan_is_acknowledged_without_processing_twice(
        self,
    ):
        reporter = MockReporter(
            should_acknowledge=True
        )
        association_report = self.make_report()

        first_acknowledgment = reporter.report(
            association_report
        )
        second_acknowledgment = reporter.report(
            association_report
        )

        self.assertTrue(first_acknowledgment)
        self.assertTrue(second_acknowledgment)
        self.assertEqual(
            reporter.processed_scan_ids,
            {"scan-123"},
        )

    def test_failed_report_returns_false(self):
        reporter = MockReporter(
            should_acknowledge=False
        )
        association_report = self.make_report()

        acknowledged = reporter.report(
            association_report
        )

        self.assertFalse(acknowledged)
        self.assertNotIn(
            association_report.scan_id,
            reporter.processed_scan_ids,
        )
        self.assertEqual(
            len(reporter.processed_scan_ids),
            0,
        )

    def test_failed_report_can_be_retried(self):
        reporter = MockReporter(
            should_acknowledge=False
        )
        association_report = self.make_report()

        first_acknowledgment = reporter.report(
            association_report
        )
        reporter.should_acknowledge = True
        retry_acknowledgment = reporter.report(
            association_report
        )

        self.assertFalse(first_acknowledgment)
        self.assertTrue(retry_acknowledgment)
        self.assertEqual(
            reporter.processed_scan_ids,
            {"scan-123"},
        )

    def test_different_scan_ids_are_processed_separately(
        self,
    ):
        reporter = MockReporter(
            should_acknowledge=True
        )

        reporter.report(
            self.make_report("scan-123")
        )
        reporter.report(
            self.make_report("scan-456")
        )

        self.assertEqual(
            reporter.processed_scan_ids,
            {"scan-123", "scan-456"},
        )


if __name__ == "__main__":
    unittest.main()