<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  Brain, FolderOpen, Minus, X, RefreshCw, Box, Loader2, Check, Upload,
  CheckCircle, AlertCircle, XCircle
} from 'lucide-vue-next'

const { t, locale } = useI18n()

// 当前插件信息
interface CustomParam {
  key: string
  type: 'select' | 'number' | 'slider' | 'checkbox' | 'text'
  label: string | Record<string, string>
  default?: unknown
  options?: { value: unknown; label: string | Record<string, string> }[]
  min?: number
  max?: number
  step?: number
}

interface PluginUIConfig {
  type?: 'default' | 'custom'
  entry?: string
  modelSelector?: boolean
  confidenceSlider?: { default: number; min: number; max: number; step?: number } | boolean
  iouSlider?: { default: number; min: number; max: number; step?: number } | boolean
  autoInfer?: { default: boolean } | boolean
  customParams?: CustomParam[]
}

const pluginId = ref<string | null>(null)
const pluginName = ref<string>('')
const pluginUIConfig = ref<PluginUIConfig | null>(null)
const customParamValues = ref<Record<string, unknown>>({})
const pluginPath = ref<string>('')
const customUIUrl = ref<string | null>(null)
const customIframeRef = ref<HTMLIFrameElement | null>(null)

// 获取本地化文本
const getLocalizedText = (text: string | Record<string, string>): string => {
  if (typeof text === 'string') return text
  return text[locale.value] || text['en-US'] || text['zh-CN'] || Object.values(text)[0] || ''
}

// 推理服务相关状态
const inferenceServerRunning = ref(false)
const selectedModel = ref<ModelItem | null>(null)
const isInferring = ref(false)

type ToastType = 'success' | 'warning' | 'error'
type ToastKey = 'inference.importSuccess' | 'inference.importFailed' | 'inference.fileExists'

const toastVisible = ref(false)
const toastType = ref<ToastType>('success')
const toastMessageKey = ref<ToastKey>('inference.importSuccess')
let toastTimer: number | null = null

const showToast = (type: ToastType, key: ToastKey) => {
  toastType.value = type
  toastMessageKey.value = key
  toastVisible.value = true
  if (toastTimer !== null) {
    window.clearTimeout(toastTimer)
  }
  toastTimer = window.setTimeout(() => {
    toastVisible.value = false
  }, 2000)
}

// 主题和语言同步（通过 URL 参数获取）
const theme = ref<'light' | 'dark'>('dark')

// 从 URL 参数获取主题、语言和插件 ID
const syncThemeAndLocale = () => {
  // Vue Router hash 模式下，参数在 hash 之后
  // URL 格式: http://localhost:5173/#/inference?theme=dark&pluginId=xxx
  let search = window.location.search
  if (!search && window.location.hash.includes('?')) {
    search = '?' + window.location.hash.split('?')[1]
  }
  
  const urlParams = new URLSearchParams(search)
  const themeParam = urlParams.get('theme')
  const localeParam = urlParams.get('locale')
  const pluginParam = urlParams.get('pluginId')
  
  console.log('[Inference] URL params:', { theme: themeParam, locale: localeParam, pluginId: pluginParam })
  
  if (themeParam === 'light' || themeParam === 'dark') {
    theme.value = themeParam
    document.documentElement.setAttribute('data-theme', themeParam)
  }
  if (localeParam) {
    locale.value = localeParam
  }
  if (pluginParam) {
    pluginId.value = pluginParam
  }
}

