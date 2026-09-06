# Pretrained Recognition Baseline

## Purpose

This baseline confirms that Stock-n-Stash can run a pretrained multi-object detector on images from three logical camera views and convert its detections into the project’s `RecognitionReport` format.

This is a pipeline baseline, not proof that the model can identify every grocery product or exact product brand.

## Selected Model

- Model: YOLOv8n
- Model file: `models/yolov8n.pt`
- Task: Multi-object detection
- Ultralytics version: `8.4.142`
- Python version: `3.12.13`
- Development platform: macOS
- Confidence threshold: `0.50`

YOLOv8n was selected as the initial baseline because it is lightweight and compatible with the planned Hailo object-detection pipeline.

## Planned Edge Hardware

The production system will use:

- Raspberry Pi Compute Module 5
- Hailo-8L accelerator
- Two MIPI IMX415 cameras
- One USB IMX415 camera

The `.pt` model currently runs on the development Mac. The Raspberry Pi deployment will eventually use a Hailo-compatible compiled model, such as a `.hef` file.

## Camera Inputs

The recognizer accepts exactly three images using these logical camera identifiers:

- `overhead`
- `rear_left`
- `front_right`

Each detected object includes its source camera so later processing can associate detections across views.

## Recognition Output

Each detection is converted into a `RecognizedItem` containing:

- Object name
- Confidence score
- Source camera
- Unknown flag
- Normalized bounding box

Bounding boxes use:

```text
(x_min, y_min, x_max, y_max)
```

All coordinates are normalized between `0.0` and `1.0`.

## Initial Result

At a confidence threshold of `0.25`, the detector returned two overlapping `clock` detections from `front_right`:

- `clock`, confidence `0.72`
- `clock`, confidence `0.28`

At a confidence threshold of `0.50`, only the `0.72` detection remained.

No detections were returned from `overhead` or `rear_left`.

The ground-truth contents of these images must be recorded before this result can be classified as correct or incorrect.

## Findings

- The three-camera recognition interface works.
- The model can return zero or multiple detections per image.
- The confidence threshold successfully removes weak detections.
- Detection results retain their correct source-camera association.
- The generic pretrained model does not guarantee exact grocery or brand recognition.
- One physical item may still be detected by multiple cameras.
- Multi-camera association and duplicate suppression will be implemented separately in issue #16.

## Testing

The YOLO integration has unit tests covering:

- Model loading
- Confidence-threshold validation
- Scan-ID validation
- Three-image validation
- Missing image files
- Conversion from YOLO detections to `RecognizedItem`
- Correct source-camera association

The complete project test suite currently contains 73 passing tests.

## Model and Dataset Storage

Large model files and datasets are excluded from Git:

- `models/*.pt`
- `models/*.onnx`
- `models/*.hef`
- `datasets/`

Source code, tests, configuration, dependency versions, and documentation remain tracked in GitHub.

## Limitations and Next Steps

1. Record the actual contents of the baseline images.
2. Test recognizable grocery-related objects such as bottles, apples, bananas, or oranges.
3. Compare predictions against known ground truth.
4. Define the Prototype 1 grocery set with the team in issue #5.
5. Implement multi-camera association and duplicate suppression in issue #16.
6. Benchmark and deploy a Hailo-compatible model after the Raspberry Pi and Hailo-8L arrive.

## Licensing Note

The installed Ultralytics package reports an AGPL-3.0 license. Licensing requirements must be reviewed before any future commercial deployment.