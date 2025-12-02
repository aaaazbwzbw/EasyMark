# Dataset Page Guide

The Dataset page manages annotation data versioning and export.

---

## 1. Enter Dataset Page

After completing annotations, click **Dataset Management** in the left sidebar.

---

## 2. Sync Version

Hover over a project to reveal the **Sync button**. Click it to update or create a version for that project.

> Versions are snapshots of project annotation states, useful for rollback and management.

---

## 3. Export Dataset

### 3.1 Select Export Content

Select categories from different projects and versions:

- **Supported**: Cross-project, cross-version multi-select
- **Not supported**: Same categories from different versions within the same project

### 3.2 Execute Export

1. After selecting categories, the **Export Dataset** button appears in the top-right
2. Configure:
   - Export path
   - Export format (YOLO, VOC, COCO, etc.)
   - Dataset split ratios (train/val/test)
3. Click **Start Export**

---

## 4. Version Management

Right-click a version in the left tree for these options:

| Action | Description |
|--------|-------------|
| **Delete** | Remove this version |
| **Rollback** | After confirmation, restore project to this version's state |

> **Rollback Note**: Rollback restores the project's annotation state to when this version was created. Unsaved changes will be lost.

---

## 5. Relation to Other Pages

- **Project page**: Dataset versions are snapshots of project annotation states
- **Training page**: Training tasks use dataset versions as data sources
- **Plugins page**: New export plugins extend available export formats
