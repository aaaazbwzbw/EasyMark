# EasyMark Inference Plugin Development Guide

This document describes how to develop inference plugins for EasyMark. Inference plugins provide AI-assisted capabilities like object detection and image segmentation.

## Directory Structure

```
infer-plugins/
└── your-plugin/
    ├── manifest.json      # Plugin metadata (required)
    ├── infer_service.py   # Python inference service (required)
    ├── logo.svg           # Plugin icon (optional)
    └── ui/
        └── index.html     # Custom UI (optional)
```

## Quick Start

### 1. Create manifest.json

```json
{
  "id": "infer.your-plugin",
  "name": {
    "zh-CN": "你的插件",
    "en-US": "Your Plugin"
  },
  "version": "1.0.0",
  "type": "inference",
  "description": {
    "zh-CN": "插件描述",
    "en-US": "Plugin description"
  },
  "author": "Your Name",
  
  "inference": {
    "icon": "logo.svg",
    "defaultIcon": "brain",
    "serviceEntry": "infer_service.py",
    "serviceType": "stdio",
    "supportedTasks": ["detect", "segment"],
    "interactionMode": "auto",
    "ui": {
      "type": "custom",
      "entry": "index.html"
    }
  },
  
  "python": {
    "minVersion": "3.10",
    "requirements": [
      "numpy>=1.20.0",
      "opencv-python>=4.6.0"
    ],
    "pytorch": {
      "packages": ["torch>=2.0.0", "torchvision>=0.15.0"],
      "gpu": {
        "minCuda": "12.4",
        "indexUrl": "https://download.pytorch.org/whl/cu124"
      }
    }
  }
}
```

### 2. Implement Python Inference Service

The Python service communicates with the main process via stdin/stdout using JSON format.

```python
#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import json
import sys

# Global variables
_model = None
_current_image = None

# Environment variables
DATA_ROOT = os.environ.get("EASYMARK_DATA_PATH", "")
PLUGIN_PATH = os.environ.get("EASYMARK_PLUGIN_PATH", "")

def _send(payload: dict) -> None:
    """Send JSON response to main process"""
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()

def _log(msg: str) -> None:
    """Output log (to stderr, shown in console)"""
    sys.stderr.write(f"[your_plugin] {msg}\n")
    sys.stderr.flush()

def _handle_load_model(request_id: str, weights: str) -> None:
    """Load model"""
    global _model
    # Implement model loading logic
    _model = load_your_model(weights)
    _send({
        "requestId": request_id,
        "cmd": "load_model",
        "success": True
    })

def _handle_set_image(request_id: str, image_path: str) -> None:
    """Set current image"""
    global _current_image
    import cv2
    _current_image = cv2.imread(image_path)
    _current_image = cv2.cvtColor(_current_image, cv2.COLOR_BGR2RGB)
    _send({
        "requestId": request_id,
        "cmd": "set_image",
        "success": True
    })

def _handle_infer(request_id: str, payload: dict) -> None:
    """Execute inference"""
    conf = payload.get("conf", 0.5)
    prompt = payload.get("prompt", "")  # Text prompt (text mode)
    points = payload.get("points", [])  # Prompt points (prompt mode)
    
    # Execute your inference logic
    results = your_inference(_current_image, conf=conf)
    
    # Convert results to EasyMark format
    annotations = []
    for r in results:
        ann = {
            "type": "rect",  # or "polygon"
            "categoryName": r.label,
            "confidence": float(r.score),
            "data": {
                "x": r.x / img_w,      # Normalized coordinates
                "y": r.y / img_h,
                "width": r.w / img_w,
                "height": r.h / img_h
            }
        }
        annotations.append(ann)
    
    _send({
        "requestId": request_id,
        "cmd": "infer",
        "success": True,
        "annotations": annotations
    })

def main():
    _log("Service started")
    for line in sys.stdin:
        req = json.loads(line.strip())
        cmd = req.get("cmd")
        request_id = req.get("requestId", "")
        
        if cmd == "load_model":
            _handle_load_model(request_id, req.get("weights", ""))
        elif cmd == "set_image":
            _handle_set_image(request_id, req.get("path", ""))
        elif cmd == "infer":
            _handle_infer(request_id, req)
        elif cmd == "unload":
            _model = None
            _send({"requestId": request_id, "cmd": "unload", "success": True})
        elif cmd == "shutdown":
            _send({"requestId": request_id, "cmd": "shutdown", "success": True})
            sys.exit(0)

if __name__ == "__main__":
    main()
```

### 3. Create Custom UI (Optional)

