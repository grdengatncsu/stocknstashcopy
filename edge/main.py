"""Interactive demonstration of the scan states without physical hardware.

This file is intentionally a simulation. It shows the expected event order,
but it is not the production loop that will run on the Compute Module 5.
"""

from edge.state_machine import StateMachine


def show_status(machine: StateMachine) -> None:
    """Print the workflow fields that are useful while debugging a scan."""
    print(f"State: {machine.state.name}")
    print(f"Scan ID: {machine.scan_id}")
    print(f"Capture valid: {machine.cap_valid}")
    print(f"Recognition valid: {machine.rec_valid}")
    print(f"Report acknowledged: {machine.report_ack}")
    print()


def main() -> None:
    """Simulate one successful scan from item detection through reset."""
    machine = StateMachine()

    # HARDWARE TODO: Replace these manual events with a production loop. That
    # loop must read weight_detected, weight_present, weight_stable, and
    # platform_empty from the calibrated NAU7802/load-cell adapter. Issue #31
    # verified /dev/i2c-1 and address 0x2A; issue #33 still blocks final mounted
    # thresholds and issue #55 owns the adapter integration.
    print("Weight detected")
    machine.handle_weight_detected()
    show_status(machine)
    print("Weight is stabilizing...")
    machine.handle_stabilizing(weight_present=True, weight_stable=False)
    print("Weight is stable...")
    machine.handle_stabilizing(weight_present=True, weight_stable=True)
    show_status(machine)

    # HARDWARE TODO: The real CAPTURE stage must enable the final GPIO-driven
    # lighting, wait for it to settle, and use the physical MIPI/USB cameras.
    # Lighting GPIO pins and camera port/device assignments are not finalized.
    print("Capture incomplete")
    machine.handle_capture(capture_complete=False)
    show_status(machine)
    print("Capture complete")
    machine.handle_capture(capture_complete=True)
    show_status(machine)
    print("Recognition incomplete")
    machine.handle_recognize(recognition_complete=False)
    show_status(machine)
    print("Recognition complete")
    machine.handle_recognize(recognition_complete=True)
    show_status(machine)

    # INTEGRATION TODO (#46): Compose the existing HttpReporter here with the
    # deployed Pi base URL and an explicitly approved review threshold. This
    # demo calls StateMachine directly so it intentionally does not send data.
    print("Report acknowledged")
    machine.handle_report(acknowledged=True)
    show_status(machine)
    print("Platform still occupied")
    machine.handle_reset(platform_empty=False)
    show_status(machine)
    print("Platform empty")
    machine.handle_reset(platform_empty=True)
    show_status(machine)


if __name__ == "__main__":
    main()
