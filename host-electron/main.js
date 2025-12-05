const { app, BrowserWindow, ipcMain, dialog, shell } = require('electron')
const path = require('path')
const fs = require('fs')
const { spawn } = require('child_process')
const AdmZip = require('adm-zip')
const Unrar = require('unrar-promise')

let mainWindow = null
let inferenceWindow = null
let goProcess = null

// ========== 日志系统 ==========
const MAX_LOG_SIZE = 128 * 1024 // 128KB
let logFilePath = null
let logStream = null

function initLogger(dataPath) {
  logFilePath = path.join(dataPath, 'log.log')
  
  // 确保目录存在
  const logDir = path.dirname(logFilePath)
  if (!fs.existsSync(logDir)) {
    fs.mkdirSync(logDir, { recursive: true })
  }
  
  // 清空日志文件（每次启动都清空）
  fs.writeFileSync(logFilePath, '', 'utf-8')
  
  // 创建写入流
  logStream = fs.createWriteStream(logFilePath, { flags: 'a' })
  
  // 重定向 console
  const originalLog = console.log
  const originalError = console.error
  const originalWarn = console.warn
  
  const writeLog = (level, ...args) => {
    const timestamp = new Date().toISOString()
    const message = args.map(a => typeof a === 'object' ? JSON.stringify(a) : String(a)).join(' ')
    const line = `[${timestamp}] [${level}] ${message}\n`
    
    // 检查文件大小并截断
    try {
      const stats = fs.statSync(logFilePath)
      if (stats.size + line.length > MAX_LOG_SIZE) {
        // 读取现有内容，截断前面的部分
        const content = fs.readFileSync(logFilePath, 'utf-8')
        const excess = stats.size + line.length - MAX_LOG_SIZE
        let start = excess
        // 找到下一个换行符保持行完整性
        for (let i = start; i < content.length; i++) {
          if (content[i] === '\n') {
            start = i + 1
            break
          }
        }
        fs.writeFileSync(logFilePath, content.slice(start) + line, 'utf-8')
      } else {
        fs.appendFileSync(logFilePath, line, 'utf-8')
      }
    } catch (e) {
      // 忽略日志写入错误
    }
  }
  
  console.log = (...args) => {
    originalLog.apply(console, args)
    writeLog('INFO', ...args)
  }
  
  console.error = (...args) => {
    originalError.apply(console, args)
    writeLog('ERROR', ...args)
  }
  
  console.warn = (...args) => {
    originalWarn.apply(console, args)
    writeLog('WARN', ...args)
  }
  
  console.log('[Logger] Initialized:', logFilePath)
}

// ========== 推理子进程相关 ==========
let inferProcess = null
let inferBuffer = ''
const inferPending = new Map()  // requestId -> { resolve, reject }
let inferNextId = 1

// 判断是否为打包后的生产环境
const isPacked = app.isPackaged

// 获取资源路径（打包后在 resources 目录，开发时在项目目录）
function getResourcePath(...segments) {
  if (isPacked) {
    return path.join(process.resourcesPath, ...segments)
  }
  return path.join(__dirname, '..', ...segments)
}

// 配置文件路径（打包后使用用户数据目录）
function getConfigDir() {
  if (isPacked) {
    return path.join(app.getPath('userData'), 'config')
  }
  return path.join(__dirname, '..', 'backend-go', 'config')
}

const CONFIG_DIR = getConfigDir()
const CONFIG_PATH = path.join(CONFIG_DIR, 'paths.json')

