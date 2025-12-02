# EasyMark User Guide

EasyMark is a professional computer vision annotation tool supporting bounding boxes, polygons, keypoints, and more. It integrates AI-assisted annotation, dataset management, and model training into a complete workflow.

---

## 1. Annotation

### 1.1 Create/Select Project

1. After launching the app, click the **Project Manager** on the left sidebar
2. Click the **Project Menu** in the top-right to select or create a project

### 1.2 Import Images or Datasets

After creating a project, you can import images or existing datasets:

- **Supported dataset formats**: YOLO, VOC, COCO
- **Import methods**:
  - **Copy**: Copy files to the project directory
  - **Hard Link**: Create hard links to save disk space
  - **External Index**: Index only without copying, keeping original location

### 1.3 Configure Annotation Categories

Switch annotation types in the right panel:

| Type | Description |
|------|-------------|
| **Bounding Box** | Commonly used for object detection |
| **Polygon** | Instance/semantic segmentation |
| **Keypoints** | Pose estimation, requires configuring point semantics and binding to a bbox category |

> **Keypoints Note**: Keypoints must be bound to a bounding box category. During inference, detections with the same bbox category will automatically use the bound keypoint semantics.

### 1.4 Start Annotating

1. Click an image in the left image list to display it on the canvas
2. Select a category first (bbox and keypoint categories can be selected together, but polygon cannot be combined with other types)
3. Draw annotations:
   - **Bounding Box**: Hold left mouse button and drag
   - **Polygon/Keypoints**: Hold `Alt` + left mouse button to draw
   - **Cancel drawing**: Right-click on empty area to cancel unfinished polygon/keypoint drawing

### 1.5 Save Annotations

- Click the **Save button** to save manually
- Or enable **Auto-save** in the bottom toolbar

### 1.6 AI-Assisted Annotation

Two built-in annotation assistance plugins:

| Plugin | Usage |
|--------|-------|
| **SAM2** | After loading model, select a polygon or bbox category, hold `Shift` and click target for segmentation inference |
| **YOLO** | After selecting model, switching images triggers automatic inference |

Click the **Plugin button** in the bottom toolbar to open the plugin panel.

> **Note**: Inference results require **manual saving**. Auto-save does not save inference results.

---

## 2. Dataset

### 2.1 Enter Dataset Page

After completing annotations, click **Dataset Management** in the left sidebar.

### 2.2 Sync Version

Hover over a project and click the **Sync button** to update or create a dataset version.

### 2.3 Export Dataset

1. Select categories from different projects and versions (cross-project, cross-version multi-select supported, but same categories from different versions within the same project are not supported)
2. Click the **Export Dataset** button that appears in the top-right
3. Select export path, format, and dataset split ratios
4. Click **Start Export**

### 2.4 Version Management

Right-click a version in the left tree:

- **Delete**: Remove the version
- **Rollback**: After confirmation, restore the project to this version's state

---

## 3. Training

### 3.1 Enter Training Page

Click **Training Management** in the left sidebar.

### 3.2 Prerequisites

Before training, ensure:
- Required plugins are ready
- Plugin dependencies are installed (see Python Environment Management section)

### 3.3 Create Training Set

1. Click the **Create Training Set** button
2. Select categories from versions (same categories from different versions within the same project supported, but cross-project category selection is not supported)
3. Enter a training set name and confirm

Saved training sets can be deleted or updated on the right panel (Update: replace training set categories with currently selected ones).

### 3.4 Create Training Task

1. Click **New Training Task**
2. Configure training parameters:
   - Training name
   - Select training set or choose dataset from directory
   - Configure data split (train/val ratios)
   - Select training type (bbox, polygon, keypoints)
   - Select training plugin, model, and version
   - Configure training hyperparameters
3. Click confirm to start training

### 3.5 Training Results

- Completed training appears in **Training History** on the right
- Can delete or open directory to view models and metrics
- Trained models are automatically imported into the YOLO inference plugin

---

## 4. Plugins

### 4.1 Plugin System

The plugin system framework is complete, supporting:
- Dataset import/export
- Training
- Inference-assisted annotation

> For plugin development, refer to example source code and documentation in the `host-plugins` directory.

### 4.2 Built-in Plugins

| Plugin | Function |
|--------|----------|
| **SAM2** | Segmentation-assisted annotation |
| **Format Converter** | Supports YOLO, COCO, VOC formats |
| **Ultralytics YOLO** | Inference and training |

### 4.3 Install Plugins

Plugins from other sources come as `.rar` or `.zip` archives. Click **Install from Disk** to install.

### 4.4 Uninstall & Update

- **Uninstall**: Also removes the plugin's deployed virtual environment
- **Update**: Simply install over the existing version

> Plugin marketplace coming soon.

---

## 5. Python Environment Management

### 5.1 Enter Environment Management

Click **Python Environment Management** in the left sidebar.

### 5.2 Dependency Detection

Upon entering the page, the system automatically checks dependency installation status for plugins requiring Python environments. Uninstalled plugins will show clear warnings.

### 5.3 Prerequisites

Before deploying environments, ensure:
- Python is installed on your system
- Python is added to system PATH environment variable

### 5.4 Deployment Process

1. Click the plugin you want to configure
2. View the plugin's required dependencies
3. Click **Create Virtual Environment**
4. After environment creation, click **Install Dependencies**

> **Tip**: Dependency installation may fail in some cases. Use pip commands to manually install appropriate dependency versions.

---

## 6. Keyboard Shortcuts

| Shortcut | Function |
|----------|----------|
| `Ctrl + S` | Save current image annotations |
| `Ctrl + Shift + S` | Save as negative sample |
| `← / →` | Previous / next image |
| `Ctrl + ← / →` | Jump to previous/next unannotated image |
| `Backspace` | Delete selected annotation |
| `V` | Toggle keypoint visibility |
| `Ctrl + 0` | Reset canvas view |

---

## 7. FAQ

### No images or datasets visible
- Check if **Data Root Directory** is correctly configured in Settings
- Confirm backend service is running normally

### Training or export unresponsive
- Check log output at bottom of Training page
- Confirm Python environment and GPU drivers are properly installed

### Plugin installation failed
- Confirm archive format is `.zip` or `.rar`
- Check error message in installation dialog

### Dependency installation failed
- Check network connection
- Try manually installing with pip commands
- Confirm Python version compatibility

### Inference plugin fails to load model (c10.dll error)
- **Cause**: System is missing Microsoft Visual C++ Redistributable runtime library
- **Solution**: Download and install [VC++ Redistributable](https://aka.ms/vs/17/release/vc_redist.x64.exe), then restart the application
