# EasyMark 推理插件 manifest.json 规范

本文档详细说明推理插件 `manifest.json` 文件的完整格式。

## 完整示例

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

## 字段说明

### 基础信息

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | 插件唯一标识，格式: `infer.<name>` |
| `name` | object | ✅ | 多语言插件名称 |
| `version` | string | ✅ | 版本号，遵循 semver |
| `type` | string | ✅ | 必须为 `"inference"` |
| `description` | object | ✅ | 多语言插件描述 |
| `author` | string | ❌ | 作者名称 |

### name / description 格式

```json
{
  "zh-CN": "中文名称",
  "en-US": "English Name"
}
```

### inference 配置

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `icon` | string | ❌ | 自定义图标文件名（相对于插件目录） |
| `defaultIcon` | string | ❌ | 内置图标名称（见下表） |
| `serviceEntry` | string | ✅ | Python 服务入口文件 |
| `serviceType` | string | ✅ | 通信类型，目前仅支持 `"stdio"` |
| `supportedTasks` | array | ✅ | 支持的任务类型 |
| `interactionMode` | string | ❌ | 交互模式（默认 `"auto"`） |
| `ui` | object | ❌ | UI 配置 |

#### 内置图标 (defaultIcon)

| 值 | 说明 |
|------|------|
| `brain` | 大脑图标（AI/神经网络） |
| `box` | 方框图标（目标检测） |
| `scan` | 扫描图标 |
| `wand` | 魔术棒图标 |
| `sparkles` | 星星图标（分割） |
| `cpu` | CPU 图标 |
| `zap` | 闪电图标（快速） |
| `text-search` | 文本搜索图标 |

#### supportedTasks 可选值

| 值 | 说明 |
|------|------|
| `detect` | 目标检测（输出矩形框） |
| `segment` | 图像分割（输出多边形） |
| `classify` | 图像分类 |
| `keypoint` | 关键点检测 |

#### interactionMode 可选值

| 值 | 说明 | 行为 |
|------|------|------|
| `auto` | 自动推理 | 切换图片时自动执行推理 |
| `prompt` | 提示点模式 | 用户 Shift+点击 提供提示点后推理 |
| `box` | 框选模式 | 用户框选示例目标后推理全图 |
| `text` | 文本模式 | 使用选中类别名作为文本提示推理 |

### ui 配置

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `type` | string | ✅ | `"default"` 使用默认 UI，`"custom"` 使用自定义 |
| `entry` | string | ❌ | 自定义 UI 入口文件（`type: custom` 时必需） |
| `modelSelector` | boolean | ❌ | 是否显示模型选择器（默认 UI） |
| `confidenceSlider` | object/boolean | ❌ | 置信度滑块配置 |
| `iouSlider` | object/boolean | ❌ | IOU 滑块配置 |
| `autoInfer` | object/boolean | ❌ | 自动推理开关配置 |

#### 滑块配置格式

```json
{
  "default": 0.5,
  "min": 0,
  "max": 1,
  "step": 0.01
}
```

### python 配置

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `minVersion` | string | ❌ | 最低 Python 版本 |
| `requirements` | array | ✅ | pip 依赖列表 |
| `pytorch` | object | ❌ | PyTorch 配置 |

#### requirements 格式

支持标准 pip 格式：
```json
[
  "numpy>=1.20.0",
  "numpy>=1.20.0,<2",
  "opencv-python==4.8.0",
  "pillow"
]
```

#### pytorch 配置

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

| 字段 | 说明 |
|------|------|
| `packages` | PyTorch 包列表 |
| `gpu.minCuda` | 最低 CUDA 版本 |
| `gpu.indexUrl` | GPU 版本 pip 源 |
| `cpu.indexUrl` | CPU 版本 pip 源 |

## 现有插件 manifest 参考

### SAM-2 (交互式分割)

```json
{
  "id": "infer.sam2",
  "inference": {
    "interactionMode": "prompt",
    "supportedTasks": ["segment"]
  }
}
```

### YOLOE (目标检测 + Visual Prompt)

```json
{
  "id": "infer.yoloe",
  "inference": {
    "interactionMode": "box",
    "supportedTasks": ["detect"]
  }
}
```

### Ultralytics (通用 YOLO)

```json
{
  "id": "ultralytics",
  "inference": {
    "interactionMode": "auto",
    "supportedTasks": ["detect", "segment"]
  }
}
```

## 注意事项

1. **id 必须唯一**：推荐格式 `infer.<your-name>`
2. **国际化**：`name` 和 `description` 必须提供 `zh-CN` 和 `en-US`
3. **依赖版本**：建议指定版本范围，避免兼容性问题
4. **PyTorch GPU**：如果插件需要 GPU，必须配置 `pytorch.gpu`