function ensurePathsConfig() {
  const defaultConfig = {
    dataPath: 'D:\\EasyMark\\Data',
  }

  try {
    if (!fs.existsSync(CONFIG_DIR)) {
      fs.mkdirSync(CONFIG_DIR, { recursive: true })
    }

    let pathsFromConfig = null

    if (fs.existsSync(CONFIG_PATH)) {
      try {
        const raw = fs.readFileSync(CONFIG_PATH, 'utf-8')
        pathsFromConfig = JSON.parse(raw)
      } catch {
        pathsFromConfig = null
      }
    }

    const effectiveConfig = pathsFromConfig && typeof pathsFromConfig === 'object'
      ? { ...defaultConfig, ...pathsFromConfig }
      : defaultConfig

    // 只确保数据目录存在（幂等）
    if (typeof effectiveConfig.dataPath === 'string' && effectiveConfig.dataPath.trim()) {
      try {
        fs.mkdirSync(effectiveConfig.dataPath, { recursive: true })
      } catch {
        // ignore directory failure
      }
    }

    if (!pathsFromConfig) {
      fs.writeFileSync(CONFIG_PATH, JSON.stringify({ dataPath: effectiveConfig.dataPath }, null, 2), 'utf-8')
    }
  } catch {
    // keep app running even if path initialization fails
  }
}

// 安装内置插件到用户插件目录
function installBuiltinPlugins() {
  if (!isPacked) return // 开发模式不需要安装
  
  const dataPath = getDataPath()
  const userPluginsDir = path.join(dataPath, 'plugins')
  const builtinPluginsDir = getResourcePath('builtin-plugins')
  
  // 确保用户插件目录存在
  if (!fs.existsSync(userPluginsDir)) {
    fs.mkdirSync(userPluginsDir, { recursive: true })
  }
  
  // 检查内置插件目录是否存在
  if (!fs.existsSync(builtinPluginsDir)) {
    console.log('[BuiltinPlugins] No builtin plugins directory found')
    return
  }
  
  // 读取内置插件列表
  const builtinPlugins = fs.readdirSync(builtinPluginsDir)
  console.log('[BuiltinPlugins] Found builtin plugins:', builtinPlugins)
  
  for (const pluginName of builtinPlugins) {
    const srcDir = path.join(builtinPluginsDir, pluginName)
    const destDir = path.join(userPluginsDir, pluginName)
    
    // 检查是否是目录
    if (!fs.statSync(srcDir).isDirectory()) continue
    
    // 检查用户插件目录中是否已存在该插件
    if (fs.existsSync(destDir)) {
      console.log(`[BuiltinPlugins] Plugin ${pluginName} already exists, skipping`)
      continue
    }
    
    // 递归复制插件目录
    console.log(`[BuiltinPlugins] Installing plugin ${pluginName}...`)
    copyDirRecursive(srcDir, destDir)
    console.log(`[BuiltinPlugins] Plugin ${pluginName} installed`)
  }
}

// 递归复制目录
function copyDirRecursive(src, dest) {
  fs.mkdirSync(dest, { recursive: true })
  const entries = fs.readdirSync(src, { withFileTypes: true })
  
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name)
    const destPath = path.join(dest, entry.name)
    
    if (entry.isDirectory()) {
      copyDirRecursive(srcPath, destPath)
    } else {
      fs.copyFileSync(srcPath, destPath)
    }
  }
}

function createWindow() {
  mainWindow = new BrowserWindow({
    title: 'EasyMark',
    width: 1600,
    height: 900,
    minWidth: 960,
    minHeight: 600,
    frame: false,
    titleBarStyle: 'hiddenInset',
    backgroundColor: '#18181b',
    icon: path.join(__dirname, isPacked ? 'logo.ico' : 'logo.png'),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  })

  if (isPacked) {
    // 打包后加载本地文件
    const appPath = getResourcePath('app', 'index.html')
    mainWindow.loadFile(appPath)
  } else {
    // 开发模式加载开发服务器
    const devUrl = 'http://localhost:5173'
    mainWindow.loadURL(devUrl)
  }

  // 当主窗口关闭时，先停止推理进程，再关闭推理小窗
  mainWindow.on('closed', () => {
    // 先停止推理进程，避免进程尝试向已销毁的窗口发送消息
    stopInferProcess()
    mainWindow = null
    if (inferenceWindow && !inferenceWindow.isDestroyed()) {
      inferenceWindow.close()
    }
  })
}

