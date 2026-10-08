#!/usr/bin/env python3
import cv2
import json
import os
import time
from ultralytics import YOLO

# Local storage configuration
SAVE_FOLDER = "scans"
DATA_FILE = "local_history.json"
os.makedirs(SAVE_FOLDER, exist_ok=True)

# Create JSON database file if it doesn't exist
if not os.path.exists(DATA_FILE):
    with open(DATA_FILE, "w") as f:
        json.dump([], f)

# Load lightweight YOLO model (downloads automatically on first run)
model = YOLO("yolov8n.pt") 
cap = cv2.VideoCapture(0)

CONFIDENCE_THRESHOLD = 0.60

def save_scan_locally(detected_items, frame):
    timestamp = int(time.time())
    
    # 1. Save frame as image file
    image_filename = os.path.join(SAVE_FOLDER, f"scan_{timestamp}.jpg")
    cv2.imwrite(image_filename, frame)
    
    # 2. Record scan metadata
    scan_record = {
        "id": timestamp,
        "date_time": time.strftime("%Y-%m-%d %H:%M:%S"),
        "items_detected": detected_items,
        "total_items": len(detected_items),
        "image_path": image_filename
    }

    # 3. Append to local JSON dataset
    with open(DATA_FILE, "r") as f:
        history = json.load(f)

    history.append(scan_record)

    with open(DATA_FILE, "w") as f:
        json.dump(history, f, indent=4)

    print(f"\n[SAVED] {len(detected_items)} item(s) -> {image_filename}\n")

print("Scanner active! Press 's' to save scan locally, or 'q' to quit.")

while cap.isOpened():
    ret, frame = cap.read()
    if not ret:
        break

    results = model(frame, verbose=False)[0]
    detected_classes = []

    for box in results.boxes:
        conf = float(box.conf[0])
        if conf >= CONFIDENCE_THRESHOLD:
            cls_id = int(box.cls[0])
            class_name = model.names[cls_id]
            detected_classes.append(class_name)

            x1, y1, x2, y2 = map(int, box.xyxy[0])
            cv2.rectangle(frame, (x1, y1), (x2, y2), (0, 255, 0), 2)
            cv2.putText(frame, f"{class_name} {conf:.2f}", (x1, y1 - 10),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 2)

    cv2.imshow("Grocery Scanner Platform", frame)
    key = cv2.waitKey(1) & 0xFF

    if key == ord('s'):
        if detected_classes:
            save_scan_locally(detected_classes, frame)
        else:
            print("[INFO] No items detected to save.")

    elif key == ord('q'):
        break

cap.release()
cv2.destroyAllWindows()
