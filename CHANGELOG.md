# Changelog

All notable changes to EasyMark will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2025-12-02

> **EasyMark** is the first release after a complete rewrite of **YoloMarkFlow**.
> 
> Compared to the original project, EasyMark features a brand new architecture 
> (Electron + Vue 3 + Go), a modern UI, a powerful plugin system, and 
> comprehensive dataset management capabilities.
> 
> The Go backend handles all file operations, resulting in massive performance 
> improvements for large-scale datasets.

### Added
- **Annotation**
  - Bounding box annotation for object detection
  - Polygon annotation for segmentation
  - Keypoints annotation for pose estimation
  - Auto-save and manual save modes

- **AI Assistance**
  - SAM2 interactive segmentation plugin
  - YOLO automatic detection plugin
  - Trained models auto-import to inference

- **Dataset Management**
  - Import from COCO, YOLO, VOC formats
  - Export to COCO, YOLO, VOC formats
  - Dataset versioning and rollback
  - Cross-project category selection for export

- **Training**
  - Ultralytics YOLOv8/v11 training support
  - Detection, segmentation, pose estimation tasks
  - Training history and metrics visualization

- **Plugin System**
  - Dataset import/export plugins
  - Inference plugins with Python backend
  - Training plugins with progress tracking

- **User Experience**
  - Dark/Light theme support
  - Keyboard shortcuts
  - Multi-language (zh-CN, en-US)