// 杀掉占用端口的进程
function killPortProcess(port) {
  return new Promise((resolve) => {
    const { execSync } = require('child_process')
    try {
      // 查找占用端口的进程 PID
      const result = execSync(`netstat -ano | findstr :${port} | findstr LISTENING`, { encoding: 'utf-8' })
      const lines = result.trim().split('\n')
      const pids = new Set()
      for (const line of lines) {
        const parts = line.trim().split(/\s+/)
        const pid = parts[parts.length - 1]
        if (pid && !isNaN(parseInt(pid))) {
          pids.add(pid)
        }
      }
      for (const pid of pids) {
        try {
          execSync(`taskkill /F /PID ${pid}`, { encoding: 'utf-8' })
          console.log(`[Go Backend] Killed process on port ${port}, PID: ${pid}`)
        } catch {}
      }
    } catch {
      // 没有进程占用端口
    }
    resolve()
  })
}

async function startGoBackend() {
  // 先杀掉可能占用端口的旧进程
  await killPortProcess(18080)
  
  if (isPacked) {
    // 打包后启动编译好的可执行文件
    const backendExe = getResourcePath('backend', 'easymark-backend.exe')
    const configPath = getConfigDir()
    
    console.log('[Go Backend] Starting:', backendExe)
    console.log('[Go Backend] Config path:', configPath)
    
    goProcess = spawn(backendExe, [], {
      env: { ...process.env, EASYMARK_CONFIG_PATH: configPath },
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false
    })
    
    goProcess.stdout.on('data', (data) => {
      console.log('[Go Backend]', data.toString().trim())
    })
    
    goProcess.stderr.on('data', (data) => {
      console.error('[Go Backend Error]', data.toString().trim())
    })
    
    goProcess.on('error', (err) => {
      console.error('[Go Backend] Failed to start:', err)
    })
    
    goProcess.on('close', (code) => {
      console.log('[Go Backend] Exited with code:', code)
      goProcess = null
    })
  } else {
    // 开发模式：使用 go run
    const backendPath = path.join(__dirname, '..', 'backend-go')
    goProcess = spawn('go', ['run', 'main.go'], {
      cwd: backendPath,
      stdio: 'ignore',
    })
  }
}

app.whenReady().then(async () => {
  ensurePathsConfig()
  // 初始化日志系统
  const dataPath = getDataPath()
  initLogger(dataPath)
  // 安装内置插件
  installBuiltinPlugins()
  // 打包后自动启动 Go 后端
  if (isPacked) {
    await startGoBackend()
  }
  createWindow()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('before-quit', () => {
  if (goProcess) {
    goProcess.kill()
  }
})

ipcMain.on('window-minimize', () => {
  if (mainWindow) {
    mainWindow.minimize()
  }
})

ipcMain.on('window-toggle-maximize', () => {
  if (!mainWindow) return
  if (mainWindow.isMaximized()) {
    mainWindow.unmaximize()
  } else {
    mainWindow.maximize()
  }
})

ipcMain.on('window-close', () => {
  if (mainWindow) {
    mainWindow.close()
  }
})

// 创建模型推理小窗口
function createInferenceWindow(theme = 'dark', locale = 'zh-CN', pluginId = '') {
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.focus()
    return
  }

  inferenceWindow = new BrowserWindow({
    width: 250,
    height: 700,
    minWidth: 220,
    minHeight: 420,
    maxWidth: 340,
    frame: false,
    titleBarStyle: 'hiddenInset',
    backgroundColor: theme === 'light' ? '#ffffff' : '#18181b',
    resizable: true,
    parent: mainWindow,  // 设置父窗口，与主窗口同层级
    icon: path.join(__dirname, isPacked ? 'logo.ico' : 'logo.png'),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  })

  // 构建 URL 参数
  const queryParams = `theme=${theme}&locale=${locale}${pluginId ? `&pluginId=${pluginId}` : ''}`
  
  if (isPacked) {
    // 打包后加载本地文件
    const appPath = getResourcePath('app', 'index.html')
    inferenceWindow.loadFile(appPath, { hash: `/inference?${queryParams}` })
  } else {
    const devUrl = `http://localhost:5173/#/inference?${queryParams}`
    inferenceWindow.loadURL(devUrl)
  }

  inferenceWindow.on('closed', () => {
    inferenceWindow = null
    // 通知主窗口小窗已关闭
    try {
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send('inference-window-closed')
      }
    } catch (e) {
      // 窗口可能已被销毁，忽略错误
    }
  })
}

