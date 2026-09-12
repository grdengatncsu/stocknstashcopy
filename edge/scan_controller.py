from pathlib import Path
from edge.state_machine import StateMachine, State
from edge.cameras.three_camera_capture import ThreeCameraCapture
from edge.recognition.recognizer import Recognizer, RecognitionReport 
from edge.reporting.reporter import Reporter
from edge.association.associator import Associator, AssociationReport

class ScanController:
    """Controls the scanning process by coordinating the state machine, camera capture, recognition, and reporting."""
    def __init__(self, state_machine: StateMachine, capture: ThreeCameraCapture, recognizer: Recognizer, reporter: Reporter, associator: Associator):
        self.state_machine = state_machine
        self.capture = capture
        self.recognizer = recognizer
        self.reporter = reporter 
        self.associator = associator

        self.current_report: RecognitionReport | None = None
        self.captured_images: dict[str, Path] | None = None
        self.current_association_report: AssociationReport | None = None
    def process_capture(self) -> None:
        """Capture three images and update the state machine."""
        if self.state_machine.state != State.CAPTURE:
            raise RuntimeError("Cannot capture images when not in CAPTURE state.")
        try:
            self.captured_images = self.capture.capture_all(scan_id=self.state_machine.scan_id)
        except (FileNotFoundError, OSError):
            self.state_machine.handle_error()
            return
        capture_complete = len(self.captured_images) == 3

        self.state_machine.handle_capture(capture_complete=capture_complete)
    def process_recognition(self) -> None:
        """Recognize items and update the state machine."""
        if self.state_machine.state != State.RECOGNIZE:
            raise RuntimeError(
                "Cannot recognize items when not in RECOGNIZE state."
        )

        if self.captured_images is None:
            raise RuntimeError(
                "No captured images available for recognition."
        )
        scan_id = self.state_machine.scan_id
        if scan_id is None:
            raise RuntimeError(
                "No scan ID available for recognition."
        )
        try:
            self.current_report = self.recognizer.recognize(
                scan_id=scan_id,
                captured_images=self.captured_images,
            )
            self.current_association_report = self.associator.associate(self.current_report)
        except (OSError, ValueError):
            self.state_machine.handle_error()
            return

        recognition_complete = self.current_association_report is not None

        self.state_machine.handle_recognize(
            recognition_complete=recognition_complete
        )
    def process_report(self) -> None:
        """Report the association results and update the state machine."""
        if self.state_machine.state != State.REPORT:
            raise RuntimeError("Cannot report when not in REPORT state.")
        if self.current_association_report is None:
            raise RuntimeError("No association report available for reporting.")
        try:
            acknowledged = self.reporter.report(self.current_association_report)
        except (OSError, ValueError):
            self.state_machine.handle_error()
            return

        self.state_machine.handle_report(acknowledged=acknowledged)
    def process_reset(self, platform_empty: bool) -> None:
        """Reset the scan controller and update the state machine."""
        if self.state_machine.state != State.RESET:
            raise RuntimeError("Cannot reset when not in RESET state.")
        self.state_machine.handle_reset(platform_empty=platform_empty)
        if self.state_machine.state == State.IDLE:
            self.current_report = None
            self.captured_images = None
            self.current_association_report = None
    def process_current_state(self, weight_detected: bool = False, weight_present: bool = False, weight_stable: bool = False, platform_empty: bool = False, recovered: bool = False) -> None:
        """Process the current state of the state machine."""
        if self.state_machine.state == State.IDLE:
            if weight_detected:
                self.state_machine.handle_weight_detected()
        elif self.state_machine.state == State.STABILIZING: 
            self.state_machine.handle_stabilizing(weight_present=weight_present, weight_stable=weight_stable)
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


        