<template>
  <div class="python-env-view">
    <!-- 左侧栏：基于 Python 的插件列表 -->
    <aside class="env-sidebar">
      <header class="sidebar-header">
        <h2 class="sidebar-title">{{ t('pythonEnv.plugins') }}</h2>
      </header>
      <div class="sidebar-body">
        <div v-if="loadingPlugins" class="sidebar-loading">
          <Loader2 :size="16" class="animate-spin" />
        </div>
        <div v-else-if="pythonPlugins.length === 0" class="sidebar-empty">
          {{ t('pythonEnv.noPlugins') }}
        </div>
        <div v-else class="plugin-list">
          <div 
            v-for="plugin in pythonPlugins" 
            :key="plugin.id"
            class="plugin-item"
            :class="{ 'plugin-item--active': selectedPlugin?.id === plugin.id }"
            @click="selectPlugin(plugin)"
          >
            <div class="plugin-item-info">
              <span class="plugin-item-name">{{ getLocalizedText(plugin.name) }}</span>
              <span class="plugin-item-version">v{{ plugin.version }}</span>
            </div>
            <!-- 依赖状态 -->
            <div class="plugin-item-status">
              <span 
                v-if="getPluginDepsStatus(plugin.id) === 'loading'" 
                class="status-badge status-badge--loading"
              >
                <Loader2 :size="12" class="animate-spin" />
              </span>
              <span 
                v-else-if="getPluginDepsStatus(plugin.id) === 'ready'" 
                class="status-badge status-badge--ready"
                :title="t('pythonEnv.depsReady')"
              >
                <CheckCircle :size="12" />
              </span>
              <span 
                v-else-if="getMissingDepsCount(plugin.id) > 0" 
                class="status-badge status-badge--missing"
                :title="t('pythonEnv.depsMissing', { count: getMissingDepsCount(plugin.id) })"
              >
                <AlertTriangle :size="12" />
                <span>{{ getMissingDepsCount(plugin.id) }}</span>
              </span>
              <span 
                v-else 
                class="status-badge status-badge--no-venv"
                :title="t('pythonEnv.noVenv')"
              >
                <Circle :size="12" />
              </span>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="env-main">
      <!-- 顶部栏：Python 版本信息 -->
      <header class="env-header">
        <div class="header-left">
          <div class="python-info">
            <span class="python-label">Python</span>
            <span v-if="loadingPython" class="python-version">
              <Loader2 :size="14" class="animate-spin" />
            </span>
            <span v-else-if="pythonVersion" class="python-version python-version--ok">
              {{ pythonVersion }}
            </span>
            <span v-else class="python-version python-version--missing">
              {{ t('pythonEnv.notInstalled') }}
            </span>
          </div>
          <button 
            v-if="!pythonVersion && !loadingPython" 
            class="btn-primary"
            @click="showPythonDownloadDialog"
          >
            <Download :size="14" />
            {{ t('pythonEnv.downloadPython') }}
          </button>
        </div>
        <div class="header-right">
          <button class="btn-icon" @click="showSettingsDialog = true" :title="t('pythonEnv.settings')">
            <Settings :size="18" />
          </button>
        </div>
      </header>

      <!-- 中间内容区：选中的插件详情 -->
      <div class="env-content">
        <div v-if="!selectedPlugin" class="env-empty">
          <Package :size="48" class="env-empty-icon" />
          <p>{{ t('pythonEnv.selectPluginHint') }}</p>
        </div>
        <div v-else class="plugin-detail">
          <div class="detail-header">
            <h3 class="detail-name">{{ getLocalizedText(selectedPlugin.name) }}</h3>
            <span class="detail-version">v{{ selectedPlugin.version }}</span>
          </div>
          
          <!-- 虚拟环境状态 -->
          <div class="detail-section">
            <h4 class="section-title">{{ t('pythonEnv.venvStatus') }}</h4>
            <div class="venv-status">
              <div class="venv-path">
                <span class="label">{{ t('pythonEnv.venvPath') }}:</span>
                <code>{{ getVenvPath(selectedPlugin.id) }}</code>
              </div>
              <div class="venv-actions">
                <button 
                  v-if="!hasVenv(selectedPlugin.id)"
                  class="btn-primary"
                  :disabled="creatingVenv"
                  @click="createVenv(selectedPlugin)"
                >
                  <Loader2 v-if="creatingVenv" :size="14" class="animate-spin" />
                  <FolderPlus v-else :size="14" />
                  {{ t('pythonEnv.createVenv') }}
                </button>
                <button 
                  v-else
                  class="btn-secondary"
                  :disabled="deletingVenv"
                  @click="deleteVenv(selectedPlugin)"
                >
                  <Loader2 v-if="deletingVenv" :size="14" class="animate-spin" />
                  <Trash2 v-else :size="14" />
                  {{ t('pythonEnv.deleteVenv') }}
                </button>
              </div>
            </div>
          </div>

          <!-- 依赖列表 -->
          <div class="detail-section">
            <h4 class="section-title">{{ t('pythonEnv.dependencies') }}</h4>
            <div v-if="!selectedPluginDeps" class="deps-empty">
              {{ t('pythonEnv.noDeps') }}
            </div>
            <div v-else class="deps-list">
              <div 
                v-for="dep in selectedPluginDeps" 
                :key="dep.name"
                class="dep-item"
                :class="{ 
                  'dep-item--missing': !dep.installed,
                  'dep-item--pytorch': dep.isPytorch
                }"
              >
                <div class="dep-left">
                  <span class="dep-name">{{ dep.name }}</span>
                  <!-- PyTorch GPU 不兼容警告 -->
                  <span v-if="dep.isPytorch && selectedPlugin.python?.pytorch?.gpu && !isGpuCompatible(selectedPlugin)" class="dep-warning">
                    <AlertTriangle :size="12" />
                    {{ t('pythonEnv.gpuNotCompatible', { minCuda: selectedPlugin.python.pytorch.gpu.minCuda }) }}
                  </span>
                </div>
                <div class="dep-right">
                  <!-- PyTorch 特殊处理 -->
                  <template v-if="dep.isPytorch">
                    <span v-if="dep.installed" class="dep-status dep-status--ok">
                      <CheckCircle :size="12" />
                      {{ dep.pytorchType?.toUpperCase() || t('pythonEnv.installed') }}
                    </span>
                    <span v-else class="dep-status dep-status--missing">
                      {{ t('pythonEnv.notInstalled') }}
                    </span>
                    <!-- PyTorch 版本选择/切换按钮 -->
                    <div v-if="hasVenv(selectedPlugin.id)" class="pytorch-inline-actions">
                      <button 
                        v-if="supportsPytorchVersion(selectedPlugin, 'gpu')"
                        class="btn-pytorch-small"
                        :class="{ 'btn-pytorch-small--active': dep.pytorchType === 'gpu' }"
                        :disabled="installingPytorch || !isGpuCompatible(selectedPlugin)"
                        @click="handlePytorchVersionClick('gpu', dep.installed)"
                      >
                        <Loader2 v-if="installingPytorch" :size="12" class="animate-spin" />
                        <span v-else>GPU</span>
                      </button>
                      <button 
                        v-if="supportsPytorchVersion(selectedPlugin, 'cpu')"
                        class="btn-pytorch-small"
                        :class="{ 'btn-pytorch-small--active': dep.pytorchType === 'cpu' }"
                        :disabled="installingPytorch"
                        @click="handlePytorchVersionClick('cpu', dep.installed)"
                      >
                        <Loader2 v-if="installingPytorch" :size="12" class="animate-spin" />
                        <span v-else>CPU</span>
                      </button>
                    </div>
                  </template>
                  <!-- 普通依赖 -->
                  <template v-else>
                    <span v-if="dep.installed" class="dep-status dep-status--ok">
                      <CheckCircle :size="12" />
                      {{ dep.version || t('pythonEnv.installed') }}
                    </span>
                    <span v-else class="dep-status dep-status--missing">
                      {{ t('pythonEnv.notInstalled') }}
                    </span>
                    <button 
                      v-if="dep.installed"
                      class="btn-icon-small btn-uninstall"
                      :disabled="uninstallingDep === dep.name"
                      :title="t('pythonEnv.uninstallDep')"
                      @click="uninstallDep(dep.name)"
                    >
                      <Loader2 v-if="uninstallingDep === dep.name" :size="12" class="animate-spin" />
                      <Trash2 v-else :size="12" />
                    </button>
                  </template>
                </div>
              </div>
            </div>
            <div v-if="hasVenv(selectedPlugin.id) && (hasMissingDeps(selectedPlugin.id) || installingDeps)" class="deps-actions">
              <button 
                v-if="!installingDeps"
                class="btn-primary"
                @click="installDeps(selectedPlugin)"
              >
                <Download :size="14" />
                {{ t('pythonEnv.installDeps') }}
              </button>
              <button 
                v-else
                class="btn-danger"
                @click="stopInstall(selectedPlugin)"
              >
                <StopCircle :size="14" />
                {{ t('pythonEnv.stopInstall') }}
              </button>
            </div>
          </div>

          <!-- GPU 状态信息 -->
          <div v-if="hasPytorchConfig(selectedPlugin)" class="detail-section">
            <h4 class="section-title">{{ t('pythonEnv.gpuStatus') }}</h4>
            <div class="gpu-status-info">
              <span v-if="gpuInfo.hasNvidia" class="gpu-ok">
                <CheckCircle :size="12" />
                {{ gpuInfo.gpuName }} (CUDA {{ gpuInfo.cudaVersion }})
              </span>
              <span v-else class="gpu-none">
                {{ t('pythonEnv.noGpu') }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部：终端面板 -->
      <div class="env-terminal" :style="{ height: terminalHeight + 'px' }">
        <div class="terminal-resize-handle" @mousedown="startResizeTerminal"></div>
        <div class="terminal-header">
          <span class="terminal-title">
            <Terminal :size="14" />
            {{ t('pythonEnv.terminal') }}
            <Loader2 v-if="installingDeps || installingPytorch" :size="14" class="animate-spin terminal-loading" />
          </span>
          <div class="terminal-actions">
            <button class="btn-icon" @click="copyTerminalContent" :title="t('training.logs.copy')">
              <Copy :size="14" />
            </button>
            <button class="btn-icon" @click="clearTerminal" :title="t('training.logs.clear')">
              <Trash2 :size="14" />
            </button>
          </div>
        </div>
        <div ref="terminalContainerRef" class="terminal-container"></div>
        <!-- 终端命令输入框 -->
        <div v-if="selectedPlugin && hasVenv(selectedPlugin.id)" class="terminal-input-bar">
          <input
            v-model="customCommand"
            type="text"
            class="terminal-input"
            :placeholder="t('pythonEnv.customCommandPlaceholder')"
            :disabled="runningCommand"
            @keyup.enter="runCustomCommand"
          />
          <button 
            class="btn-run-command"
            :disabled="runningCommand || !customCommand.trim()"
            @click="runCustomCommand"
          >
            <Loader2 v-if="runningCommand" :size="14" class="animate-spin" />
            <Play v-else :size="14" />
          </button>
        </div>
      </div>
    </main>

    <!-- Python 下载引导对话框 -->
    <div v-if="showPythonDialog" class="modal-overlay" @click.self="showPythonDialog = false">
      <div class="modal">
        <h2 class="modal-title">{{ t('pythonEnv.downloadPythonTitle') }}</h2>
        <div class="modal-content">
          <p>{{ t('pythonEnv.downloadPythonDesc') }}</p>
          <div class="modal-tips">
            <AlertTriangle :size="16" class="tip-icon" />
            <div class="tip-content">
              <p><strong>{{ t('pythonEnv.importantTips') }}</strong></p>
              <ul>
                <li>{{ t('pythonEnv.tip1') }}</li>
                <li>{{ t('pythonEnv.tip2') }}</li>
              </ul>
            </div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showPythonDialog = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary" @click="openPythonDownload">
            <ExternalLink :size="14" />
            {{ t('pythonEnv.openDownload') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 设置对话框 -->
    <div v-if="showSettingsDialog" class="modal-overlay" @click.self="showSettingsDialog = false">
      <div class="modal">
        <h2 class="modal-title">{{ t('pythonEnv.settingsTitle') }}</h2>
        <div class="modal-content">
          <!-- 默认下载源 -->
          <div class="form-group">
            <label class="form-label">{{ t('pythonEnv.pipSource') }}</label>
            <select v-model="pipSource" class="form-select">
              <option value="">{{ t('pythonEnv.pipSourceDefault') }}</option>
              <option value="https://pypi.tuna.tsinghua.edu.cn/simple">{{ t('pythonEnv.pipSourceTsinghua') }}</option>
              <option value="https://mirrors.aliyun.com/pypi/simple">{{ t('pythonEnv.pipSourceAliyun') }}</option>
              <option value="https://pypi.douban.com/simple">{{ t('pythonEnv.pipSourceDouban') }}</option>
            </select>
          </div>
          <!-- HTTP 代理 -->
          <div class="form-group">
            <label class="form-label">{{ t('pythonEnv.httpProxy') }}</label>
            <input 
              type="text" 
              v-model="httpProxy" 
              class="form-input"
              placeholder="http://127.0.0.1:7890"
            />
          </div>
          <!-- SOCKS5 代理 -->
          <div class="form-group">
            <label class="form-label">{{ t('pythonEnv.socks5Proxy') }}</label>
            <input 
              type="text" 
              v-model="socks5Proxy" 
              class="form-input"
              placeholder="socks5://127.0.0.1:1080"
            />
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showSettingsDialog = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary" @click="saveSettings">
            {{ t('common.save') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, shallowRef } from 'vue'
import { 
  Loader2, CheckCircle, AlertTriangle, Circle, Download, Settings, Package, 
  FolderPlus, Trash2, Terminal, Copy, ExternalLink, Play, StopCircle 
} from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import notification from '../utils/notification'
import { useGlobalWs } from '../composables/useGlobalWs'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const { t, locale } = useI18n()
const { subscribe } = useGlobalWs()

// ============ 类型定义 ============
type PytorchConfig = {
  packages: string[]  // 如 ["torch>=2.0.0", "torchvision>=0.15.0"]
  gpu?: { 
    minCuda: string   // 如 "11.8"
    indexUrl: string 
  }
  cpu?: { 
    indexUrl: string 
  }
}

type Plugin = {
  id: string
  name: string | Record<string, string>
  version: string
  type: string
  description: string | Record<string, string>
  python?: {
    dependencies?: string[]
    requirements?: string[]
    pytorch?: PytorchConfig
  }
}

type DependencyInfo = {
  name: string
  installed: boolean
  version?: string
  isPytorch?: boolean       // 是否是 PyTorch 依赖
  pytorchType?: 'gpu' | 'cpu'  // PyTorch 类型
}

type GpuInfo = {
  hasNvidia: boolean
  cudaVersion: string
  gpuName: string
}

type PluginEnvStatus = {
  hasVenv: boolean
  dependencies: DependencyInfo[]
  pytorchInstalled: boolean
  pytorchType?: 'gpu' | 'cpu'  // 当前安装的类型
  loading: boolean
}

// ============ 状态 ============
const loadingPlugins = ref(true)
const loadingPython = ref(true)
const pythonVersion = ref<string | null>(null)
const pythonPlugins = ref<Plugin[]>([])
const selectedPlugin = ref<Plugin | null>(null)
const pluginEnvStatus = ref<Record<string, PluginEnvStatus>>({})

// GPU 信息
const gpuInfo = ref<GpuInfo>({ hasNvidia: false, cudaVersion: '', gpuName: '' })
const installingPytorch = ref(false)
const customPytorch = ref('')

// 推荐的 PyTorch 版本（仅用于展示和作为自定义输入参考，不强绑定插件）
const recommendedPytorch = {
  gpu: {
    packages: ['torch==2.2.2+cu118', 'torchvision==0.17.2+cu118'],
    indexUrl: 'https://download.pytorch.org/whl/cu118',
    get display() {
      return this.packages.join(' ')
    }
  },
  cpu: {
    packages: ['torch==2.2.2', 'torchvision==0.17.2'],
    indexUrl: 'https://download.pytorch.org/whl/cpu',
    get display() {
      return this.packages.join(' ')
    }
  }
} as const

// 操作状态
const creatingVenv = ref(false)
const deletingVenv = ref(false)
const installingDeps = ref(false)
const uninstallingDep = ref<string | null>(null)
const customCommand = ref('')
const runningCommand = ref(false)

// 正在安装的插件列表（用于页面刷新后恢复状态）
const installingPluginIds = ref<string[]>([])

// PyTorch 选中的版本（未安装时用于记录用户选择）
const selectedPytorchVersion = ref<Record<string, 'gpu' | 'cpu'>>({})

// 对话框
const showPythonDialog = ref(false)
const showSettingsDialog = ref(false)

// 设置
const pipSource = ref('')
const httpProxy = ref('')
const socks5Proxy = ref('')

// 终端
const terminalHeight = ref(200)
const terminalContainerRef = ref<HTMLElement | null>(null)
const terminal = shallowRef<XTerm | null>(null)
const fitAddon = shallowRef<FitAddon | null>(null)
const isResizingTerminal = ref(false)

// 终端输出缓存 (sessionStorage key)
const TERMINAL_CACHE_KEY = 'pythonEnv_terminal_cache'

// 终端主题
const terminalThemes: Record<string, any> = {
  dark: {
    background: '#0d0d0d',
    foreground: '#c8c8c8',
    cursor: '#c8c8c8',
    cursorAccent: '#0d0d0d',
    selectionBackground: 'rgba(255, 255, 255, 0.3)'
  },
  light: {
    background: '#f5f5f5',
    foreground: '#383a42',
    cursor: '#383a42',
    cursorAccent: '#f5f5f5',
    selectionBackground: 'rgba(0, 0, 0, 0.2)'
  }
}

// ============ 计算属性 ============
const selectedPluginDeps = computed<DependencyInfo[] | null>(() => {
  if (!selectedPlugin.value) return null
  const status = pluginEnvStatus.value[selectedPlugin.value.id]
  const deps = status?.dependencies || []
  
  // 如果插件需要 PyTorch，将其添加到依赖列表开头
  if (hasPytorchConfig(selectedPlugin.value)) {
    const pytorchPackages = selectedPlugin.value.python?.pytorch?.packages || []
    const pytorchName = pytorchPackages.join(' + ') || 'PyTorch'
    const gpuCompatible = isGpuCompatible(selectedPlugin.value)
    const minCuda = selectedPlugin.value.python?.pytorch?.gpu?.minCuda
    const pluginId = selectedPlugin.value.id
    
    // 已安装时显示实际类型，未安装时显示选中的类型
    const installedType = status?.pytorchType
    const selectedType = selectedPytorchVersion.value[pluginId] || (gpuCompatible ? 'gpu' : 'cpu')
    const displayType = installedType || selectedType
    
    const pytorchDep: DependencyInfo = {
      name: pytorchName + (displayType === 'gpu' && minCuda ? ` (CUDA ${minCuda}+)` : ' (CPU)'),
      installed: status?.pytorchInstalled || false,
      version: installedType?.toUpperCase(),
      isPytorch: true,
      pytorchType: installedType || selectedType
    }
    return [pytorchDep, ...deps]
  }
  
  return deps.length > 0 ? deps : null
})

// ============ 工具函数 ============
const getLocalizedText = (text: string | Record<string, string>): string => {
  if (typeof text === 'string') return text
  if (!text) return ''
  return text[locale.value] || text['zh-CN'] || text['en-US'] || Object.values(text)[0] || ''
}

const getPluginDepsStatus = (pluginId: string): 'loading' | 'ready' | 'missing' | 'no-venv' => {
  const status = pluginEnvStatus.value[pluginId]
  if (!status) return 'no-venv'
  if (status.loading) return 'loading'
  if (!status.hasVenv) return 'no-venv'
  const missingCount = status.dependencies.filter(d => !d.installed).length
  return missingCount > 0 ? 'missing' : 'ready'
}

const getMissingDepsCount = (pluginId: string): number => {
  const status = pluginEnvStatus.value[pluginId]
  if (!status) return 0
  return status.dependencies.filter(d => !d.installed).length
}

const hasVenv = (pluginId: string): boolean => {
  return pluginEnvStatus.value[pluginId]?.hasVenv || false
}

const hasMissingDeps = (pluginId: string): boolean => {
  return getMissingDepsCount(pluginId) > 0
}

// 存储实际路径
const dataPath = ref('')

const getVenvPath = (pluginId: string): string => {
  if (!dataPath.value) return `plugins_python_venv/${pluginId}`
  return `${dataPath.value}/plugins_python_venv/${pluginId}`
}

// ============ API 调用 ============
const checkPythonVersion = async () => {
  loadingPython.value = true
  try {
    const res = await fetch('http://localhost:18080/api/python/version')
    if (res.ok) {
      const data = await res.json()
      pythonVersion.value = data.version || null
    }
  } catch (e) {
    console.error('Check Python version error:', e)
    pythonVersion.value = null
  } finally {
    loadingPython.value = false
  }
}

const loadPythonPlugins = async () => {
  loadingPlugins.value = true
  try {
    const res = await fetch('http://localhost:18080/api/plugins')
    if (res.ok) {
      const data = await res.json()
      // 过滤出有 python 配置的插件（支持 dependencies 或 requirements）
      pythonPlugins.value = (data.plugins || []).filter((p: Plugin) => {
        const py = p.python as any
        return py?.dependencies?.length || py?.requirements?.length
      })
    }
  } catch (e) {
    console.error('Load plugins error:', e)
  } finally {
    loadingPlugins.value = false
  }
}

const checkPluginEnvStatus = async (plugin: Plugin) => {
  pluginEnvStatus.value[plugin.id] = {
    hasVenv: false,
    dependencies: [],
    pytorchInstalled: false,
    loading: true
  }
  
  try {
    const res = await fetch(`http://localhost:18080/api/python/env-status?pluginId=${encodeURIComponent(plugin.id)}`)
    if (res.ok) {
      const data = await res.json()
      const deps = data.dependencies || []
      pluginEnvStatus.value[plugin.id] = {
        hasVenv: data.hasVenv || false,
        dependencies: deps,
        pytorchInstalled: data.pytorchInstalled || false,
        pytorchType: data.pytorchType || undefined,
        loading: false
      }
    }
  } catch (e) {
    console.error('Check plugin env status error:', e)
    pluginEnvStatus.value[plugin.id] = { hasVenv: false, dependencies: [], pytorchInstalled: false, loading: false }
  }
}

const checkAllPluginEnvStatus = async () => {
  for (const plugin of pythonPlugins.value) {
    await checkPluginEnvStatus(plugin)
  }
}

// ============ 操作函数 ============
const selectPlugin = (plugin: Plugin) => {
  selectedPlugin.value = plugin
  // 检查该插件是否正在安装中（用于页面刷新后恢复状态）
  if (installingPluginIds.value.includes(plugin.id)) {
    installingDeps.value = true
    installingPytorch.value = true
  } else {
    installingDeps.value = false
    installingPytorch.value = false
  }
}

const showPythonDownloadDialog = () => {
  showPythonDialog.value = true
}

const openPythonDownload = () => {
  window.electronAPI?.openExternal?.('https://www.python.org/ftp/python/3.11.0/python-3.11.0-amd64.exe')
  showPythonDialog.value = false
}

const createVenv = async (plugin: Plugin) => {
  creatingVenv.value = true
  appendLog(t('pythonEnv.creatingVenv', { name: getLocalizedText(plugin.name) }), 'info')
  
  try {
    const res = await fetch('http://localhost:18080/api/python/create-venv', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pluginId: plugin.id })
    })
    
    if (!res.ok) {
      throw new Error('Create venv failed')
    }
    
    // 读取流式响应，等待完成
    const reader = res.body?.getReader()
    const decoder = new TextDecoder()
    let success = false
    
    if (reader) {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        
        const text = decoder.decode(value)
        const lines = text.split('\n').filter(Boolean)
        for (const line of lines) {
          try {
            const msg = JSON.parse(line)
            if (msg.type === 'output') {
              writeRaw(msg.message)
            } else if (msg.type === 'done') {
              success = msg.success
            }
          } catch {}
        }
      }
    }
    
    // 刷新状态
    await checkPluginEnvStatus(plugin)
    
    if (success) {
      appendLog(t('pythonEnv.venvCreated'), 'info')
      notification.success(t('pythonEnv.venvCreated'))
    } else {
      appendLog(t('pythonEnv.venvCreateFailed'), 'error')
      notification.error(t('pythonEnv.venvCreateFailed'))
    }
  } catch (e) {
    console.error('Create venv error:', e)
    appendLog(t('pythonEnv.venvCreateFailed'), 'error')
    notification.error(t('pythonEnv.venvCreateFailed'))
  } finally {
    creatingVenv.value = false
  }
}