// 打开模型推理窗口
ipcMain.on('open-inference-window', (_event, options = {}) => {
  const { theme, locale, pluginId } = options
  createInferenceWindow(theme, locale, pluginId)
})

// 推理窗口控制
ipcMain.on('inference-window-minimize', () => {
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.minimize()
  }
})

ipcMain.on('inference-window-close', () => {
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.close()
  }
})

// 向推理窗口发送主题变化
ipcMain.on('broadcast-theme-change', (_event, theme) => {
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.webContents.send('theme-changed', theme)
  }
})

// 向推理窗口发送语言变化
ipcMain.on('broadcast-locale-change', (_event, locale) => {
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.webContents.send('locale-changed', locale)
  }
})

// ========== 推理相关 IPC ==========

// 存储当前图片路径（由主窗口设置，供推理窗口获取）
let currentImagePath = null

// 主窗口通知图片切换
ipcMain.on('notify-image-changed', (_event, imagePath) => {
  currentImagePath = imagePath
  // 转发给推理窗口
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.webContents.send('image-changed', imagePath)
  }
})

// 转发模型下载进度到推理窗口
ipcMain.on('forward-download-progress', (_event, data) => {
  if (inferenceWindow && !inferenceWindow.isDestroyed()) {
    inferenceWindow.webContents.send('model-download-progress', data)
  }
})

// 推理窗口获取当前图片路径
ipcMain.handle('get-current-image-path', () => {
  return currentImagePath
})

// 推理窗口发送推理结果到主窗口
ipcMain.on('inference-results', (_event, results) => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('inference-results', results)
  }
})

// 小窗日志转发到主窗口
ipcMain.on('forward-log', (_event, data) => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('forwarded-log', data)
  }
})

// 小窗参数变化，转发到主窗口触发重新推理
ipcMain.on('inference-params-changed', (_event, params) => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('inference-params-changed', params)
  }
})

// 小窗模型变化，转发到主窗口触发重新推理
ipcMain.on('inference-model-changed', () => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('inference-model-changed')
  }
})

// 类别相关接口（供插件使用）
ipcMain.handle('get-project-categories', async () => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    return await mainWindow.webContents.executeJavaScript('window.__easymark_getCategories?.()')
  }
  return { success: false, categories: [] }
})

ipcMain.handle('get-selected-category', async () => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    return await mainWindow.webContents.executeJavaScript('window.__easymark_getSelectedCategory?.()')
  }
  return { success: false, category: null }
})

ipcMain.handle('select-category', async (_event, categoryId) => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    return await mainWindow.webContents.executeJavaScript(`window.__easymark_selectCategory?.(${categoryId})`)
  }
  return { success: false }
})

// 显示通知（转发到主窗口）
ipcMain.on('show-notification', (_event, { type, message }) => {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('show-notification', { type, message })
  }
})

ipcMain.handle('paths-select-directory', async (_event, options = {}) => {
  const { defaultPath } = options
  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory', 'createDirectory'],
    defaultPath,
  })

  if (result.canceled || !result.filePaths || result.filePaths.length === 0) {
    return null
  }

  return result.filePaths[0]
})

ipcMain.handle('paths-select-files', async (_event, options = {}) => {
  const { defaultPath, filters } = options
  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openFile', 'multiSelections'],
    defaultPath,
    filters:
      filters || [
        {
          name: 'Images',
          extensions: ['png', 'jpg', 'jpeg', 'bmp', 'gif', 'webp'],
        },
      ],
  })

  if (result.canceled || !result.filePaths || result.filePaths.length === 0) {
    return []
  }

  return result.filePaths
})