// 加载插件配置
const loadPluginConfig = async () => {
  console.log('[Inference] loadPluginConfig, pluginId:', pluginId.value)
  if (!pluginId.value) {
    console.log('[Inference] No pluginId, using default UI')
    return
  }
  
  try {
    const resp = await fetch(`http://localhost:18080/api/plugins?type=inference`)
    if (resp.ok) {
      const data = await resp.json()
      console.log('[Inference] Loaded plugins:', data.plugins?.map((p: any) => p.id))
      const plugin = (data.plugins || []).find((p: any) => p.id === pluginId.value)
      console.log('[Inference] Found plugin:', plugin?.id, 'ui type:', plugin?.inference?.ui?.type)
      if (plugin) {
        // 获取插件名称
        if (typeof plugin.name === 'string') {
          pluginName.value = plugin.name
        } else if (plugin.name) {
          pluginName.value = plugin.name[locale.value] || plugin.name['zh-CN'] || plugin.name['en-US'] || ''
        }
        
        // 保存插件路径
        pluginPath.value = plugin.path || ''
        
        // 获取 UI 配置
        if (plugin.inference?.ui) {
          pluginUIConfig.value = plugin.inference.ui
          const ui = plugin.inference.ui
          
          // 检查是否是自定义 UI
          if (ui.type === 'custom' && ui.entry && pluginId.value) {
            // 通过后端加载插件 UI
            customUIUrl.value = `http://localhost:18080/api/plugins/${pluginId.value}/ui/${ui.entry}?theme=${theme.value}&locale=${locale.value}&pluginId=${pluginId.value}`
            console.log('[Inference] Using custom UI:', customUIUrl.value)
          } else {
            // 使用默认 UI，应用默认值
            customUIUrl.value = null
            if (typeof ui.confidenceSlider === 'object' && ui.confidenceSlider.default) {
              confidenceThreshold.value = ui.confidenceSlider.default
            }
            if (typeof ui.iouSlider === 'object' && ui.iouSlider.default) {
              nmsThreshold.value = ui.iouSlider.default
            }
            
            // 初始化自定义参数默认值
            if (ui.customParams) {
              for (const param of ui.customParams) {
                if (param.default !== undefined) {
                  customParamValues.value[param.key] = param.default
                }
              }
            }
          }
        }
      }
    }
  } catch (e) {
    console.error('Failed to load plugin config:', e)
  }
}

// 监听主进程发送的主题/语言变化
if (window.electronAPI?.onThemeChanged) {
  window.electronAPI.onThemeChanged((newTheme: string) => {
    theme.value = newTheme as 'light' | 'dark'
    document.documentElement.setAttribute('data-theme', newTheme)
  })
}
if (window.electronAPI?.onLocaleChanged) {
  window.electronAPI.onLocaleChanged((newLocale: string) => {
    locale.value = newLocale
  })
}

// 滑块值
const confidenceThreshold = ref(0.3)
const nmsThreshold = ref(0.5)

// Tab 状态
const activeTab = ref<'trained' | 'imported'>('trained')

// 模型列表
interface ModelItem {
  name: string
  path: string
  createdAt: string
}

const trainedModels = ref<ModelItem[]>([])
const importedModels = ref<ModelItem[]>([])
const isLoading = ref(false)
const isDragging = ref(false)

// Tab 切换
const handleSelectTrainedTab = () => {
  activeTab.value = 'trained'
}

const handleSelectImportedTab = async () => {
  activeTab.value = 'imported'
  await loadImportedModels()
}

// 加载已训练模型
const loadTrainedModels = async () => {
  try {
    const resp = await fetch('http://localhost:18080/api/training/outputs')
    if (resp.ok) {
      const data = await resp.json()
      trainedModels.value = (data.outputs || []).map((o: any) => ({
        name: o.name || o.taskId,
        path: o.modelPath,
        createdAt: o.createdAt || ''
      }))
    }
  } catch (e) {
    console.error('Failed to load trained models:', e)
  }
}

// 加载导入的模型
const loadImportedModels = async () => {
  try {
    const resp = await fetch('http://localhost:18080/api/inference/models')
    if (resp.ok) {
      const data = await resp.json()
      importedModels.value = data.models || []
    }
  } catch (e) {
    console.error('Failed to load imported models:', e)
  }
}

