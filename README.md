<div align="center">

# EasyMark

<img src="docs/assets/logo.png" alt="EasyMark Logo" width="120">

**Professional Computer Vision Annotation Tool**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-blue.svg)]()
[![Electron](https://img.shields.io/badge/Electron-28-47848F.svg?logo=electron)](https://www.electronjs.org/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D.svg?logo=vue.js)](https://vuejs.org/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?logo=go)](https://golang.org/)
[![Latest Release](https://img.shields.io/github/v/release/aaaazbwzbw/EasyMark)](https://github.com/aaaazbwzbw/EasyMark/releases/latest)
[![Total Downloads](https://img.shields.io/github/downloads/aaaazbwzbw/EasyMark/total)](https://github.com/aaaazbwzbw/EasyMark/releases)

[English](README.md) | [简体中文](README_zh-CN.md)

---

## Download

Get the latest version of EasyMark from the releases page:

<a href="https://github.com/aaaazbwzbw/EasyMark/releases/latest"><img src="https://img.shields.io/badge/Download_Latest_Version-4FC08D?style=for-the-badge&logo=windows" alt="Download Latest Version"></a>


</div>

---

## Overview

EasyMark is a modern, high-performance annotation tool designed for computer vision tasks. It supports multiple annotation types with AI-assisted labeling, comprehensive dataset management, and integrated model training capabilities.

<div align="center">

<img src="docs/assets/demo1.gif" alt="EasyMark AI Segmentation Demo" width="49%">
<img src="docs/assets/demo2.gif" alt="EasyMark Auto Detection Demo" width="49%">

</div>

## Features

### Annotation

| Type | Use Case | Operation |
|------|----------|-----------|
| **Bounding Box** | Object Detection | Click & Drag |
| **Polygon** | Instance/Semantic Segmentation | Alt + Click |
| **Keypoints** | Pose Estimation | Alt + Click (with skeleton binding) |

### AI-Assisted Annotation

Boost your labeling efficiency with built-in AI plugins:

- **SAM2** - Segment Anything Model for interactive segmentation
- **YOLO** - Automatic object detection on image switch

### Dataset Management

- **Format Support**: YOLO, VOC, COCO import/export
- **Version Control**: Snapshot, rollback, and manage dataset versions
- **Flexible Export**: Cross-project merging with custom train/val/test splits

### Model Training

- Built-in **Ultralytics YOLO** training pipeline
- Automatic model deployment to inference plugins
- Training history with metrics visualization

### Plugin System

Extend EasyMark with custom plugins for:
- Dataset import/export formats
- Training frameworks
- Inference backends

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | Vue 3 + TypeScript + Vite + TailwindCSS |
| Desktop | Electron 28 |
| Backend | Go 1.21+ |
| Plugins | Python 3.10+ |

## Quick Start

### Prerequisites

- **Node.js** 18+
- **Go** 1.21+
- **Python** 3.10+ (for AI plugins)

### Installation

```bash
# Clone the repository
git clone https://github.com/aaaazbwzbw/EasyMark.git
cd easymark

# Install frontend dependencies
cd frontend && npm install

# Install Electron dependencies
cd ../host-electron && npm install

# Build backend
cd ../backend-go && go build
```

### Development

```bash
# Terminal 1: Start backend
cd backend-go && ./backend-go

# Terminal 2: Start frontend dev server
cd frontend && npm run dev

# Terminal 3: Start Electron
cd host-electron && npm run dev
```

### Build

```bash
# Build for production
cd host-electron && npm run build
```

## Project Structure

```
easymark/
├── frontend/              # Vue 3 frontend application
│   └── src/docs/          # User documentation
├── host-electron/         # Electron main process
├── backend-go/            # Go backend service
├── host-plugins/          # Built-in plugins
│   ├── infer-plugins/     # Inference plugins (SAM2, YOLO)
│   ├── train_python/      # Training plugin
│   └── dataset-common/    # Dataset format converters
└── docs/                  # Development documentation
```

## Documentation

- **User Guide**: Available in the app's Help page
- **Plugin Development**: See `docs/plugin-development-guide.md`
- **API Reference**: See `docs/plugin-api-reference.md`

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

> 中文版：[贡献指南](CONTRIBUTING_zh-CN.md)

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Roadmap

- [ ] Plugin Marketplace
- [ ] Cloud Sync
- [ ] Team Collaboration
- [ ] More AI Models Support
- [ ] Video Annotation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Ultralytics](https://github.com/ultralytics/ultralytics) for YOLO
- [Segment Anything](https://github.com/facebookresearch/segment-anything-2) for SAM2
- [Electron](https://www.electronjs.org/)
- [Vue.js](https://vuejs.org/)

---

<div align="center">

**If you find EasyMark useful, please consider giving it a star!**

</div>