const deleteVenv = async (plugin: Plugin) => {
  deletingVenv.value = true
  appendLog(t('pythonEnv.deletingVenv', { name: getLocalizedText(plugin.name) }), 'info')
  
  try {
    const res = await fetch('http://localhost:18080/api/python/delete-venv', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pluginId: plugin.id })
    })
    
    if (res.ok) {
      await checkPluginEnvStatus(plugin)
      appendLog(t('pythonEnv.venvDeleted'), 'info')
    }
  } catch (e) {
    console.error('Delete venv error:', e)
    appendLog(t('pythonEnv.venvDeleteFailed'), 'error')
  } finally {
    deletingVenv.value = false
  }
}

const installDeps = async (plugin: Plugin) => {
  installingDeps.value = true
  // 添加到正在安装列表
  if (!installingPluginIds.value.includes(plugin.id)) {
    installingPluginIds.value.push(plugin.id)
  }
  const status = pluginEnvStatus.value[plugin.id]
  const missing = status?.dependencies.filter(d => !d.installed).map(d => d.name) || []
  
  appendLog(t('pythonEnv.installingDeps', { name: getLocalizedText(plugin.name) }), 'info')
  
  try {
    // 如果需要 PyTorch 且未安装，先安装 PyTorch
    if (hasPytorchConfig(plugin) && !status?.pytorchInstalled) {
      const gpuCompatible = isGpuCompatible(plugin)
      const pytorchVersion = selectedPytorchVersion.value[plugin.id] || (gpuCompatible ? 'gpu' : 'cpu')
      
      appendLog(t('pythonEnv.installingPytorch', { version: pytorchVersion.toUpperCase() }), 'cmd')
      installingPytorch.value = true
      
      const pytorchConfig = plugin.python?.pytorch
      const packages = pytorchConfig?.packages || []
      const indexUrl = pytorchConfig?.[pytorchVersion]?.indexUrl || ''
      
      const pytorchRes = await fetch('http://localhost:18080/api/python/install-pytorch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          pluginId: plugin.id,
          version: pytorchVersion,
          packages,
          indexUrl
        })
      })
      
      if (!pytorchRes.ok) {
        throw new Error('PyTorch install request failed')
      }
      // PyTorch 安装完成后会通过 WebSocket 推送 pip_done
      // 等待 PyTorch 安装完成后再安装其他依赖
      return
    }
    
    // 安装普通依赖
    const body: any = { pluginId: plugin.id, packages: missing }
    if (pipSource.value) body.indexUrl = pipSource.value
    if (httpProxy.value) body.proxy = httpProxy.value
    
    const res = await fetch('http://localhost:18080/api/python/install-deps', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    
    if (!res.ok) {
      throw new Error('Request failed')
    }
    // 后端立即返回 taskId，实际输出通过 WebSocket 推送
    // 状态更新在 pip_done 消息处理中完成
  } catch (e) {
    console.error('Install deps error:', e)
    appendLog(t('pythonEnv.depsInstallFailed'), 'error')
    notification.error(t('pythonEnv.depsInstallFailed'))
    installingDeps.value = false
  }
}

