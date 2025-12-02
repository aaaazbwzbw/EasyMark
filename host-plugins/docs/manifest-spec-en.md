# EasyMark Inference Plugin manifest.json Specification

This document details the complete format of inference plugin `manifest.json` files.

## Complete Example

```json
{
  "id": "infer.sam2",
  "name": {
    "zh-CN": "SAM-2 分割",
    "en-US": "SAM-2 Segmentation"
  },
  "version": "1.0.0",
  "type": "inference",
  "description": {
    "zh-CN": "基于 SAM-2 的交互式图像分割，支持点击提示和找相似目标",
    "en-US": "Interactive image segmentation based on SAM-2"
  },
  "author": "EasyMark Team",
  
  "inference": {
    "icon": "logo.svg",
    "defaultIcon": "sparkles",
    "serviceEntry": "infer_service.py",
    "serviceType": "stdio",
    "supportedTasks": ["segment"],
    "interactionMode": "prompt",
    "ui": {
      "type": "custom",
      "entry": "index.html"
    }
  },
  
  "python": {
    "minVersion": "3.10",
    "requirements": [
      "numpy>=1.20.0,<2",
      "opencv-python>=4.6.0",
      "pillow>=9.0.0",
      "scipy>=1.10.0"
    ],
    "pytorch": {
      "packages": ["torch>=2.0.0", "torchvision>=0.15.0"],
      "gpu": {
        "minCuda": "12.4",
        "indexUrl": "https://download.pytorch.org/whl/cu124"
      },
      "cpu": {
        "indexUrl": "https://download.pytorch.org/whl/cpu"
      }
    }
  }
}
```

## Field Descriptions

### Basic Information

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | ✅ | Unique plugin identifier, format: `infer.<name>` |
| `name` | object | ✅ | Multilingual plugin name |
| `version` | string | ✅ | Version number, follows semver |
| `type` | string | ✅ | Must be `"inference"` |
| `description` | object | ✅ | Multilingual plugin description |
| `author` | string | ❌ | Author name |

### name / description Format

```json
{
  "zh-CN": "中文名称",
  "en-US": "English Name"
}
```

### inference Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `icon` | string | ❌ | Custom icon filename (relative to plugin directory) |
| `defaultIcon` | string | ❌ | Built-in icon name (see table below) |
| `serviceEntry` | string | ✅ | Python service entry file |
| `serviceType` | string | ✅ | Communication type, currently only `"stdio"` |
| `supportedTasks` | array | ✅ | Supported task types |
| `interactionMode` | string | ❌ | Interaction mode (default `"auto"`) |
| `ui` | object | ❌ | UI configuration |

#### Built-in Icons (defaultIcon)

| Value | Description |
|-------|-------------|
| `brain` | Brain icon (AI/neural network) |
| `box` | Box icon (object detection) |
| `scan` | Scan icon |
| `wand` | Magic wand icon |
| `sparkles` | Sparkles icon (segmentation) |
| `cpu` | CPU icon |
| `zap` | Lightning icon (fast) |
| `text-search` | Text search icon |

#### supportedTasks Values

| Value | Description |
|-------|-------------|
| `detect` | Object detection (outputs bounding boxes) |
| `segment` | Image segmentation (outputs polygons) |
| `classify` | Image classification |
| `keypoint` | Keypoint detection |

#### interactionMode Values

| Value | Description | Behavior |
|-------|-------------|----------|
| `auto` | Auto inference | Automatically infer when switching images |
| `prompt` | Prompt mode | User Shift+click to provide prompt points |
| `box` | Box mode | User draws box on example target |
| `text` | Text mode | Uses selected category name as text prompt |

### ui Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | ✅ | `"default"` for default UI, `"custom"` for custom |
| `entry` | string | ❌ | Custom UI entry file (required when `type: custom`) |
| `modelSelector` | boolean | ❌ | Show model selector (default UI) |
| `confidenceSlider` | object/boolean | ❌ | Confidence slider config |
| `iouSlider` | object/boolean | ❌ | IOU slider config |
| `autoInfer` | object/boolean | ❌ | Auto-inference toggle config |

#### Slider Configuration Format

```json
{
  "default": 0.5,
  "min": 0,
  "max": 1,
  "step": 0.01
}
```

### python Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `minVersion` | string | ❌ | Minimum Python version |
| `requirements` | array | ✅ | pip dependencies list |
| `pytorch` | object | ❌ | PyTorch configuration |

#### requirements Format

Supports standard pip format:
```json
[
  "numpy>=1.20.0",
  "numpy>=1.20.0,<2",
  "opencv-python==4.8.0",
  "pillow"
]
```

#### pytorch Configuration

```json
{
  "packages": ["torch>=2.0.0", "torchvision>=0.15.0"],
  "gpu": {
    "minCuda": "12.4",
    "indexUrl": "https://download.pytorch.org/whl/cu124"
  },
  "cpu": {
    "indexUrl": "https://download.pytorch.org/whl/cpu"
  }
}
```

| Field | Description |
|-------|-------------|
| `packages` | PyTorch package list |
| `gpu.minCuda` | Minimum CUDA version |
| `gpu.indexUrl` | GPU version pip index URL |
| `cpu.indexUrl` | CPU version pip index URL |

## Reference Plugin Manifests

### SAM-2 (Interactive Segmentation)

```json
{
  "id": "infer.sam2",
  "inference": {
    "interactionMode": "prompt",
    "supportedTasks": ["segment"]
  }
}
```

### YOLOE (Detection + Visual Prompt)

```json
{
  "id": "infer.yoloe",
  "inference": {
    "interactionMode": "box",
    "supportedTasks": ["detect"]
  }
}
```

### Ultralytics (General YOLO)

```json
{
  "id": "ultralytics",
  "inference": {
    "interactionMode": "auto",
    "supportedTasks": ["detect", "segment"]
  }
}
```

## Notes

1. **id must be unique**: Recommended format `infer.<your-name>`
2. **Internationalization**: `name` and `description` must provide both `zh-CN` and `en-US`
3. **Dependency versions**: Recommend specifying version ranges to avoid compatibility issues
4. **PyTorch GPU**: If plugin requires GPU, must configure `pytorch.gpu`
