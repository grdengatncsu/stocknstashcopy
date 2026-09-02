import unittest

from edge.state_machine import State, StateMachine


def machine_in_stabilizing():
    machine = StateMachine()
    machine.handle_weight_detected()
    return machine


def machine_in_capture():
    machine = machine_in_stabilizing()
    machine.handle_stabilizing(weight_present=True, weight_stable=True)
    return machine


def machine_in_recognize():
    machine = machine_in_capture()
    machine.handle_capture(capture_complete=True)
    return machine


def machine_in_report():
    machine = machine_in_recognize()
    machine.handle_recognize(recognition_complete=True)
    return machine


def machine_in_reset():
    machine = machine_in_report()
    machine.handle_report(acknowledged=True)
    return machine


class TestStartupAndWeightHandling(unittest.TestCase):
    def test_machine_starts_idle_with_clear_flags(self):
        machine = StateMachine()

        self.assertEqual(machine.state, State.IDLE)
        self.assertFalse(machine.cap_valid)
        self.assertFalse(machine.rec_valid)
        self.assertFalse(machine.report_ack)

    def test_weight_detected_moves_idle_to_stabilizing(self):
        machine = StateMachine()

        machine.handle_weight_detected()

        self.assertEqual(machine.state, State.STABILIZING)

    def test_weight_detected_is_ignored_outside_idle(self):
        machine = machine_in_stabilizing()

        machine.handle_weight_detected()

        self.assertEqual(machine.state, State.STABILIZING)

    def test_unstable_weight_remains_stabilizing(self):
        machine = machine_in_stabilizing()

        machine.handle_stabilizing(weight_present=True, weight_stable=False)

        self.assertEqual(machine.state, State.STABILIZING)

    def test_removed_weight_returns_to_idle(self):
        machine = machine_in_stabilizing()

        machine.handle_stabilizing(weight_present=False, weight_stable=False)

        self.assertEqual(machine.state, State.IDLE)

    def test_missing_weight_overrides_stable_signal(self):
        machine = machine_in_stabilizing()

        machine.handle_stabilizing(weight_present=False, weight_stable=True)

        self.assertEqual(machine.state, State.IDLE)

    def test_stable_weight_moves_to_capture(self):
        machine = machine_in_stabilizing()

        machine.handle_stabilizing(weight_present=True, weight_stable=True)

        self.assertEqual(machine.state, State.CAPTURE)


class TestCaptureHandling(unittest.TestCase):
    def test_incomplete_capture_remains_in_capture(self):
        machine = machine_in_capture()

        machine.handle_capture(capture_complete=False)

        self.assertEqual(machine.state, State.CAPTURE)
        self.assertFalse(machine.cap_valid)

    def test_successful_capture_advances_and_sets_flag(self):
        machine = machine_in_capture()

        machine.handle_capture(capture_complete=True)

        self.assertEqual(machine.state, State.RECOGNIZE)
        self.assertTrue(machine.cap_valid)

    def test_valid_capture_is_reused_after_recovery(self):
        machine = machine_in_capture()
        machine.cap_valid = True

        machine.handle_capture(capture_complete=False)

        self.assertEqual(machine.state, State.RECOGNIZE)
        self.assertTrue(machine.cap_valid)

    def test_capture_event_is_ignored_in_wrong_state(self):
        machine = StateMachine()

        machine.handle_capture(capture_complete=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertFalse(machine.cap_valid)


class TestRecognitionHandling(unittest.TestCase):
    def test_incomplete_recognition_remains_in_recognize(self):
        machine = machine_in_recognize()

        machine.handle_recognize(recognition_complete=False)

        self.assertEqual(machine.state, State.RECOGNIZE)
        self.assertFalse(machine.rec_valid)

    def test_successful_recognition_advances_and_sets_flag(self):
        machine = machine_in_recognize()

        machine.handle_recognize(recognition_complete=True)

        self.assertEqual(machine.state, State.REPORT)
        self.assertTrue(machine.rec_valid)

    def test_valid_recognition_is_reused_after_recovery(self):
        machine = machine_in_recognize()
        machine.rec_valid = True

        machine.handle_recognize(recognition_complete=False)

        self.assertEqual(machine.state, State.REPORT)
        self.assertTrue(machine.rec_valid)

    def test_recognition_event_is_ignored_in_wrong_state(self):
        machine = StateMachine()

        machine.handle_recognize(recognition_complete=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertFalse(machine.rec_valid)


class TestReportHandling(unittest.TestCase):
    def test_missing_acknowledgment_remains_in_report(self):
        machine = machine_in_report()

        machine.handle_report(acknowledged=False)

        self.assertEqual(machine.state, State.REPORT)
        self.assertFalse(machine.report_ack)

    def test_acknowledgment_advances_and_sets_flag(self):
        machine = machine_in_report()

        machine.handle_report(acknowledged=True)

        self.assertEqual(machine.state, State.RESET)
        self.assertTrue(machine.report_ack)

    def test_saved_acknowledgment_is_reused_after_recovery(self):
        machine = machine_in_report()
        machine.report_ack = True

        machine.handle_report(acknowledged=False)

        self.assertEqual(machine.state, State.RESET)
        self.assertTrue(machine.report_ack)

    def test_report_event_is_ignored_in_wrong_state(self):
        machine = StateMachine()

        machine.handle_report(acknowledged=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertFalse(machine.report_ack)


class TestResetHandling(unittest.TestCase):
    def test_occupied_platform_remains_in_reset_and_preserves_flags(self):
        machine = machine_in_reset()

        machine.handle_reset(platform_empty=False)

        self.assertEqual(machine.state, State.RESET)
        self.assertTrue(machine.cap_valid)
        self.assertTrue(machine.rec_valid)
        self.assertTrue(machine.report_ack)

    def test_empty_platform_clears_scan_and_returns_to_idle(self):
        machine = machine_in_reset()

        machine.handle_reset(platform_empty=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertFalse(machine.cap_valid)
        self.assertFalse(machine.rec_valid)
        self.assertFalse(machine.report_ack)

    def test_reset_event_is_ignored_in_wrong_state(self):
        machine = StateMachine()

        machine.handle_reset(platform_empty=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertFalse(machine.cap_valid)
        self.assertFalse(machine.rec_valid)
        self.assertFalse(machine.report_ack)


if __name__ == "__main__":
    unittest.main()
class TestRecoveryHandling(unittest.TestCase):
    def test_unresolved_error_remains_in_error(self):
        machine = machine_in_capture()
        machine.handle_error()

        machine.handle_recovery(recovered=False)

        self.assertEqual(machine.state, State.ERROR)
        self.assertEqual(machine.error_source, State.CAPTURE)

    def test_missing_error_source_remains_in_error(self):
        machine = StateMachine()
        machine.state = State.ERROR

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.ERROR)
        self.assertIsNone(machine.error_source)

    def test_idle_error_recovers_to_idle(self):
        machine = StateMachine()
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertIsNone(machine.error_source)

    def test_stabilizing_error_retries_stabilizing(self):
        machine = machine_in_stabilizing()
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.STABILIZING)
        self.assertIsNone(machine.error_source)

    def test_failed_capture_retries_capture(self):
        machine = machine_in_capture()
        original_scan_id = machine.scan_id
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.CAPTURE)
        self.assertFalse(machine.cap_valid)
        self.assertEqual(machine.scan_id, original_scan_id)
        self.assertIsNone(machine.error_source)

    def test_valid_capture_skips_to_recognize(self):
        machine = machine_in_capture()
        machine.cap_valid = True
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.RECOGNIZE)
        self.assertTrue(machine.cap_valid)
        self.assertIsNone(machine.error_source)

    def test_failed_recognition_retries_recognize(self):
        machine = machine_in_recognize()
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.RECOGNIZE)
        self.assertFalse(machine.rec_valid)
        self.assertIsNone(machine.error_source)

    def test_valid_recognition_skips_to_report(self):
        machine = machine_in_recognize()
        machine.rec_valid = True
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.REPORT)
        self.assertTrue(machine.rec_valid)
        self.assertIsNone(machine.error_source)

    def test_unacknowledged_report_retries_report(self):
        machine = machine_in_report()
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.REPORT)
        self.assertFalse(machine.report_ack)
        self.assertIsNone(machine.error_source)

    def test_acknowledged_report_skips_to_reset(self):
        machine = machine_in_report()
        machine.report_ack = True
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.RESET)
        self.assertTrue(machine.report_ack)
        self.assertIsNone(machine.error_source)

    def test_reset_error_recovers_to_reset(self):
        machine = machine_in_reset()
        machine.handle_error()

        machine.handle_recovery(recovered=True)

        self.assertEqual(machine.state, State.RESET)
        self.assertIsNone(machine.error_source)
