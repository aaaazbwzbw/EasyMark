# Changelog

All notable changes to EasyMark will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.2] - 2025-12-27

### Added
- **SAM2**
  - Continuous prompt refinement with `Ctrl + Shift + Click` (append prompt points)
- **Python Environment Management**
  - Allow binding an external virtual environment for Python plugins
- **Badges**
  - Persist the corner mark status of the Python environment, and no longer check every time the page is switched.

### Changed
- Plugin installation/update now uses overlay copy instead of deleting the entire plugin directory (prevents losing downloaded weights)

### Fixed
- Fixed IPC structured clone error caused by sending Vue Proxy objects through Electron IPC
- Improved SAM2 mask-to-polygon post-processing to reduce jagged edges
- Fixed dataset export category conflict selection where the checkbox could remain checked after warning
- Fixed `dataset-common` executable build/packaging so it is included in release artifacts
- Reduced redundant `/api/python/plugins-deps-summary` checks triggered by navigation

---

## [1.0.1] - 2025-12-05

### Added
- **Classification Category Annotation**
  - Added annotation logic for classification categories
  - Import/export for classification datasets will be improved in future versions (feedback on commonly used classification models and data formats is welcome)

### Fixed
- Fixed incorrect path passed when clicking the "Open Model Download Directory" button in the training panel
- Fixed YOLOv11 series model download failure issue (modelId format mismatch)
- Fixed incomplete built-in format conversion plugin issue (exe file was not uploaded)

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

