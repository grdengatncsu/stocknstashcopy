"""Greedy position-based matching of observations from different cameras."""

from math import dist

from edge.association.associator import AssociatedItem, AssociationReport
from edge.recognition.recognizer import RecognizedItem, RecognitionReport


class PositionAssociator:
    """Group same-name observations that occur near one another on the platform.

    This is a greedy algorithm: observations are processed in report order and
    assigned to the closest compatible group. That keeps the prototype easy to
    inspect while preventing two detections from the same camera from being
    mistaken for independent views of one item.
    """

    def __init__(self, maximum_distance: float = 0.1):
        """Set the largest normalized platform distance allowed for a match."""
        if maximum_distance <= 0:
            raise ValueError("maximum_distance must be a positive value")
        self.maximum_distance = maximum_distance

    def calculate_distance(
        self,
        pos1: tuple[float, float],
        pos2: tuple[float, float],
    ) -> float:
        """Calculate the Euclidean distance between two positions."""
        return dist(pos1, pos2)

    def can_associate(
        self,
        observation: RecognizedItem,
        associated_item: AssociatedItem,
    ) -> bool:
        """Return whether an observation is a safe match for an existing group."""

        # Matching names is the first guard against combining different product
        # types that happen to sit close together.
        if observation.name != associated_item.name:
            return False

        # Unknown detections carry too little information to merge safely.
        if observation.unknown or any(existing.unknown for existing in associated_item.observations):
            return False

        if observation.platform_position is None or associated_item.platform_position is None:
            return False

        # One camera can see two separate instances of the same product. Never
        # collapse those detections into the same physical-item group.
        if any(existing.source_camera == observation.source_camera for existing in associated_item.observations):
            return False

        if self.calculate_distance(observation.platform_position, associated_item.platform_position) > self.maximum_distance:
            return False

        return True

    def add_observation(
        self,
        observation: RecognizedItem,
        associated_item: AssociatedItem,
    ) -> None:
        """Add an observation to an associated item and recalculate the group averages."""
        associated_item.observations.append(observation)
        associated_item.confidence = sum(
            obs.confidence for obs in associated_item.observations
        ) / len(associated_item.observations)

        # The group centroid moves as new camera views are added. Later
        # observations are compared with this averaged position.
        valid_positions = [
            obs.platform_position
            for obs in associated_item.observations
            if obs.platform_position is not None
        ]
        if valid_positions:
            associated_item.platform_position = (
                sum(pos[0] for pos in valid_positions) / len(valid_positions),
                sum(pos[1] for pos in valid_positions) / len(valid_positions),
            )
        else:
            associated_item.platform_position = None

    def create_associated_item(
        self,
        scan_id: str,
        item_number: int,
        observation: RecognizedItem,
    ) -> AssociatedItem:
        """Create a new associated item from a single observation."""
        return AssociatedItem(
            association_id=f"{scan_id}:item-{item_number}",
            name=observation.name,
            confidence=observation.confidence,
            observations=[observation],
            platform_position=observation.platform_position,
        )

    def associate(self, recognition_report: RecognitionReport) -> AssociationReport:
        """Greedily group observations by name, camera source, and position."""
        if not recognition_report.scan_id:
            raise ValueError("scan_id must be provided in the recognition report")
        associated_items: list[AssociatedItem] = []

        for observation in recognition_report.items:
            closest_associated_item: AssociatedItem | None = None
            closest_distance = float("inf")

            # An observation can fit more than one nearby group, so select the
            # nearest valid group rather than the first one encountered.
            for associated_item in associated_items:
                if not self.can_associate(observation, associated_item):
                    continue
                distance = self.calculate_distance(
                    observation.platform_position,
                    associated_item.platform_position,
                )
                if distance < closest_distance:
                    closest_distance = distance
                    closest_associated_item = associated_item
            if closest_associated_item is not None:
                self.add_observation(observation, closest_associated_item)
            else:
                new_associated_item = self.create_associated_item(
                    recognition_report.scan_id,
                    len(associated_items) + 1,
                    observation,
                )
                associated_items.append(new_associated_item)

        return AssociationReport(scan_id=recognition_report.scan_id, items=associated_items)