// 处理来自插件 UI iframe 的消息
const handlePluginMessage = async (event: MessageEvent) => {
  const msg = event.data
  if (msg.type !== 'plugin-ui-request') return
  
  const { requestId, action, data } = msg
  
  try {
    let result: any = { success: true }
    
    switch (action) {
      case 'loadModel':
        // 加载模型
        if (window.electronAPI?.inferenceLoadModel) {
          result = await window.electronAPI.inferenceLoadModel(data.path)
        }
        break
        
      case 'notifyModelChanged':
        // 通知模型变化
        window.electronAPI?.notifyInferenceModelChanged?.()
        break
        
      case 'notifyParamsChanged':
        // 通知参数变化
        window.electronAPI?.notifyInferenceParamsChanged?.(data)
        break
        
      case 'setParam':
        // 设置推理参数（如置信度）
        window.electronAPI?.notifyInferenceParamsChanged?.({ [data.key]: data.value } as any)
        break
        
      case 'getCategories':
        // 获取项目类别列表
        if (window.electronAPI?.getProjectCategories) {
          result = await window.electronAPI.getProjectCategories()
        }
        break
        
      case 'getSelectedCategory':
        // 获取当前选中的类别
        if (window.electronAPI?.getSelectedCategory) {
          result = await window.electronAPI.getSelectedCategory()
        }
        break
        
      case 'selectCategory':
        // 选中指定类别
        if (window.electronAPI?.selectCategory) {
          result = await window.electronAPI.selectCategory(data.categoryId)
        }
        break
        
      case 'notify':
        // 显示通知（转发到主窗口）
        window.electronAPI?.showNotification?.(data.type, data.message)
        break
        
      case 'setContinuousMode':
        // 设置连续标注模式
        window.electronAPI?.notifyInferenceParamsChanged?.({ continuousMode: data.enabled })
        break
        
      default:
        result = { success: false, error: `Unknown action: ${action}` }
    }
    
    // 发送响应
    event.source?.postMessage({
      type: 'plugin-ui-response',
      requestId,
      success: true,
      result
    }, { targetOrigin: '*' })
    
  } catch (e: any) {
    event.source?.postMessage({
      type: 'plugin-ui-response',
      requestId,
      success: false,
      error: e.message || 'Unknown error'
    }, { targetOrigin: '*' })
  }
}

// 初始化
onMounted(async () => {
  // 监听插件 UI 消息
  window.addEventListener('message', handlePluginMessage)
  
  // 监听模型下载进度并转发给 iframe
  window.electronAPI?.onModelDownloadProgress?.((data) => {
    if (customIframeRef.value?.contentWindow) {
      customIframeRef.value.contentWindow.postMessage({
        type: data.type === 'progress' ? 'model_download_progress' : 
              data.type === 'complete' ? 'model_download_complete' : 'model_download_error',
        data
      }, '*')
    }
  })
  
  // 劫持 console，转发日志到主窗口
  const originalConsole = { log: console.log, warn: console.warn, error: console.error }
  const forwardLog = (type: string, ...args: unknown[]) => {
    window.electronAPI?.forwardLog?.(type, args.map(a => 
      typeof a === 'object' ? JSON.stringify(a) : String(a)
    ))
  }
  console.log = (...args) => { originalConsole.log(...args); forwardLog('log', ...args) }
  console.warn = (...args) => { originalConsole.warn(...args); forwardLog('warn', ...args) }
  console.error = (...args) => { originalConsole.error(...args); forwardLog('error', ...args) }
  
  // 让小窗页面完全贴合当前窗口尺寸（覆盖全局 body 的 min-width 设置）
  const docEl = document.documentElement
  const body = document.body
  if (docEl) {
    docEl.style.minWidth = '0'
  }
  if (body) {
    body.style.minWidth = '0'
    body.style.width = '100%'
    body.style.overflow = 'hidden'
  }

  // 防止 Electron 默认行为吞掉拖拽事件，同时将 drop 统一委托给组件内的处理逻辑
  const handleWindowDragOver = (e: DragEvent) => {
    e.preventDefault()
  }
  const handleWindowDrop = (e: DragEvent) => {
    console.log('[Inference] window drop', e)
    handleDrop(e)
  }
  window.addEventListener('dragover', handleWindowDragOver)
  window.addEventListener('drop', handleWindowDrop)

  syncThemeAndLocale()
  
  // 加载插件配置
  await loadPluginConfig()
  
  // 确保目录存在
  await fetch('http://localhost:18080/api/inference/ensure-dir', { method: 'POST' })
  
  isLoading.value = true
  await Promise.all([loadTrainedModels(), loadImportedModels()])
  isLoading.value = false
  
  // 启动推理服务
  await startInferenceServer()
  
  // 图片切换事件现在由工作台直接处理推理，小窗不再参与
})

