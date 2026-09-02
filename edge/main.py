from edge.state_machine import StateMachine, State

def show_status(machine):
    print(f"State: {machine.state.name}")
    print(f"Scan ID: {machine.scan_id}")
    print(f"Capture valid: {machine.cap_valid}")
    print(f"Recognition valid: {machine.rec_valid}")
    print(f"Report acknowledged: {machine.report_ack}")
    print()

def main():
    machine = StateMachine()

    # Simulate events
    print("Weight detected")
    machine.handle_weight_detected()
    show_status(machine)
    print("Weight is stabilizing...")
    machine.handle_stabilizing(weight_present=True, weight_stable=False)
    print("Weight is stable...")
    machine.handle_stabilizing(weight_present=True, weight_stable=True)
    show_status(machine)
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
    print("Report acknowledged")
    machine.handle_report(acknowledged=True)
    show_status(machine)
    print("Platform still occupied")
    machine.handle_reset(platform_empty=False)
    show_status(machine)
    print("Platform empty")
    machine.handle_reset(platform_empty=True)
    show_status(machine)

if(__name__ == "__main__"):
    main()