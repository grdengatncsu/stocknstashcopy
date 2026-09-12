from pathlib import Path
from edge.recognition.recognizer import RecognitionReport, RecognizedItem
try:
    from ultralytics import YOLO
except ModuleNotFoundError:
    YOLO = None
from edge.positioning.coordinate_mapper import CoordinateMapper

class YoloRecognizer:
    """Recognize multiple items in captured images using a YOLO model."""
    
    def __init__(self, model_path: str, confidence_threshold: float = 0.25, coordinate_mapper: CoordinateMapper | None = None):
        if not 0.0 <= confidence_threshold <= 1.0:
            raise ValueError("confidence_threshold must be between 0.0 and 1.0")
        self.model_path = model_path
        self.confidence_threshold = confidence_threshold
        self.model = YOLO(model_path)
        self.coordinate_mapper = coordinate_mapper

    def recognize(self, scan_id: str, captured_images: dict[str, Path]) -> RecognitionReport:
        """Recognize items in the captured images and return a recognition report."""
        if not scan_id:
            raise ValueError("scan_id must be provided")
        if len(captured_images) != 3:
            raise ValueError("Three captured images are required")

        recognized_items: list[RecognizedItem] = []

        for camera_id, image_path in captured_images.items():
            if not image_path.exists():
                raise FileNotFoundError(f"Captured image for camera {camera_id} does not exist at {image_path}")

            results = self.model.predict(
                source=str(image_path),
                conf=self.confidence_threshold,
                verbose=False,
            )

            result = results[0]

            for box in result.boxes:
                class_id = int(box.cls.item())
                name = result.names[class_id]
                confidence = float(box.conf.item())

                xmin, ymin, xmax, ymax = box.xyxyn[0].tolist()

                bounding_box = (float(xmin), float(ymin), float(xmax), float(ymax))
                platform_position = None
                if self.coordinate_mapper is not None:
                    platform_position = self.coordinate_mapper.map_detection(camera_id, bounding_box)

                recognized_item = RecognizedItem(
                    name=name,
                    confidence=confidence,
                    source_camera=camera_id,
                    unknown=False,
                    bounding_box=bounding_box,
                    platform_position=platform_position
                )
                recognized_items.append(recognized_item)

        return RecognitionReport(scan_id=scan_id, items=recognized_items)
