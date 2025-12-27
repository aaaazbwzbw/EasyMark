const zhCN = {
  app: {
    title: 'EasyMark'
  },
  header: {
    settings: '设置'
  },
  home: {
    welcome: '欢迎使用 EasyMark。'
  },
  notifications: {
    panelTitle: '通知',
    clear: '清除通知',
    collapse: '收起',
    success: '操作成功！',
    info: '这是一条提示信息',
    warning: '请注意检查输入内容',
    error: '操作失败，请重试'
  },
  sidebar: {
    project: '项目管理器',
    dataset: '数据集管理',
    training: '训练管理',
    plugins: '插件市场',
    pythonEnv: 'Python 环境',
    ui: 'UI 测试',
    help: '帮助文档',
    newVersionAvailable: '发现新版本 v{version}，点击前往下载'
  },
  project: {
    status: {
      none: '未打开项目'
    },
    actions: {
      new: '新建项目',
      importDataset: '导入数据集',
      importImages: '导入图片'
    },
    createModal: {
      title: '新建项目',
      nameLabel: '项目名称',
      namePlaceholder: '请输入项目名称',
      create: '创建',
      cancel: '取消',
      nameRequired: '项目名称不能为空',
      nameInvalid: '项目名称不能包含特殊字符',
      errorExists: '已存在同名项目',
      errorGeneric: '创建项目失败，请稍后重试',
      errorNetwork: '网络请求失败，请检查后端服务状态'
    },
    sidebar: {
      imageList: '图片列表',
      noImages: '暂无图片',
      noFilteredImages: '当前筛选条件下无图片',
      expandImageList: '展开图片列表',
      collapseImageList: '折叠图片列表',
      selectAll: '全选'
    },
    importImagesModal: {
      title: '导入图片',
      description: '请选择导入方式',
      byDirectory: '通过文件夹导入',
      byFiles: '选择文件导入',
      cancel: '取消',
      modeLabel: '导入图片模式',
      modeCopy: '复制到项目目录（适用于通用和小型数据集）',
      modeLink: '在同一磁盘上创建硬链接（推荐：大型数据集，速度更快且节省磁盘空间）',
      modeExternal: '仅索引外部图片（不复制，依赖原始目录保持不变）',
      modeHint: '同一磁盘上的大型数据集强烈推荐使用“硬链接”模式；跨磁盘时会自动退回为复制导入。',
      modeExternalWarning: '仅索引外部图片时，请确保原始图片目录不会被移动、重命名或删除，否则索引会失效。',
      errorRootDirectory: '不能选择磁盘根目录',
      errorDirectoryInvalid: '请选择有效的文件夹',
      errorNoImages: '未找到可导入的图片',
      errorImportModeInvalid: '导入模式无效，请重新选择导入方式',
      errorGeneric: '导入图片失败，请稍后重试',
      errorNetwork: '导入请求失败，请检查后端服务状态'
    },
    importProgress: {
      scanning: '正在扫描目录，请稍候…',
      copying: '正在导入图片 {imported}/{total}（{progress}%）',
      indexing: '正在构建索引 {imported}/{total}（{progress}%）',
      completed: '成功导入 {count} 张图片',
      failed: '导入失败，请稍后重试'
    },
    deleteProgress: {
      deleting: '正在删除图片（{progress}%）',
      completed: '成功删除 {count} 张图片'
    },
    ioTaskBusy: '有其他任务正在进行中，请稍后再试',
    importDataset: {
      title: '导入数据集',
      selectDesc: '请选择数据集所在的目录，系统将自动识别数据集格式。',
      pathPlaceholder: '选择数据集目录...',
      browse: '浏览...',
      detecting: '正在识别数据集格式...',
      configureDesc: '已识别数据集格式，请确认后开始导入。',
      noPluginDetected: '未识别到支持的数据集格式，请确认目录正确或安装相应插件。',
      cancel: '取消',
      import: '开始导入',
      importing: '正在导入数据集...',
      success: '数据集导入成功',
      successWithStats: '导入完成：{images} 张图片，{categories} 个类别，{annotations} 个标注',
      errorGeneric: '操作失败，请稍后重试',
      errorNetwork: '网络请求失败',
      errorImport: '导入失败，请检查数据集格式'
    },
    images: {
      contextMenu: {
        deleteSingle: '删除图片',
        deleteMultiple: '删除选中的 {count} 张图片'
      },
      deleteConfirmTitle: '删除图片',
      deleteConfirmMessage: '确定要删除选中的 {count} 张图片吗？此操作不可撤销。',
      deleteConfirmButton: '删除',
      deleteCancelButton: '取消',
      deleteSuccess: '已删除 {count} 张图片',
      deleteExternalNote: '部分图片为外部索引，仅删除了索引，未删除原始文件。',
      deleteError: '删除图片失败，请稍后重试',
      filters: {
        all: '全部',
        annotated: '已标注',
        unannotated: '未标注',
        negative: '负样本'
      },
      badge: {
        annotated: '已标注',
        none: '未标注',
        negative: '负样本'
      }
    },
    contextMenu: {
      rename: '重命名',
      delete: '删除项目'
    },
    deleteModal: {
      title: '删除项目',
      warning: '⚠️ 此操作不可撤销！将同时删除该项目下的所有数据集版本和图片资源。',
      message: '确定要删除项目 "{name}" 吗？',
      confirm: '确认删除',
      cancel: '取消',
      success: '项目已删除',
      error: '删除失败：{msg}'
    },
    renameModal: {
      title: '重命名项目',
      nameLabel: '新名称',
      namePlaceholder: '请输入新的项目名称',
      confirm: '确认',
      cancel: '取消',
      success: '重命名成功'
    },
    categoryPanel: {
      title: '类别管理',
      empty: '暂无类别',
      comingSoon: '暂不支持',
      comingSoonDesc: '分类标注功能正在开发中，敬请期待',
      tabs: {
        bbox: '矩形框',
        keypoint: '关键点',
        polygon: '多边形',
        category: '分类'
      },
	  addCategory: '添加类别',
	  namePlaceholder: '请输入类别名称',
	  confirm: '确定',
	  createSuccess: '类别创建成功',
	  createErrorNameRequired: '请先输入类别名称',
	  createErrorGeneric: '创建类别失败，请稍后重试',
	  createErrorNetwork: '创建类别请求失败，请检查后端服务',
	  selectBboxToBind: '选择要绑定的矩形框类别',
	  keypointMustBindBbox: '关键点类别必须绑定一个矩形框类别',
	  contextMenu: {
	    configureKeypoints: '设置关键点',
	    editCategory: '编辑类别',
	    deleteCategory: '删除类别'
	  },
	  keypointBadge: '{count} 关键点',
	  keypointBadgeEmpty: '请右键设置关键点',
	  keypointConfig: {
	    title: '设置关键点',
	    titleWithName: '设置「{name}」的关键点',
	    description: '请预先定义该类别的关键点数量和含义',
	    namePlaceholder: '关键点名称',
	    addKeypoint: '新增关键点',
	    bindBbox: '绑定矩形框类别',
	    noBind: '不绑定',
	    save: '保存',
	    cancel: '取消',
	    saveSuccess: '关键点配置已保存',
	    errorEmpty: '至少需要一个有效的关键点',
	    errorTooMany: '关键点数量不能超过 64 个',
	    errorUnsupported: '当前项目不支持关键点配置，请新建项目后重试',
	    errorGeneric: '保存关键点配置失败，请稍后重试',
	    errorNetwork: '保存请求失败，请检查后端服务'
	  },
	  deleteCategory: {
	    title: '删除类别',
	    message: '确定要删除类别「{name}」吗？此操作不可撤销。',
	    warningMessage: '⚠️ 此操作将同时删除该类别的所有标注！',
	    confirm: '删除',
	    cancel: '取消',
	    success: '类别已删除',
	    errorNotFound: '类别不存在或已被删除',
	    errorGeneric: '删除类别失败，请稍后重试',
	    errorNetwork: '删除请求失败，请检查后端服务'
	  },
	  editCategory: {
	    title: '编辑类别',
	    nameLabel: '类别名称',
	    namePlaceholder: '请输入新的类别名称',
	    colorLabel: '类别颜色',
	    cancel: '取消',
	    save: '保存',
	    success: '类别已更新',
	    errorNameRequired: '类别名称不能为空',
	    errorGeneric: '更新类别失败，请稍后重试',
	    errorNetwork: '更新类别请求失败，请检查后端服务'
	  },
	  mergeCategory: {
	    title: '合并类别',
	    message: '已存在同名同类型的类别「{name}」，此操作将把当前类别的所有标注合并到目标类别中。',
	    confirm: '确定合并',
	    cancel: '取消',
	    success: '类别已合并'
	  },
	  sort: {
	    errorGeneric: '保存类别排序失败，请稍后重试',
	    errorNetwork: '保存类别排序请求失败，请检查后端服务'
	  }
    }
  },
  footer: {
    totalImages: '总图片',
    annotated: '已标注',
    unannotated: '未标注',
    negative: '负样本',
    imageStatus: '状态',
    annotationCount: '标注数',
    autoSave: '自动保存',
    status: {
      none: '未标注',
      annotated: '已标注',
      negative: '负样本'
    }
  },
  annotation: {
    toolbar: {
      saveAsNegative: '保存为负样本',
      save: '保存标注',
      prev: '上一张',
      next: '下一张',
      zoomIn: '放大',
      zoomOut: '缩小',
      reset: '重置视图',
      fullscreen: '全屏'
    },
    contextMenu: {
      delete: '删除标注',
      deletePoint: '删除该点',
      setInvisible: '设为不可见',
      setVisible: '设为可见',
      setNotExist: '设为不存在'
    },
    tips: {
      selectCategory: '请先选择一个类别',
      keypointNeedBbox: '关键点标注需要在矩形框内进行',
      keypointSelectBboxFirst: '请先选中一个矩形框再标注关键点',
      keypointNeedConfig: '请先为该类别配置关键点',
      keypointExists: '该矩形框内已存在相同类别的关键点标注',
      polygonMinPoints: '多边形至少需要3个点'
    },
    save: {
      success: '标注已保存',
      error: '保存标注失败，请稍后重试'
    },
    saveAsNegative: {
      success: '已保存为负样本',
      error: '保存失败，请稍后重试'
    }
  },
  settings: {
    general: '通用设置',
    paths: '路径设置',
    language: '界面语言',
    theme: '主题模式',
    sections: {
      language: {
        title: '界面语言',
        description: '选择 EasyMark 界面显示的语言。'
      },
      theme: {
        title: '主题外观',
        description: '选择应用的整体外观主题。'
      },
      paths: {
        title: '路径设置',
        description: '配置项目相关的数据与输出路径，以下路径为示例默认值。',
        dataPathLabel: '项目数据路径',
        datasetExportLabel: '数据集导出路径',
        modelOutputLabel: '模型训练输出路径',
      }
    },
    shortcuts: {
      title: '快捷键',
      sectionTitle: '标注快捷键',
      description: '点击快捷键按钮后按下新的组合键可修改快捷键。',
      pressKey: '请按键...',
      reset: '重置为默认',
      resetAll: '重置所有快捷键',
      actions: {
        save: '保存标注',
        saveAsNegative: '保存为负样本',
        prevImage: '上一张图片',
        nextImage: '下一张图片',
        prevUnannotated: '上一张未标注',
        nextUnannotated: '下一张未标注',
        resetView: '重置视图',
        deleteSelected: '删除选中',
        toggleKeypointVisibility: '切换关键点可见性'
      }
    }
  },
  shortcut: {
    noMoreUnannotated: '当前方向没有更多未标注的图片了'
  },
  plugins: {
    title: '插件管理',
    install: '安装插件',
    loading: '加载中...',
    empty: '暂无已安装的插件',
    emptyHint: '点击上方"安装插件"按钮导入插件包',
    dragHint: '或直接将插件压缩包拖拽到此处',
    dropToInstall: '释放以安装插件',
    uninstall: '卸载',
    // 左侧栏
    installFromDisk: '从磁盘安装',
    searchPlaceholder: '搜索插件...',
    installed: '已安装',
    market: '插件市场',
    marketComingSoon: '敬请期待',
    noResults: '无匹配结果',
    selectHint: '请从左侧选择一个插件查看详情',
    author: '作者',
    unknownAuthor: '未知作者',
    installedAt: '安装时间',
    size: '插件大小',
    readme: '介绍',
    noReadme: '该插件暂无介绍',
    uninstallAndDelete: '卸载并删除插件',
    installTitle: '安装插件',
    installDesc: '请选择插件压缩包文件（支持 .zip 和 .rar 格式）',
    selectFile: '选择插件文件...',
    selectFileTitle: '选择插件压缩包',
    browse: '浏览...',
    cancel: '取消',
    installing: '安装中...',
    uninstallTitle: '卸载插件',
    uninstallConfirm: '确定要卸载插件 "{name}" 吗？此操作不可撤销。',
    uninstallingWithVenv: '正在卸载插件 {name} 及其虚拟环境…',
    uninstallWithVenvSuccess: '插件 {name} 已卸载，虚拟环境已删除',
    uninstallFailed: '卸载插件失败',
    types: {
      dataset: '数据集插件',
      'import-dataset': '数据集导入',
      'export-dataset': '数据集导出',
      training: '模型训练',
      inference: '模型推理',
      default: '通用插件'
    },
    installError: {
      file_path_required: '请选择插件文件',
      unzip_failed: '解压插件失败',
      unrar_failed: 'RAR 解压失败，请确保系统已安装 unrar 或 7z',
      unsupported_format: '不支持的文件格式',
      manifest_not_found: '插件包中未找到 manifest.json',
      invalid_manifest: 'manifest.json 格式无效',
      manifest_missing_fields: '插件清单缺少必要字段',
      invalid_plugin_id: '插件ID无效',
      install_failed: '安装失败',
      network: '网络请求失败',
      unknown: '未知错误'
    },
    // 依赖相关
    missingDepsHint: '缺失 {count} 个依赖，点击安装',
    installingDeps: '正在为 {name} 安装依赖...',
    installDepsSuccess: '依赖安装成功',
    installDepsFailed: '依赖安装失败',
    terminal: '终端输出'
  },
  pythonEnv: {
    title: 'Python 环境',
    plugins: '基于 Python 的插件',
    noPlugins: '暂无基于 Python 的插件',
    selectPluginHint: '请从左侧选择一个插件查看详情',
    // Python 版本
    notInstalled: '未安装',
    downloadPython: '下载 Python',
    downloadPythonTitle: '下载并安装 Python',
    downloadPythonDesc: '检测到系统未安装 Python，请下载并安装 Python 3.11 或更高版本。',
    importantTips: '重要提示',
    tip1: '安装时请务必勾选「Add Python to PATH」选项',
    tip2: '安装完成后请重启本程序',
    openDownload: '打开下载页',
    // 设置
    settings: '设置',
    settingsTitle: 'Python 环境设置',
    pipSource: '默认下载源',
    pipSourceDefault: '官方源 (pypi.org)',
    pipSourceTsinghua: '清华镜像',
    pipSourceAliyun: '阿里云镜像',
    pipSourceDouban: '豆瓣镜像',
    httpProxy: 'HTTP 代理',
    socks5Proxy: 'SOCKS5 代理',
    // 虚拟环境
    venvStatus: '虚拟环境状态',
    venvPath: '路径',
    createVenv: '创建虚拟环境',
    deleteVenv: '删除虚拟环境',
    selectExternalVenv: '选择虚拟环境',
    removeExternalVenv: '移除虚拟环境使用',
    invalidVenvPath: '虚拟环境路径无效，请选择正确的虚拟环境目录',
    bindVenvFailed: '绑定虚拟环境失败',
    unbindVenvFailed: '移除虚拟环境失败',
    noVenv: '未创建虚拟环境',
    creatingVenv: '正在为 {name} 创建虚拟环境...',
    venvCreated: '虚拟环境创建成功',
    venvCreateFailed: '虚拟环境创建失败',
    deletingVenv: '正在删除 {name} 的虚拟环境...',
    venvDeleted: '虚拟环境已删除',
    venvDeleteFailed: '虚拟环境删除失败',
    // 依赖
    dependencies: '依赖列表',
    noDeps: '该插件无 Python 依赖',
    installed: '已安装',
    installDeps: '安装依赖',
    stopInstall: '停止安装',
    stoppingInstall: '正在停止安装...',
    installStopped: '安装已停止',
    installingDeps: '正在为 {name} 安装依赖...',
    depsInstalled: '依赖安装完成',
    depsInstallFailed: '依赖安装失败',
    depsReady: '依赖已就绪',
    depsMissing: '缺失 {count} 个依赖',
    terminal: '终端输出',
    // PyTorch
    gpuStatus: 'GPU 状态',
    noGpu: '未检测到 NVIDIA GPU',
    pytorchReady: 'PyTorch 已安装',
    pytorchHint: '请选择 PyTorch 版本进行安装，或使用下方自定义版本',
    recommended: '(推荐)',
    noGpuAvailable: '无可用 GPU',
    installingPytorch: '正在安装 PyTorch {version} 版本...',
    pytorchInstalled: 'PyTorch 安装完成',
    pytorchInstallFailed: 'PyTorch 安装失败',
    pytorchSwitchHint: '切换到其他 PyTorch 版本（将覆盖现有版本）',
    pytorchRecommendedGpu: '推荐 GPU 版本：{version}',
    pytorchRecommendedCpu: '推荐 CPU 版本：{version}',
    customPytorchLabel: '自定义 PyTorch 包（可选）',
    customPytorchPlaceholder: '例如：torch==2.2.2+cu118 torchvision==0.17.2+cu118',
    // 依赖卸载
    uninstallDep: '卸载',
    uninstallingDep: '正在卸载 {name}...',
    depUninstalled: '{name} 已卸载',
    depUninstallFailed: '{name} 卸载失败',
    // GPU 兼容性
    gpuNotCompatible: '当前 CUDA 版本不满足最低要求 (CUDA >= {minCuda})，将安装 CPU 版本',
    switchToGpu: '切换到 GPU',
    switchToCpu: '切换到 CPU',
    // 自定义命令
    customCommandPlaceholder: '输入 pip 或其他命令，如：pip list',
    commandFailed: '命令执行失败'
  },
  training: {
    title: '训练管理',
    // 顶部栏 - 系统资源
    systemResources: {
      cpu: {
        title: 'CPU',
        cores: '{count} 个逻辑核心',
        usage: '使用率'
      },
      gpu: {
        title: 'GPU',
        vram: '显存',
        usage: '核心占用',
        notAvailable: '未检测到 GPU'
      },
      python: {
        title: 'Python 环境',
        notDeployed: '环境未部署',
        deployed: '环境已部署',
        deployHint: '点击部署后将在数据目录中创建虚拟环境',
        deployedPathHint: '虚拟环境路径：{path}',
        deployButton: '部署环境',
        manageButton: '管理依赖'
      },
      plugin: {
        ready: '训练插件已就绪',
        notReady: '训练插件未就绪',
        readyHint: '可以开始创建训练任务',
        notReadyHint: '请前往 Python 环境页面安装插件依赖',
        setupButton: '前往配置'
      }
    },
    // 管理依赖对话框
    envDialog: {
      title: '管理 Python 环境',
      pythonVersion: 'Python 版本',
      pythonNotInstalled: '未检测到 Python',
      pythonVersionWarning: '建议使用 Python 3.8 - 3.12 版本',
      helpButton: '帮助',
      venvPath: '虚拟环境路径',
      venvNotDeployed: '未部署环境',
      pypiMirror: 'PyPI 镜像源',
      enableMirror: '启用镜像源',
      mirrorPlaceholder: '输入镜像源地址',
      installPackage: '安装依赖包',
      packagePlaceholder: '输入包名，如 ultralytics>=8.0.0',
      install: '安装',
      installedPackages: '已安装的依赖',
      noPackages: '暂无已安装的依赖',
      deployFirst: '请先部署环境',
      deploying: '正在部署环境...',
      deployStarted: '已开始部署环境',
      deploySuccess: '环境部署成功',
      deployError: '环境部署失败',
      installing: '正在安装...',
      installStarted: '已开始安装依赖',
      installSuccess: '安装成功',
      installError: '安装失败',
      uninstall: '卸载',
      uninstallConfirmTitle: '确认卸载依赖',
      uninstallConfirmText: '确定要卸载 {name} 吗？',
      confirmUninstall: '确认卸载',
      undeployButton: '卸载环境',
      undeployConfirmTitle: '确认卸载环境',
      undeployConfirmText: '卸载环境将删除虚拟环境目录及其内部所有已安装的依赖包。此操作不可恢复，确定要继续吗？',
      undeployStarted: '已开始卸载环境',
      confirmUndeploy: '确认卸载',
      close: '关闭'
    },
    // 训练集管理
    trainset: {
      createButton: '创建训练集',
      title: '训练集管理',
      selectCategories: '选择类别',
      noProjects: '暂无项目',
      selected: '已选择 {count} 个类别',
      trainsetInfo: '训练集信息',
      trainsetName: '训练集名称',
      namePlaceholder: '输入训练集名称',
      categoriesPreview: '已选类别预览',
      noCategoriesSelected: '请从左侧选择类别',
      save: '保存训练集',
      update: '更新训练集',
      cancelEdit: '取消编辑',
      savedTrainsets: '已保存的训练集',
      noSavedTrainsets: '暂无已保存的训练集',
      categoriesCount: '个类别',
      edit: '编辑',
      delete: '删除',
      close: '关闭',
      conflictWarning: '不能选择同一项目不同版本的相同类别',
      typeRectangle: '矩形框',
      typePolygon: '多边形',
      typeKeypoint: '关键点',
      images: '张图片',
      annotations: '个标注',
      saveSuccess: '训练集保存成功',
      updateSuccess: '训练集更新成功',
      saveError: '训练集保存失败',
      deleteConfirmTitle: '确认删除训练集',
      deleteConfirmText: '确定要删除训练集 "{name}" 吗？此操作不可恢复。',
      confirmDelete: '确认删除',
      deleteSuccess: '训练集删除成功',
      deleteError: '训练集删除失败'
    },
    // 训练任务
    tasks: {
      title: '训练任务',
      newTask: '新建训练任务',
      empty: '暂无训练任务',
      emptyHint: '点击上方按钮创建新的训练任务',
      status: {
        running: '进行中',
        failed: '失败',
        success: '成功',
        pending: '等待中',
        error: '错误',
        completed: '已完成'
      },
      epoch: 'Epoch',
      progress: '进度',
      menu: {
        stop: '终止任务',
        delete: '删除任务',
        retrain: '重新训练'
      },
      metrics: {
        trainLoss: '训练损失',
        valLoss: '验证损失',
        boxLoss: 'Box Loss',
        clsLoss: 'Cls Loss',
        mAP50: 'mAP50',
        mAP5095: 'mAP50-95'
      },
      completedNotify: '训练任务 {name} 已完成',
      stopSuccess: '任务已停止',
      stopFailed: '停止任务失败',
      deleteSuccess: '任务已删除',
      deleteFailed: '删除任务失败',
      retrainHint: '请在新建任务对话框中配置训练参数'
    },
    // 日志窗口
    logs: {
      title: '日志输出',
      clear: '清空',
      copy: '复制',
      copied: '日志已复制到剪贴板',
      copyFailed: '复制失败',
      autoScroll: '自动滚动',
      empty: '暂无日志输出'
    },
    // 训练历史
    history: {
      title: '训练历史',
      empty: '暂无训练记录',
      emptyHint: '完成训练后将在此处显示',
      modelPath: '模型路径',
      openFolder: '打开目录',
      delete: '删除',
      deleteSuccess: '训练记录已删除',
      deleteFailed: '删除训练记录失败',
      duration: '训练时长',
      completedAt: '完成时间'
    },
    // 新建训练任务对话框
    newTaskDialog: {
      title: '新建训练任务',
      taskName: '训练名称',
      taskNamePlaceholder: '输入训练任务名称',
      datasetSource: '数据来源',
      fromTrainset: '从已配置的训练集',
      fromDirectory: '从目录选择',
      selectTrainset: '选择训练集',
      selectTrainsetPlaceholder: '请选择训练集',
      datasetPath: '数据集路径',
      pathPlaceholder: '选择数据集目录...',
      browse: '浏览',
      detecting: '正在检测格式...',
      detectedFormat: '检测到格式',
      formatUnknown: '未识别格式',
      trainValSplit: '数据划分',
      trainType: '训练类型',
      trainTypes: {
        detection: '矩形框检测',
        pose: '关键点检测',
        segmentation: '实例分割',
        custom: '其他'
      },
      trainPlugin: '训练插件',
      loadingPlugins: '加载插件中...',
      selectPlugin: '请选择插件',
      noPluginsForType: '当前训练类型暂无可用插件',
      dependencies: '依赖状态',
      checkingDeps: '检查依赖中...',
      envNotDeployed: 'Python 环境未部署',
      depsMissing: '缺少 {count} 个依赖',
      depsOk: '依赖已满足',
      installDeps: '一键安装',
      installingDeps: '正在安装依赖...',
      installDepsFailed: '依赖安装失败',
      model: '训练模型',
      selectModel: '请选择模型',
      downloadModel: '下载',
      downloadingModel: '正在下载模型 {model}',
      modelDownloaded: '模型 {model} 下载完成',
      downloadModelFailed: '模型下载失败',
      cancelDownload: '取消',
      downloadCancelled: '下载已取消',
      openModelsDir: '打开模型目录',
      refreshModelStatus: '刷新状态',
      installDepsComplete: '依赖安装完成',
      hyperparams: '训练超参数',
      epochs: '训练轮数',
      batchSize: '批次大小',
      imageSize: '图像尺寸',
      advancedParams: '高级参数',
      cancel: '取消',
      start: '开始训练',
      starting: '正在启动...',
      trainingStarted: '训练任务 {name} 已启动',
      startFailed: '启动训练失败',
      validationError: '请填写必要信息',
      torchCpuOnly: 'PyTorch 仅支持 CPU（需要 GPU 版本）',
      reinstallTorchCuda: '安装 GPU 版本',
      reinstallingTorch: '正在重新安装 PyTorch (GPU 版本)...',
      torchCudaInstalled: 'PyTorch GPU 版本安装完成',
      noTorchConfig: '插件未配置 PyTorch CUDA 版本信息'
    },
    errors: {
      pluginPathMissing: '插件路径缺失，请重新安装插件'
    }
  },
  dataset: {
    title: '数据集管理',
    loading: '加载中...',
    openExportDir: '打开导出目录',
    empty: {
      title: '暂无数据集版本',
      desc: '请在项目节点右侧点击「同步」按钮创建首个版本'
    },
    tree: {
      projects: '项目',
      noProjects: '暂无项目',
      versions: '版本',
      categories: '类别'
    },
    sync: {
      button: '同步',
      title: '创建数据集版本',
      desc: '将当前项目的所有图片、类别、标注保存为一个只读版本',
      noteLabel: '版本备注',
      notePlaceholder: '可选，描述本次版本的内容',
      confirm: '创建版本',
      cancel: '取消',
      creating: '创建中...',
      success: '成功创建版本 v{version}',
      error: '创建版本失败'
    },
    version: {
      info: '版本信息',
      createdAt: '创建时间',
      note: '备注',
      noNote: '无备注',
      editNote: '编辑备注',
      stats: '统计信息',
      images: '图片',
      categories: '类别',
      annotations: '标注',
      rollback: '从此版本回溯',
      delete: '删除版本'
    },
    health: {
      title: '数据健康度',
      progress: '标注完成度',
      progressDesc: '共 {count} 张图片已全部完成标注',
      avgAnnotations: '平均标注/图',
      checkEmpty: '无空标注',
      checkBalance: '类别分布略有失衡'
    },
    rollback: {
      title: '回溯到版本 v{version}',
      warning: '此操作将把当前项目的数据恢复到该版本的状态。',
      affected: '影响的项目',
      confirmLabel: '我已了解上述风险，确认执行回溯',
      confirm: '执行回溯',
      cancel: '取消',
      success: '已成功回溯到版本 v{version}',
      error: '回溯失败'
    },
    deleteVersion: {
      title: '删除版本',
      confirm: '确定要删除版本 v{version} 吗？此操作不可撤销。',
      success: '版本已删除',
      error: '删除失败'
    },
    export: {
      button: '导出所选类别...',
      selected: '已选 {count} 个类别',
      selectedFrom: '来自 {projects} 个项目 / {versions} 个版本',
      conflictWarning: '同一项目内不能选择不同版本的同名类别'
    },
    exportWizard: {
      title: '导出数据集',
      categories: '已选 {count} 个类别',
      images: '预计 {count} 张图片',
      annotations: '预计 {count} 个标注',
      outputPath: '导出路径',
      pathPlaceholder: '选择导出目录',
      format: '导出格式',
      customFormat: '特殊格式',
      plugin: '导出插件',
      noPlugin: '暂无支持该格式的插件',
      split: '数据集划分',
      train: '训练集',
      val: '验证集',
      test: '测试集',
      splitWarning: '划分比例之和应为100%（当前为{total}%）',
      confirm: '开始导出',
      exporting: '导出中...',
      success: '导出成功！路径：{path}',
      error: '导出失败：{msg}'
    },
    exportHistory: {
      title: '导出历史',
      openFolder: '打开目录'
    },
    project: {
      info: '项目信息',
      noVersions: '当前项目还没有任何数据集版本',
      noVersionsHint: '请点击左侧的「同步」按钮创建首个版本',
      stats: '当前状态',
      lastSync: '最近同步',
      never: '从未同步'
    },
    category: {
      type: {
        bbox: '矩形框',
        keypoint: '关键点',
        polygon: '多边形',
        mask: '分割掩码'
      }
    }
  },
  inference: {
    title: '模型推理辅助',
    confidenceThreshold: '置信度阈值',
    nmsThreshold: 'NMS 阈值',
    trainedModels: '已训练模型',
    importedModels: '导入的模型',
    noTrainedModels: '暂无已训练的模型',
    noImportedModels: '暂无导入的模型',
    dropModelHint: '将模型文件拖拽到此处，或点击右上角按钮打开目录手动放入（文件名必须唯一）',
    openFolder: '打开模型目录',
    importSuccess: '模型导入成功',
    importFailed: '模型导入失败',
    fileExists: '同名模型已存在',
    selectPolygonCategory: '请先选中一个多边形或矩形框类别',
    downloading: '{name} 下载中 {progress}%',
    downloadComplete: '{name} 下载完成',
    downloadFailed: '{name} 下载失败: {error}',
    pluginDepsNotReady: '{name} 依赖未就绪，请先在 Python 环境管理中安装依赖',
    trainingPluginNotReady: '训练插件依赖未就绪，请先在 Python 环境管理中安装依赖'
  },
  common: {
    save: '保存',
    cancel: '取消',
    minimize: '最小化',
    close: '关闭'
  },
  help: {
    title: '帮助与文档',
    description: '欢迎使用 EasyMark！这是一个高效的计算机视觉标注工具。',
    contactCard: {
      title: '联系我们',
      desc: '如有问题或建议，欢迎联系我们'
    },
    nav: {
      overview: '总览',
      project: '项目页',
      dataset: '数据集页',
      training: '训练页',
      inference: '推理窗口'
    },
    sections: {
      project: {
        title: '📁 项目管理',
        items: [
          '创建项目：点击侧边栏"+"按钮，输入名称创建新项目',
          '导入图片：右键项目选择"导入图片"，支持拖拽导入',
          '类别管理：右侧面板添加矩形框、关键点、多边形类别',
          '标注操作：选择类别后在图片上绘制，右键可编辑标注',
          '图片筛选：支持按全部/已标注/未标注/负样本筛选'
        ]
      },
      shortcuts: {
        title: '⌨️ 快捷键',
        items: [
          'Ctrl+S：保存当前图片标注',
          'Ctrl+Shift+S：保存为负样本（无标注图片）',
          '← / →：切换上一张/下一张图片',
          'Ctrl+← / Ctrl+→：跳转上/下一张未标注图片',
          'Backspace：删除选中的标注',
          'V：切换关键点可见性',
          'Ctrl+0：重置画布视图'
        ]
      },
      dataset: {
        title: '📊 数据集管理',
        items: [
          '同步版本：选择项目后点击同步按钮，创建当前标注状态的数据集快照',
          '版本管理：支持查看历史版本、回溯和删除版本',
          '数据导出：勾选需要的类别，选择格式（YOLO/COCO等）后导出',
          '数据划分：导出时可设置训练集/验证集/测试集的比例'
        ]
      },
      training: {
        title: '🧠 模型训练',
        items: [
          '环境部署：首次使用需点击"部署环境"安装 Python 依赖',
          '创建任务：选择数据集版本和模型后，配置参数创建训练任务',
          '任务监控：实时查看训练日志、损失曲线和模型指标',
          '模型推理：训练完成后可使用模型对新图片进行推理'
        ]
      },
      plugins: {
        title: '🧩 插件系统',
        items: [
          '插件安装：将插件压缩包拖拽到插件页面即可安装',
          '插件管理：右键点击已安装插件可进行卸载操作',
          '支持类型：数据集导入插件、训练框架插件等'
        ]
      },
      settings: {
        title: '⚙️ 系统设置',
        items: [
          '数据目录：设置项目和数据集的存储路径',
          '主题切换：支持深色/浅色主题',
          '语言切换：支持中文/英文界面'
        ]
      }
    }
  }
}

export default zhCN