// 清理
onUnmounted(async () => {
  // 移除插件 UI 消息监听
  window.removeEventListener('message', handlePluginMessage)
  
  // 停止推理服务
  await stopInferenceServer()
})

// 拖拽导入
const handleDragOver = (e: DragEvent) => {
  if (activeTab.value !== 'imported') return
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
  isDragging.value = true
}

const handleDragLeave = () => {
  isDragging.value = false
}

const handleDrop = async (e: DragEvent) => {
  if (activeTab.value !== 'imported') return
  e.preventDefault()
  e.stopPropagation()
  console.log('[Inference] handleDrop start', e)
  isDragging.value = false
  
  const dt = e.dataTransfer
  if (!dt) return
  console.log('[Inference] dataTransfer', {
    filesLength: dt.files ? dt.files.length : 0,
    itemsLength: dt.items ? dt.items.length : 0,
    types: dt.types
  })
  let fileList: File[] = []
  if (dt.files && dt.files.length > 0) {
    fileList = Array.from(dt.files)
  } else if (dt.items && dt.items.length > 0) {
    fileList = Array.from(dt.items)
      .filter((item) => item.kind === 'file')
      .map((item) => item.getAsFile())
      .filter((f): f is File => !!f)
  }
  
  for (let index = 0; index < fileList.length; index++) {
    const file = fileList[index]
    if (!file) continue
    const name = file.name || ''
    const lower = name.toLowerCase()
    if (!lower.endsWith('.pt') && !lower.endsWith('.onnx')) {
      console.log('[Inference] skip non-model file', name)
      continue
    }
    
    console.log('[Inference] candidate file (upload)', { name })
    try {
      const formData = new FormData()
      formData.append('file', file)
      const resp = await fetch('http://localhost:18080/api/inference/upload-model', {
        method: 'POST',
        body: formData
      })
      if (!resp.ok) {
        const text = await resp.text()
        let data: any = null
        try {
          data = text ? JSON.parse(text) : null
        } catch {
          data = null
        }
        if (resp.status === 409 && data && data.error === 'file_exists') {
          showToast('warning', 'inference.fileExists')
        } else {
          showToast('error', 'inference.importFailed')
          console.error('[Inference] upload failed', text)
        }
        continue
      }
      showToast('success', 'inference.importSuccess')
      await loadImportedModels()
    } catch (e) {
      showToast('error', 'inference.importFailed')
      console.error('Failed to upload model:', e)
    }
  }
}

// ========== 推理服务管理（通过 IPC 调用 Electron 主进程管理的 Python 子进程） ==========

// 启动推理服务
const startInferenceServer = async () => {
  try {
    // 传递当前插件 ID，确保重新打开时能正确找到插件
    const result = await window.electronAPI?.inferenceStartService?.(pluginId.value || undefined)
    if (result?.success) {
      inferenceServerRunning.value = true
      console.log('[Inference] Service started via IPC, pluginId:', pluginId.value)
    }
  } catch (e) {
    console.error('[Inference] Failed to start service:', e)
  }
}

// 停止推理服务
const stopInferenceServer = async () => {
  try {
    await window.electronAPI?.inferenceStopService?.()
    inferenceServerRunning.value = false
    console.log('[Inference] Service stopped via IPC')
  } catch (e) {
    console.error('[Inference] Failed to stop service:', e)
  }
}