// 打开系统中的文件或目录
ipcMain.handle('open-path', async (_event, targetPath) => {
  try {
    if (!targetPath || typeof targetPath !== 'string') {
      return { success: false, error: 'empty_path' }
    }
    const err = await shell.openPath(targetPath)
    if (err) {
      return { success: false, error: err }
    }
    return { success: true }
  } catch (e) {
    return { success: false, error: String(e) }
  }
})

// 使用系统默认浏览器打开外部链接
ipcMain.handle('open-external', async (_event, url) => {
  try {
    if (!url || typeof url !== 'string') {
      return { success: false, error: 'empty_url' }
    }
    await shell.openExternal(url)
    return { success: true }
  } catch (e) {
    return { success: false, error: String(e) }
  }
})

// 获取应用版本
ipcMain.handle('get-app-version', () => {
  return app.getVersion()
})

// 选择单个文件（用于插件安装等）
ipcMain.handle('select-file', async (_event, options = {}) => {
  const { defaultPath, filters, title } = options
  const result = await dialog.showOpenDialog(mainWindow, {
    title: title || 'Select File',
    properties: ['openFile'],
    defaultPath,
    filters: filters || [],
  })

  if (result.canceled || !result.filePaths || result.filePaths.length === 0) {
    return null
  }

  return result.filePaths[0]
})

// 获取配置中的数据路径
function getDataPath() {
  try {
    if (fs.existsSync(CONFIG_PATH)) {
      const raw = fs.readFileSync(CONFIG_PATH, 'utf-8')
      const config = JSON.parse(raw)
      return config.dataPath || 'D:\\EasyMark\\Data'
    }
  } catch {}
  return 'D:\\EasyMark\\Data'
}

// ========== 推理子进程管理（行协议） ==========

// 获取 Python 可执行文件路径（与 Go 后端保持一致）
function getInferPythonExe() {
  const dataPath = getDataPath()
  // 使用与 Go 后端相同的虚拟环境路径
  const venvPython = path.join(dataPath, 'EasyMark_python_venv', 'Scripts', 'python.exe')
  console.log('[Infer] Checking venv Python:', venvPython, 'exists:', fs.existsSync(venvPython))
  if (fs.existsSync(venvPython)) {
    console.log('[Infer] Using venv Python')
    return venvPython
  }
  // 回退到系统 python
  console.log('[Infer] Using system Python (venv not found)')
  return 'python'
}

// 当前激活的推理插件 ID
let activeInferPluginId = null

// 查找推理插件脚本路径
function findInferPluginScript(pluginId) {
  const dataPath = getDataPath()
  
  // 只从用户数据目录的 plugins 文件夹读取，禁止使用开发目录或内置目录
  const searchDirs = []
  
  // 用户安装的插件目录（开发和打包模式统一使用）
  searchDirs.push(path.join(dataPath, 'plugins'))
  
  for (const dir of searchDirs) {
    if (!fs.existsSync(dir)) continue
    
    // 遍历插件目录
    const entries = fs.readdirSync(dir, { withFileTypes: true })
    for (const entry of entries) {
      if (!entry.isDirectory()) continue
      
      const manifestPath = path.join(dir, entry.name, 'manifest.json')
      if (!fs.existsSync(manifestPath)) continue
      
      try {
        const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'))
        
        // 检查是否是目标插件
        if (manifest.id === pluginId || entry.name === pluginId) {
          // 检查是否支持推理
          const types = Array.isArray(manifest.type) ? manifest.type : [manifest.type]
          if (!types.includes('inference')) continue
          
          const serviceEntry = manifest.inference?.serviceEntry || 'infer_service.py'
          const scriptPath = path.join(dir, entry.name, serviceEntry)
          
          if (fs.existsSync(scriptPath)) {
            console.log('[Infer] Found plugin:', pluginId, 'at', scriptPath)
            return {
              scriptPath,
              pluginDir: path.join(dir, entry.name),
              manifest
            }
          }
        }
      } catch (e) {
        console.warn('[Infer] Failed to parse manifest:', manifestPath, e.message)
      }
    }
  }
  
  return null
}

