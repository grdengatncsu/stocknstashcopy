# Camera-to-Platform Coordinate Mapping

## Purpose

Each camera observes groceries from a different angle. A bounding-box position from one camera cannot be compared directly with a bounding-box position from another camera.

The coordinate mapper converts each camera detection into a shared normalized platform position. The association component can then compare observations using the same coordinate system.

## Shared Platform Coordinate System

Platform positions use normalized `(x, y)` coordinates:

- `(0.0, 0.0)` represents one corner of the platform.
- `(1.0, 1.0)` represents the opposite corner.
- Valid coordinates must remain between `0.0` and `1.0`.

A mapped position outside these bounds is rejected and returned as `None`.

## Bounding-Box Format

YOLO detections use normalized bounding boxes:

```text
(x_min, y_min, x_max, y_max)
```

Each coordinate is relative to the width or height of the camera image.

## Detection Anchor Points

Different camera views use different points from the bounding box.

### Overhead Camera

The overhead camera uses the center of the bounding box:

```text
x = (x_min + x_max) / 2
y = (y_min + y_max) / 2
```

### Angled Cameras

The angled cameras use the bottom-center of the bounding box:

```text
x = (x_min + x_max) / 2
y = y_max
```

The bottom-center approximates the location where the grocery item touches the platform.

## Perspective Transformation

Each camera has its own 3×3 homography matrix. The matrix transforms that camera’s anchor point into the shared platform coordinate system.

The mapper stores matrices using the camera’s logical ID:

- `overhead`
- `rear_left`
- `front_right`

If a camera does not have a configured matrix, the mapper returns `None`. This prevents an uncalibrated detection from creating a false cross-camera association.

## Provisional Testing

Unit tests use identity matrices and known artificial matrices. An identity matrix leaves the selected anchor point unchanged, making the mapping behavior predictable during testing.

These test matrices are not final camera-calibration values.

## Physical Calibration

Final homography matrices depend on the completed platform and fixed camera mounts.

Calibration will require:

1. Mounting every camera in its final position.
2. Marking at least four known reference points on the platform.
3. Capturing an image from each camera.
4. Measuring the image coordinates of the reference points.
5. Matching those image points to their normalized platform coordinates.
6. Calculating a homography matrix for each camera.
7. Testing several additional platform positions to measure mapping error.
8. Recalibrating if a camera or mount moves.

The calibration values must not be finalized until the platform geometry and camera positions are fixed.

## Current Limitations

The bottom-center of an angled detection is only an estimate of the item’s platform contact point. Tall objects, heavy occlusion, inaccurate bounding boxes, and objects extending beyond the platform may reduce mapping accuracy.

Detections without a valid platform position remain available for recognition, but the conservative association algorithm will not merge them across cameras.