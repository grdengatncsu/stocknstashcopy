"""Coordinator that performs the work associated with each scan state."""

from pathlib import Path

from edge.association.associator import Associator, AssociationReport
from edge.cameras.three_camera_capture import ThreeCameraCapture
from edge.errors import EdgeComponentError
from edge.recognition.recognizer import Recognizer, RecognitionReport
from edge.reporting.reporter import Reporter
from edge.state_machine import State, StateMachine


class ScanController:
    """Connect capture, recognition, association, and reporting components.

    Dependencies are supplied by the caller so mock components can be replaced
    with physical implementations without changing this workflow class.
    """

    def __init__(
        self,
        state_machine: StateMachine,
        capture: ThreeCameraCapture,
        recognizer: Recognizer,
        reporter: Reporter,
        associator: Associator,
    ):
        """Store pipeline components and initialize empty per-scan results."""
        self.state_machine = state_machine
        self.capture = capture
        self.recognizer = recognizer
        self.reporter = reporter
        self.associator = associator

        self.current_report: RecognitionReport | None = None
        self.captured_images: dict[str, Path] | None = None
        self.current_association_report: AssociationReport | None = None

    def process_capture(self) -> None:
        """Capture all three views and record whether capture completed."""
        if self.state_machine.state != State.CAPTURE:
            raise RuntimeError("Cannot capture images when not in CAPTURE state.")

        # HARDWARE TODO: Turn on the platform lighting here, verify its output,
        # and wait the measured settling time before capturing. Ensure lighting
        # returns to its safe state on both success and exception paths.
        try:
            self.captured_images = self.capture.capture_all(
                scan_id=self.state_machine.scan_id
            )
        except (EdgeComponentError, FileNotFoundError, OSError):
            # Camera and filesystem failures use the common recovery path.
            self.state_machine.handle_error()
            return
        capture_complete = len(self.captured_images) == 3

        self.state_machine.handle_capture(capture_complete=capture_complete)

    def process_recognition(self) -> None:
        """Detect items, combine camera views, and record completion."""
        if self.state_machine.state != State.RECOGNIZE:
            raise RuntimeError(
                "Cannot recognize items when not in RECOGNIZE state."
            )

        if self.captured_images is None:
            raise RuntimeError("No captured images available for recognition.")
        scan_id = self.state_machine.scan_id
        if scan_id is None:
            raise RuntimeError("No scan ID available for recognition.")
        try:
            self.current_report = self.recognizer.recognize(
                scan_id=scan_id,
                captured_images=self.captured_images,
            )
            # Recognition yields camera observations. Association combines
            # observations that likely represent the same physical object.
            self.current_association_report = self.associator.associate(
                self.current_report
            )
        except (EdgeComponentError, OSError, ValueError):
            self.state_machine.handle_error()
            return

        recognition_complete = self.current_association_report is not None

        self.state_machine.handle_recognize(
            recognition_complete=recognition_complete
        )

    def process_report(self) -> None:
        """Send the consolidated result and record its acknowledgement."""
        if self.state_machine.state != State.REPORT:
            raise RuntimeError("Cannot report when not in REPORT state.")
        if self.current_association_report is None:
            raise RuntimeError("No association report available for reporting.")
        try:
            acknowledged = self.reporter.report(self.current_association_report)
        except (EdgeComponentError, OSError, ValueError):
            self.state_machine.handle_error()
            return

        self.state_machine.handle_report(acknowledged=acknowledged)

    def process_reset(self, platform_empty: bool) -> None:
        """Discard cached scan results once the platform is empty."""
        if self.state_machine.state != State.RESET:
            raise RuntimeError("Cannot reset when not in RESET state.")

        # HARDWARE TODO: If reset requires an actuator, command it through a
        # timeout-limited adapter and verify the final limit switch before IDLE.
        self.state_machine.handle_reset(platform_empty=platform_empty)
        if self.state_machine.state == State.IDLE:
            self.current_report = None
            self.captured_images = None
            self.current_association_report = None

    def process_current_state(
        self,
        weight_detected: bool = False,
        weight_present: bool = False,
        weight_stable: bool = False,
        platform_empty: bool = False,
        recovered: bool = False,
    ) -> None:
        """Run one action for the machine's current state.

        PLATFORM BLOCKED (#33, #54, #55): The production loop must obtain the
        four physical-state booleans from the NAU7802 adapter. Issue #31
        verified bus 1, address 0x2A, and provisional +2000/+1000 hysteresis;
        final thresholds still require the assembled platform. Callers pass
        booleans directly until that adapter is integrated and validated.
        """
        if self.state_machine.state == State.IDLE:
            if weight_detected:
                self.state_machine.handle_weight_detected()
        elif self.state_machine.state == State.STABILIZING:
            self.state_machine.handle_stabilizing(
                weight_present=weight_present,
                weight_stable=weight_stable,
            )
        elif self.state_machine.state == State.CAPTURE:
            self.process_capture()
        elif self.state_machine.state == State.RECOGNIZE:
            self.process_recognition()
        elif self.state_machine.state == State.REPORT:
            self.process_report()
        elif self.state_machine.state == State.RESET:
            self.process_reset(platform_empty=platform_empty)
        elif self.state_machine.state == State.ERROR:
            self.state_machine.handle_recovery(recovered=recovered)