// 启动推理子进程
function startInferProcess(pluginId) {
  if (inferProcess) {
    console.log('[Infer] Process already running')
    return true
  }
  
  const dataPath = getDataPath()
  
  // 如果指定了插件 ID，查找对应插件
  let scriptPath = null
  let pluginInfo = null
  
  if (pluginId) {
    pluginInfo = findInferPluginScript(pluginId)
    if (pluginInfo) {
      scriptPath = pluginInfo.scriptPath
      activeInferPluginId = pluginId
    }
  }
  
  // 如果没有指定插件或找不到，使用默认的 YOLO 推理插件
  if (!scriptPath) {
    pluginInfo = findInferPluginScript('infer.ultralytics-yolo')
    if (pluginInfo) {
      scriptPath = pluginInfo.scriptPath
      activeInferPluginId = 'infer.ultralytics-yolo'
    }
  }
  
  if (!scriptPath) {
    console.error('[Infer] No inference plugin found. Please install a plugin to {dataPath}/plugins/')
    console.error('[Infer] Data path:', dataPath)
    return false
  }
  
  // 获取 Python 解释器
  // 优先级：useVenvFrom 指定的 > 自己 ID 对应的 > 全局虚拟环境
  let pythonExe
  const manifest = pluginInfo?.manifest
  const useVenvFrom = manifest?.python?.useVenvFrom
  const pluginHasPython = !!manifest?.python
  
  // 确定使用哪个虚拟环境
  let venvPluginId = null
  if (useVenvFrom) {
    // 使用指定插件的虚拟环境
    venvPluginId = useVenvFrom
    console.log('[Infer] Using venv from plugin:', useVenvFrom)
  } else if (pluginHasPython && activeInferPluginId) {
    // 插件有 python 配置，使用自己 ID 对应的虚拟环境
    venvPluginId = activeInferPluginId
    console.log('[Infer] Using own venv:', activeInferPluginId)
  }
  
  if (venvPluginId) {
    pythonExe = path.join(dataPath, 'plugins_python_venv', venvPluginId, 'Scripts', 'python.exe')
  } else {
    // 回退到全局虚拟环境
    pythonExe = path.join(dataPath, 'EasyMark_python_venv', 'Scripts', 'python.exe')
    console.log('[Infer] Using global venv')
  }
  
  // 检查虚拟环境是否存在
  if (!fs.existsSync(pythonExe)) {
    // 开发环境下尝试使用系统 Python
    if (!isPacked) {
      console.log('[Infer] Venv not found:', pythonExe)
      console.log('[Infer] Trying system Python (dev mode)')
      pythonExe = 'python'
    } else {
      console.error('[Infer] Python venv not found:', pythonExe)
      console.error('[Infer] Please install dependencies for plugin:', venvPluginId || 'global')
      return false
    }
  }
  
  const pluginDir = pluginInfo?.pluginDir || path.dirname(scriptPath)
  
  console.log('========================================')
  console.log('[Infer] Starting inference plugin service')
  console.log('[Infer] Plugin ID:', activeInferPluginId || '(none)')
  console.log('[Infer] Plugin Path:', pluginDir)
  console.log('[Infer] Script:', scriptPath)
  console.log('[Infer] Python:', pythonExe)
  console.log('========================================')
  
  // 构建环境变量，确保 PyTorch CUDA DLL 能被找到
  const venvDir = path.dirname(path.dirname(pythonExe)) // 获取虚拟环境根目录
  const torchLibPath = path.join(venvDir, 'Lib', 'site-packages', 'torch', 'lib')
  const venvScriptsPath = path.join(venvDir, 'Scripts')
  const existingPath = process.env.PATH || ''
  
  inferProcess = spawn(pythonExe, [scriptPath], {
    env: { 
      ...process.env,
      PATH: `${torchLibPath};${venvScriptsPath};${existingPath}`,
      EASYMARK_DATA_PATH: dataPath,
      EASYMARK_PLUGIN_ID: activeInferPluginId || '',
      EASYMARK_PLUGIN_PATH: pluginDir
    },
    stdio: ['pipe', 'pipe', 'pipe']
  })
  
  inferBuffer = ''
  
  // 处理 stdout（JSON 响应）
  inferProcess.stdout.on('data', (chunk) => {
    inferBuffer += chunk.toString()
    let newlineIdx
    while ((newlineIdx = inferBuffer.indexOf('\n')) !== -1) {
      const line = inferBuffer.slice(0, newlineIdx).trim()
      inferBuffer = inferBuffer.slice(newlineIdx + 1)
      if (!line) continue
      
      try {
        const msg = JSON.parse(line)
        const reqId = msg.requestId
        if (reqId && inferPending.has(reqId)) {
          const { resolve } = inferPending.get(reqId)
          inferPending.delete(reqId)
          resolve(msg)
        }
      } catch (e) {
        console.warn('[Infer] Failed to parse response:', line)
      }
    }
  })
  
  // 处理 stderr（日志）
  inferProcess.stderr.on('data', (chunk) => {
    const text = chunk.toString().trim()
    if (text) {
      console.log('[Infer/Python]', text)
    }
  })
  
  inferProcess.on('close', (code) => {
    console.log('[Infer] Process exited with code:', code)
    inferProcess = null
    // 拒绝所有待处理的请求
    for (const [, { reject }] of inferPending) {
      reject(new Error('Process exited'))
    }
    inferPending.clear()
  })
  
  inferProcess.on('error', (err) => {
    console.error('[Infer] Process error:', err)
    inferProcess = null
  })
  
  return true
}

