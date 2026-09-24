"""In-memory reporter for acknowledgement and retry tests."""

from edge.association.associator import AssociationReport


class MockReporter:
    """Simulate a receiver without making a network request.

    Use ``HttpReporter`` in production. Keep this mock for deterministic tests
    that must not depend on the server or network.
    """

    def __init__(self, should_acknowledge: bool):
        """Choose whether calls simulate success or no acknowledgement."""
        self.processed_scan_ids: set[str] = set()
        self.should_acknowledge = should_acknowledge

    def report(self, association_report: AssociationReport) -> bool:
        """Acknowledge each scan ID while making retries idempotent."""
        if not self.should_acknowledge:
            return False
        if association_report.scan_id in self.processed_scan_ids:
            # Treat a repeat as success without processing it twice.
            return True
        self.processed_scan_ids.add(association_report.scan_id)
        return True