// 停止安装依赖
const stopInstall = async (plugin: Plugin) => {
  try {
    appendLog(t('pythonEnv.stoppingInstall'), 'info')
    const res = await fetch('http://localhost:18080/api/python/stop-install', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pluginId: plugin.id })
    })
    if (res.ok) {
      appendLog(t('pythonEnv.installStopped'), 'info')
      notification.warning(t('pythonEnv.installStopped'))
    }
  } catch (e) {
    console.error('Stop install error:', e)
  }
  // 从正在安装列表移除
  installingPluginIds.value = installingPluginIds.value.filter(id => id !== plugin.id)
  installingDeps.value = false
  installingPytorch.value = false
}

// 卸载单个依赖
const uninstallDep = async (packageName: string) => {
  if (!selectedPlugin.value) return
  const plugin = selectedPlugin.value
  
  uninstallingDep.value = packageName
  appendLog(t('pythonEnv.uninstallingDep', { name: packageName }), 'cmd')
  
  try {
    const res = await fetch('http://localhost:18080/api/python/uninstall-dep', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pluginId: plugin.id,
        package: packageName
      })
    })
    
    const data = await res.json()
    if (data.success) {
      appendLog(t('pythonEnv.depUninstalled', { name: packageName }), 'info')
      notification.success(t('pythonEnv.depUninstalled', { name: packageName }))
      await checkPluginEnvStatus(plugin)
    } else {
      appendLog(t('pythonEnv.depUninstallFailed', { name: packageName }), 'error')
      notification.error(t('pythonEnv.depUninstallFailed', { name: packageName }))
    }
  } catch (e) {
    console.error('Uninstall dep error:', e)
    appendLog(t('pythonEnv.depUninstallFailed', { name: packageName }), 'error')
    notification.error(t('pythonEnv.depUninstallFailed', { name: packageName }))
  } finally {
    uninstallingDep.value = null
  }
}

