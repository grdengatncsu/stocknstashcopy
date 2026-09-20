# Stock ’n Stash

Stock ’n Stash is a countertop grocery-scanning system that automatically identifies grocery items and adds them to a household inventory.

## Prototype 1 Goal

Prototype 1 will demonstrate the following process:

1. Detect groceries placed on the platform.
2. Wait until the measured weight is stable.
3. Capture images from three cameras under ambient lighting, with controlled lighting evaluated if needed.
4. Identify the grocery items.
5. Send the results to the inventory application.
6. Ask the user to confirm uncertain results.

## Hardware

The current prototype hardware includes:

- Raspberry Pi Compute Module 5 development kit
- Hailo-8L AI accelerator
- Two MIPI side cameras
- One overhead USB camera
- Four load cells
- NAU7802 load-cell ADC
- Controlled lighting under evaluation for consistent image capture
- Actuators and limit switches
- Wi-Fi communication with the application

Controlled lighting is not required for initial hardware bring-up, but remains an active design investigation and may be added if ambient lighting causes unreliable exposure or recognition.

## System Operating States

The embedded system follows this sequence:

```text
Idle
→ Weight Detected
→ Stable
→ Capture
→ Recognize
→ Report
→ Reset
```

## Camera Names

The three cameras use permanent position-based names:

- `overhead`
- `rear_left`
- `front_right`

Images captured during the same scan will share a scan ID and timestamp.

## Planned Software Components

- Camera capture
- Load-cell monitoring
- Lighting evaluation/control if required
- Scan state machine
- Grocery detection and recognition
- Barcode and OCR assistance
- Multi-camera result combination
- App communication
- Inventory storage
- System logging and diagnostics

## Project Structure

```text
stock-n-stash/
├── edge/
│   ├── cameras/
│   ├── load_sensor/
│   ├── lighting/
│   ├── recognition/
│   ├── state_machine/
│   └── communication/
├── app/
├── configuration/
├── documentation/
├── models/
├── test_images/
└── tests/
```

## Development Status

Current work:

- [ ] Confirm final hardware interfaces
- [ ] Create simulated load-cell input
- [ ] Create mock three-camera capture
- [ ] Implement the scan state machine
- [ ] Run a pretrained model on saved images
- [ ] Define the Prototype 1 grocery list
- [ ] Define the recognition-result format
- [ ] Connect the edge system to the application
- [ ] Test the system on the real hardware

## Running the Project

The installation and startup commands will be added after the initial software environment is created.

Planned mock-mode command:

```text
python -m edge.main --mock
```

Mock mode will use saved images and simulated sensor events so development can continue without the physical hardware.

## Recognition Result Format

Example output:

```json
{
  "scan_id": "scan-001",
  "items": [
    {
      "name": "Cheerios",
      "upc": "016000275287",
      "confidence": 0.93,
      "status": "accepted"
    },
    {
      "name": "Unknown",
      "upc": null,
      "confidence": 0.41,
      "status": "confirmation_required"
    }
  ]
}
```

## Documentation

Setup, camera configuration, model deployment, calibration, testing, and recovery procedures will be maintained in the `documentation` folder.
