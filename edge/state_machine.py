"""State and transition rules for one edge-device scan."""

from enum import IntEnum
from uuid import uuid4


class State(IntEnum):
    """Stages that one item passes through during a scan."""

    IDLE = 0b000
    STABILIZING = 0b001
    CAPTURE = 0b010
    RECOGNIZE = 0b011
    REPORT = 0b100
    RESET = 0b101
    ERROR = 0b110


class StateMachine:
    """Track scan progress and allow only valid stage transitions.

    This class records workflow state only. It does not communicate with
    sensors, cameras, GPIO pins, or the server; ``ScanController`` coordinates
    those components and reports their outcomes here.
    """

    def __init__(self):
        """Start ready for a new item with all completion flags cleared."""
        self.scan_id = None
        self.state = State.IDLE
        self.cap_valid = False
        self.rec_valid = False
        self.report_ack = False
        self.error_source = None

    def handle_weight_detected(self):
        """Start a uniquely identified scan when an item is detected."""
        if self.state == State.IDLE:
            # One scan ID connects the images, detections, and API records that
            # belong to the same physical pass through the scanner.
            self.scan_id = str(uuid4())
            self.state = State.STABILIZING

    def handle_stabilizing(self, weight_present: bool, weight_stable: bool):
        """Wait for a steady item, or cancel if it leaves the platform."""
        if self.state == State.STABILIZING:
            if not weight_present:
                self.scan_id = None
                self.state = State.IDLE
            elif weight_present and weight_stable:
                self.state = State.CAPTURE

    def handle_capture(self, capture_complete: bool):
        """Advance after all required camera views have been saved."""
        if self.state == State.CAPTURE:
            if self.cap_valid:
                self.state = State.RECOGNIZE
            elif capture_complete:
                self.cap_valid = True
                self.state = State.RECOGNIZE

    def handle_recognize(self, recognition_complete: bool):
        """Advance after recognition and association produce a report."""
        if self.state == State.RECOGNIZE:
            if self.rec_valid:
                self.state = State.REPORT
            elif recognition_complete:
                self.rec_valid = True
                self.state = State.REPORT

    def handle_report(self, acknowledged: bool):
        """Advance only after the receiver acknowledges the scan report."""
        if self.state == State.REPORT:
            if self.report_ack:
                self.state = State.RESET
            elif acknowledged:
                self.report_ack = True
                self.state = State.RESET

    def handle_reset(self, platform_empty: bool):
        """Clear completed-scan data after the item leaves the platform."""
        if self.state == State.RESET:
            if platform_empty:
                self.state = State.IDLE
                self.cap_valid = False
                self.rec_valid = False
                self.report_ack = False
                self.scan_id = None

    def handle_error(self):
        """Remember the failed stage before entering the shared error state."""
        if self.state != State.ERROR:
            self.error_source = self.state
            self.state = State.ERROR

    def handle_recovery(self, recovered: bool):
        """Resume at the first unfinished stage after recovery is confirmed."""
        if self.state == State.ERROR and self.error_source is not None and recovered:
            source = self.error_source
            if source in (State.IDLE, State.STABILIZING, State.RESET):
                self.state = source
            elif source == State.CAPTURE:
                if self.cap_valid:
                    self.state = State.RECOGNIZE
                else:
                    self.state = State.CAPTURE
            elif source == State.RECOGNIZE:
                if self.rec_valid:
                    self.state = State.REPORT
                else:
                    self.state = State.RECOGNIZE
            elif source == State.REPORT:
                if self.report_ack:
                    self.state = State.RESET
                else:
                    self.state = State.REPORT
            self.error_source = None