// 检测 GPU 信息
const checkGpuInfo = async () => {
  try {
    const res = await fetch('http://localhost:18080/api/system/gpu')
    if (res.ok) {
      const data = await res.json()
      gpuInfo.value = {
        hasNvidia: data.hasNvidia || false,
        cudaVersion: data.cudaVersion || '',
        gpuName: data.gpuName || ''
      }
    }
  } catch (e) {
    console.error('Check GPU info error:', e)
  }
}

// 处理 PyTorch 版本按钮点击
const handlePytorchVersionClick = (version: 'gpu' | 'cpu', isInstalled: boolean) => {
  if (!selectedPlugin.value) return
  
  if (isInstalled) {
    // 已安装时，点击直接切换版本
    installPytorch(version)
  } else {
    // 未安装时，只选择版本，不安装
    selectedPytorchVersion.value[selectedPlugin.value.id] = version
  }
}

// 安装 PyTorch
const installPytorch = async (version: 'gpu' | 'cpu') => {
  if (!selectedPlugin.value) return
  const plugin = selectedPlugin.value
  const pytorchConfig = plugin.python?.pytorch
  
  installingPytorch.value = true
  appendLog(t('pythonEnv.installingPytorch', { version: version.toUpperCase() }), 'cmd')
  
  try {
    // 解析自定义 PyTorch 包（可选）
    const custom = customPytorch.value.trim()
    let packages: string[]
    let indexUrl: string
    
    if (custom) {
      // 用户自定义
      packages = custom.split(/\s+/).filter(Boolean)
      indexUrl = pytorchConfig?.[version]?.indexUrl || recommendedPytorch[version].indexUrl
    } else if (pytorchConfig?.packages) {
      // 使用插件 manifest 中的配置（新格式：packages 在根级别）
      packages = [...pytorchConfig.packages]
      indexUrl = pytorchConfig[version]?.indexUrl || recommendedPytorch[version].indexUrl
    } else {
      // 回退到推荐版本
      packages = [...recommendedPytorch[version].packages]
      indexUrl = recommendedPytorch[version].indexUrl
    }

    const res = await fetch('http://localhost:18080/api/python/install-pytorch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pluginId: plugin.id,
        version,
        packages,
        indexUrl
      })
    })
    
    if (!res.ok) {
      throw new Error('Request failed')
    }
    // 后端立即返回 taskId，实际输出通过 WebSocket 推送
    // 状态更新在 pip_done 消息处理中完成
  } catch (e) {
    console.error('Install pytorch error:', e)
    appendLog(t('pythonEnv.pytorchInstallFailed'), 'error')
    notification.error(t('pythonEnv.pytorchInstallFailed'))
    installingPytorch.value = false
  }
}

