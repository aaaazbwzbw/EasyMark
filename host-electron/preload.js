const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('electronAPI', {
  // 主窗口控制
  minimize: () => ipcRenderer.send('window-minimize'),
  toggleMaximize: () => ipcRenderer.send('window-toggle-maximize'),
  close: () => ipcRenderer.send('window-close'),
  
  // 文件/目录选择
  selectDirectory: (defaultPath) =>
    ipcRenderer.invoke('paths-select-directory', { defaultPath }),
  selectFiles: (options) => ipcRenderer.invoke('paths-select-files', options || {}),
  selectFile: (options) => ipcRenderer.invoke('select-file', options || {}),
  
  // 插件安装
  installPlugin: (filePath) => ipcRenderer.invoke('install-plugin', filePath),
  
  // 打开路径
  openPath: (targetPath) => ipcRenderer.invoke('open-path', targetPath),
  
  // 使用系统浏览器打开外部链接
  openExternal: (url) => ipcRenderer.invoke('open-external', url),
  
  // 模型推理窗口
  openInferenceWindow: (options) => ipcRenderer.send('open-inference-window', options || {}),
  minimizeInference: () => ipcRenderer.send('inference-window-minimize'),
  closeInference: () => ipcRenderer.send('inference-window-close'),
  
  // 推理相关（旧接口，保留兼容）
  getCurrentImagePath: () => ipcRenderer.invoke('get-current-image-path'),
  sendInferenceResults: (results) => ipcRenderer.send('inference-results', results),
  onImageChanged: (callback) => ipcRenderer.on('image-changed', (_event, imagePath) => callback(imagePath)),
  onInferenceResults: (callback) => ipcRenderer.on('inference-results', (_event, results) => callback(results)),
  notifyImageChanged: (imagePath) => ipcRenderer.send('notify-image-changed', imagePath),
  
  // 推理服务（行协议，新接口）
  inferenceStartService: (pluginId) => ipcRenderer.invoke('inference-start-service', pluginId),
  inferenceStopService: () => ipcRenderer.invoke('inference-stop-service'),
  inferenceLoadModel: (modelPath) => ipcRenderer.invoke('inference-load-model', modelPath),
  inferenceRun: (payload) => ipcRenderer.invoke('inference-run', payload),
  inferenceSetImage: (imagePath) => ipcRenderer.invoke('inference-set-image', imagePath),
  inferenceServiceStatus: () => ipcRenderer.invoke('inference-service-status'),
  
  // 小窗通知主窗口参数变化，触发重新推理
  notifyInferenceParamsChanged: (params) => ipcRenderer.send('inference-params-changed', params),
  onInferenceParamsChanged: (callback) => ipcRenderer.on('inference-params-changed', (_event, params) => callback(params)),
  
  // 小窗通知主窗口模型变化，触发重新推理
  notifyInferenceModelChanged: () => ipcRenderer.send('inference-model-changed'),
  onInferenceModelChanged: (callback) => ipcRenderer.on('inference-model-changed', () => callback()),
  
  // 模型下载进度转发（主窗口 -> 推理小窗）
  forwardDownloadProgress: (data) => ipcRenderer.send('forward-download-progress', data),
  onModelDownloadProgress: (callback) => ipcRenderer.on('model-download-progress', (_event, data) => callback(data)),
  
  // 小窗日志转发到主窗口
  forwardLog: (type, args) => ipcRenderer.send('forward-log', { type, args }),
  onForwardedLog: (callback) => ipcRenderer.on('forwarded-log', (_event, data) => callback(data)),
  
  // 广播主题/语言变化到其他窗口
  broadcastThemeChange: (theme) => ipcRenderer.send('broadcast-theme-change', theme),
  broadcastLocaleChange: (locale) => ipcRenderer.send('broadcast-locale-change', locale),
  
  // 监听来自主进程的消息
  onThemeChanged: (callback) => ipcRenderer.on('theme-changed', (_event, theme) => callback(theme)),
  onLocaleChanged: (callback) => ipcRenderer.on('locale-changed', (_event, locale) => callback(locale)),
  onInferenceWindowClosed: (callback) => ipcRenderer.on('inference-window-closed', () => callback()),
  
  // 类别相关接口（供插件使用）
  getProjectCategories: () => ipcRenderer.invoke('get-project-categories'),
  getSelectedCategory: () => ipcRenderer.invoke('get-selected-category'),
  selectCategory: (categoryId) => ipcRenderer.invoke('select-category', categoryId),
  onCategoryChanged: (callback) => ipcRenderer.on('category-changed', (_event, category) => callback(category)),
  
  // 显示通知
  showNotification: (type, message) => ipcRenderer.send('show-notification', { type, message }),
  onShowNotification: (callback) => ipcRenderer.on('show-notification', (_event, data) => callback(data)),
  
  // 日志发送到主进程
  sendLog: (level, ...args) => ipcRenderer.send('renderer-log', { level, args }),
  
  // 获取应用版本
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),
})
