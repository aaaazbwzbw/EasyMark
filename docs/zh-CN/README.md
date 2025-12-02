# EasyMark 开发文档（中文）

## 文档目录

### 项目架构

| 文档 | 说明 |
|------|------|
| [架构概述](./architecture.md) | 整体架构、技术选型、数据流 |

### 插件开发

| 文档 | 说明 |
|------|------|
| [插件开发概述](./plugin-overview.md) | 插件系统总览 |
| [推理插件开发指南](../../host-plugins/docs/inference-plugin-guide-zh.md) | 推理插件详细开发指南 |
| [Manifest 规范](../../host-plugins/docs/manifest-spec-zh.md) | manifest.json 配置规范 |
| [Python API 规范](../../host-plugins/docs/python-api-spec-zh.md) | Python 服务 API 规范 |

### 数据集插件

| 文档 | 说明 |
|------|------|
| [通用数据集插件](../../host-plugins/dataset-common/README.md) | COCO/YOLO/VOC 格式支持 |

### 示例代码

- **推理插件示例**: `host-plugins/infer-plugins/`
  - SAM-2 分割插件
  - Ultralytics YOLO 推理插件
- **训练插件示例**: `host-plugins/train_python/`
- **数据集插件示例**: `host-plugins/dataset-common/`

## 快速链接

- [用户使用手册](../../frontend/src/docs/help-zh-CN.md)
- [项目 README](../../README_zh-CN.md)
