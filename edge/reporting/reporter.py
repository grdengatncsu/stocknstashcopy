"""Interface for delivering completed scan results to another system."""

from typing import Protocol

from edge.association.associator import AssociationReport


class Reporter(Protocol):
    """Contract shared by mock and future network-backed reporters."""

    def report(self, association_report: AssociationReport) -> bool:
        """Send a result and return whether the receiver acknowledged it."""
        ...