// 加载模型到推理服务
const loadModelToServer = async (modelPath: string) => {
  try {
    const result = await window.electronAPI?.inferenceLoadModel?.(modelPath)
    if (result?.success) {
      inferenceServerRunning.value = true  // 加载模型成功说明服务在运行
      console.log('[Inference] Model loaded via IPC:', modelPath)
      return true
    } else {
      console.error('[Inference] Failed to load model:', result?.error)
    }
  } catch (e) {
    const err = e as Error
    console.error('[Inference] Failed to load model:', err.message || err.name || String(e))
  }
  return false
}

// 选择模型（加载后通知主窗口重新推理）
const selectModel = async (model: ModelItem) => {
  if (selectedModel.value?.path === model.path) return
  
  selectedModel.value = model
  console.log('[Inference] Model selected:', model.name)
  
  // 加载模型到 Python 服务
  const success = await loadModelToServer(model.path)
  
  // 模型加载成功后，通知主窗口重新推理当前图片
  if (success) {
    window.electronAPI?.notifyInferenceModelChanged?.()
  }
}

// 滑块调整后通知主窗口，触发重新推理
const handleSliderChange = () => {
  console.log('[Inference] Params changed:', confidenceThreshold.value, nmsThreshold.value)
  // 通知主窗口参数已变化，让它用新参数重新推理
  window.electronAPI?.notifyInferenceParamsChanged?.({
    conf: confidenceThreshold.value,
    iou: nmsThreshold.value
  })
}

// 打开导入模型文件夹
const openImportFolder = async () => {
  try {
    await fetch('http://localhost:18080/api/inference/open-folder', { method: 'POST' })
  } catch (e) {
    console.error('[Inference] Failed to open folder:', e)
  }
}

// 窗口控制
const minimizeWindow = () => {
  window.electronAPI?.minimizeInference?.()
}

const closeWindow = async () => {
  // 关闭窗口前停止推理服务
  await stopInferenceServer()
  window.electronAPI?.closeInference?.()
}

// 格式化时间
const formatTime = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
}
</script>

