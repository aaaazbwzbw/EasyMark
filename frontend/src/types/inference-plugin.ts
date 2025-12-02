/**
 * 推理插件类型定义
 * Inference Plugin Type Definitions
 */

// 国际化文本
export interface I18nText {
  'zh-CN'?: string
  'en-US'?: string
  [key: string]: string | undefined
}

// 自定义参数选项
export interface ParamOption {
  value: string | number | boolean
  label: string | I18nText
}

// 自定义参数定义
export interface CustomParam {
  key: string
  type: 'select' | 'number' | 'slider' | 'checkbox' | 'text'
  label: string | I18nText
  default?: string | number | boolean
  // select 类型
  options?: ParamOption[]
  // number/slider 类型
  min?: number
  max?: number
  step?: number
  // text 类型
  maxLength?: number
  placeholder?: string | I18nText
}

// 滑块配置
export interface SliderConfig {
  default: number
  min: number
  max: number
  step?: number
}

// UI 配置
export interface InferenceUIConfig {
  modelSelector?: boolean
  confidenceSlider?: SliderConfig | boolean
  iouSlider?: SliderConfig | boolean
  autoInfer?: { default: boolean } | boolean
  customParams?: CustomParam[]
}

// 推理插件配置
export interface InferenceConfig {
  icon?: string
  defaultIcon?: string
  serviceEntry: string
  serviceType: 'stdio'
  supportedTasks: string[]
  ui?: InferenceUIConfig
}

// Python 依赖配置
export interface PythonConfig {
  minVersion?: string
  requirements?: string[]
  torchRequirements?: {
    packages: string[]
    indexUrl?: string
    description?: string
  }
}

// 推理插件 Manifest
export interface InferencePluginManifest {
  id: string
  name: string | I18nText
  version: string
  type: 'inference'
  description?: string | I18nText
  author?: string
  inference: InferenceConfig
  python?: PythonConfig
}

// 运行时推理插件信息
export interface InferencePlugin {
  id: string
  name: string
  version: string
  description: string
  icon: string | null
  defaultIcon: string
  pluginPath: string
  serviceEntry: string
  supportedTasks: string[]
  interactionMode?: 'auto' | 'prompt' | 'box' | 'text'  // auto=自动推理, prompt=需要用户提示点, box=需要用户框选示例, text=文本提示检测
  ui: InferenceUIConfig
  isActive: boolean
  isLoaded: boolean
}

// 推理服务状态
export interface InferenceServiceStatus {
  running: boolean
  activePluginId: string | null
  modelLoaded: boolean
  modelPath: string | null
}

// 推理参数
export interface InferenceParams {
  conf: number
  iou: number
  [key: string]: unknown
}

// 推理结果标注
export interface InferenceAnnotation {
  type: 'bbox' | 'polygon'
  categoryName: string
  confidence: number
  data: {
    x?: number
    y?: number
    width?: number
    height?: number
    points?: [number, number][]
    keypoints?: [number, number, number][]
  }
}

// 推理结果
export interface InferenceResult {
  success: boolean
  error?: string
  annotations: InferenceAnnotation[]
  inferTimeMs?: number
}

// 系统内置图标映射
export const BUILTIN_ICONS: Record<string, string> = {
  brain: 'Brain',
  box: 'Box',
  scan: 'Scan',
  wand: 'Wand2',
  sparkles: 'Sparkles',
  cpu: 'Cpu',
  zap: 'Zap'
}
