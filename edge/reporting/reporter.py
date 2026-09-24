"""Interface for delivering completed scan results to another system."""

from typing import Protocol

from edge.association.associator import AssociationReport


class Reporter(Protocol):
    """Contract shared by in-memory and HTTP-backed reporters.

    ``HttpReporter`` implements the local edge-to-Pi boundary. Keeping this
    protocol means the scan workflow does not need to know whether a report is
    stored in memory, sent over HTTP, or retried by another adapter.
    """

    def report(self, association_report: AssociationReport) -> bool:
        """Send a report and return whether the receiver acknowledged it."""
        ...
