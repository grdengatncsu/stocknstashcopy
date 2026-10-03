"""Tests for simulated platform presence signals."""

import unittest

from edge.presence import PresenceSignals, SimulatedPresenceSource


class TestSimulatedPresenceSource(unittest.TestCase):
    def setUp(self):
        self.source = SimulatedPresenceSource()

    def test_simulated_sequence_matches_scan_lifecycle(self):
        expected = [
            PresenceSignals(
                weight_detected=True,
                weight_present=True,
            ),
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            PresenceSignals(
                weight_present=True,
                weight_stable=True,
            ),
            PresenceSignals(
                platform_empty=True,
            ),
        ]

        actual = [
            self.source.read()
            for _ in range(len(expected))
        ]

        self.assertEqual(actual, expected)

    def test_sequence_advances_one_read_at_a_time(self):
        first = self.source.read()
        second = self.source.read()

        self.assertTrue(first.weight_detected)
        self.assertTrue(first.weight_present)
        self.assertFalse(first.weight_stable)

        self.assertFalse(second.weight_detected)
        self.assertTrue(second.weight_present)
        self.assertTrue(second.weight_stable)

    def test_exhausted_sequence_reports_empty_platform(self):
        for _ in range(6):
            self.source.read()

        signals = self.source.read()

        self.assertEqual(
            signals,
            PresenceSignals(platform_empty=True),
        )


if __name__ == "__main__":
    unittest.main()