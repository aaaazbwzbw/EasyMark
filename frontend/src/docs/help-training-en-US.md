# Training Page Guide

The Training page is for creating and managing model training tasks.

---

## 1. Enter Training Page

Click **Training Management** in the left sidebar.

---

## 2. Prerequisites

Before training, ensure:

- Required plugins are ready
- Plugin dependencies are installed (see Python Environment Management section)

---

## 3. Create Training Set

### 3.1 Steps

1. Click **Create Training Set** button
2. Select categories from versions
3. Enter training set name and confirm

### 3.2 Category Selection Rules

| Supported | Not Supported |
|-----------|---------------|
| Same categories from different versions within the same project | Cross-project category selection |

### 3.3 Manage Training Sets

Saved training sets can be managed on the right panel:

- **Delete**: Remove the training set
- **Update**: Replace training set categories with currently selected ones

---

## 4. Create Training Task

Click **New Training Task** and configure:

| Setting | Description |
|---------|-------------|
| Training Name | Task identifier |
| Data Source | Select training set or choose dataset from directory |
| Data Split | Configure train/val ratios |
| Training Type | Bbox, polygon, keypoints |
| Training Plugin | Select supported training plugin |
| Model & Version | Choose appropriate model and version |
| Training Parameters | Configure hyperparameters (epochs, batch size, etc.) |

Click confirm to start training after configuration.

---

## 5. Training Results

After training completes:

- Appears in **Training History** on the right
- Can delete or open directory to view models and metrics
- Trained models are **automatically imported** into YOLO inference plugin for immediate use

---

## 6. Relation to Other Pages

- **Dataset page**: Source of training set data
- **Inference plugin**: Trained models are automatically available
- **Plugins page**: Extends training plugins and model support
