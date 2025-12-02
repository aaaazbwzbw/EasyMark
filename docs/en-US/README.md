# EasyMark Development Documentation (English)

## Documentation Index

### Project Architecture

| Document | Description |
|----------|-------------|
| [Architecture Overview](./architecture.md) | Overall architecture, tech stack, data flow |

### Plugin Development

| Document | Description |
|----------|-------------|
| [Plugin Development Overview](./plugin-overview.md) | Plugin system overview |
| [Inference Plugin Guide](../../host-plugins/docs/inference-plugin-guide-en.md) | Detailed inference plugin guide |
| [Manifest Specification](../../host-plugins/docs/manifest-spec-en.md) | manifest.json configuration spec |
| [Python API Specification](../../host-plugins/docs/python-api-spec-en.md) | Python service API spec |

### Dataset Plugins

| Document | Description |
|----------|-------------|
| [Common Dataset Plugin](../../host-plugins/dataset-common/README.md) | COCO/YOLO/VOC format support |

### Example Code

- **Inference Plugin Examples**: `host-plugins/infer-plugins/`
  - SAM-2 segmentation plugin
  - Ultralytics YOLO inference plugin
- **Training Plugin Example**: `host-plugins/train_python/`
- **Dataset Plugin Example**: `host-plugins/dataset-common/`

## Quick Links

- [User Manual](../../frontend/src/docs/help-en-US.md)
- [Project README](../../README.md)
