from enum import IntEnum

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
        self.state = State.IDLE

    def handle_weight_detected(self):
        """Handle weight detected event."""
        if self.state == State.IDLE:
            self.state = State.STABILIZING
    def handle_stabilizing(self, weight_present: bool, weight_stable: bool):
        """Handle stabilizing event."""
        if self.state == State.STABILIZING:
            if not weight_present:
                self.state = State.IDLE
            elif weight_present and weight_stable:
                self.state = State.CAPTURE