// 检查插件是否需要 PyTorch
const hasPytorchConfig = (plugin: Plugin): boolean => {
  return !!(plugin.python?.pytorch?.packages?.length)
}

// 检查 GPU 是否满足最低要求
const isGpuCompatible = (plugin: Plugin): boolean => {
  const minCuda = plugin.python?.pytorch?.gpu?.minCuda
  if (!minCuda || !gpuInfo.value.hasNvidia) return false
  // 比较 CUDA 版本
  const current = gpuInfo.value.cudaVersion
  if (!current) return false
  return compareVersions(current, minCuda) >= 0
}

// 版本比较工具
const compareVersions = (v1: string, v2: string): number => {
  const parts1 = v1.split('.').map(Number)
  const parts2 = v2.split('.').map(Number)
  for (let i = 0; i < Math.max(parts1.length, parts2.length); i++) {
    const p1 = parts1[i] || 0
    const p2 = parts2[i] || 0
    if (p1 > p2) return 1
    if (p1 < p2) return -1
  }
  return 0
}

// 检查是否支持特定版本
const supportsPytorchVersion = (plugin: Plugin, version: 'gpu' | 'cpu'): boolean => {
  return !!plugin.python?.pytorch?.[version]
}

// 运行自定义命令
const runCustomCommand = async () => {
  if (!selectedPlugin.value || !customCommand.value.trim()) return
  
  const cmd = customCommand.value.trim()
  runningCommand.value = true
  appendLog(`$ ${cmd}`, 'cmd')
  
  try {
    const res = await fetch('http://localhost:18080/api/python/run-command', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pluginId: selectedPlugin.value.id,
        command: cmd
      })
    })
    
    if (!res.ok) {
      throw new Error('Request failed')
    }
    // 清空输入框
    customCommand.value = ''
  } catch (e) {
    console.error('Run command error:', e)
    appendLog(t('pythonEnv.commandFailed'), 'error')
    runningCommand.value = false
  }
}

