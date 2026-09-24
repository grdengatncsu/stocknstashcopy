"""Deliver consolidated scan results to downstream systems."""

from edge.reporting.http_reporter import HttpReporter
from edge.reporting.reporter import Reporter

__all__ = ["HttpReporter", "Reporter"]
