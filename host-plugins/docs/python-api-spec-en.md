# EasyMark Inference Plugin Python Backend API Specification

This document details the communication protocol and API specification for inference plugin Python services.

## Communication Protocol

Plugin Python services communicate with the Electron main process via **standard input/output (stdio)**:

- **Input**: Read JSON commands line by line from `stdin`
- **Output**: Write JSON responses to `stdout` (one per line)
- **Logs**: Write log information to `stderr` (displayed in console)

## Environment Variables

The following environment variables are available when the Python service starts:

| Variable | Description | Example |
|----------|-------------|---------|
| `EASYMARK_DATA_PATH` | Data root directory | `D:\EasyMark\Data` |
| `EASYMARK_PLUGIN_PATH` | Plugin directory | `D:\EasyMark\Data\plugins\infer.sam2` |

## Command Format

### Request Format

```json
{
  "requestId": "uuid-string",
  "cmd": "command_name",
  "param1": "value1",
  "param2": "value2"
}
```

### Response Format

```json
{
  "requestId": "uuid-string",
  "cmd": "command_name",
  "success": true,
  "result_field": "value"
}
```

Or error response:

```json
{
  "requestId": "uuid-string",
  "cmd": "command_name",
  "success": false,
  "error": "Error message"
}
```

## Standard Commands

### load_model - Load Model

**Request**:
```json
{
  "requestId": "xxx",
  "cmd": "load_model",
  "weights": "sam2_hiera_tiny.pt"
}
```

**Response**:
```json
{
  "requestId": "xxx",
  "cmd": "load_model",
  "success": true,
  "modelPath": "sam2_hiera_tiny.pt",
  "device": "cuda"
}
```

### set_image - Set Current Image

**Request**:
```json
{
  "requestId": "xxx",
  "cmd": "set_image",
  "path": "project_item/123/images/001.jpg"
}
```

`path` can be:
- Relative path (relative to `EASYMARK_DATA_PATH`)
- Absolute path (external images)

**Response**:
```json
{
  "requestId": "xxx",
  "cmd": "set_image",
  "success": true,
  "imagePath": "/full/path/to/image.jpg",
  "imageSize": {
    "width": 1920,
    "height": 1080
  }
}
```

### infer - Execute Inference

**Request**:
```json
{
  "requestId": "xxx",
  "cmd": "infer",
  "conf": 0.5,
  "iou": 0.5,
  "prompt": "watermelon",
  "points": [
    {"x": 0.5, "y": 0.3, "type": "positive"},
    {"x": 0.2, "y": 0.8, "type": "negative"}
  ],
  "findSimilar": false,
  "outputType": "polygon"
}
```

**Parameter Description**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `conf` | number | Confidence threshold (0-1) |
| `iou` | number | NMS IOU threshold (0-1) |
| `prompt` | string | Text prompt (`text` mode) |
| `points` | array | Prompt points list (`prompt` mode) |
| `findSimilar` | boolean | Find similar targets (SAM specific) |
| `outputType` | string | Output type: `"rect"` or `"polygon"` |

**Response**:
```json
{
  "requestId": "xxx",
  "cmd": "infer",
  "success": true,
  "annotations": [
    {
      "type": "rect",
      "categoryName": "cat",
      "confidence": 0.95,
      "data": {
        "x": 0.1,
        "y": 0.2,
        "width": 0.3,
        "height": 0.4
      }
    }
  ]
}
```

### unload - Unload Model

**Request**:
```json
{
  "requestId": "xxx",
  "cmd": "unload"
}
```

**Response**:
```json
{
  "requestId": "xxx",
  "cmd": "unload",
  "success": true
}
```

### shutdown - Shutdown Service

**Request**:
```json
{
  "requestId": "xxx",
  "cmd": "shutdown"
}
```

**Response**:
```json
{
  "requestId": "xxx",
  "cmd": "shutdown",
  "success": true
}
```

After receiving this command, the service should call `sys.exit(0)` to exit.

## Extended Commands

Plugins can define their own extended commands. For example, SAM-2's `set_find_similar`:

**Request**:
```json
{
  "requestId": "xxx",
  "cmd": "set_find_similar",
  "enabled": true
}
```

**Response**:
```json
{
  "requestId": "xxx",
  "cmd": "set_find_similar",
  "success": true
}
```

## Annotation Result Format

### Bounding Box (rect)

```json
{
  "type": "rect",
  "categoryName": "cat",
  "confidence": 0.95,
  "data": {
    "x": 0.1,
    "y": 0.2,
    "width": 0.3,
    "height": 0.4
  }
}
```

- `x`, `y`: Top-left corner coordinates (**normalized**, 0-1)
- `width`, `height`: Width and height (**normalized**, 0-1)

