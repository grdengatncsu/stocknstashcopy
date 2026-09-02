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