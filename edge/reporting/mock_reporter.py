"""In-memory reporter for exercising acknowledgement and retry behavior."""

from edge.association.associator import AssociationReport


class MockReporter:
    """Simulate a receiver without performing network requests."""

    def __init__(self, should_acknowledge: bool):
        """Choose whether calls should simulate success or no acknowledgement."""
        self.processed_scan_ids: set[str] = set()
        self.should_acknowledge = should_acknowledge

    def report(self, association_report: AssociationReport) -> bool:
        """Acknowledge a scan at most once while making retries idempotent."""
        if not self.should_acknowledge:
            return False
        if association_report.scan_id in self.processed_scan_ids:
            # Network clients often retry when an acknowledgement is lost. A
            # repeated scan ID should succeed without processing it twice.
            return True
        self.processed_scan_ids.add(association_report.scan_id)
        return True
