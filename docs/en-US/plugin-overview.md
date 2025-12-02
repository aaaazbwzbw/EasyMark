# EasyMark Plugin Development Overview

EasyMark uses a plugin architecture supporting three types of extensions.

## Plugin Types

| Type | ID Prefix | Description |
|------|-----------|-------------|
| **Dataset Plugin** | `dataset.*` | Dataset format import/export |
| **Inference Plugin** | `infer.*` | AI-assisted annotation inference |
| **Training Plugin** | `training` | Model training |

## Built-in Plugins

### Dataset Plugins

| Plugin | ID | Features |
|--------|-----|----------|
| Common Format Converter | `dataset.common` | COCO, YOLO, VOC format import/export |

### Inference Plugins

| Plugin | ID | Features |
|--------|-----|----------|
| SAM-2 Segmentation | `infer.sam2` | Interactive segmentation with point prompts |
| YOLO Inference | `infer.ultralytics-yolo` | Object detection, segmentation, pose estimation |

### Training Plugins

| Plugin | ID | Features |
|--------|-----|----------|
| Ultralytics YOLO | `ultralytics` | YOLOv8/v11 detection, segmentation, pose training |

## Plugin Directory Structure

```
plugin-name/
├── manifest.json       # Plugin config (required)
├── infer_service.py    # Python service (inference plugins)
├── main.py             # Python entry (training plugins)
├── common-importer.exe # Go executable (dataset plugins)
├── logo.svg            # Plugin icon (optional)
├── ui/
│   └── index.html      # Custom UI (optional)
└── README.md           # Documentation (optional)
```

## Development Documentation

For detailed development guides, see `host-plugins/docs/`:

- **Inference Plugin Guide**: `inference-plugin-guide-en.md`
- **Manifest Specification**: `manifest-spec-en.md`
- **Python API Specification**: `python-api-spec-en.md`

## Plugin Communication

### Dataset Plugins (Go/Executable)

Communicate via stdin/stdout with the main program:

```bash
# Detect format
echo '{"rootPath": "/path/to/dataset"}' | plugin detect

# Import dataset
echo '{"rootPath": "/path/to/dataset", "params": {}}' | plugin import

# Export dataset
echo '{"format": "yolo", "outputDir": "/path/to/output", ...}' | plugin export
```

### Inference Plugins (Python)

Communicate via stdio JSON protocol with Electron:

```python
# Receive commands
for line in sys.stdin:
    req = json.loads(line)
    cmd = req.get("cmd")  # load_model, set_image, infer, unload, shutdown
    
# Send response
sys.stdout.write(json.dumps(response) + "\n")
sys.stdout.flush()
```

## Environment Variables

Environment variables available to Python plugins:

| Variable | Description |
|----------|-------------|
| `EASYMARK_DATA_PATH` | Data root directory |
| `EASYMARK_PLUGIN_PATH` | Current plugin directory |

## Coordinate Normalization

EasyMark uses **normalized coordinates (0-1)**:

```
normalized_x = pixel_x / image_width
normalized_y = pixel_y / image_height
```

## Installing Plugins

1. Package plugin as `.zip` or `.rar`
2. Click "Install from Disk" in EasyMark Plugins page
3. Select the archive to complete installation

## Python Environment

For plugins requiring Python:

1. Go to "Python Environment Management" page
2. Select plugin and click "Create Virtual Environment"
3. Click "Install Dependencies"

> Dependency installation may take some time, please be patient.
