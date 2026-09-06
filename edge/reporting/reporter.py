from typing import Protocol
from edge.recognition.recognizer import RecognitionReport

class Reporter(Protocol):
 def report(self, recognition_report: RecognitionReport) -> bool:
  """Report the recognition results to an external system."""
  ...