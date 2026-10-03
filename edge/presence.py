from dataclasses import dataclass
from typing import Protocol

@dataclass(frozen=True)
class PresenceSignals:
    weight_detected: bool = False
    weight_present: bool = False
    weight_stable: bool = False
    platform_empty: bool = False

class PresenceSource(Protocol):
    def read(self) -> PresenceSignals:
        """Return the current presence signals from the platform."""
        ...

class SimulatedPresenceSource:
    """Simulate presence signals for testing and development."""

    def __init__(self):
        self._step = 0

    def read(self) -> PresenceSignals:
        sequence = [
            #IDLE -> STABILIZING
            PresenceSignals(
                weight_detected=True,
                weight_present=True,
            ),
            #STABILIZING -> CAPTURE
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            #CAPTURE -> RECOGNIZE
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            #RECOGNIZE -> REPORT
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            #REPORT -> RESET
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            #RESET -> IDLE
            PresenceSignals(
                platform_empty=True,
            ),
        ]

        if self._step >= len(sequence):
            return PresenceSignals(platform_empty=True)

        signals = sequence[self._step]
        self._step += 1
        return signals