class TestErrorHandling(unittest.TestCase):
    def test_error_records_the_state_that_failed(self):
        machine = machine_in_capture()
        original_scan_id = machine.scan_id

        machine.handle_error()

        self.assertEqual(machine.state, State.ERROR)
        self.assertEqual(machine.error_source, State.CAPTURE)
        self.assertEqual(machine.scan_id, original_scan_id)

    def test_repeated_error_does_not_overwrite_source(self):
        machine = machine_in_capture()

        machine.handle_error()
        machine.handle_error()

        self.assertEqual(machine.state, State.ERROR)
        self.assertEqual(machine.error_source, State.CAPTURE)

class TestScanIdHandling(unittest.TestCase):
    def test_weight_detection_creates_scan_id(self):
        machine = StateMachine()

        self.assertIsNone(machine.scan_id)

        machine.handle_weight_detected()

        self.assertIsNotNone(machine.scan_id)
        self.assertIsInstance(machine.scan_id, str)

    def test_scan_id_is_preserved_through_scan(self):
        machine = StateMachine()
        machine.handle_weight_detected()
        original_scan_id = machine.scan_id

        machine.handle_stabilizing(
            weight_present=True,
            weight_stable=True,
        )
        self.assertEqual(machine.scan_id, original_scan_id)

        machine.handle_capture(capture_complete=True)
        self.assertEqual(machine.scan_id, original_scan_id)

        machine.handle_recognize(recognition_complete=True)
        self.assertEqual(machine.scan_id, original_scan_id)

        machine.handle_report(acknowledged=True)
        self.assertEqual(machine.scan_id, original_scan_id)

    def test_removed_weight_clears_scan_id(self):
        machine = StateMachine()
        machine.handle_weight_detected()

        self.assertIsNotNone(machine.scan_id)

        machine.handle_stabilizing(
            weight_present=False,
            weight_stable=False,
        )

        self.assertEqual(machine.state, State.IDLE)
        self.assertIsNone(machine.scan_id)

    def test_reset_clears_id_and_next_scan_gets_new_id(self):
        machine = machine_in_reset()
        first_scan_id = machine.scan_id

        machine.handle_reset(platform_empty=True)

        self.assertEqual(machine.state, State.IDLE)
        self.assertIsNone(machine.scan_id)

        machine.handle_weight_detected()
        second_scan_id = machine.scan_id

        self.assertIsNotNone(second_scan_id)
        self.assertNotEqual(first_scan_id, second_scan_id)
    