### Polygon

```json
{
  "type": "polygon",
  "categoryName": "cat",
  "confidence": 0.95,
  "polygon": [
    [0.1, 0.2],
    [0.3, 0.2],
    [0.35, 0.4],
    [0.3, 0.5],
    [0.1, 0.5]
  ]
}
```

- `polygon`: Array of vertex coordinates, each point is `[x, y]` (**normalized**, 0-1)

### Bounding Box with Keypoints (YOLO-Pose)

YOLO-Pose and other pose estimation models output bounding box + keypoints:

```json
{
  "type": "rect",
  "categoryName": "person",
  "confidence": 0.95,
  "data": {
    "x": 0.1,
    "y": 0.1,
    "width": 0.2,
    "height": 0.5,
    "keypoints": [
      [0.15, 0.12, 2],
      [0.16, 0.12, 2],
      [0.14, 0.12, 2],
      [0.18, 0.13, 2],
      [0.12, 0.13, 2],
      [0.20, 0.20, 2],
      [0.10, 0.20, 2],
      [0.22, 0.30, 2],
      [0.08, 0.30, 2],
      [0.23, 0.38, 2],
      [0.07, 0.38, 2],
      [0.18, 0.40, 2],
      [0.12, 0.40, 2],
      [0.19, 0.52, 2],
      [0.11, 0.52, 2],
      [0.20, 0.65, 2],
      [0.10, 0.65, 2]
    ]
  }
}
```

**Field Description**:

- `x`, `y`, `width`, `height`: Bounding box (normalized 0-1)
- `keypoints`: Array of keypoints, each point is `[x, y, visibility]`
  - `x`, `y`: Keypoint coordinates (**normalized** 0-1)
  - `visibility`: Visibility, 0=invisible, 1=occluded, 2=visible

**COCO 17 Keypoint Order** (YOLO-Pose default):

| Index | Keypoint | Index | Keypoint |
|-------|----------|-------|----------|
| 0 | nose | 9 | left_wrist |
| 1 | left_eye | 10 | right_wrist |
| 2 | right_eye | 11 | left_hip |
| 3 | left_ear | 12 | right_hip |
| 4 | right_ear | 13 | left_knee |
| 5 | left_shoulder | 14 | right_knee |
| 6 | right_shoulder | 15 | left_ankle |
| 7 | left_elbow | 16 | right_ankle |
| 8 | right_elbow | | |

**Python Conversion Example**:

```python
# Convert YOLO-Pose result to EasyMark format
def convert_pose_result(result, img_w, img_h):
    annotations = []
    for box, kpts, conf in zip(result.boxes, result.keypoints, result.boxes.conf):
        x1, y1, x2, y2 = box.xyxy[0].tolist()
        
        # Convert keypoints
        keypoints = []
        for kp in kpts.data[0]:
            kx, ky, kc = kp.tolist()  # x, y, confidence
            visibility = 2 if kc > 0.5 else (1 if kc > 0.2 else 0)
            keypoints.append([kx / img_w, ky / img_h, visibility])
        
        ann = {
            "type": "rect",
            "categoryName": "person",
            "confidence": float(conf),
            "data": {
                "x": x1 / img_w,
                "y": y1 / img_h,
                "width": (x2 - x1) / img_w,
                "height": (y2 - y1) / img_h,
                "keypoints": keypoints
            }
        }
        annotations.append(ann)
    return annotations
```

**Automatic Keypoint Binding**:

Plugins only need to return the correct `categoryName` and `keypoints` array. EasyMark automatically handles keypoint-to-category binding:

1. If `categoryName` (e.g., `"person"`) matches an existing bbox category in the project;
2. And that category has a bound keypoint category (skeleton definition);
3. Then points in the `keypoints` array will be automatically mapped to skeleton parts by **index order**.

> **Important**: The **number and order** of keypoints must match the skeleton definition bound to the category in the project. For example, COCO format requires 17 points in order: nose, left_eye, right_eye, left_ear, right_ear, left_shoulder, right_shoulder... (see table above).

## Complete Example Code

