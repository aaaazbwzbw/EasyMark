# EasyMark 推理插件开发指南

本文档介绍如何为 EasyMark 开发推理插件。推理插件可以为标注工具提供 AI 辅助能力，如目标检测、图像分割等。

## 目录结构

```
infer-plugins/
└── your-plugin/
    ├── manifest.json      # 插件元数据（必需）
    ├── infer_service.py   # Python 推理服务（必需）
    ├── logo.svg           # 插件图标（可选）
    └── ui/
        └── index.html     # 自定义 UI（可选）
```

## 快速开始

### 1. 创建 manifest.json

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

### 2. 实现 Python 推理服务

Python 服务通过标准输入/输出与主进程通信，使用 JSON 格式。

```python
#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import json
import sys

# 全局变量
_model = None
_current_image = None

# 环境变量
DATA_ROOT = os.environ.get("EASYMARK_DATA_PATH", "")
PLUGIN_PATH = os.environ.get("EASYMARK_PLUGIN_PATH", "")

def _send(payload: dict) -> None:
    """发送 JSON 响应到主进程"""
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()

def _log(msg: str) -> None:
    """输出日志（到 stderr，会显示在控制台）"""
    sys.stderr.write(f"[your_plugin] {msg}\n")
    sys.stderr.flush()

def _handle_load_model(request_id: str, weights: str) -> None:
    """加载模型"""
    global _model
    # 实现模型加载逻辑
    _model = load_your_model(weights)
    _send({
        "requestId": request_id,
        "cmd": "load_model",
        "success": True
    })

def _handle_set_image(request_id: str, image_path: str) -> None:
    """设置当前图像"""
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
    """执行推理"""
    conf = payload.get("conf", 0.5)
    prompt = payload.get("prompt", "")  # 文本提示（text 模式）
    points = payload.get("points", [])  # 提示点（prompt 模式）
    
    # 执行你的推理逻辑
    results = your_inference(_current_image, conf=conf)
    
    # 转换结果为 EasyMark 格式
    annotations = []
    for r in results:
        ann = {
            "type": "rect",  # 或 "polygon"
            "categoryName": r.label,
            "confidence": float(r.score),
            "data": {
                "x": r.x / img_w,      # 归一化坐标
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

### 3. 创建自定义 UI（可选）

如果需要自定义界面（如模型选择、参数调整），创建 `ui/index.html`：

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
    // 与主窗口通信
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
    
    // 监听响应
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
    
    // 加载模型
    async function loadModel(modelPath) {
      const result = await callParent('loadModel', { path: modelPath })
      if (result.success) {
        callParent('notifyModelChanged')
      }
    }
    
    // 通知参数变化
    function onConfChange(value) {
      callParent('notifyParamsChanged', { conf: value })
    }
  </script>
</body>
</html>
```

## 交互模式

插件通过 `interactionMode` 定义与用户的交互方式：

| 模式 | 说明 | 典型用例 |
|------|------|----------|
| `auto` | 切换图片自动推理 | YOLO 目标检测 |
| `prompt` | 需要用户点击提供提示点 | SAM 分割 |
| `box` | 需要用户框选示例目标 | YOLOE Visual Prompt |
| `text` | 使用选中类别名作为文本提示 | Grounding-DINO |

## UI 接口参考

插件 UI 可通过 `callParent()` 调用以下接口：

### 模型相关

```javascript
// 加载模型
await callParent('loadModel', { path: 'model.pt' })

// 通知模型已变化（触发主窗口重新推理）
callParent('notifyModelChanged')
```

### 参数相关

```javascript
// 通知参数变化
callParent('notifyParamsChanged', { 
  conf: 0.5,           // 置信度
  iou: 0.5,            // NMS 阈值
  findSimilar: false   // 找相似开关
})

// 设置单个参数
callParent('setParam', { key: 'conf', value: 0.5 })
```

### 类别相关

```javascript
// 获取项目类别列表
const result = await callParent('getCategories')
// result: { success: true, categories: [{ id, name, type, color }] }

// 获取当前选中的类别
const result = await callParent('getSelectedCategory')
// result: { success: true, category: { id, name, type, color } | null }

// 选中指定类别
await callParent('selectCategory', { categoryId: 1 })
```

### 通知

```javascript
// 显示通知
callParent('notify', { type: 'success', message: '操作成功' })
callParent('notify', { type: 'error', message: '操作失败' })
```

### 特殊功能

```javascript
// 设置 AMG 模式（SAM 找相似功能）
await callParent('setAmg', { enabled: true })
```

## Python 命令参考

主进程发送到 Python 服务的命令：

| 命令 | 参数 | 说明 |
|------|------|------|
| `load_model` | `weights` | 加载指定模型 |
| `set_image` | `path` | 设置当前图像 |
| `infer` | `conf`, `iou`, `prompt`, `points` | 执行推理 |
| `unload` | - | 卸载模型 |
| `shutdown` | - | 关闭服务 |

### 推理参数详解

```json
{
  "requestId": "xxx",
  "cmd": "infer",
  "conf": 0.5,
  "iou": 0.5,
  "prompt": "西瓜",
  "points": [
    {"x": 0.5, "y": 0.3, "type": "positive"},
    {"x": 0.2, "y": 0.8, "type": "negative"}
  ]
}
```

## 标注结果格式

### 矩形框

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

### 多边形

```json
{
  "type": "polygon",
  "categoryName": "cat",
  "confidence": 0.95,
  "polygon": [[0.1, 0.2], [0.3, 0.2], [0.3, 0.5], [0.1, 0.5]]
}
```

## 现有插件参考

- **SAM-2** (`infer.sam2`): 交互式分割，支持点提示和找相似功能
- **YOLOE** (`infer.yoloe`): 目标检测，支持 Visual Prompt
- **Ultralytics** (`ultralytics`): 通用 YOLO 模型推理

## 常见问题

### Q: 如何调试 Python 服务？
A: 使用 `_log()` 输出日志到 stderr，会显示在 Electron 控制台。

### Q: 模型文件放在哪里？
A: 建议放在插件目录下的 `weights/` 文件夹，可通过 `PLUGIN_PATH` 环境变量获取路径。

### Q: 如何支持 GPU？
A: 在 `manifest.json` 中配置 `python.pytorch.gpu`，EasyMark 会自动安装对应 CUDA 版本的 PyTorch。
