# Plugins & Inference Guide

This page covers the plugin system and Python environment management.

---

## 1. Plugin System

### 1.1 Overview

The plugin system framework is complete, supporting:

- Dataset import/export
- Training
- Inference-assisted annotation

> For plugin development, refer to example source code and documentation in the `host-plugins` directory.

### 1.2 Built-in Plugins

| Plugin | Function |
|--------|----------|
| **SAM2** | Segmentation-assisted annotation |
| **Format Converter** | Supports YOLO, COCO, VOC formats |
| **Ultralytics YOLO** | Inference and training |

### 1.3 Install Plugins

Plugins from other sources come as `.rar` or `.zip` archives.

Click **Install from Disk** to install.

### 1.4 Uninstall & Update

| Action | Description |
|--------|-------------|
| **Uninstall** | Also removes the plugin's deployed virtual environment |
| **Update** | Simply install over the existing version |

> Plugin marketplace coming soon.

---

## 2. Python Environment Management

### 2.1 Enter Environment Management

Click **Python Environment Management** in the left sidebar.

### 2.2 Dependency Detection

Upon entering the page, the system automatically checks dependency installation status for plugins requiring Python environments.

Uninstalled plugins will show clear warnings.

### 2.3 Prerequisites

Before deploying environments, ensure:

- Python is installed on your system
- Python is added to system PATH environment variable

### 2.4 Deployment Process

1. Click the plugin you want to configure
2. View the plugin's required dependencies
3. Click **Create Virtual Environment**
4. After environment creation, click **Install Dependencies**

> **Tip**: Dependency installation may fail in some cases. Use pip commands to manually install appropriate dependency versions.

---

## 3. AI-Assisted Annotation Plugins

### 3.1 Usage

Click the **Plugin button** in the bottom toolbar to open the plugin panel.

### 3.2 Built-in Assistance Plugins

| Plugin | Usage |
|--------|-------|
| **SAM2** | After loading model, select polygon or bbox category, hold `Shift` and click target for segmentation |
| **YOLO** | After selecting model, switching images triggers automatic inference |

### 3.3 Automatic Keypoint Binding

When using pose estimation models (e.g. YOLO-Pose):

1. Model's output bbox category name matches a bbox category in the project
2. That bbox category has a keypoint category bound to it
3. Output keypoints automatically use the bound keypoint semantics

> **Note**: Keypoint count and order must match the bound skeleton definition.

### 3.4 Saving Inference Results

Inference results require **manual saving**. Auto-save does not save inference results.