// 停止推理子进程
function stopInferProcess() {
  if (!inferProcess) return
  
  // 发送 shutdown 命令
  try {
    const shutdownMsg = JSON.stringify({ requestId: 'shutdown', cmd: 'shutdown' }) + '\n'
    inferProcess.stdin.write(shutdownMsg)
  } catch (e) {
    // ignore
  }
  
  // 强制终止
  setTimeout(() => {
    if (inferProcess) {
      inferProcess.kill()
      inferProcess = null
    }
  }, 1000)
}

// 发送命令到 Python 子进程
function sendInferCommand(cmd, payload = {}) {
  return new Promise((resolve, reject) => {
    if (!inferProcess) {
      reject(new Error('Inference process not running'))
      return
    }
    
    const requestId = String(inferNextId++)
    const msg = { requestId, cmd, ...payload }
    
    inferPending.set(requestId, { resolve, reject })
    
    // 设置超时（SAM2 等大模型加载较慢，延长超时时间）
    setTimeout(() => {
      if (inferPending.has(requestId)) {
        inferPending.delete(requestId)
        reject(new Error('模型加载超时，请重试 / Model loading timeout, please retry'))
      }
    }, 32000)
    
    try {
      inferProcess.stdin.write(JSON.stringify(msg) + '\n')
    } catch (e) {
      inferPending.delete(requestId)
      reject(e)
    }
  })
}

// IPC: 启动推理服务
ipcMain.handle('inference-start-service', async (_event, pluginId) => {
  const success = startInferProcess(pluginId)
  return { success }
})

// IPC: 停止推理服务
ipcMain.handle('inference-stop-service', async () => {
  stopInferProcess()
  return { success: true }
})

// IPC: 加载模型
ipcMain.handle('inference-load-model', async (_event, modelPath) => {
  try {
    // 如果进程未启动，先启动
    if (!inferProcess) {
      startInferProcess()
      // 等待进程就绪
      await new Promise(r => setTimeout(r, 500))
    }
    
    const result = await sendInferCommand('load_model', { weights: modelPath })
    return result
  } catch (e) {
    return { success: false, error: e.message }
  }
})

// IPC: 设置图像（SAM-2 需要）
ipcMain.handle('inference-set-image', async (_event, imagePath) => {
  try {
    if (!inferProcess) {
      return { success: false, error: 'Inference service not running' }
    }
    
    const result = await sendInferCommand('set_image', { path: imagePath })
    return result
  } catch (e) {
    return { success: false, error: e.message }
  }
})