```python
#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import json
import sys
import traceback
import numpy as np
import cv2

# Global state
_model = None
_current_image = None
_current_image_path = None

DATA_ROOT = os.environ.get("EASYMARK_DATA_PATH", "")
PLUGIN_PATH = os.environ.get("EASYMARK_PLUGIN_PATH", "")


def _send(payload: dict) -> None:
    """Send response"""
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def _log(msg: str) -> None:
    """Output log"""
    sys.stderr.write(f"[my_plugin] {msg}\n")
    sys.stderr.flush()


def _handle_load_model(request_id: str, weights: str) -> None:
    global _model
    try:
        # Model path handling
        if not os.path.isabs(weights):
            weights = os.path.join(PLUGIN_PATH, "weights", weights)
        
        _log(f"Loading model: {weights}")
        # _model = load_your_model(weights)
        
        _send({
            "requestId": request_id,
            "cmd": "load_model",
            "success": True,
            "modelPath": weights
        })
    except Exception as e:
        _log(f"Load failed: {e}")
        _log(traceback.format_exc())
        _send({
            "requestId": request_id,
            "cmd": "load_model",
            "success": False,
            "error": str(e)
        })


def _handle_set_image(request_id: str, path: str) -> None:
    global _current_image, _current_image_path
    try:
        # Path handling
        path = path.replace('/', os.sep).replace('\\', os.sep)
        if not os.path.isabs(path) and DATA_ROOT:
            path = os.path.join(DATA_ROOT, path)
        
        if not os.path.exists(path):
            raise FileNotFoundError(f"Image not found: {path}")
        
        _log(f"Setting image: {path}")
        image = cv2.imread(path)
        _current_image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
        _current_image_path = path
        
        _send({
            "requestId": request_id,
            "cmd": "set_image",
            "success": True,
            "imagePath": path,
            "imageSize": {
                "width": _current_image.shape[1],
                "height": _current_image.shape[0]
            }
        })
    except Exception as e:
        _log(f"Set image failed: {e}")
        _send({
            "requestId": request_id,
            "cmd": "set_image",
            "success": False,
            "error": str(e)
        })


def _handle_infer(request_id: str, payload: dict) -> None:
    try:
        if _model is None:
            raise RuntimeError("Model not loaded")
        if _current_image is None:
            raise RuntimeError("No image set")
        
        conf = payload.get("conf", 0.5)
        img_h, img_w = _current_image.shape[:2]
        
        _log(f"Running inference, conf={conf}")
        
        # Execute inference
        # results = _model.predict(_current_image, conf=conf)
        
        # Convert results
        annotations = []
        # for r in results:
        #     annotations.append({
        #         "type": "rect",
        #         "categoryName": r.label,
        #         "confidence": float(r.conf),
        #         "data": {
        #             "x": r.x1 / img_w,
        #             "y": r.y1 / img_h,
        #             "width": (r.x2 - r.x1) / img_w,
        #             "height": (r.y2 - r.y1) / img_h
        #         }
        #     })
        
        _send({
            "requestId": request_id,
            "cmd": "infer",
            "success": True,
            "annotations": annotations
        })
    except Exception as e:
        _log(f"Inference failed: {e}")
        _log(traceback.format_exc())
        _send({
            "requestId": request_id,
            "cmd": "infer",
            "success": False,
            "error": str(e)
        })


def _handle_unload(request_id: str) -> None:
    global _model, _current_image, _current_image_path
    _model = None
    _current_image = None
    _current_image_path = None
    _log("Model unloaded")
    _send({
        "requestId": request_id,
        "cmd": "unload",
        "success": True
    })


def main():
    _log("Service started (stdio mode)")
    _log(f"DATA_ROOT: {DATA_ROOT}")
    _log(f"PLUGIN_PATH: {PLUGIN_PATH}")
    
    try:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            
            try:
                req = json.loads(line)
                request_id = req.get("requestId", "")
                cmd = req.get("cmd", "")
                
                if cmd == "load_model":
                    _handle_load_model(request_id, req.get("weights", ""))
                elif cmd == "set_image":
                    _handle_set_image(request_id, req.get("path", ""))
                elif cmd == "infer":
                    _handle_infer(request_id, req)
                elif cmd == "unload":
                    _handle_unload(request_id)
                elif cmd == "shutdown":
                    _log("Shutting down")
                    _send({
                        "requestId": request_id,
                        "cmd": "shutdown",
                        "success": True
                    })
                    sys.exit(0)
                else:
                    _send({
                        "requestId": request_id,
                        "cmd": cmd,
                        "success": False,
                        "error": f"Unknown command: {cmd}"
                    })
            except json.JSONDecodeError as e:
                _log(f"Invalid JSON: {e}")
            except Exception as e:
                _log(f"Error: {e}")
                _log(traceback.format_exc())
    
    except KeyboardInterrupt:
        _log("Interrupted")
    
    sys.exit(0)


if __name__ == "__main__":
    main()
```

## Debugging Tips

1. **Use `_log()` for debug output**: Logs appear in Electron console
2. **Catch all exceptions**: Avoid service crashes, return `success: false` with error message
3. **Check image paths**: Handle both relative and absolute paths correctly
4. **Normalize coordinates**: All coordinates must be normalized to 0-1 range
