from edge.cameras.mock_camera import MockCamera
from edge.cameras.three_camera_capture import ThreeCameraCapture
from edge.state_machine import StateMachine


def main():
    machine = StateMachine()
    machine.handle_weight_detected()
    machine.handle_stabilizing(
        weight_present=True,
        weight_stable=True,
    )

    print(f"FSM before capture: {machine.state.name}")
    print(f"Scan ID: {machine.scan_id}")

    cameras = [
        MockCamera(
            "camera_1",
            "mock_camera_inputs/camera_1.jpg",
        ),
        MockCamera(
            "camera_2",
            "mock_camera_inputs/camera_2.jpg",
        ),
        MockCamera(
            "camera_3",
            "mock_camera_inputs/camera_3.jpg",
        ),
    ]

    capture = ThreeCameraCapture(cameras)

    try:
        captured_images = capture.capture_all(
            scan_id=machine.scan_id
        )
    except (FileNotFoundError, OSError) as error:
        print(f"Capture failed: {error}")

        machine.handle_error()

        print(f"State: {machine.state.name}")
        print(f"Error source: {machine.error_source.name}")
        return

    capture_complete = len(captured_images) == 3
    machine.handle_capture(
        capture_complete=capture_complete
    )

    print(f"FSM after capture: {machine.state.name}")
    print(f"Capture valid: {machine.cap_valid}")
    print("Three-camera capture complete:")

    for camera_id, image_path in captured_images.items():
        print(f"{camera_id}: {image_path}")


if __name__ == "__main__":
    main()