const saveSettings = async () => {
  try {
    await fetch('http://localhost:18080/api/python/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pipSource: pipSource.value,
        httpProxy: httpProxy.value,
        socks5Proxy: socks5Proxy.value
      })
    })
    showSettingsDialog.value = false
  } catch (e) {
    console.error('Save settings error:', e)
  }
}

const loadSettings = async () => {
  try {
    const res = await fetch('http://localhost:18080/api/python/settings')
    if (res.ok) {
      const data = await res.json()
      pipSource.value = data.pipSource || ''
      httpProxy.value = data.httpProxy || ''
      socks5Proxy.value = data.socks5Proxy || ''
    }
  } catch (e) {
    console.error('Load settings error:', e)
  }
}

// ============ 终端函数 ============
const getCurrentTheme = () => document.documentElement.getAttribute('data-theme') === 'light' ? 'light' : 'dark'

const updateTerminalTheme = () => {
  if (!terminal.value) return
  const theme = getCurrentTheme()
  terminal.value.options.theme = terminalThemes[theme]
}

const initTerminal = () => {
  if (!terminalContainerRef.value || terminal.value) return
  
  const theme = getCurrentTheme()
  const term = new XTerm({
    theme: terminalThemes[theme],
    fontSize: 12,
    fontFamily: "'Consolas', 'Monaco', 'Courier New', monospace",
    cursorBlink: false,
    disableStdin: true,
    scrollback: 5000,
    convertEol: true // 将 \n 转换为 \r\n，避免换行问题
  })
  
  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(terminalContainerRef.value)
  fit.fit()
  
  terminal.value = term
  fitAddon.value = fit
  
  // 恢复缓存的终端内容
  const cachedContent = sessionStorage.getItem(TERMINAL_CACHE_KEY)
  if (cachedContent) {
    term.write(cachedContent)
  }
  
  // 监听容器大小变化
  const resizeObserver = new ResizeObserver(() => {
    nextTick(() => fitAddon.value?.fit())
  })
  resizeObserver.observe(terminalContainerRef.value)
  
  // 监听主题变化
  const themeObserver = new MutationObserver(() => updateTerminalTheme())
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
}

const clearTerminal = () => {
  terminal.value?.clear()
  sessionStorage.removeItem(TERMINAL_CACHE_KEY)
}

const copyTerminalContent = async () => {
  if (!terminal.value) return
  const buffer = terminal.value.buffer.active
  const lines: string[] = []
  for (let i = 0; i < buffer.length; i++) {
    const line = buffer.getLine(i)
    if (line) lines.push(line.translateToString(true))
  }
  const text = lines.join('\n').trimEnd()
  if (text) {
    await navigator.clipboard.writeText(text)
  }
}

