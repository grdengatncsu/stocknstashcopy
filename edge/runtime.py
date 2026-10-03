import logging, signal
from threading import Event
from edge.association.position_associator import PositionAssociator
from edge.cameras.mock_camera import MockCamera
from edge.cameras.three_camera_capture import ThreeCameraCapture
from edge.recognition.mock_recognizer import MockRecognizer
from edge.reporting.http_reporter import HttpReporter
from edge.scan_controller import ScanController
from edge.state_machine import State, StateMachine
from edge.presence import PresenceSource, SimulatedPresenceSource

from edge.runtime_config import RuntimeConfig, load_runtime_config

logger = logging.getLogger(__name__)

def build_development_controller(config: RuntimeConfig) -> ScanController:
    cameras = [
        MockCamera(
            camera_id=camera_id,
            image_path=config.mock_image_directory / f"{camera_id}.jpg",
        )
        for camera_id in config.camera_ids
    ]

    capture = ThreeCameraCapture(
        cameras=cameras,
        expected_camera_ids=config.camera_ids,
        output_root=config.capture_directory
    )
    recognizer = MockRecognizer(
        predictions={
            config.camera_ids[0]: ("milk", 0.9, (0.5, 0.5)),
            config.camera_ids[1]: ("milk", 0.9, (0.53, 0.51)),
            config.camera_ids[2]: ("milk", 0.9, (0.49, 0.52)),
        }
    )
    associator = PositionAssociator(maximum_distance=config.association_distance)
    reporter = HttpReporter(
        base_url=config.api_base_url,
        review_confidence_threshold=config.recognition_threshold,
        timeout_seconds=config.report_timeout_seconds,
        )

    state_machine = StateMachine()

    return ScanController(state_machine=state_machine, capture=capture, recognizer=recognizer, associator=associator, reporter=reporter)

def process_runtime_step(controller: ScanController, presence_source: PresenceSource) -> None:
    """Run one step of the runtime loop, handling state transitions."""
    signals = presence_source.read()
    
    controller.process_current_state(
        weight_detected=signals.weight_detected,
        weight_present=signals.weight_present,
        weight_stable=signals.weight_stable,
        platform_empty=signals.platform_empty,
)

def run_simulated_scan(
    controller: ScanController,
    presence_source: PresenceSource,
    max_steps: int = 20,
) -> None:
    """Run a simulated scan, stepping through the state machine."""
    for _ in range(max_steps):
        previous_state = controller.state_machine.state
        previous_scan_id = controller.state_machine.scan_id

        process_runtime_step(controller = controller, presence_source = presence_source,)

        current_state = controller.state_machine.state
        current_scan_id = controller.state_machine.scan_id

        logger.info(
            "state = %s -> %s scan_id = %s" , 
            previous_state.name,
            current_state.name,
            current_scan_id or previous_scan_id,
        )

        if current_state != State.IDLE:
            scan_started = True

        if scan_started and current_state == State.IDLE:
            logger.info("Simulated scan complete.")
            return
    raise RuntimeError(f"Simulated scan did not complete within {max_steps} runtime steps.")

def run_forever(controller: ScanController, presence_source: PresenceSource, stop_event: Event, poll_interval_seconds: float) -> None:
    logger.info("Edge runtime started.")

    while not stop_event.is_set():
        previous_state = controller.state_machine.state
        previous_scan_id = controller.state_machine.scan_id

        process_runtime_step(controller = controller, presence_source = presence_source,)

        current_state = controller.state_machine.state
        current_scan_id = controller.state_machine.scan_id

        if current_state != previous_state:
            logger.info(
                "state = %s -> %s scan_id = %s" , 
                previous_state.name,
                current_state.name,
                current_scan_id or previous_scan_id,
            )
        stop_event.wait(poll_interval_seconds)

    logger.info("Edge runtime stopped.")

def install_signal_handlers(stop_event: Event) -> None:
    def request_shutdown(signum, _frame):
        signal_name = signal.Signals(signum).name

        logger.info(
            "Received %s; requesting shutdown.",
            signal_name,
        )

        stop_event.set()

    signal.signal(signal.SIGINT, request_shutdown)
    signal.signal(signal.SIGTERM, request_shutdown)

def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
    config = load_runtime_config()
    controller = build_development_controller(config)
    presence_source = SimulatedPresenceSource()

    stop_event = Event()
    install_signal_handlers(stop_event)

    run_forever(controller=controller, presence_source=presence_source, stop_event=stop_event, poll_interval_seconds=config.poll_interval_seconds,)

if __name__ == "__main__":
    main()