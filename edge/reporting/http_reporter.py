"""HTTP implementation of the edge-to-Pi reporting boundary."""

from __future__ import annotations

import json
import logging
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from edge.association.associator import AssociatedItem, AssociationReport
from edge.errors import ReportingError

logger = logging.getLogger(__name__)

class HttpReporter:
    """Send association reports to the local Go API.

    ``review_confidence_threshold`` is deliberately required instead of hidden
    in this class: the final product threshold is still an open policy choice,
    and changing it determines whether an item reaches inventory or the review
    queue.  The caller must therefore choose it visibly in configuration.
    """

    def __init__(
        self,
        base_url: str,
        review_confidence_threshold: float,
        timeout_seconds: float = 5.0,
    ) -> None:
        """Store and validate the API endpoint and reporting policy."""
        if not base_url.strip():
            raise ValueError("base_url must be provided")
        if not 0.0 <= review_confidence_threshold <= 1.0:
            raise ValueError(
                "review_confidence_threshold must be between 0.0 and 1.0"
            )
        if timeout_seconds <= 0:
            raise ValueError("timeout_seconds must be positive")

        self.scan_url = f"{base_url.rstrip('/')}/api/scans"
        self.review_confidence_threshold = review_confidence_threshold
        self.timeout_seconds = timeout_seconds

    def _item_payload(self, item: AssociatedItem) -> dict[str, Any]:
        """Translate one physical association into the API wire format."""
        if not item.association_id.strip():
            raise ValueError("every associated item must include an association_id")
        if not 0.0 <= item.confidence <= 1.0:
            raise ValueError("item confidence must be between 0.0 and 1.0")

        uncertain = (
            item.confidence < self.review_confidence_threshold
            or not item.name.strip()
            or any(observation.unknown for observation in item.observations)
        )

        return {
            "association_id": item.association_id,
            "name": item.name.strip() or "unknown",
            # UPC lookup is not part of recognition yet.  JSON null preserves
            # that distinction instead of inventing a product identifier.
            "upc": None,
            "confidence": item.confidence,
            # An AssociatedItem represents one physical object. Aggregating
            # quantities by UPC belongs downstream, after identity is known.
            "quantity": 1,
            "requires_review": uncertain,
        }

    def build_payload(self, report: AssociationReport) -> dict[str, Any]:
        """Build the documented ``POST /api/scans`` request body."""
        if not report.scan_id.strip():
            raise ValueError("association report must include a scan_id")
        if not report.items:
            raise ValueError("association report must contain at least one item")

        return {
            "scan_id": report.scan_id,
            "items": [self._item_payload(item) for item in report.items],
        }

    def report(self, association_report: AssociationReport) -> bool:
        """Submit a scan and return whether the API acknowledged that scan.

        Connection failures and server-side 5xx responses return ``False`` so
        the state machine keeps the same report for an idempotent retry. Client
        errors and malformed acknowledgements raise ``ReportingError`` because
        retrying an unchanged payload cannot repair them.
        """
        body = json.dumps(self.build_payload(association_report)).encode("utf-8")
        request = Request(
            self.scan_url,
            data=body,
            headers={"Content-Type": "application/json"},
            method="POST",
        )

        try:
            # urllib is sufficient for this one request and avoids adding an
            # extra runtime dependency to the constrained edge installation.
            with urlopen(request, timeout=self.timeout_seconds) as response:
                response_body = response.read()
        except HTTPError as error:
            status_code = error.code
            error.close()

            if 500 <= status_code <= 599:
                logger.warning(
                    "Pi API returned HTTP %s for scan report %s; retrying...", status_code, association_report.scan_id
                )
                return False
            raise ReportingError(
                f"Pi API rejected scan report with HTTP {status_code}"
            ) from error
        except (URLError, TimeoutError) as error:
            logger.warning("Pi API unreachable for scan report %s (%s); retrying...", association_report.scan_id, type(error).__name__,)
            return False

        try:
            acknowledgement = json.loads(response_body)
        except (json.JSONDecodeError, UnicodeDecodeError) as error:
            raise ReportingError("Pi API returned invalid JSON") from error

        if not isinstance(acknowledgement, dict):
            raise ReportingError("Pi API acknowledgement must be a JSON object")
        if acknowledgement.get("scan_id") != association_report.scan_id:
            raise ReportingError("Pi API acknowledged a different scan_id")

        if acknowledgement.get("status") not in {
            "accepted",
            "already_processed",
        }:
            raise ReportingError("Pi API returned an unknown acknowledgement status")
        return True