<template>
  <div class="inference-window" :class="{ 'theme-light': theme === 'light' }">
    <!-- 自定义标题栏 -->
    <header class="inference-header">
      <div class="inference-header__left">
        <Brain :size="18" />
        <span class="inference-header__title">{{ t('inference.title') }}</span>
      </div>
      <div class="inference-header__right">
        <button
          v-if="activeTab === 'imported'"
          class="inference-header__btn"
          :title="t('inference.openFolder')"
          @click="openImportFolder"
        >
          <FolderOpen :size="16" />
        </button>
        <button class="inference-header__btn" :title="t('common.minimize')" @click="minimizeWindow">
          <Minus :size="16" />
        </button>
        <button class="inference-header__btn inference-header__btn--close" :title="t('common.close')" @click="closeWindow">
          <X :size="16" />
        </button>
      </div>
    </header>

    <!-- 内容区域 -->
    <main class="inference-content">
      <!-- 自定义 UI（iframe） -->
      <iframe
        v-if="customUIUrl"
        ref="customIframeRef"
        :src="customUIUrl"
        class="inference-custom-iframe"
        frameborder="0"
        sandbox="allow-scripts allow-same-origin"
      />

      <!-- 默认 UI -->
      <template v-else>
        <!-- 滑块区域：标题 + 滑块 + 数值 同一行 -->
        <div class="inference-sliders">
        <div class="inference-slider">
          <span class="inference-slider__label">{{ t('inference.confidenceThreshold') }}</span>
          <input
            v-model.number="confidenceThreshold"
            type="range"
            min="0"
            max="1"
            step="0.01"
            class="inference-slider__input"
            @change="handleSliderChange"
          />
          <span class="inference-slider__value">{{ confidenceThreshold.toFixed(2) }}</span>
        </div>
        <div class="inference-slider">
          <span class="inference-slider__label">{{ t('inference.nmsThreshold') }}</span>
          <input
            v-model.number="nmsThreshold"
            type="range"
            min="0"
            max="1"
            step="0.01"
            class="inference-slider__input"
            @change="handleSliderChange"
          />
          <span class="inference-slider__value">{{ nmsThreshold.toFixed(2) }}</span>
        </div>
        
      </div>

      <!-- 自定义参数区域（根据插件配置动态渲染） -->
      <div v-if="pluginUIConfig?.customParams?.length" class="inference-custom-params">
        <template v-for="param in pluginUIConfig.customParams" :key="param.key">
          <!-- 数字输入 -->
          <div v-if="param.type === 'number'" class="inference-slider">
            <span class="inference-slider__label">{{ getLocalizedText(param.label) }}</span>
            <input
              v-model.number="customParamValues[param.key]"
              type="number"
              :min="param.min"
              :max="param.max"
              :step="param.step || 1"
              class="inference-number-input"
            />
          </div>
          <!-- 下拉选择 -->
          <div v-else-if="param.type === 'select'" class="inference-select-row">
            <span class="inference-slider__label">{{ getLocalizedText(param.label) }}</span>
            <select v-model="customParamValues[param.key]" class="inference-select">
              <option
                v-for="opt in param.options"
                :key="String(opt.value)"
                :value="opt.value"
              >
                {{ typeof opt.label === 'string' ? opt.label : getLocalizedText(opt.label) }}
              </option>
            </select>
          </div>
          <!-- 开关 -->
          <div v-else-if="param.type === 'checkbox'" class="inference-checkbox-row">
            <label class="inference-checkbox-label">
              <input
                v-model="customParamValues[param.key]"
                type="checkbox"
                class="inference-checkbox"
              />
              <span>{{ getLocalizedText(param.label) }}</span>
            </label>
          </div>
        </template>
      </div>

      <!-- Tab 切换 -->
      <div class="inference-tabs">
        <button
          class="inference-tab"
          :class="{ 'inference-tab--active': activeTab === 'trained' }"
          @click="handleSelectTrainedTab"
        >
          {{ t('inference.trainedModels') }}
        </button>
        <button
          class="inference-tab"
          :class="{ 'inference-tab--active': activeTab === 'imported' }"
          @click="handleSelectImportedTab"
        >
          <span class="inference-tab__label">{{ t('inference.importedModels') }}</span>
          <RefreshCw
            :size="14"
            class="inference-tab__refresh-icon"
          />
        </button>
      </div>

      <!-- 模型列表（作为拖拽区域） -->
      <div
        class="inference-models"
        :class="{ 'inference-models--dragging': isDragging }"
        @dragover="handleDragOver"
        @dragleave="handleDragLeave"
        @drop="handleDrop"
      >
        <!-- 已训练模型 -->
        <template v-if="activeTab === 'trained'">
          <div v-if="trainedModels.length === 0" class="inference-models__empty">
            {{ t('inference.noTrainedModels') }}
          </div>
          <div
            v-for="model in trainedModels"
            :key="model.path"
            class="inference-model-item"
            :class="{ 'inference-model-item--selected': selectedModel?.path === model.path }"
            @click="selectModel(model)"
          >
            <Box :size="16" class="inference-model-item__icon" />
            <div class="inference-model-item__info">
              <span class="inference-model-item__name">{{ model.name }}</span>
              <span class="inference-model-item__time">{{ formatTime(model.createdAt) }}</span>
            </div>
            <Loader2
              v-if="selectedModel?.path === model.path && isInferring"
              :size="14"
              class="inference-model-item__loading animate-spin"
            />
            <Check
              v-else-if="selectedModel?.path === model.path"
              :size="14"
              class="inference-model-item__check"
            />
          </div>
        </template>

        <!-- 导入的模型 -->
        <template v-else>
          <div
            v-if="importedModels.length === 0"
            class="inference-models__empty inference-models__drop-hint"
            @dragover.prevent.stop="handleDragOver"
            @dragleave.stop="handleDragLeave"
            @drop.prevent.stop="handleDrop"
          >
            <Upload :size="24" />
            <span>{{ t('inference.dropModelHint') }}</span>
          </div>
          <div
            v-for="model in importedModels"
            :key="model.path"
            class="inference-model-item"
            :class="{ 'inference-model-item--selected': selectedModel?.path === model.path }"
            @click="selectModel(model)"
          >
            <Box :size="16" class="inference-model-item__icon" />
            <div class="inference-model-item__info">
              <span class="inference-model-item__name">{{ model.name }}</span>
              <span class="inference-model-item__time">{{ formatTime(model.createdAt) }}</span>
            </div>
            <Loader2
              v-if="selectedModel?.path === model.path && isInferring"
              :size="14"
              class="inference-model-item__loading animate-spin"
            />
            <Check
              v-else-if="selectedModel?.path === model.path"
              :size="14"
              class="inference-model-item__check"
            />
          </div>
        </template>
      </div>
      </template>
    </main>

    <!-- 悬浮 Toast 提示 -->
    <transition name="inference-toast">
      <div
        v-if="toastVisible"
        class="inference-toast"
        :class="`inference-toast--${toastType}`"
      >
        <CheckCircle v-if="toastType === 'success'" :size="16" />
        <AlertCircle v-else-if="toastType === 'warning'" :size="16" />
        <XCircle v-else :size="16" />
        <span class="inference-toast__message">{{ t(toastMessageKey) }}</span>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.inference-window {
  width: 100%;
  max-width: 340px;
  height: 100vh;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  position: relative;
  background: var(--color-bg-primary);
  color: var(--color-fg-primary);
  font-family: 'Inter', system-ui, sans-serif;
  overflow: hidden;
}

