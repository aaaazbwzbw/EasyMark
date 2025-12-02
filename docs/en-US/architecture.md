# EasyMark Architecture Overview

## Overall Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Electron Main Process                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Window Mgmt │  │ Plugin Svc  │  │ Python Process Mgmt │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
           │                    │                    │
           │ IPC                │ stdio              │ spawn
           ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Vue Frontend  │  │ Inference Plugin│  │ Training Plugin │
│   (Renderer)    │  │   (Python)      │  │   (Python)      │
└────────┬────────┘  └─────────────────┘  └─────────────────┘
         │ HTTP/WS
         ▼
┌─────────────────┐
│   Go Backend    │
│   (:18080)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   File System   │
│   (SQLite/JSON) │
└─────────────────┘
```

## Technology Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| **Desktop** | Electron | Cross-platform, web tech stack, mature ecosystem |
| **Frontend** | Vue 3 + TypeScript | Reactive, type-safe, Composition API |
| **UI** | TailwindCSS + shadcn/ui | Modern design, highly customizable |
| **Backend** | Go | High performance, single binary, concurrent |
| **Storage** | SQLite + JSON | Lightweight, no installation, easy backup |
| **Plugins** | Python | Rich AI ecosystem, PyTorch/TensorFlow support |

## Core Modules

### 1. Frontend (frontend/)

```
src/
├── views/              # Page views
│   ├── ProjectView     # Main annotation interface
│   ├── DatasetView     # Dataset management
│   ├── TrainingView    # Training management
│   └── PluginsView     # Plugin management
├── components/         # Reusable components
│   ├── AnnotationCanvas # Canvas (core)
│   └── ...
├── composables/        # Composition functions
└── locales/            # i18n
```

**Annotation Canvas** is the core component:
- Canvas rendering & interaction
- Bbox/polygon/keypoint drawing
- Zoom, pan, selection operations

### 2. Backend (backend-go/)

```
backend-go/
├── main.go           # Router entry
├── project.go        # Project management
├── images.go         # Image management
├── categories.go     # Category management
├── annotations.go    # Annotation management
└── remaining.go      # Other features
```

**Responsibilities:**
- CRUD for projects/images/categories/annotations
- Dataset version management
- Plugin invocation & management
- Python environment management
- Training task scheduling

### 3. Electron Main Process (host-electron/)

```
host-electron/
├── main.ts           # Main process entry
├── plugin-service.ts # Plugin service management
└── preload.ts        # Preload script
```

**Responsibilities:**
- Window creation & management
- Inference plugin process management (stdio)
- File system access
- System-level API calls

## Data Flow

### Annotation Flow

```
User Action → Canvas Component → Frontend State → HTTP API → Go Backend → SQLite
```

### Inference Flow

```
User Trigger → Frontend → IPC → Electron → Python Process → Return Results → Render to Canvas
```

### Training Flow

```
User Config → HTTP API → Go Backend → Spawn Python Training → WebSocket Push Logs
```

## Data Storage

### Project Data

```
{DataRoot}/
├── projects/
│   └── {project-id}/
│       ├── images/           # Image files
│       └── project.db        # SQLite database
├── datasets/
│   └── {version-id}/         # Dataset version snapshots
├── plugins/                  # Installed plugins
├── training/                 # Training outputs
└── models/                   # Inference models
```

### SQLite Tables

- `images` - Image metadata
- `categories` - Category definitions (keypoint skeleton stored in `meta` field)
- `annotations` - Annotation data

## Plugin System

### Plugin Types

| Type | Communication | Language |
|------|---------------|----------|
| Dataset Plugin | stdin/stdout | Go (compiled to exe) |
| Inference Plugin | stdio JSON | Python |
| Training Plugin | stdio + WebSocket | Python |

### Plugin Lifecycle

```
Install → Load manifest → Create venv → Install deps → Ready
                                              ↓
                                        Start Service ↔ Call
                                              ↓
                                         Unload/Stop
```

## Design Principles

1. **Frontend-Backend Separation**: Frontend handles display, backend handles data & files
2. **Plugin-based Extension**: Core features built-in, AI capabilities via plugins
3. **Configurable Paths**: All data paths from settings, no hardcoding
4. **i18n First**: All user-visible text supports multiple languages
5. **Offline First**: No cloud dependency, local data storage
