from edge.cameras.mock_camera import MockCamera
from.cameras.three_camera_capture import ThreeCameraCapture

def main():
    cameras = [MockCamera("camera_1", "mock_camera_inputs/camera_1.jpg"), MockCamera("camera_2", "mock_camera_inputs/camera_2.jpg"), MockCamera("camera_3", "mock_camera_inputs/camera_3.jpg")]
    capture = ThreeCameraCapture(cameras)
    captured_images = capture.capture_all(scan_id="mock_scan_001")
    print("Three-camera capture complete:")

    for camera_id, image_path in captured_images.items():
        print(f"{camera_id}: {image_path}")

if __name__ == "__main__":
    main()
