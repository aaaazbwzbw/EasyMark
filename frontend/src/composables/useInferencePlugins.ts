/**
 * 推理插件管理 Composable
 * Inference Plugin Management Composable
 */
import { ref, computed } from 'vue'
import type { 
  InferencePlugin, 
  InferenceServiceStatus, 
  InferenceParams,
  InferenceResult,
  InferenceUIConfig,
  InferenceAnnotation
} from '../types/inference-plugin'

const API_BASE = 'http://localhost:18080'

// 全局状态
const inferencePlugins = ref<InferencePlugin[]>([])
const activePluginId = ref<string | null>(null)
const isLoading = ref(false)
const serviceStatus = ref<InferenceServiceStatus>({
  running: false,
  activePluginId: null,
  modelLoaded: false,
  modelPath: null
})

// 推理参数
const inferenceParams = ref<InferenceParams>({
  conf: 0.25,
  iou: 0.45
})

// 自动推理开关
const autoInference = ref(true)

// 当前激活的插件
const activePlugin = computed(() => {
  if (!activePluginId.value) return null
  return inferencePlugins.value.find(p => p.id === activePluginId.value) || null
})

// 获取插件的 UI 配置
const activePluginUI = computed<InferenceUIConfig | null>(() => {
  return activePlugin.value?.ui || null
})

/**
 * 加载推理插件列表
 */
async function loadPlugins(): Promise<void> {
  isLoading.value = true
  try {
    const resp = await fetch(`${API_BASE}/api/plugins?type=inference`)
    if (resp.ok) {
      const data = await resp.json()
      inferencePlugins.value = (data.plugins || []).map((p: any) => ({
        id: p.id,
        name: p.name,
        version: p.version,
        description: p.description || '',
        icon: p.inference?.icon || null,
        defaultIcon: p.inference?.defaultIcon || 'brain',
        pluginPath: p.pluginPath || '',
        serviceEntry: p.inference?.serviceEntry || '',
        supportedTasks: p.inference?.supportedTasks || [],
        interactionMode: p.inference?.interactionMode || 'auto',
        ui: p.inference?.ui || {},
        isActive: false,
        isLoaded: false
      }))
    }
  } catch (e) {
    console.error('Failed to load inference plugins:', e)
  } finally {
    isLoading.value = false
  }
}

/**
 * 激活推理插件
 */
async function activatePlugin(pluginId: string): Promise<boolean> {
  // 如果已经是当前插件，则切换关闭
  if (activePluginId.value === pluginId) {
    await deactivatePlugin()
    return false
  }

  // 先停止当前插件
  if (activePluginId.value) {
    await deactivatePlugin()
  }

  try {
    // 启动推理服务（传递插件 ID）
    const result = await window.electronAPI?.inferenceStartService?.(pluginId)
    if (!result?.success) {
      console.error('Failed to start inference service')
      return false
    }

    activePluginId.value = pluginId
    serviceStatus.value.running = true
    serviceStatus.value.activePluginId = pluginId

    // 更新插件状态
    const plugin = inferencePlugins.value.find(p => p.id === pluginId)
    if (plugin) {
      plugin.isActive = true
    }

    return true
  } catch (e) {
    console.error('Failed to activate plugin:', e)
    return false
  }
}

/**
 * 停用当前推理插件
 */
async function deactivatePlugin(): Promise<void> {
  if (!activePluginId.value) return

  try {
    await window.electronAPI?.inferenceStopService?.()
  } catch (e) {
    console.error('Failed to stop inference service:', e)
  }

  // 更新插件状态
  const plugin = inferencePlugins.value.find(p => p.id === activePluginId.value)
  if (plugin) {
    plugin.isActive = false
    plugin.isLoaded = false
  }

  activePluginId.value = null
  serviceStatus.value = {
    running: false,
    activePluginId: null,
    modelLoaded: false,
    modelPath: null
  }
}

/**
 * 加载模型
 */
async function loadModel(modelPath: string): Promise<boolean> {
  if (!activePluginId.value) {
    console.error('No active plugin')
    return false
  }

  try {
    const result = await window.electronAPI?.inferenceLoadModel?.(modelPath)
    if (result?.success) {
      serviceStatus.value.modelLoaded = true
      serviceStatus.value.modelPath = modelPath

      const plugin = inferencePlugins.value.find(p => p.id === activePluginId.value)
      if (plugin) {
        plugin.isLoaded = true
      }
      return true
    }
  } catch (e) {
    console.error('Failed to load model:', e)
  }
  return false
}

/**
 * 执行推理
 */
async function runInference(payload: {
  imagePath?: string
  projectId?: string
  path?: string
}): Promise<InferenceResult> {
  if (!activePluginId.value || !serviceStatus.value.modelLoaded) {
    return {
      success: false,
      error: 'No model loaded',
      annotations: []
    }
  }

  try {
    const inferPayload = {
      projectId: payload.projectId || '',
      path: payload.path || '',
      imagePath: payload.imagePath,
      conf: inferenceParams.value.conf,
      iou: inferenceParams.value.iou
    }
    const result = await window.electronAPI?.inferenceRun?.(inferPayload as any)

    return {
      success: result?.success || false,
      error: result?.error,
      annotations: (result?.annotations || []) as InferenceAnnotation[],
      inferTimeMs: result?.inferTimeMs
    }
  } catch (e) {
    return {
      success: false,
      error: String(e),
      annotations: []
    }
  }
}

/**
 * 更新推理参数
 */
function updateParams(params: Partial<InferenceParams>): void {
  inferenceParams.value = { ...inferenceParams.value, ...params }
}

/**
 * 获取插件显示名称（根据语言）
 */
function getPluginName(plugin: InferencePlugin, locale: string): string {
  if (typeof plugin.name === 'string') {
    return plugin.name
  }
  return (plugin.name as any)?.[locale] || (plugin.name as any)?.['en-US'] || plugin.id
}

export function useInferencePlugins() {
  return {
    // 状态
    inferencePlugins,
    activePluginId,
    activePlugin,
    activePluginUI,
    isLoading,
    serviceStatus,
    inferenceParams,
    autoInference,

    // 方法
    loadPlugins,
    activatePlugin,
    deactivatePlugin,
    loadModel,
    runInference,
    updateParams,
    getPluginName
  }
}