If you need a custom interface (model selection, parameter adjustment), create `ui/index.html`:

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: system-ui; padding: 12px; }
    .model-btn { padding: 8px 12px; margin: 4px; cursor: pointer; }
    .model-btn.active { background: #3b82f6; color: white; }
  </style>
</head>
<body>
  <div id="models"></div>
  <div id="status"></div>
  
  <script>
    // Communicate with main window
    let requestIdCounter = 0
    const pendingRequests = new Map()
    
    function callParent(action, data = {}) {
      return new Promise((resolve, reject) => {
        const requestId = ++requestIdCounter
        pendingRequests.set(requestId, { resolve, reject })
        window.parent.postMessage({
          type: 'plugin-ui-request',
          requestId,
          action,
          data
        }, '*')
      })
    }
    
    // Listen for responses
    window.addEventListener('message', (event) => {
      const msg = event.data
      if (msg.type === 'plugin-ui-response') {
        const pending = pendingRequests.get(msg.requestId)
        if (pending) {
          pendingRequests.delete(msg.requestId)
          pending.resolve(msg.result)
        }
      }
    })
    
    // Load model
    async function loadModel(modelPath) {
      const result = await callParent('loadModel', { path: modelPath })
      if (result.success) {
        callParent('notifyModelChanged')
      }
    }
    
    // Notify parameter change
    function onConfChange(value) {
      callParent('notifyParamsChanged', { conf: value })
    }
  </script>
</body>
</html>
```

## Interaction Modes

Plugins define user interaction via `interactionMode`:

| Mode | Description | Typical Use Case |
|------|-------------|------------------|
| `auto` | Auto-infer on image switch | YOLO detection |
| `prompt` | Requires user click for prompt points | SAM segmentation |
| `box` | Requires user to draw box on example | YOLOE Visual Prompt |
| `text` | Uses selected category name as text prompt | Grounding-DINO |

## UI API Reference

Plugin UI can call these APIs via `callParent()`:

### Model Related

```javascript
// Load model
await callParent('loadModel', { path: 'model.pt' })

// Notify model changed (triggers re-inference)
callParent('notifyModelChanged')
```

### Parameters Related

```javascript
// Notify parameter change
callParent('notifyParamsChanged', { 
  conf: 0.5,           // Confidence threshold
  iou: 0.5,            // NMS threshold
  findSimilar: false   // Find similar toggle
})

// Set single parameter
callParent('setParam', { key: 'conf', value: 0.5 })
```

### Category Related

```javascript
// Get project categories
const result = await callParent('getCategories')
// result: { success: true, categories: [{ id, name, type, color }] }

// Get currently selected category
const result = await callParent('getSelectedCategory')
// result: { success: true, category: { id, name, type, color } | null }

// Select a category
await callParent('selectCategory', { categoryId: 1 })
```

### Notifications

```javascript
// Show notification
callParent('notify', { type: 'success', message: 'Operation successful' })
callParent('notify', { type: 'error', message: 'Operation failed' })
```

### Special Features

```javascript
// Set AMG mode (SAM find-similar feature)
await callParent('setAmg', { enabled: true })
```

## Python Command Reference

Commands sent from main process to Python service:

| Command | Parameters | Description |
|---------|------------|-------------|
| `load_model` | `weights` | Load specified model |
| `set_image` | `path` | Set current image |
| `infer` | `conf`, `iou`, `prompt`, `points` | Execute inference |
| `unload` | - | Unload model |
| `shutdown` | - | Shutdown service |

### Inference Parameters Detail

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
  ]
}
```

## Annotation Result Format

### Bounding Box

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

### Polygon

```json
{
  "type": "polygon",
  "categoryName": "cat",
  "confidence": 0.95,
  "polygon": [[0.1, 0.2], [0.3, 0.2], [0.3, 0.5], [0.1, 0.5]]
}
```

## Reference Plugins

- **SAM-2** (`infer.sam2`): Interactive segmentation with point prompts and find-similar
- **YOLOE** (`infer.yoloe`): Object detection with Visual Prompt support
- **Ultralytics** (`ultralytics`): General YOLO model inference

## FAQ

### Q: How to debug Python service?
A: Use `_log()` to output logs to stderr, which will appear in Electron console.

### Q: Where to put model files?
A: Recommend placing in `weights/` folder under plugin directory, accessible via `PLUGIN_PATH` environment variable.

### Q: How to enable GPU support?
A: Configure `python.pytorch.gpu` in `manifest.json`, EasyMark will automatically install PyTorch with appropriate CUDA version.