// IPC: 执行推理
ipcMain.handle('inference-run', async (_event, payload) => {
  try {
    if (!inferProcess) {
      return { success: false, error: 'Inference service not running', annotations: [] }
    }
    
    const result = await sendInferCommand('infer', payload)
    return result
  } catch (e) {
    return { success: false, error: e.message, annotations: [] }
  }
})

// IPC: 获取推理服务状态
ipcMain.handle('inference-service-status', async () => {
  return { running: !!inferProcess }
})

// 应用退出时停止推理子进程
app.on('before-quit', () => {
  stopInferProcess()
})

ipcMain.handle('install-plugin', async (_event, filePath) => {
  try {
    const ext = path.extname(filePath).toLowerCase()
    const dataPath = getDataPath()
    const pluginsDir = path.join(dataPath, 'plugins')
    
    // 确保插件目录存在
    fs.mkdirSync(pluginsDir, { recursive: true })
    
    // 创建临时目录
    const tempDir = path.join(app.getPath('temp'), `plugin-install-${Date.now()}`)
    fs.mkdirSync(tempDir, { recursive: true })
    
    try {
      // 解压
      if (ext === '.zip') {
        const zip = new AdmZip(filePath)
        zip.extractAllTo(tempDir, true)
      } else if (ext === '.rar') {
        await Unrar.unrar(filePath, tempDir)
      } else {
        return { success: false, error: 'unsupported_format' }
      }
      
      // 查找 manifest.json
      let manifestPath = null
      let pluginRoot = null
      
      const findManifest = (dir) => {
        const entries = fs.readdirSync(dir, { withFileTypes: true })
        for (const entry of entries) {
          const fullPath = path.join(dir, entry.name)
          if (entry.isFile() && entry.name === 'manifest.json') {
            return { manifestPath: fullPath, pluginRoot: dir }
          }
          if (entry.isDirectory()) {
            const result = findManifest(fullPath)
            if (result) return result
          }
        }
        return null
      }
      
      const found = findManifest(tempDir)
      if (!found) {
        return { success: false, error: 'manifest_not_found' }
      }
      manifestPath = found.manifestPath
      pluginRoot = found.pluginRoot
      
      // 读取并验证 manifest
      const manifestData = fs.readFileSync(manifestPath, 'utf-8')
      let manifest
      try {
        manifest = JSON.parse(manifestData)
      } catch {
        return { success: false, error: 'invalid_manifest' }
      }
      
      if (!manifest.id || !manifest.name) {
        return { success: false, error: 'manifest_missing_fields' }
      }
      
      // 安全检查 ID
      if (manifest.id.includes('/') || manifest.id.includes('\\')) {
        return { success: false, error: 'invalid_plugin_id' }
      }
      
      // 复制到插件目录
      const destDir = path.join(pluginsDir, manifest.id)
      
      // 如果已存在，先删除
      if (fs.existsSync(destDir)) {
        fs.rmSync(destDir, { recursive: true, force: true })
      }
      
      // 复制目录
      const copyDir = (src, dest) => {
        fs.mkdirSync(dest, { recursive: true })
        const entries = fs.readdirSync(src, { withFileTypes: true })
        for (const entry of entries) {
          const srcPath = path.join(src, entry.name)
          const destPath = path.join(dest, entry.name)
          if (entry.isDirectory()) {
            copyDir(srcPath, destPath)
          } else {
            fs.copyFileSync(srcPath, destPath)
          }
        }
      }
      
      copyDir(pluginRoot, destDir)
      
      return { success: true, plugin: manifest }
    } finally {
      // 清理临时目录
      try {
        fs.rmSync(tempDir, { recursive: true, force: true })
      } catch {}
    }
  } catch (err) {
    console.error('Install plugin error:', err)
    return { success: false, error: 'install_failed', message: err.message }
  }
})

// ========== 前端日志接收 ==========
ipcMain.on('renderer-log', (_event, { level, args }) => {
  const prefix = '[Renderer]'
  switch (level) {
    case 'error':
      console.error(prefix, ...args)
      break
    case 'warn':
      console.warn(prefix, ...args)
      break
    default:
      console.log(prefix, ...args)
  }
})
