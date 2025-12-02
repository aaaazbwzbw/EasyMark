# EasyMark 推理插件 Python 后端接口规范

本文档详细说明推理插件 Python 服务的通信协议和接口规范。

## 通信协议

插件 Python 服务通过 **标准输入/输出 (stdio)** 与 Electron 主进程通信：

- **输入**: 从 `stdin` 逐行读取 JSON 命令
- **输出**: 向 `stdout` 写入 JSON 响应（每行一个）
- **日志**: 向 `stderr` 写入日志信息（会显示在控制台）

## 环境变量

Python 服务启动时，以下环境变量可用：

| 变量 | 说明 | 示例 |
|------|------|------|
| `EASYMARK_DATA_PATH` | 数据根目录 | `D:\EasyMark\Data` |
| `EASYMARK_PLUGIN_PATH` | 插件目录 | `D:\EasyMark\Data\plugins\infer.sam2` |

## 命令格式

### 请求格式

```json
{
  "requestId": "uuid-string",
  "cmd": "command_name",
  "param1": "value1",
  "param2": "value2"
}
```

### 响应格式

```json
{
  "requestId": "uuid-string",
  "cmd": "command_name",
  "success": true,
  "result_field": "value"
}
```

或错误响应：

```json
{
  "requestId": "uuid-string",
  "cmd": "command_name",
  "success": false,
  "error": "错误信息"
}
```

## 标准命令

### load_model - 加载模型

**请求**:
```json
{
  "requestId": "xxx",
  "cmd": "load_model",
  "weights": "sam2_hiera_tiny.pt"
}
```

**响应**:
```json
{
  "requestId": "xxx",
  "cmd": "load_model",
  "success": true,
  "modelPath": "sam2_hiera_tiny.pt",
  "device": "cuda"
}
```

### set_image - 设置当前图像

**请求**:
```json
{
  "requestId": "xxx",
  "cmd": "set_image",
  "path": "project_item/123/images/001.jpg"
}
```

`path` 可能是：
- 相对路径（相对于 `EASYMARK_DATA_PATH`）
- 绝对路径（外部图片）

**响应**:
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

### infer - 执行推理

**请求**:
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
  ],
  "findSimilar": false,
  "outputType": "polygon"
}
```

**参数说明**:

| 参数 | 类型 | 说明 |
|------|------|------|
| `conf` | number | 置信度阈值 (0-1) |
| `iou` | number | NMS IOU 阈值 (0-1) |
| `prompt` | string | 文本提示（`text` 模式使用） |
| `points` | array | 提示点列表（`prompt` 模式使用） |
| `findSimilar` | boolean | 是否找相似目标（SAM 特有） |
| `outputType` | string | 输出类型: `"rect"` 或 `"polygon"` |

**响应**:
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

### unload - 卸载模型

**请求**:
```json
{
  "requestId": "xxx",
  "cmd": "unload"
}
```

**响应**:
```json
{
  "requestId": "xxx",
  "cmd": "unload",
  "success": true
}
```

### shutdown - 关闭服务

**请求**:
```json
{
  "requestId": "xxx",
  "cmd": "shutdown"
}
```

**响应**:
```json
{
  "requestId": "xxx",
  "cmd": "shutdown",
  "success": true
}
```

收到此命令后，服务应调用 `sys.exit(0)` 退出。

## 扩展命令

插件可以定义自己的扩展命令。例如 SAM-2 的 `set_find_similar`:

**请求**:
```json
{
  "requestId": "xxx",
  "cmd": "set_find_similar",
  "enabled": true
}
```

**响应**:
```json
{
  "requestId": "xxx",
  "cmd": "set_find_similar",
  "success": true
}
```

## 标注结果格式

### 矩形框 (rect)

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

- `x`, `y`: 左上角坐标（**归一化**，0-1）
- `width`, `height`: 宽高（**归一化**，0-1）

### 多边形 (polygon)

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

- `polygon`: 顶点坐标数组，每个点为 `[x, y]`（**归一化**，0-1）

### 带关键点的矩形框 (YOLO-Pose)

YOLO-Pose 等姿态估计模型输出矩形框 + 关键点：

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

**字段说明**：

- `x`, `y`, `width`, `height`: 矩形框（归一化 0-1）
- `keypoints`: 关键点数组，每个点为 `[x, y, visibility]`
  - `x`, `y`: 关键点坐标（**归一化** 0-1）
  - `visibility`: 可见性，0=不可见, 1=遮挡, 2=可见

**COCO 17 关键点顺序**（YOLO-Pose 默认）：

| 索引 | 关键点 | 索引 | 关键点 |
|------|--------|------|--------|
| 0 | 鼻子 (nose) | 9 | 左手腕 (left_wrist) |
| 1 | 左眼 (left_eye) | 10 | 右手腕 (right_wrist) |
| 2 | 右眼 (right_eye) | 11 | 左髋 (left_hip) |
| 3 | 左耳 (left_ear) | 12 | 右髋 (right_hip) |
| 4 | 右耳 (right_ear) | 13 | 左膝 (left_knee) |
| 5 | 左肩 (left_shoulder) | 14 | 右膝 (right_knee) |
| 6 | 右肩 (right_shoulder) | 15 | 左踝 (left_ankle) |
| 7 | 左肘 (left_elbow) | 16 | 右踝 (right_ankle) |
| 8 | 右肘 (right_elbow) | | |

**Python 转换示例**：

```python
# YOLO-Pose 结果转 EasyMark 格式
def convert_pose_result(result, img_w, img_h):
    annotations = []
    for box, kpts, conf in zip(result.boxes, result.keypoints, result.boxes.conf):
        x1, y1, x2, y2 = box.xyxy[0].tolist()
        
        # 转换关键点
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

**关键点自动绑定机制**：

插件只需要正确返回 `categoryName` 和 `keypoints` 数组，EasyMark 会自动处理关键点与类别的绑定：

1. 如果 `categoryName`（如 `"person"`）与项目中已创建的矩形框类别匹配；
2. 且该类别已绑定关键点类别（骨架定义）；
3. 则 `keypoints` 数组中的各点会按**索引顺序**自动映射到骨架定义中的各部位。

> **重要**：关键点的**数量和顺序**必须与项目中类别绑定的骨架定义一致。例如 COCO 格式要求 17 个点，顺序为：鼻子、左眼、右眼、左耳、右耳、左肩、右肩...（见上表）。

## 完整示例代码

```python
#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import json
import sys
import traceback
import numpy as np
import cv2

# 全局状态
_model = None
_current_image = None
_current_image_path = None

DATA_ROOT = os.environ.get("EASYMARK_DATA_PATH", "")
PLUGIN_PATH = os.environ.get("EASYMARK_PLUGIN_PATH", "")


def _send(payload: dict) -> None:
    """发送响应"""
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def _log(msg: str) -> None:
    """输出日志"""
    sys.stderr.write(f"[my_plugin] {msg}\n")
    sys.stderr.flush()


def _handle_load_model(request_id: str, weights: str) -> None:
    global _model
    try:
        # 模型路径处理
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
        # 路径处理
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
        
        # 执行推理
        # results = _model.predict(_current_image, conf=conf)
        
        # 转换结果
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

## 调试技巧

1. **使用 `_log()` 输出调试信息**：日志会显示在 Electron 控制台
2. **捕获所有异常**：避免服务崩溃，返回 `success: false` 和错误信息
3. **检查图像路径**：注意相对路径和绝对路径的处理
4. **坐标归一化**：所有坐标必须归一化到 0-1 范围
