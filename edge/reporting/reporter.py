from typing import Protocol
from edge.recognition.recognizer import RecognitionReport
from edge.association.associator import AssociationReport

class Reporter(Protocol):
 def report(self, association_report: AssociationReport) -> bool:
  """Report the association results to an external system."""
  ...