const getTimeStr = () => {
  const now = new Date()
  return `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
}

const appendToCache = (text: string) => {
  const current = sessionStorage.getItem(TERMINAL_CACHE_KEY) || ''
  // 限制缓存大小为 100KB
  const newContent = current + text
  if (newContent.length > 100000) {
    sessionStorage.setItem(TERMINAL_CACHE_KEY, newContent.slice(-50000))
  } else {
    sessionStorage.setItem(TERMINAL_CACHE_KEY, newContent)
  }
}

const writeRaw = (text: string) => {
  if (!terminal.value) return
  terminal.value.write(text)
  appendToCache(text)
}

const appendLog = (text: string, type: 'cmd' | 'info' | 'error' = 'info') => {
  if (!terminal.value) return
  const time = `\x1b[90m[${getTimeStr()}]\x1b[0m `
  
  let line = ''
  if (type === 'cmd') {
    line = time + `\x1b[36m$ ${text}\x1b[0m`
  } else if (type === 'error') {
    line = time + `\x1b[31m${text}\x1b[0m`
  } else {
    line = time + text
  }
  terminal.value.writeln(line)
  appendToCache(line + '\r\n')
}

// 拖拽调整终端高度
const startResizeTerminal = (e: MouseEvent) => {
  isResizingTerminal.value = true
  const startY = e.clientY
  const startHeight = terminalHeight.value
  
  const onMouseMove = (e: MouseEvent) => {
    const delta = startY - e.clientY
    terminalHeight.value = Math.max(100, Math.min(500, startHeight + delta))
    nextTick(() => fitAddon.value?.fit())
  }
  
  const onMouseUp = () => {
    isResizingTerminal.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }
  
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// 加载数据路径
const loadDataPath = async () => {
  try {
    const res = await fetch('http://localhost:18080/api/settings/paths')
    if (res.ok) {
      const data = await res.json()
      dataPath.value = data.dataPath || ''
    }
  } catch (e) {
    console.error('Load data path error:', e)
  }
}

// WebSocket 取消订阅函数
let unsubscribePipOutput: (() => void) | null = null
let unsubscribePipDone: (() => void) | null = null

// 查询正在安装的任务
const loadInstallingTasks = async () => {
  try {
    const res = await fetch('http://localhost:18080/api/python/installing-tasks')
    if (res.ok) {
      const data = await res.json()
      if (data.installingPlugins && data.installingPlugins.length > 0) {
        installingPluginIds.value = data.installingPlugins
        // 如果当前选中的插件正在安装，恢复状态
        if (selectedPlugin.value && data.installingPlugins.includes(selectedPlugin.value.id)) {
          installingDeps.value = true
          installingPytorch.value = true
        }
      }
    }
  } catch (e) {
    console.error('Load installing tasks error:', e)
  }
}

// ============ 生命周期 ============
onMounted(async () => {
  nextTick(() => initTerminal())
  await loadDataPath()
  await loadSettings()
  await checkPythonVersion()
  await checkGpuInfo()
  await loadPythonPlugins()
  await checkAllPluginEnvStatus()
  
  // 主动查询正在安装的任务（页面刷新后恢复状态）
  await loadInstallingTasks()
  
  // 订阅 pip 输出消息
  unsubscribePipOutput = subscribe('pip_output', (msg) => {
    if (msg.message) {
      writeRaw(msg.message)
    }
  })
  
  // 订阅 pip 完成消息
  unsubscribePipDone = subscribe('pip_done', async (msg) => {
    // 从正在安装列表中移除已完成的插件
    if (msg.pluginId) {
      installingPluginIds.value = installingPluginIds.value.filter(id => id !== msg.pluginId)
    }
    
    // 记录是否是 PyTorch 安装完成（在重置状态前判断）
    const wasPytorchInstall = installingPytorch.value
    
    installingPytorch.value = false
    runningCommand.value = false
    
    // 刷新状态
    if (selectedPlugin.value) {
      await checkPluginEnvStatus(selectedPlugin.value)
      
      // 如果是 PyTorch 安装完成且成功，检查是否还有其他依赖需要安装
      if (wasPytorchInstall && msg.success) {
        const status = pluginEnvStatus.value[selectedPlugin.value.id]
        const missingDeps = status?.dependencies.filter(d => !d.installed) || []
        if (missingDeps.length > 0) {
          // 继续安装其他依赖，但不再安装 PyTorch
          appendLog(t('pythonEnv.installingDeps', { name: getLocalizedText(selectedPlugin.value.name) }), 'info')
          const body: any = { pluginId: selectedPlugin.value.id, packages: missingDeps.map(d => d.name) }
          if (pipSource.value) body.indexUrl = pipSource.value
          if (httpProxy.value) body.proxy = httpProxy.value
          
          const res = await fetch('http://localhost:18080/api/python/install-deps', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
          })
          if (!res.ok) {
            appendLog(t('pythonEnv.depsInstallFailed'), 'error')
            notification.error(t('pythonEnv.depsInstallFailed'))
            installingDeps.value = false
          }
          return
        }
      }
    }
    
    // 所有安装完成
    if (msg.success) {
      appendLog(t('pythonEnv.depsInstalled'), 'info')
      notification.success(t('pythonEnv.depsInstalled'))
    } else {
      appendLog(t('pythonEnv.depsInstallFailed'), 'error')
      notification.error(t('pythonEnv.depsInstallFailed'))
    }
    installingDeps.value = false
  })
  
  // 订阅正在安装的任务（用于页面刷新后恢复状态）
  subscribe('pip_installing', (msg) => {
    const plugins = msg.data as string[]
    if (plugins && plugins.length > 0) {
      // 存储正在安装的插件列表
      installingPluginIds.value = plugins
      // 如果当前选中的插件正在安装中，恢复状态
      if (selectedPlugin.value && plugins.includes(selectedPlugin.value.id)) {
        installingDeps.value = true
        installingPytorch.value = true
      }
    }
  })
})

onUnmounted(() => {
  // 取消订阅
  unsubscribePipOutput?.()
  unsubscribePipDone?.()
  terminal.value?.dispose()
})
</script>

<style scoped>
.python-env-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: row;
  background: var(--color-bg-app);
}

/* ========== 左侧栏 ========== */
.env-sidebar {
  width: 280px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-sidebar);
  border-right: 1px solid var(--color-border-subtle);
}

.sidebar-header {
  display: flex;
  align-items: center;
  height: 48px;
  padding: 0 16px;
  border-bottom: 1px solid var(--color-border-subtle);
}

.sidebar-title {
  font-size: 14px;
  font-weight: 600;
  margin: 0;
  color: var(--color-fg);
}

.sidebar-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.sidebar-loading, .sidebar-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  color: var(--color-fg-muted);
  font-size: 12px;
}

.plugin-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.plugin-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.plugin-item:hover {
  background: var(--color-bg-sidebar-hover);
}

.plugin-item--active {
  background: var(--color-bg-sidebar-hover);
}

.plugin-item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.plugin-item-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-fg);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.plugin-item-version {
  font-size: 10px;
  color: var(--color-fg-muted);
}

.plugin-item-status {
  flex-shrink: 0;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
}

.status-badge--loading { color: var(--color-fg-muted); }
.status-badge--ready { color: #22c55e; }
.status-badge--missing { color: #f59e0b; }
.status-badge--no-venv { color: var(--color-fg-muted); opacity: 0.5; }

/* ========== 右侧主内容区 ========== */
.env-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.env-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 16px;
  border-bottom: 1px solid var(--color-border-subtle);
  background: var(--color-bg-sidebar);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.python-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.python-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-fg);
}

.python-version {
  font-size: 13px;
  font-family: monospace;
}

.python-version--ok { color: #22c55e; }
.python-version--missing { color: #ef4444; }

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.env-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.env-content::-webkit-scrollbar {
  display: none;
}

.env-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-fg-muted);
}

.env-empty-icon {
  opacity: 0.4;
  margin-bottom: 12px;
}

/* 插件详情 */
.plugin-detail {
  max-width: 800px;
}

.detail-header {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 24px;
}

.detail-name {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: var(--color-fg);
}

.detail-version {
  font-size: 12px;
  color: var(--color-fg-muted);
}

.detail-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-fg);
  margin: 0 0 12px;
}

.venv-status {
  padding: 16px;
  background: var(--color-bg-sidebar);
  border-radius: 8px;
}

.venv-path {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 12px;
}

.venv-path .label {
  color: var(--color-fg-muted);
}

.venv-path code {
  font-family: monospace;
  color: var(--color-fg);
  background: var(--color-bg-app);
  padding: 4px 8px;
  border-radius: 4px;
}

.venv-actions {
  display: flex;
  gap: 8px;
}

/* 依赖列表 */
.deps-empty {
  padding: 16px;
  background: var(--color-bg-sidebar);
  border-radius: 8px;
  font-size: 12px;
  color: var(--color-fg-muted);
  text-align: center;
}

.deps-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  background: var(--color-bg-sidebar);
  border-radius: 8px;
}

.dep-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 12px;
}

.dep-item--missing {
  background: rgba(245, 158, 11, 0.1);
}

.dep-name {
  font-family: monospace;
  color: var(--color-fg);
}

.dep-status {
  display: flex;
  align-items: center;
  gap: 4px;
}

.dep-status--ok { color: #22c55e; }
.dep-status--missing { color: #f59e0b; }

.dep-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-icon-small {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-fg-muted);
  cursor: pointer;
  transition: all 0.15s;
}

.btn-icon-small:hover:not(:disabled) {
  background: var(--color-bg-sidebar-hover);
  color: var(--color-fg);
}

.btn-icon-small:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-uninstall:hover:not(:disabled) {
  color: #ef4444;
}

.deps-actions {
  margin-top: 12px;
}

.dep-item--pytorch {
  background: var(--color-bg-sidebar);
  border-radius: 6px;
  padding: 8px 12px;
}

.dep-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.dep-warning {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #eab308;
}

.pytorch-inline-actions {
  display: flex;
  gap: 4px;
  margin-left: 8px;
}

.btn-pytorch-small {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-bg-app);
  color: var(--color-fg-muted);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-pytorch-small:hover:not(:disabled) {
  background: var(--color-bg-sidebar-hover);
  color: var(--color-fg);
}

.btn-pytorch-small:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-pytorch-small--active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.btn-pytorch-small--active:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

/* ========== GPU 状态 ========== */
.gpu-status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  padding: 8px 12px;
  background: var(--color-bg-sidebar);
  border-radius: 6px;
}

/* ========== PyTorch ========== */
.pytorch-status {
  padding: 16px;
  background: var(--color-bg-sidebar);
  border-radius: 8px;
}

.pytorch-gpu-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 12px;
}

.pytorch-gpu-info .label {
  color: var(--color-fg-muted);
}

.pytorch-gpu-info .gpu-ok {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #22c55e;
}

.pytorch-gpu-info .gpu-none {
  color: var(--color-fg-muted);
}

.pytorch-installed {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #22c55e;
  margin-bottom: 8px;
}

.pytorch-version-info {
  font-size: 11px;
  color: var(--color-fg-muted);
  margin-left: 4px;
}

.pytorch-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pytorch-hint {
  font-size: 12px;
  color: var(--color-fg-muted);
  margin: 0;
}

.pytorch-recommend {
  margin: 8px 0;
}

.pytorch-recommend__line {
  font-size: 11px;
  color: var(--color-fg-muted);
  margin: 2px 0;
  opacity: 0.8;
}

.pytorch-buttons {
  display: flex;
  gap: 8px;
}

.btn-pytorch {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 6px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-bg-app);
  color: var(--color-fg);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-pytorch:hover:not(:disabled) {
  background: var(--color-bg-sidebar-hover);
}

.btn-pytorch:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-pytorch--recommended {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.btn-pytorch--recommended:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

.text-green {
  color: #22c55e;
}

.pytorch-warning {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: rgba(234, 179, 8, 0.1);
  border: 1px solid rgba(234, 179, 8, 0.3);
  border-radius: 6px;
  color: #eab308;
  font-size: 12px;
  margin-bottom: 12px;
}

.pytorch-type-badge {
  display: inline-flex;
  padding: 2px 6px;
  background: var(--color-accent);
  color: #fff;
  font-size: 10px;
  font-weight: 500;
  border-radius: 4px;
  margin-left: 8px;
}

/* ========== 终端面板 ========== */
.env-terminal {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-sidebar);
  border-top: 1px solid var(--color-border-subtle);
}

.terminal-resize-handle {
  height: 4px;
  background: transparent;
  cursor: ns-resize;
  transition: background 0.15s;
}

.terminal-resize-handle:hover {
  background: var(--color-accent);
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 3px 12px;
  border-bottom: 1px solid var(--color-border-subtle);
  height: 28px;
  flex-shrink: 0;
}

.terminal-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-fg);
}

.terminal-loading {
  color: var(--color-accent);
  margin-left: 4px;
}

.terminal-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.terminal-container {
  flex: 1;
  overflow: hidden;
  background: #0d0d0d;
}

/* 隐藏 xterm 滚动条 */
.terminal-container :deep(.xterm-viewport) {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar) {
  display: none;
}

[data-theme='light'] .terminal-container {
  background: #f5f5f5;
}

.terminal-input-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--color-bg-app);
  border-top: 1px solid var(--color-border-subtle);
}

.terminal-input {
  flex: 1;
  padding: 6px 10px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-bg-sidebar);
  color: var(--color-fg);
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
}

.terminal-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.terminal-input:disabled {
  opacity: 0.6;
}

.btn-run-command {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 28px;
  border-radius: 4px;
  border: none;
  background: var(--color-accent);
  color: #fff;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-run-command:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

.btn-run-command:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ========== 按钮 ========== */
.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  border: none;
  background: var(--color-accent);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover {
  background: var(--color-accent-hover);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  border: 1px solid var(--color-border-subtle);
  background: transparent;
  color: var(--color-fg);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-secondary:hover {
  background: var(--color-bg-sidebar-hover);
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-danger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  border: none;
  background: #dc2626;
  color: white;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-danger:hover {
  background: #b91c1c;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--color-fg-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.btn-icon:hover {
  background: var(--color-bg-sidebar-hover);
  color: var(--color-fg);
}

/* ========== 对话框 ========== */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  width: 480px;
  max-width: 90%;
  background: var(--color-bg-sidebar);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border-subtle);
  color: var(--color-fg);
}

.modal-content {
  padding: 20px;
}

.modal-content p {
  margin: 0 0 16px;
  font-size: 13px;
  color: var(--color-fg-muted);
  line-height: 1.6;
}

.modal-tips {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: rgba(245, 158, 11, 0.1);
  border-radius: 8px;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.tip-icon {
  flex-shrink: 0;
  color: #f59e0b;
}

.tip-content p {
  margin: 0 0 8px;
  font-size: 12px;
}

.tip-content ul {
  margin: 0;
  padding-left: 16px;
  font-size: 12px;
  color: var(--color-fg-muted);
}

.tip-content li {
  margin: 4px 0;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 20px;
  border-top: 1px solid var(--color-border-subtle);
}

/* 表单 */
.form-group {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-fg);
  margin-bottom: 6px;
}

.form-input, .form-select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  background: var(--color-bg-app);
  color: var(--color-fg);
  font-size: 13px;
  box-sizing: border-box;
}

.form-input:focus, .form-select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.form-select {
  cursor: pointer;
}

/* 动画 */
.animate-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
