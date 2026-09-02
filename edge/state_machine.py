from enum import IntEnum
from uuid import uuid4

class State(IntEnum):
    """State of the state machine."""
    IDLE = 0b000
    STABILIZING = 0b001
    CAPTURE = 0b010
    RECOGNIZE = 0b011
    REPORT = 0b100
    RESET = 0b101
    ERROR = 0b110

class StateMachine:
    """State machine for the edge device."""
    def __init__(self):
        self.scan_id = None
        self.state = State.IDLE
        self.cap_valid = False
        self.rec_valid = False
        self.report_ack = False
        self.error_source = None
    def handle_weight_detected(self):
        """Handle weight detected event."""
        if self.state == State.IDLE:
            self.scan_id = str(uuid4())
            self.state = State.STABILIZING
    def handle_stabilizing(self, weight_present: bool, weight_stable: bool):
        """Handle stabilizing event."""
        if self.state == State.STABILIZING:
            if not weight_present:
                self.scan_id = None
                self.state = State.IDLE
            elif weight_present and weight_stable:
                self.state = State.CAPTURE
    def handle_capture(self, capture_complete: bool):
        """Handle capture event."""
        if self.state == State.CAPTURE:
            if self.cap_valid:
                self.state = State.RECOGNIZE
            elif capture_complete:
                self.cap_valid = True
                self.state = State.RECOGNIZE

    def handle_recognize(self, recognition_complete: bool):
        """Handle recognize event."""
        if self.state == State.RECOGNIZE:
            if self.rec_valid:
                self.state = State.REPORT
            elif recognition_complete:
                self.rec_valid = True
                self.state = State.REPORT
    def handle_report(self, acknowledged: bool):
        """Handle report event."""
        if self.state == State.REPORT:
            if self.report_ack:
                self.state = State.RESET
            elif acknowledged:
                self.report_ack = True
                self.state = State.RESET
    def handle_reset(self, platform_empty: bool):
        """Handle reset event."""
        if self.state == State.RESET:
            if platform_empty:
                self.state = State.IDLE
                self.cap_valid = False
                self.rec_valid = False
                self.report_ack = False
                self.scan_id = None
    def handle_error(self):
        """Handle error event."""
        if self.state != State.ERROR:
            self.error_source = self.state
            self.state = State.ERROR
        