/* 标题栏 */
.inference-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 32px;
  padding: 0 8px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border-primary);
  -webkit-app-region: drag;
  flex-shrink: 0;
}

.inference-header__left {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-fg-primary);
}

.inference-header__title {
  font-size: 12px;
  font-weight: 500;
}

.inference-header__right {
  display: flex;
  align-items: center;
  gap: 2px;
  -webkit-app-region: no-drag;
}

.inference-header__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--color-fg-muted);
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.15s, color 0.15s;
}

.inference-header__btn:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-fg-primary);
}

.inference-header__btn--close:hover {
  background: #e81123;
  color: white;
}

/* 内容区域 */
.inference-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 8px;
  gap: 8px;
  overflow: hidden;
}

/* 自定义 UI iframe */
.inference-custom-iframe {
  width: 100%;
  height: 100%;
  flex: 1;
  border: none;
  border-radius: 6px;
  background: var(--color-bg-secondary);
}

/* 滑块区域：单行布局（标题 + 滑块 + 数值） */
.inference-sliders {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
  max-width: 250px;
}

.inference-slider {
  display: flex;
  align-items: center;
  gap: 4px;
}

.inference-slider__label {
  font-size: 10px;
  color: var(--color-fg-primary);
  font-weight: 500;
  white-space: nowrap;
  flex: 0 0 auto;
}

.inference-slider__value {
  font-size: 10px;
  font-weight: 600;
  color: var(--color-accent);
  flex: 0 0 40px;
  text-align: right;
}

.inference-slider__input {
  flex: 1 1 auto;
  min-width: 0;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: var(--color-bg-tertiary);
  border-radius: 2px;
  cursor: pointer;
}

.inference-slider__input::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  background: var(--color-accent);
  border-radius: 50%;
  cursor: pointer;
  border: 2px solid white;
  box-shadow: 0 1px 2px rgba(0,0,0,0.3);
}

/* 自定义参数区域 */
.inference-custom-params {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
  max-width: 250px;
}

.inference-number-input {
  width: 60px;
  padding: 2px 4px;
  font-size: 10px;
  border: 1px solid var(--color-border-primary);
  border-radius: 3px;
  background: var(--color-bg-secondary);
  color: var(--color-fg-primary);
  text-align: center;
}

