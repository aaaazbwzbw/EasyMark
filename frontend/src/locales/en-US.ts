const enUS = {
  app: {
    title: 'EasyMark'
  },
  header: {
    settings: 'Settings'
  },
  home: {
    welcome: 'Welcome to EasyMark.'
  },
  notifications: {
    panelTitle: 'Notifications',
    clear: 'Clear',
    collapse: 'Collapse',
    success: 'Operation succeeded!',
    info: 'This is an info message',
    warning: 'Please check your input',
    error: 'Operation failed, please retry'
  },
  sidebar: {
    project: 'Project manager',
    dataset: 'Dataset manager',
    training: 'Training manager',
    plugins: 'Plugin marketplace',
    pythonEnv: 'Python Environment',
    ui: 'UI playground',
    help: 'Help & Docs',
    newVersionAvailable: 'New version v{version} available, click to download'
  },
  project: {
    status: {
      none: 'No project opened'
    },
    actions: {
      new: 'New project',
      importDataset: 'Import dataset',
      importImages: 'Import images'
    },
    createModal: {
      title: 'New project',
      nameLabel: 'Project name',
      namePlaceholder: 'Enter project name',
      create: 'Create',
      cancel: 'Cancel',
      nameRequired: 'Project name is required',
      nameInvalid: 'Project name cannot contain special characters',
      errorExists: 'A project with this name already exists',
      errorGeneric: 'Failed to create project. Please try again later',
      errorNetwork: 'Network request failed. Please check backend service'
    },
    sidebar: {
      imageList: 'Image list',
      noImages: 'No images yet',
      noFilteredImages: 'No images match current filter',
      expandImageList: 'Expand image list',
      collapseImageList: 'Collapse image list',
      selectAll: 'Select all'
    },
    importImagesModal: {
      title: 'Import images',
      description: 'Please choose how to import images',
      byDirectory: 'Import from folder',
      byFiles: 'Import from files',
      cancel: 'Cancel',
      modeLabel: 'Import mode',
      modeCopy: 'Copy into project directory (for general use and smaller datasets)',
      modeLink: 'Create hard links on the same drive (recommended for large datasets: much faster and saves disk space)',
      modeExternal: 'Index external images only (no copy, relies on original folder staying unchanged)',
      modeHint: 'For large datasets on the same drive, the hard link mode is strongly recommended. When the source is on another drive, it will automatically fall back to copy.',
      modeExternalWarning: 'When indexing external images only, please ensure the original image folder will not be moved, renamed, or deleted, otherwise the index will break.',
      errorRootDirectory: 'Cannot select a drive root',
      errorDirectoryInvalid: 'Please select a valid folder',
      errorNoImages: 'No importable images were found',
      errorImportModeInvalid: 'Import mode is invalid. Please choose a valid mode and try again',
      errorGeneric: 'Failed to import images. Please try again later',
      errorNetwork: 'Import request failed. Please check backend service'
    },
    importProgress: {
      scanning: 'Scanning directory, please wait…',
      copying: 'Importing images {imported}/{total} ({progress}%)',
      indexing: 'Building index {imported}/{total} ({progress}%)',
      completed: 'Successfully imported {count} images',
      failed: 'Image import failed. Please try again later'
    },
    deleteProgress: {
      deleting: 'Deleting images ({progress}%)',
      completed: 'Successfully deleted {count} images'
    },
    ioTaskBusy: 'Another task is in progress. Please try again later',
    importDataset: {
      title: 'Import Dataset',
      selectDesc: 'Select the dataset directory. The system will automatically detect the format.',
      pathPlaceholder: 'Select dataset directory...',
      browse: 'Browse...',
      detecting: 'Detecting dataset format...',
      configureDesc: 'Dataset format detected. Confirm to start import.',
      noPluginDetected: 'No supported dataset format detected. Please verify the directory or install the required plugin.',
      cancel: 'Cancel',
      import: 'Start Import',
      importing: 'Importing dataset...',
      success: 'Dataset imported successfully',
      successWithStats: 'Import completed: {images} images, {categories} categories, {annotations} annotations',
      errorGeneric: 'Operation failed. Please try again later',
      errorNetwork: 'Network request failed',
      errorImport: 'Import failed. Please check the dataset format'
    },
    images: {
      contextMenu: {
        deleteSingle: 'Delete image',
        deleteMultiple: 'Delete {count} selected images'
      },
      deleteConfirmTitle: 'Delete images',
      deleteConfirmMessage: 'Are you sure you want to delete {count} selected images? This action cannot be undone.',
      deleteConfirmButton: 'Delete',
      deleteCancelButton: 'Cancel',
      deleteSuccess: 'Deleted {count} images',
      deleteExternalNote: 'Some images are external; only indexes were removed and original files were kept.',
      deleteError: 'Failed to delete images. Please try again later',
      filters: {
        all: 'All',
        annotated: 'Annotated',
        unannotated: 'Unannotated',
        negative: 'Negative samples'
      },
      badge: {
        annotated: 'Done',
        none: 'Todo',
        negative: 'Neg'
      }
    },
    contextMenu: {
      rename: 'Rename',
      delete: 'Delete project'
    },
    deleteModal: {
      title: 'Delete Project',
      warning: '⚠️ This action cannot be undone! All dataset versions and image resources under this project will be deleted.',
      message: 'Are you sure you want to delete project "{name}"?',
      confirm: 'Delete',
      cancel: 'Cancel',
      success: 'Project deleted',
      error: 'Failed to delete: {msg}'
    },
    renameModal: {
      title: 'Rename Project',
      nameLabel: 'New name',
      namePlaceholder: 'Enter new project name',
      confirm: 'Confirm',
      cancel: 'Cancel',
      success: 'Renamed successfully'
    },
    categoryPanel: {
      title: 'Category management',
      empty: 'No categories yet',
      comingSoon: 'Not Available',
      comingSoonDesc: 'Classification annotation is under development. Stay tuned!',
      tabs: {
        bbox: 'Bounding box',
        keypoint: 'Keypoints',
        polygon: 'Polygon',
        category: 'Category'
      },
	  addCategory: 'Add category',
	  namePlaceholder: 'Enter category name',
	  confirm: 'Create',
	  createSuccess: 'Category created successfully',
	  createErrorNameRequired: 'Please enter a category name first',
	  createErrorGeneric: 'Failed to create category. Please try again later',
	  createErrorNetwork: 'Create category request failed. Please check backend service',
	  selectBboxToBind: 'Select bbox category to bind',
	  keypointMustBindBbox: 'Keypoint category must bind to a bbox category',
	  contextMenu: {
	    configureKeypoints: 'Configure keypoints',
	    editCategory: 'Edit category',
	    deleteCategory: 'Delete category'
	  },
	  keypointBadge: '{count} keypoints',
	  keypointBadgeEmpty: 'Right-click to set keypoints',
	  keypointConfig: {
	    title: 'Configure keypoints',
	    titleWithName: 'Configure keypoints for "{name}"',
	    description: 'Pre-define the number and meaning of keypoints for this category',
	    namePlaceholder: 'Keypoint name',
	    addKeypoint: 'Add keypoint',
	    bindBbox: 'Bind to bbox category',
	    noBind: 'No binding',
	    save: 'Save',
	    cancel: 'Cancel',
	    saveSuccess: 'Keypoint configuration saved',
	    errorEmpty: 'At least one valid keypoint is required',
	    errorTooMany: 'Cannot exceed 64 keypoints',
	    errorUnsupported: 'This project does not support keypoint configuration. Please create a new project',
	    errorGeneric: 'Failed to save keypoint configuration. Please try again later',
	    errorNetwork: 'Save request failed. Please check backend service'
	  },
	  deleteCategory: {
	    title: 'Delete category',
	    message: 'Are you sure you want to delete category "{name}"? This action cannot be undone.',
	    warningMessage: '⚠️ This will also delete all annotations of this category!',
	    confirm: 'Delete',
	    cancel: 'Cancel',
	    success: 'Category deleted',
	    errorNotFound: 'Category does not exist or has already been deleted',
	    errorGeneric: 'Failed to delete category. Please try again later',
	    errorNetwork: 'Delete request failed. Please check backend service'
	  },
	  editCategory: {
	    title: 'Edit category',
	    nameLabel: 'Category name',
	    namePlaceholder: 'Enter a new category name',
	    colorLabel: 'Category color',
	    cancel: 'Cancel',
	    save: 'Save',
	    success: 'Category updated',
	    errorNameRequired: 'Category name cannot be empty',
	    errorExists: 'A category with this name already exists. Please choose another name',
	    errorGeneric: 'Failed to update category. Please try again later',
	    errorNetwork: 'Update category request failed. Please check backend service'
	  },
	  mergeCategory: {
	    title: 'Merge Category',
	    message: 'A category with the same name and type "{name}" already exists. This will merge all annotations from the current category into the target category.',
	    confirm: 'Confirm Merge',
	    cancel: 'Cancel',
	    success: 'Categories merged'
	  },
	  sort: {
	    errorGeneric: 'Failed to save category order. Please try again later',
	    errorNetwork: 'Sort save request failed. Please check backend service'
	  }
    }
  },
  footer: {
    totalImages: 'Total',
    annotated: 'Annotated',
    unannotated: 'Unannotated',
    negative: 'Negative',
    imageStatus: 'Status',
    annotationCount: 'Annotations',
    autoSave: 'Auto Save',
    status: {
      none: 'Unannotated',
      annotated: 'Annotated',
      negative: 'Negative'
    }
  },
  annotation: {
    toolbar: {
      saveAsNegative: 'Save as Negative',
      save: 'Save',
      prev: 'Previous',
      next: 'Next',
      zoomIn: 'Zoom In',
      zoomOut: 'Zoom Out',
      reset: 'Reset View',
      fullscreen: 'Fullscreen'
    },
    contextMenu: {
      delete: 'Delete Annotation',
      deletePoint: 'Delete Point',
      setInvisible: 'Set as Invisible',
      setVisible: 'Set as Visible',
      setNotExist: 'Set as Not Exist'
    },
    tips: {
      selectCategory: 'Please select a category first',
      keypointNeedBbox: 'Keypoint annotation must be inside a bounding box',
      keypointSelectBboxFirst: 'Please select a bounding box first before annotating keypoints',
      keypointNeedConfig: 'Please configure keypoints for this category first',
      keypointExists: 'A keypoint annotation of this category already exists in this bounding box',
      polygonMinPoints: 'Polygon requires at least 3 points'
    },
    save: {
      success: 'Annotations saved',
      error: 'Failed to save annotations. Please try again'
    },
    saveAsNegative: {
      success: 'Saved as negative sample',
      error: 'Failed to save. Please try again'
    }
  },
  settings: {
    general: 'General',
    paths: 'Paths',
    language: 'Language',
    theme: 'Theme',
    sections: {
      language: {
        title: 'Language',
        description: 'Select the display language for the EasyMark interface.'
      },
      theme: {
        title: 'Theme',
        description: 'Choose the overall appearance theme of the application.'
      },
      paths: {
        title: 'Paths',
        description: 'Configure project data and output paths. The following values are example defaults.',
        dataPathLabel: 'Project data path',
        datasetExportLabel: 'Dataset export path',
        modelOutputLabel: 'Model training output path',
        currentPath: 'Current path',
        browse: 'Browse...'
      }
    },
    shortcuts: {
      title: 'Shortcuts',
      sectionTitle: 'Annotation Shortcuts',
      description: 'Click on a shortcut button and press a new key combination to change it.',
      pressKey: 'Press key...',
      reset: 'Reset to default',
      resetAll: 'Reset All Shortcuts',
      actions: {
        save: 'Save Annotations',
        saveAsNegative: 'Save as Negative Sample',
        prevImage: 'Previous Image',
        nextImage: 'Next Image',
        prevUnannotated: 'Previous Unannotated',
        nextUnannotated: 'Next Unannotated',
        resetView: 'Reset View',
        deleteSelected: 'Delete Selected',
        toggleKeypointVisibility: 'Toggle Keypoint Visibility'
      }
    }
  },
  shortcut: {
    noMoreUnannotated: 'No more unannotated images in this direction'
  },
  plugins: {
    title: 'Plugin Manager',
    install: 'Install Plugin',
    loading: 'Loading...',
    empty: 'No plugins installed',
    emptyHint: 'Click "Install Plugin" button above to import a plugin package',
    dragHint: 'Or drag and drop a plugin archive here',
    dropToInstall: 'Drop to install plugin',
    uninstall: 'Uninstall',
    // Sidebar
    installFromDisk: 'Install from disk',
    searchPlaceholder: 'Search plugins...',
    installed: 'Installed',
    market: 'Plugin Market',
    marketComingSoon: 'Coming soon',
    noResults: 'No results',
    selectHint: 'Select a plugin from the left to view details',
    author: 'Author',
    unknownAuthor: 'Unknown author',
    installedAt: 'Installed at',
    size: 'Size',
    readme: 'README',
    noReadme: 'No description available',
    uninstallAndDelete: 'Uninstall and delete',
    installTitle: 'Install Plugin',
    installDesc: 'Select a plugin archive file (.zip or .rar format)',
    selectFile: 'Select plugin file...',
    selectFileTitle: 'Select Plugin Archive',
    browse: 'Browse...',
    cancel: 'Cancel',
    installing: 'Installing...',
    uninstallTitle: 'Uninstall Plugin',
    uninstallConfirm: 'Are you sure you want to uninstall "{name}"? This action cannot be undone.',
    uninstallingWithVenv: 'Uninstalling plugin {name} and its virtual environment…',
    uninstallWithVenvSuccess: 'Plugin {name} and its virtual environment have been removed',
    uninstallFailed: 'Failed to uninstall plugin',
    types: {
      dataset: 'Dataset Plugin',
      'import-dataset': 'Dataset Import',
      'export-dataset': 'Dataset Export',
      training: 'Model Training',
      inference: 'Model Inference',
      default: 'General Plugin'
    },
    installError: {
      file_path_required: 'Please select a plugin file',
      unzip_failed: 'Failed to extract plugin',
      unrar_failed: 'RAR extraction failed. Please ensure unrar or 7z is installed',
      unsupported_format: 'Unsupported file format',
      manifest_not_found: 'manifest.json not found in plugin package',
      invalid_manifest: 'Invalid manifest.json format',
      manifest_missing_fields: 'Plugin manifest missing required fields',
      invalid_plugin_id: 'Invalid plugin ID',
      install_failed: 'Installation failed',
      network: 'Network request failed',
      unknown: 'Unknown error'
    },
    // Dependencies
    missingDepsHint: 'Missing {count} dependencies, click to install',
    installingDeps: 'Installing dependencies for {name}...',
    installDepsSuccess: 'Dependencies installed successfully',
    installDepsFailed: 'Failed to install dependencies',
    terminal: 'Terminal Output'
  },
  pythonEnv: {
    title: 'Python Environment',
    plugins: 'Python-based Plugins',
    noPlugins: 'No Python-based plugins',
    selectPluginHint: 'Select a plugin from the left to view details',
    // Python version
    notInstalled: 'Not Installed',
    downloadPython: 'Download Python',
    downloadPythonTitle: 'Download and Install Python',
    downloadPythonDesc: 'Python is not installed on your system. Please download and install Python 3.11 or higher.',
    importantTips: 'Important Tips',
    tip1: 'Make sure to check "Add Python to PATH" during installation',
    tip2: 'Restart this program after installation',
    openDownload: 'Open Download Page',
    // Settings
    settings: 'Settings',
    settingsTitle: 'Python Environment Settings',
    pipSource: 'Default Package Source',
    pipSourceDefault: 'Official (pypi.org)',
    pipSourceTsinghua: 'Tsinghua Mirror',
    pipSourceAliyun: 'Aliyun Mirror',
    pipSourceDouban: 'Douban Mirror',
    httpProxy: 'HTTP Proxy',
    socks5Proxy: 'SOCKS5 Proxy',
    // Virtual environment
    venvStatus: 'Virtual Environment Status',
    venvPath: 'Path',
    createVenv: 'Create Virtual Environment',
    deleteVenv: 'Delete Virtual Environment',
    selectExternalVenv: 'Select Virtual Environment',
    removeExternalVenv: 'Remove Virtual Environment',
    invalidVenvPath: 'Invalid virtual environment path. Please select a valid venv directory',
    bindVenvFailed: 'Failed to bind virtual environment',
    unbindVenvFailed: 'Failed to remove virtual environment',
    noVenv: 'No virtual environment',
    creatingVenv: 'Creating virtual environment for {name}...',
    venvCreated: 'Virtual environment created successfully',
    venvCreateFailed: 'Failed to create virtual environment',
    deletingVenv: 'Deleting virtual environment for {name}...',
    venvDeleted: 'Virtual environment deleted',
    venvDeleteFailed: 'Failed to delete virtual environment',
    // Dependencies
    dependencies: 'Dependencies',
    noDeps: 'This plugin has no Python dependencies',
    installed: 'Installed',
    installDeps: 'Install Dependencies',
    stopInstall: 'Stop Install',
    stoppingInstall: 'Stopping installation...',
    installStopped: 'Installation stopped',
    installingDeps: 'Installing dependencies for {name}...',
    depsInstalled: 'Dependencies installed successfully',
    depsInstallFailed: 'Failed to install dependencies',
    depsReady: 'Dependencies ready',
    depsMissing: 'Missing {count} dependencies',
    terminal: 'Terminal Output',
    // PyTorch
    gpuStatus: 'GPU Status',
    noGpu: 'No NVIDIA GPU detected',
    pytorchReady: 'PyTorch is installed',
    pytorchHint: 'Select a PyTorch version to install, or use a custom version below',
    recommended: '(Recommended)',
    noGpuAvailable: 'No GPU available',
    installingPytorch: 'Installing PyTorch {version} version...',
    pytorchInstalled: 'PyTorch installed successfully',
    pytorchInstallFailed: 'Failed to install PyTorch',
    pytorchSwitchHint: 'Switch to another PyTorch version (will override existing)',
    pytorchRecommendedGpu: 'Recommended GPU: {version}',
    pytorchRecommendedCpu: 'Recommended CPU: {version}',
    customPytorchLabel: 'Custom PyTorch packages (optional)',
    customPytorchPlaceholder: 'e.g. torch==2.2.2+cu118 torchvision==0.17.2+cu118',
    // Dependency uninstall
    uninstallDep: 'Uninstall',
    uninstallingDep: 'Uninstalling {name}...',
    depUninstalled: '{name} uninstalled',
    depUninstallFailed: 'Failed to uninstall {name}',
    // GPU compatibility
    gpuNotCompatible: 'CUDA version does not meet minimum requirement (CUDA >= {minCuda}), will install CPU version',
    switchToGpu: 'Switch to GPU',
    switchToCpu: 'Switch to CPU',
    // Custom command
    customCommandPlaceholder: 'Enter pip or other command, e.g. pip list',
    commandFailed: 'Command execution failed'
  },
  training: {
    title: 'Training Manager',
    // Top bar - System resources
    systemResources: {
      cpu: {
        title: 'CPU',
        cores: '{count} logical cores',
        usage: 'Usage'
      },
      gpu: {
        title: 'GPU',
        vram: 'VRAM',
        usage: 'Core Usage',
        notAvailable: 'No GPU detected'
      },
      python: {
        title: 'Python Environment',
        notDeployed: 'Not deployed',
        deployed: 'Deployed',
        deployHint: 'Click to create a virtual environment in the data directory',
        deployedPathHint: 'Virtual environment path: {path}',
        deployButton: 'Deploy',
        manageButton: 'Manage Dependencies'
      },
      plugin: {
        ready: 'Training Plugin Ready',
        notReady: 'Training Plugin Not Ready',
        readyHint: 'You can start creating training tasks',
        notReadyHint: 'Go to Python Environment page to install plugin dependencies',
        setupButton: 'Go to Setup'
      }
    },
    // Manage dependencies dialog
    envDialog: {
      title: 'Manage Python Environment',
      pythonVersion: 'Python Version',
      pythonNotInstalled: 'Python not detected',
      pythonVersionWarning: 'Python 3.8 - 3.12 recommended',
      helpButton: 'Help',
      venvPath: 'Virtual Environment Path',
      venvNotDeployed: 'Not deployed',
      pypiMirror: 'PyPI Mirror',
      enableMirror: 'Enable mirror',
      mirrorPlaceholder: 'Enter mirror URL',
      installPackage: 'Install Package',
      packagePlaceholder: 'Enter package name, e.g. ultralytics>=8.0.0',
      install: 'Install',
      installedPackages: 'Installed Packages',
      noPackages: 'No packages installed',
      deployFirst: 'Please deploy environment first',
      deploying: 'Deploying environment...',
      deployStarted: 'Environment deployment started',
      deploySuccess: 'Environment deployed successfully',
      deployError: 'Failed to deploy environment',
      installing: 'Installing...',
      installStarted: 'Dependency installation started',
      installSuccess: 'Installation successful',
      installError: 'Installation failed',
      uninstall: 'Uninstall',
      uninstallConfirmTitle: 'Confirm Uninstall Package',
      uninstallConfirmText: 'Are you sure you want to uninstall {name}?',
      confirmUninstall: 'Confirm Uninstall',
      undeployButton: 'Uninstall Environment',
      undeployConfirmTitle: 'Confirm Uninstall Environment',
      undeployConfirmText: 'This will delete the virtual environment directory and all installed packages. This action cannot be undone. Are you sure you want to continue?',
      undeployStarted: 'Environment uninstall started',
      confirmUndeploy: 'Confirm Uninstall',
      close: 'Close'
    },
    // Trainset management
    trainset: {
      createButton: 'Create Trainset',
      title: 'Trainset Management',
      selectCategories: 'Select Categories',
      noProjects: 'No projects',
      selected: '{count} categories selected',
      trainsetInfo: 'Trainset Info',
      trainsetName: 'Trainset Name',
      namePlaceholder: 'Enter trainset name',
      categoriesPreview: 'Selected Categories Preview',
      noCategoriesSelected: 'Please select categories from the left',
      save: 'Save Trainset',
      update: 'Update Trainset',
      cancelEdit: 'Cancel Edit',
      savedTrainsets: 'Saved Trainsets',
      noSavedTrainsets: 'No saved trainsets',
      categoriesCount: 'categories',
      edit: 'Edit',
      delete: 'Delete',
      close: 'Close',
      conflictWarning: 'Cannot select same category from different versions of the same project',
      typeRectangle: 'Rectangle',
      typePolygon: 'Polygon',
      typeKeypoint: 'Keypoint',
      images: 'images',
      annotations: 'annotations',
      saveSuccess: 'Trainset saved successfully',
      updateSuccess: 'Trainset updated successfully',
      saveError: 'Failed to save trainset',
      deleteConfirmTitle: 'Confirm Delete Trainset',
      deleteConfirmText: 'Are you sure you want to delete trainset "{name}"? This action cannot be undone.',
      confirmDelete: 'Confirm Delete',
      deleteSuccess: 'Trainset deleted successfully',
      deleteError: 'Failed to delete trainset'
    },
    // Training tasks
    tasks: {
      title: 'Training Tasks',
      newTask: 'New Training Task',
      empty: 'No training tasks',
      emptyHint: 'Click the button above to create a new training task',
      status: {
        running: 'Running',
        failed: 'Failed',
        success: 'Success',
        pending: 'Pending',
        error: 'Error',
        completed: 'Completed'
      },
      epoch: 'Epoch',
      progress: 'Progress',
      menu: {
        stop: 'Stop Task',
        delete: 'Delete Task',
        retrain: 'Retrain'
      },
      metrics: {
        trainLoss: 'Train Loss',
        valLoss: 'Val Loss',
        boxLoss: 'Box Loss',
        clsLoss: 'Cls Loss',
        mAP50: 'mAP50',
        mAP5095: 'mAP50-95'
      },
      completedNotify: 'Training task {name} completed',
      stopSuccess: 'Task stopped',
      stopFailed: 'Failed to stop task',
      deleteSuccess: 'Task deleted',
      deleteFailed: 'Failed to delete task',
      retrainHint: 'Please configure training parameters in the new task dialog'
    },
    // Log window
    logs: {
      title: 'Log Output',
      clear: 'Clear',
      copy: 'Copy',
      copied: 'Logs copied to clipboard',
      copyFailed: 'Copy failed',
      autoScroll: 'Auto Scroll',
      empty: 'No log output'
    },
    // Training history
    history: {
      title: 'Training History',
      empty: 'No training records',
      emptyHint: 'Completed trainings will appear here',
      modelPath: 'Model Path',
      openFolder: 'Open Folder',
      delete: 'Delete',
      deleteSuccess: 'Training record deleted',
      deleteFailed: 'Failed to delete training record',
      duration: 'Duration',
      completedAt: 'Completed'
    },
    // New training task dialog
    newTaskDialog: {
      title: 'New Training Task',
      taskName: 'Task Name',
      taskNamePlaceholder: 'Enter training task name',
      datasetSource: 'Data Source',
      fromTrainset: 'From configured trainset',
      fromDirectory: 'From directory',
      selectTrainset: 'Select Trainset',
      selectTrainsetPlaceholder: 'Select a trainset',
      datasetPath: 'Dataset Path',
      pathPlaceholder: 'Select dataset directory...',
      browse: 'Browse',
      detecting: 'Detecting format...',
      detectedFormat: 'Detected Format',
      formatUnknown: 'Unknown format',
      trainValSplit: 'Data Split',
      trainType: 'Training Type',
      trainTypes: {
        detection: 'Object Detection',
        pose: 'Pose Estimation',
        segmentation: 'Instance Segmentation',
        custom: 'Custom'
      },
      trainPlugin: 'Training Plugin',
      loadingPlugins: 'Loading plugins...',
      selectPlugin: 'Select a plugin',
      noPluginsForType: 'No plugins available for this type',
      dependencies: 'Dependencies',
      checkingDeps: 'Checking dependencies...',
      envNotDeployed: 'Python environment not deployed',
      depsMissing: '{count} dependencies missing',
      depsOk: 'All dependencies satisfied',
      installDeps: 'Install',
      installingDeps: 'Installing dependencies...',
      installDepsFailed: 'Failed to install dependencies',
      model: 'Model',
      selectModel: 'Select a model',
      downloadModel: 'Download',
      downloadingModel: 'Downloading model {model}',
      modelDownloaded: 'Model {model} downloaded',
      downloadModelFailed: 'Failed to download model',
      cancelDownload: 'Cancel',
      downloadCancelled: 'Download cancelled',
      openModelsDir: 'Open models folder',
      refreshModelStatus: 'Refresh status',
      installDepsComplete: 'Dependencies installed',
      hyperparams: 'Hyperparameters',
      epochs: 'Epochs',
      batchSize: 'Batch Size',
      imageSize: 'Image Size',
      advancedParams: 'Advanced Parameters',
      cancel: 'Cancel',
      start: 'Start Training',
      starting: 'Starting...',
      trainingStarted: 'Training task {name} started',
      startFailed: 'Failed to start training',
      validationError: 'Please fill in the required information',
      torchCpuOnly: 'PyTorch CPU-only (GPU version required)',
      reinstallTorchCuda: 'Install GPU Version',
      reinstallingTorch: 'Reinstalling PyTorch (GPU version)...',
      torchCudaInstalled: 'PyTorch GPU version installed',
      noTorchConfig: 'Plugin has no PyTorch CUDA configuration'
    },
    errors: {
      pluginPathMissing: 'Plugin path is missing, please reinstall the plugin'
    }
  },
  dataset: {
    title: 'Dataset Management',
    loading: 'Loading...',
    openExportDir: 'Open Export Directory',
    empty: {
      title: 'No Dataset Versions',
      desc: 'Click the "Sync" button next to a project to create the first version'
    },
    tree: {
      projects: 'Projects',
      noProjects: 'No projects',
      versions: 'Versions',
      categories: 'Categories'
    },
    sync: {
      button: 'Sync',
      title: 'Create Dataset Version',
      desc: 'Save all images, categories, and annotations as a read-only version',
      noteLabel: 'Version Note',
      notePlaceholder: 'Optional, describe this version',
      confirm: 'Create Version',
      cancel: 'Cancel',
      creating: 'Creating...',
      success: 'Successfully created version v{version}',
      error: 'Failed to create version'
    },
    version: {
      info: 'Version Info',
      createdAt: 'Created',
      note: 'Note',
      noNote: 'No note',
      editNote: 'Edit Note',
      stats: 'Statistics',
      images: 'Images',
      categories: 'Categories',
      annotations: 'Annotations',
      rollback: 'Rollback to this version',
      delete: 'Delete Version'
    },
    health: {
      title: 'Data Health',
      progress: 'Annotation Progress',
      progressDesc: '{count} images fully annotated',
      avgAnnotations: 'Avg Ann/Img',
      checkEmpty: 'No Empty Images',
      checkBalance: 'Slight Imbalance'
    },
    rollback: {
      title: 'Rollback to v{version}',
      warning: 'This will restore the project data to this version state.',
      affected: 'Affected Project',
      confirmLabel: 'I understand the risks and confirm the rollback',
      confirm: 'Rollback',
      cancel: 'Cancel',
      success: 'Rolled back to v{version}',
      error: 'Rollback failed'
    },
    deleteVersion: {
      title: 'Delete Version',
      confirm: 'Are you sure you want to delete v{version}? This cannot be undone.',
      success: 'Version deleted',
      error: 'Delete failed'
    },
    export: {
      button: 'Export Selected...',
      selected: '{count} categories selected',
      selectedFrom: 'from {projects} projects / {versions} versions',
      conflictWarning: 'Cannot select same-name categories from different versions in the same project'
    },
    exportWizard: {
      title: 'Export Dataset',
      categories: '{count} categories',
      images: '~{count} images',
      annotations: '~{count} annotations',
      outputPath: 'Output Path',
      pathPlaceholder: 'Select export directory',
      format: 'Format',
      customFormat: 'Custom Format',
      plugin: 'Export Plugin',
      noPlugin: 'No plugin supports this format',
      split: 'Dataset Split',
      train: 'Train',
      val: 'Validation',
      test: 'Test',
      splitWarning: 'Split must sum to 100% (current: {total}%)',
      confirm: 'Start Export',
      exporting: 'Exporting...',
      success: 'Export completed! Path: {path}',
      error: 'Export failed: {msg}'
    },
    exportHistory: {
      title: 'Export History',
      openFolder: 'Open Folder'
    },
    project: {
      info: 'Project Info',
      noVersions: 'This project has no dataset versions yet',
      noVersionsHint: 'Click the "Sync" button to create the first version',
      stats: 'Current Status',
      lastSync: 'Last Sync',
      never: 'Never synced'
    },
    category: {
      type: {
        bbox: 'Bounding Box',
        keypoint: 'Keypoint',
        polygon: 'Polygon',
        mask: 'Segmentation Mask'
      }
    }
  },
  inference: {
    title: 'Model Inference Assistant',
    confidenceThreshold: 'Confidence Threshold',
    nmsThreshold: 'NMS Threshold',
    trainedModels: 'Trained Models',
    importedModels: 'Imported Models',
    noTrainedModels: 'No trained models yet',
    noImportedModels: 'No imported models yet',
    dropModelHint: 'Drag and drop model files here, or click the button in the top right to open the folder and add manually (filename must be unique)',
    openFolder: 'Open Model Folder',
    importSuccess: 'Model imported successfully',
    importFailed: 'Failed to import model',
    fileExists: 'A model with this name already exists',
    selectPolygonCategory: 'Please select a polygon or bounding box category first',
    downloading: '{name} downloading {progress}%',
    downloadComplete: '{name} download complete',
    downloadFailed: '{name} download failed: {error}',
    pluginDepsNotReady: '{name} dependencies not ready, please install in Python Environment first',
    trainingPluginNotReady: 'Training plugin dependencies not ready, please install in Python Environment first'
  },
  common: {
    save: 'Save',
    cancel: 'Cancel',
    minimize: 'Minimize',
    close: 'Close'
  },
  help: {
    title: 'Help & Docs',
    description: 'Welcome to EasyMark! An efficient computer vision annotation tool.',
    contactCard: {
      title: 'Contact Us',
      desc: 'Feel free to reach out with questions or suggestions'
    },
    nav: {
      overview: 'Overview',
      project: 'Project',
      dataset: 'Dataset',
      training: 'Training',
      inference: 'Inference'
    },
    sections: {
      project: {
        title: '📁 Project Management',
        items: [
          'Create Project: Click the "+" button in sidebar to create a new project',
          'Import Images: Right-click project and select "Import Images", supports drag & drop',
          'Category Management: Add bbox, keypoint, or polygon categories in the right panel',
          'Annotation: Select a category and draw on images, right-click to edit',
          'Image Filter: Filter by all/annotated/unannotated/negative samples'
        ]
      },
      shortcuts: {
        title: '⌨️ Shortcuts',
        items: [
          'Ctrl+S: Save current image annotations',
          'Ctrl+Shift+S: Save as negative sample (no annotations)',
          '← / →: Switch to previous/next image',
          'Ctrl+← / Ctrl+→: Jump to previous/next unannotated image',
          'Backspace: Delete selected annotation',
          'V: Toggle keypoint visibility',
          'Ctrl+0: Reset canvas view'
        ]
      },
      dataset: {
        title: '📊 Dataset Management',
        items: [
          'Sync Version: Click the sync button to create a snapshot of current annotations',
          'Version Control: View history, rollback, or delete dataset versions',
          'Export Data: Select categories and export in YOLO/COCO or other formats',
          'Data Split: Configure train/val/test ratios during export'
        ]
      },
      training: {
        title: '🧠 Model Training',
        items: [
          'Environment Setup: Deploy Python environment on first use',
          'Create Task: Select dataset version and model, then configure training parameters',
          'Task Monitoring: View real-time logs, loss curves, and model metrics',
          'Model Inference: Use trained models for inference on new images'
        ]
      },
      plugins: {
        title: '🧩 Plugin System',
        items: [
          'Install Plugin: Drag and drop plugin archive to the plugins page',
          'Manage Plugins: Right-click installed plugins to uninstall',
          'Supported Types: Dataset import plugins, training framework plugins, etc.'
        ]
      },
      settings: {
        title: '⚙️ System Settings',
        items: [
          'Data Directory: Set storage path for projects and datasets',
          'Theme: Switch between dark and light themes',
          'Language: Switch between Chinese and English'
        ]
      }
    }
  }
}

export default enUS
