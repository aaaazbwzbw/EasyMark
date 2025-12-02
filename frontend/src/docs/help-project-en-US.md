# Project Page Guide

The Project page is the core workspace for image annotation.

---

## 1. Create/Select Project

1. After launching the app, click the **Project Manager** on the left sidebar
2. Click the **Project Menu** in the top-right to select or create a project

---

## 2. Import Images or Datasets

After creating a project, you can import images or existing datasets.

### Supported Dataset Formats

- YOLO
- VOC
- COCO

### Import Methods

| Method | Description |
|--------|-------------|
| **Copy** | Copy files to project directory |
| **Hard Link** | Create hard links to save disk space |
| **External Index** | Index only without copying, keeping original location |

---

## 3. Configure Annotation Categories

Switch annotation types in the right panel:

| Type | Description |
|------|-------------|
| **Bounding Box** | Commonly used for object detection |
| **Polygon** | Instance/semantic segmentation |
| **Keypoints** | Pose estimation |

### Keypoint Special Configuration

Keypoints require additional setup:
1. Define semantics for each point (e.g., nose, left eye, right eye)
2. Bind to a bounding box category

> **Note**: After binding, inference results with the same bbox category will automatically use the bound keypoint semantics.

---

## 4. Start Annotating

### 4.1 Select Image

Click an image in the left image list to display it on the canvas.

### 4.2 Select Category

- Bbox and keypoint categories can be selected together
- Polygon categories **cannot** be combined with other types

### 4.3 Draw Annotations

| Annotation Type | Operation |
|-----------------|-----------|
| **Bounding Box** | Hold left mouse button and drag |
| **Polygon** | Hold `Alt` + left click to add vertices |
| **Keypoints** | Hold `Alt` + left click in sequence |

### 4.4 Cancel Drawing

Right-click on empty area to cancel unfinished polygon/keypoint drawing.

---

## 5. Save Annotations

- Click the **Save button** to save manually
- Or enable **Auto-save** in the bottom toolbar

---

## 6. AI-Assisted Annotation

Click the **Plugin button** in the bottom toolbar to open the plugin panel.

### Built-in Assistance Plugins

| Plugin | Usage |
|--------|-------|
| **SAM2** | After loading model, select polygon or bbox category, hold `Shift` and click target for segmentation |
| **YOLO** | After selecting model, switching images triggers automatic inference |

> **Note**: Inference results require **manual saving**. Auto-save does not save inference results.

---

## 7. Keyboard Shortcuts

| Shortcut | Function |
|----------|----------|
| `Ctrl + S` | Save current image annotations |
| `Ctrl + Shift + S` | Save as negative sample |
| `← / →` | Previous / next image |
| `Ctrl + ← / →` | Jump to previous/next unannotated image |
| `Backspace` | Delete selected annotation |
| `V` | Toggle keypoint visibility |
| `Ctrl + 0` | Reset canvas view |
