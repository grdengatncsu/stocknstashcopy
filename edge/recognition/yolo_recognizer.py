"""Ultralytics YOLO baseline for recognition development on saved images."""

from pathlib import Path

from edge.recognition.recognizer import RecognitionReport, RecognizedItem

# Keep this module importable without the large Ultralytics dependency so unit
# tests can replace YOLO with a lightweight fake.
try:
    from ultralytics import YOLO
except ModuleNotFoundError:
    YOLO = None

from edge.positioning.coordinate_mapper import CoordinateMapper


class YoloRecognizer:
    """Recognize item observations in each of three camera images.

    HARDWARE TODO: This is the portable development baseline. Replace or wrap
    it with a Hailo-8L implementation that loads the approved HEF model through
    the Hailo runtime. Record the deployed model path, input resolution, label
    map, and confidence threshold during model deployment.
    """

    def __init__(
        self,
        model_path: str,
        confidence_threshold: float = 0.25,
        coordinate_mapper: CoordinateMapper | None = None,
    ):
        """Load a model and configure the minimum accepted confidence score."""
        if not 0.0 <= confidence_threshold <= 1.0:
            raise ValueError("confidence_threshold must be between 0.0 and 1.0")
        if YOLO is None:
            raise RuntimeError(
                "Ultralytics is not installed; run 'pip install -r requirements.txt' "
                "to use YoloRecognizer"
            )
        self.model_path = model_path
        self.confidence_threshold = confidence_threshold
        self.model = YOLO(model_path)
        self.coordinate_mapper = coordinate_mapper

    def recognize(
        self,
        scan_id: str,
        captured_images: dict[str, Path],
    ) -> RecognitionReport:
        """Recognize items in the captured images and return a recognition report."""
        if not scan_id:
            raise ValueError("scan_id must be provided")
        if len(captured_images) != 3:
            raise ValueError("Three captured images are required")

        recognized_items: list[RecognizedItem] = []

        for camera_id, image_path in captured_images.items():
            if not image_path.exists():
                raise FileNotFoundError(
                    f"Captured image for camera {camera_id} does not exist "
                    f"at {image_path}"
                )

            results = self.model.predict(
                source=str(image_path),
                conf=self.confidence_threshold,
                verbose=False,
            )

            # Exactly one image is submitted per call, so the first result
            # corresponds to the current camera even when it has no boxes.
            result = results[0]

            for box in result.boxes:
                class_id = int(box.cls.item())
                name = result.names[class_id]
                confidence = float(box.conf.item())

                # ``xyxyn`` uses normalized (left, top, right, bottom) values.
                xmin, ymin, xmax, ymax = box.xyxyn[0].tolist()

                bounding_box = (float(xmin), float(ymin), float(xmax), float(ymax))
                platform_position = None
                if self.coordinate_mapper is not None:
                    # Shared platform coordinates let association compare
                    # observations from different camera angles.
                    platform_position = self.coordinate_mapper.map_detection(
                        camera_id,
                        bounding_box,
                    )

                recognized_item = RecognizedItem(
                    name=name,
                    confidence=confidence,
                    source_camera=camera_id,
                    unknown=False,
                    bounding_box=bounding_box,
                    platform_position=platform_position,
                )
                recognized_items.append(recognized_item)

        return RecognitionReport(scan_id=scan_id, items=recognized_items)
