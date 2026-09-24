"""Stable pipeline exceptions shared by hardware and network adapters.

Vendor libraries all raise different exception types.  Adapters translate
those details into these small, project-owned exceptions so the controller can
recover without importing camera, accelerator, sensor, or HTTP dependencies.
"""


class EdgeComponentError(RuntimeError):
    """Base class for an expected edge-component failure."""


class CaptureError(EdgeComponentError):
    """A camera could not produce a complete image."""


class RecognitionError(EdgeComponentError):
    """The recognition or association stage could not produce a report."""


class ReportingError(EdgeComponentError):
    """A report was rejected or its acknowledgement was invalid."""