.inference-number-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.inference-select-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.inference-select {
  flex: 1;
  padding: 2px 4px;
  font-size: 10px;
  border: 1px solid var(--color-border-primary);
  border-radius: 3px;
  background: var(--color-bg-secondary);
  color: var(--color-fg-primary);
  cursor: pointer;
}

.inference-select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.inference-checkbox-row {
  display: flex;
  align-items: center;
}

.inference-checkbox-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  color: var(--color-fg-primary);
  cursor: pointer;
}

.inference-checkbox {
  width: 14px;
  height: 14px;
  cursor: pointer;
}

/* Tab 切换 */
.inference-tabs {
  display: flex;
  gap: 2px;
  padding: 2px;
  background: var(--color-bg-tertiary);
  border-radius: 4px;
  flex-shrink: 0;
}

.inference-tab {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 4px 6px;
  font-size: 10px;
  border: none;
  background: transparent;
  color: var(--color-fg-muted);
  cursor: pointer;
  border-radius: 3px;
  transition: background 0.15s, color 0.15s;
}

.inference-tab__label {
  flex: 1 1 auto;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.inference-tab__refresh-icon {
  flex: 0 0 auto;
}

.inference-tab:hover {
  background: var(--color-bg-secondary);
}

.inference-tab--active {
  background: var(--color-accent);
  color: white;
  font-weight: 500;
}

.inference-tab--active:hover {
  background: var(--color-accent);
  color: white;
}

/* 模型列表 */
.inference-models {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  border: 1px solid var(--color-border-primary);
  border-radius: 4px;
  background: var(--color-bg-secondary);
  /* 自定义细滚动条，替代默认样式 */
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-subtle) transparent;
}

.inference-models::-webkit-scrollbar {
  width: 6px;
}

.inference-models::-webkit-scrollbar-track {
  background: transparent;
}

.inference-models::-webkit-scrollbar-thumb {
  background-color: rgba(148, 163, 184, 0.7);
  border-radius: 999px;
}

.inference-models--dragging {
  border-color: var(--color-accent);
  background: var(--color-accent-subtle);
}

.inference-models__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 6px;
  padding: 12px;
  color: var(--color-fg-muted);
  font-size: 10px;
  text-align: center;
}

.inference-models__drop-hint {
  padding: 12px;
  text-align: center;
}

.inference-model-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--color-border-subtle);
  cursor: pointer;
  transition: background 0.15s;
}

.inference-model-item:hover {
  background: var(--color-bg-tertiary);
}

.inference-model-item:last-child {
  border-bottom: none;
}

.inference-model-item__icon {
  color: var(--color-accent);
  flex-shrink: 0;
}

.inference-model-item__info {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.inference-model-item__name {
  font-size: 10px;
  font-weight: 500;
  color: var(--color-fg-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inference-model-item__time {
  font-size: 9px;
  color: var(--color-fg-muted);
}

.inference-model-item--selected {
  background: var(--color-accent-subtle);
  border-left: 2px solid var(--color-accent);
}

.inference-model-item--selected:hover {
  background: var(--color-accent-subtle);
}

.inference-model-item__check {
  color: var(--color-accent);
  flex-shrink: 0;
}

.inference-model-item__loading {
  color: var(--color-accent);
  flex-shrink: 0;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 悬浮 Toast 提示 */
.inference-toast {
  position: absolute;
  left: 50%;
  bottom: 16px;
  transform: translateX(-50%);
  max-width: 90%;
  padding: 6px 10px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.35);
  z-index: 50;
}

.inference-toast--success {
  background: rgba(22, 163, 74, 0.96);
  color: #ecfdf5;
}

.inference-toast--warning {
  background: rgba(234, 179, 8, 0.96);
  color: #fefce8;
}

.inference-toast--error {
  background: rgba(239, 68, 68, 0.96);
  color: #fef2f2;
}

.inference-toast__message {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.inference-toast-enter-active,
.inference-toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.inference-toast-enter-from,
.inference-toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(8px);
}

.inference-toast-enter-to,
.inference-toast-leave-from {
  opacity: 1;
  transform: translateX(-50%) translateY(0);
}
</style>
