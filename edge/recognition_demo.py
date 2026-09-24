"""Run the portable YOLO baseline on saved development images."""

from pathlib import Path

from edge.recognition.yolo_recognizer import YoloRecognizer


def main() -> None:
    """Print model detections for a repeatable three-camera fixture set."""
    # HARDWARE TODO: Replace these saved paths with output from the physical
    # camera adapter once the MIPI and USB camera assignments are verified.
    captured_images = {
        "overhead": Path("mock_camera_inputs/camera_1.jpg"),
        "rear_left": Path("mock_camera_inputs/camera_2.jpg"),
        "front_right": Path("mock_camera_inputs/camera_3.jpg"),
    }

    # HARDWARE TODO: The .pt file runs the development baseline. Production
    # must use the approved Hailo HEF model and its hardware-backed recognizer.
    recognizer = YoloRecognizer(
        model_path="models/yolov8n.pt",
        confidence_threshold=0.50,
    )
    report = recognizer.recognize(
        scan_id="baseline_scan_001",
        captured_images=captured_images,
    )

    print(f"Scan ID: {report.scan_id}")

    for item in report.items:
        print(
            f"Item: {item.name}, Confidence: {item.confidence:.2f}, "
            f"Source Camera: {item.source_camera}, Bounding Box: {item.bounding_box}"
        )


if __name__ == "__main__":
    main()
