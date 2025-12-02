# EasyMark 插件开发概述

EasyMark 采用插件化架构，支持三种类型的插件扩展。

## 插件类型

| 类型 | ID 前缀 | 说明 |
|------|---------|------|
| **数据集插件** | `dataset.*` | 数据集格式导入导出 |
| **推理插件** | `infer.*` | AI 辅助标注推理 |
| **训练插件** | `training` | 模型训练 |

## 内置插件

### 数据集插件

| 插件 | ID | 功能 |
|------|-----|------|
| 通用格式转换 | `dataset.common` | COCO、YOLO、VOC 格式导入导出 |

### 推理插件

| 插件 | ID | 功能 |
|------|-----|------|
| SAM-2 分割 | `infer.sam2` | 交互式分割，点击提示 |
| YOLO 推理 | `infer.ultralytics-yolo` | 目标检测、分割、姿态估计 |

### 训练插件

| 插件 | ID | 功能 |
|------|-----|------|
| Ultralytics YOLO | `ultralytics` | YOLOv8/v11 检测、分割、姿态估计训练 |

## 插件目录结构

```
plugin-name/
├── manifest.json       # 插件配置（必需）
├── infer_service.py    # Python 服务（推理插件）
├── main.py             # Python 入口（训练插件）
├── common-importer.exe # Go 可执行文件（数据集插件）
├── logo.svg            # 插件图标（可选）
├── ui/
│   └── index.html      # 自定义 UI（可选）
└── README.md           # 说明文档（可选）
```

## 开发文档

详细开发文档请参考 `host-plugins/docs/` 目录：

- **推理插件开发指南**: `inference-plugin-guide-zh.md`
- **Manifest 规范**: `manifest-spec-zh.md`
- **Python API 规范**: `python-api-spec-zh.md`

## 插件通信机制

### 数据集插件 (Go/可执行文件)

通过标准输入/输出 (stdin/stdout) 与主程序通信：

```bash
# 探测格式
echo '{"rootPath": "/path/to/dataset"}' | plugin detect

# 导入数据集
echo '{"rootPath": "/path/to/dataset", "params": {}}' | plugin import

# 导出数据集
echo '{"format": "yolo", "outputDir": "/path/to/output", ...}' | plugin export
```

### 推理插件 (Python)

通过 stdio JSON 协议与 Electron 通信：

```python
# 接收命令
for line in sys.stdin:
    req = json.loads(line)
    cmd = req.get("cmd")  # load_model, set_image, infer, unload, shutdown
    
# 发送响应
sys.stdout.write(json.dumps(response) + "\n")
sys.stdout.flush()
```

## 环境变量

Python 插件可用的环境变量：

| 变量 | 说明 |
|------|------|
| `EASYMARK_DATA_PATH` | 数据根目录 |
| `EASYMARK_PLUGIN_PATH` | 当前插件目录 |

## 坐标归一化

EasyMark 使用 **归一化坐标 (0-1)**：

```
normalized_x = pixel_x / image_width
normalized_y = pixel_y / image_height
```

## 安装插件

1. 将插件打包为 `.zip` 或 `.rar`
2. 在 EasyMark 插件页面点击"从磁盘安装"
3. 选择压缩包完成安装

## Python 环境

需要 Python 环境的插件：

1. 进入 "Python 环境管理" 页面
2. 选择插件并点击 "创建虚拟环境"
3. 点击 "安装依赖"

> 依赖安装可能需要较长时间，请耐心等待。
