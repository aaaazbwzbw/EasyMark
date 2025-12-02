<template>
  <div 
    class="plugins-view"
    @dragover.prevent="onDragOver"
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
    :class="{ 'plugins-view--dragover': isDragOver }"
  >
    <!-- 拖拽提示遮罩 -->
    <div v-if="isDragOver" class="plugins-drop-overlay">
      <Package :size="48" />
      <p>{{ t('plugins.dropToInstall') }}</p>
    </div>

    <!-- 左侧栏 -->
    <aside class="plugins-sidebar">
      <header class="sidebar-header">
        <h2 class="sidebar-title">{{ t('plugins.title') }}</h2>
        <button class="sidebar-install-btn" @click="openInstallDialog" :title="t('plugins.installFromDisk')">
          <Plug :size="18" />
          <span class="sidebar-install-text">{{ t('plugins.installFromDisk') }}</span>
        </button>
      </header>
      <div class="sidebar-body">
        <div class="sidebar-search">
          <input 
            type="text" 
            class="search-input" 
            v-model="searchQuery"
            :placeholder="t('plugins.searchPlaceholder')"
          />
        </div>
        <div class="sidebar-panels">
          <!-- 已安装折叠面板 -->
          <div class="collapse-panel">
            <button class="collapse-header" @click="togglePanel('installed')">
              <component :is="expandedPanels.installed ? ChevronDown : ChevronRight" :size="16" />
              <span>{{ t('plugins.installed') }}</span>
              <span class="collapse-count">{{ filteredInstalledPlugins.length }}</span>
            </button>
            <div v-if="expandedPanels.installed" class="collapse-content">
              <div v-if="loading" class="panel-loading">
                <Loader2 :size="16" class="plugins-loading-icon animate-spin" />
              </div>
              <div v-else-if="filteredInstalledPlugins.length === 0" class="panel-empty">
                {{ t('plugins.noResults') }}
              </div>
              <div v-else class="plugin-list">
                <div 
                  v-for="plugin in filteredInstalledPlugins" 
                  :key="plugin.id" 
                  class="plugin-item"
                  :class="{ 'plugin-item--active': selectedPlugin?.id === plugin.id }"
                  @click="selectPlugin(plugin)"
                  @contextmenu.prevent="showContextMenu($event, plugin)"
                >
                  <img 
                    v-if="plugin.logo" 
                    :src="plugin.logo" 
                    class="plugin-item-logo" 
                    @error="onLogoError(plugin)"
                  />
                  <component v-else :is="getPluginIcon(plugin.type)" :size="18" class="plugin-item-icon" />
                  <div class="plugin-item-info">
                    <span class="plugin-item-name">{{ getLocalizedText(plugin.name) }}</span>
                    <span class="plugin-item-version">v{{ plugin.version }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <!-- 插件市场折叠面板 -->
          <div class="collapse-panel">
            <button class="collapse-header" @click="togglePanel('market')">
              <component :is="expandedPanels.market ? ChevronDown : ChevronRight" :size="16" />
              <span>{{ t('plugins.market') }}</span>
            </button>
            <div v-if="expandedPanels.market" class="collapse-content">
              <div class="panel-empty">{{ t('plugins.marketComingSoon') }}</div>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="plugins-main">
      <div v-if="!selectedPlugin" class="plugins-empty">
          <Puzzle :size="48" class="plugins-empty-icon" />
          <p>{{ t('plugins.selectHint') }}</p>
        </div>
        <div v-else class="plugin-detail">
          <div class="detail-header">
            <div class="detail-icon">
              <img 
                v-if="selectedPlugin.logo" 
                :src="selectedPlugin.logo" 
                class="detail-logo"
                @error="onLogoError(selectedPlugin)"
              />
              <component v-else :is="getPluginIcon(selectedPlugin.type)" :size="32" />
            </div>
            <div class="detail-main">
              <div class="detail-title-row">
                <h2 class="detail-name">{{ getLocalizedText(selectedPlugin.name) }}</h2>
                <span class="detail-version">v{{ selectedPlugin.version }}</span>
              </div>
              <div class="detail-sub-row">
                <span class="detail-author">{{ selectedPlugin.author || t('plugins.unknownAuthor') }}</span>
                <span class="detail-dot">·</span>
                <span class="detail-type">{{ getPluginTypeText(selectedPlugin.type) }}</span>
                <span v-if="selectedPlugin.installedAt" class="detail-dot">·</span>
                <span v-if="selectedPlugin.installedAt" class="detail-installed">{{ selectedPlugin.installedAt }}</span>
              </div>
            </div>
            <div class="detail-actions">
              <button class="detail-primary-btn" @click="confirmDelete(selectedPlugin)">
                {{ t('plugins.uninstall') }}
              </button>
            </div>
          </div>
          <p class="detail-desc">{{ getLocalizedText(selectedPlugin.description) }}</p>
          <div class="detail-meta">
            <div class="meta-item">
              <span class="meta-label">{{ t('plugins.author') }}</span>
              <span class="meta-value">{{ selectedPlugin.author || '-' }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">{{ t('plugins.installedAt') }}</span>
              <span class="meta-value">{{ selectedPlugin.installedAt }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">{{ t('plugins.size') }}</span>
              <span class="meta-value">{{ formatSize(selectedPlugin.size) }}</span>
            </div>
          </div>
          <!-- README 区域 -->
          <div class="detail-readme">
          <div v-if="loadingReadme" class="readme-loading">
            <Loader2 :size="20" class="plugins-loading-icon animate-spin" />
          </div>
          <div v-else-if="pluginReadme" class="readme-content" v-html="pluginReadme"></div>
          <div v-else class="readme-empty">{{ t('plugins.noReadme') }}</div>
        </div>
      </div>
    </main>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div 
        v-if="contextMenu.visible" 
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <button class="context-menu-item context-menu-item--danger" @click="handleContextMenuUninstall">
          <Trash2 :size="16" />
          {{ t('plugins.uninstallAndDelete') }}
        </button>
      </div>
    </Teleport>

    <!-- 安装插件对话框 -->
    <div v-if="isInstallDialogOpen" class="plugins-modal-overlay" @click.self="closeInstallDialog">
      <div class="plugins-modal">
        <h2 class="plugins-modal__title">{{ t('plugins.installTitle') }}</h2>
        <p class="plugins-modal__desc">{{ t('plugins.installDesc') }}</p>
        
        <div class="plugins-modal__file-input">
          <input 
            type="text" 
            class="plugins-modal__input" 
            :value="selectedFilePath" 
            readonly 
            :placeholder="t('plugins.selectFile')"
          />
          <button class="plugins-modal__browse" @click="browseFile">
            {{ t('plugins.browse') }}
          </button>
        </div>

        <div v-if="installError" class="plugins-modal__error">
          {{ installError }}
        </div>

        <div class="plugins-modal__actions">
          <button class="plugins-modal__btn" @click="closeInstallDialog">
            {{ t('plugins.cancel') }}
          </button>
          <button 
            class="plugins-modal__btn plugins-modal__btn--primary" 
            @click="installPlugin"
            :disabled="!selectedFilePath || installing"
          >
            <Loader2 v-if="installing" :size="16" class="plugins-loading-icon animate-spin" />
            {{ installing ? t('plugins.installing') : t('plugins.install') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="isDeleteDialogOpen" class="plugins-modal-overlay" @click.self="closeDeleteDialog">
      <div class="plugins-modal">
        <h2 class="plugins-modal__title">{{ t('plugins.uninstallTitle') }}</h2>
        <p class="plugins-modal__desc">
          {{ t('plugins.uninstallConfirm', { name: getLocalizedText(deleteTarget?.name || '') }) }}
        </p>
        <div class="plugins-modal__actions">
          <button class="plugins-modal__btn" @click="closeDeleteDialog">
            {{ t('plugins.cancel') }}
          </button>
          <button 
            class="plugins-modal__btn plugins-modal__btn--danger" 
            @click="uninstallPlugin"
          >
            {{ t('plugins.uninstall') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Package, Plug, Loader2, Puzzle, Trash2, ChevronDown, ChevronRight, Database, DatabaseBackup, Brain } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import notification from '../utils/notification'

const { t, locale } = useI18n()

type Plugin = {
  id: string
  name: string | Record<string, string>
  version: string
  type: string
  description: string | Record<string, string>
  author?: string
  installedAt: string
  path: string
  logo?: string
  size?: number
}

// 获取本地化文本
const getLocalizedText = (text: string | Record<string, string>): string => {
  if (typeof text === 'string') return text
  if (!text) return ''
  return text[locale.value] || text['zh-CN'] || text['en-US'] || Object.values(text)[0] || ''
}

// 获取插件类型文本（支持数组类型）
const getPluginTypeText = (pluginType: string | string[]): string => {
  if (Array.isArray(pluginType)) {
    return pluginType.map(pt => t('plugins.types.' + pt)).join(' + ')
  }
  return t('plugins.types.' + pluginType)
}

const plugins = ref<Plugin[]>([])
const loading = ref(true)
const isInstallDialogOpen = ref(false)
const isDeleteDialogOpen = ref(false)
const selectedFilePath = ref('')
const installError = ref('')
const installing = ref(false)
const deleteTarget = ref<Plugin | null>(null)
const isDragOver = ref(false)

// 新增：左侧栏状态
const searchQuery = ref('')
const selectedPlugin = ref<Plugin | null>(null)
const expandedPanels = ref({ installed: true, market: false })

// README 相关
const loadingReadme = ref(false)
const pluginReadme = ref('')

// 右键菜单
const contextMenu = ref({ visible: false, x: 0, y: 0, plugin: null as Plugin | null })

// 过滤后的已安装插件
const filteredInstalledPlugins = computed(() => {
  if (!searchQuery.value.trim()) return plugins.value
  const q = searchQuery.value.toLowerCase()
  return plugins.value.filter(p => 
    getLocalizedText(p.name).toLowerCase().includes(q) || 
    getLocalizedText(p.description).toLowerCase().includes(q)
  )
})

// 切换折叠面板
const togglePanel = (panel: 'installed' | 'market') => {
  expandedPanels.value[panel] = !expandedPanels.value[panel]
}

// 选择插件
const selectPlugin = async (plugin: Plugin) => {
  selectedPlugin.value = plugin
  await loadPluginReadme(plugin)
}

// 加载插件 README
const loadPluginReadme = async (plugin: Plugin) => {
  loadingReadme.value = true
  pluginReadme.value = ''
  try {
    const res = await fetch(`http://localhost:18080/api/plugins/${encodeURIComponent(plugin.id)}/readme`)
    if (res.ok) {
      const text = await res.text()
      if (text) {
        // 使用 marked 标准解析 Markdown
        pluginReadme.value = marked.parse(text) as string
      }
    }
  } catch (e) {
    console.error('Load plugin readme error:', e)
  } finally {
    loadingReadme.value = false
  }
}

// 格式化插件大小
const formatSize = (size?: number): string => {
  if (!size || size <= 0) return '-'
  const kb = size / 1024
  if (kb < 1024) return kb.toFixed(1) + ' KB'
  const mb = kb / 1024
  if (mb < 1024) return mb.toFixed(1) + ' MB'
  const gb = mb / 1024
  return gb.toFixed(1) + ' GB'
}

// Logo 加载失败处理
const onLogoError = (plugin: Plugin) => {
  plugin.logo = undefined
}

// 右键菜单
const showContextMenu = (event: MouseEvent, plugin: Plugin) => {
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    plugin
  }
}

const hideContextMenu = () => {
  contextMenu.value.visible = false
  contextMenu.value.plugin = null
}

const handleContextMenuUninstall = () => {
  if (contextMenu.value.plugin) {
    confirmDelete(contextMenu.value.plugin)
  }
  hideContextMenu()
}

// 点击外部关闭菜单
const handleDocumentClick = () => {
  if (contextMenu.value.visible) {
    hideContextMenu()
  }
}

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick)
})

// electronAPI 类型定义已移至 src/electron.d.ts

const getPluginIcon = (type: string) => {
  switch (type) {
    case 'dataset': return Database
    case 'import-dataset': return DatabaseBackup
    case 'export-dataset': return DatabaseBackup
    case 'training': return Brain
    default: return Puzzle
  }
}

const loadPlugins = async () => {
  loading.value = true
  try {
    const res = await fetch('http://localhost:18080/api/plugins')
    if (res.ok) {
      const data = await res.json()
      plugins.value = data.plugins || []
    }
  } catch (e) {
    console.error('Load plugins error:', e)
  } finally {
    loading.value = false
  }
}

const openInstallDialog = () => {
  selectedFilePath.value = ''
  installError.value = ''
  isInstallDialogOpen.value = true
}

const closeInstallDialog = () => {
  isInstallDialogOpen.value = false
}

const browseFile = async () => {
  // 使用 Electron 的文件选择对话框
  if (window.electronAPI?.selectFile) {
    const filePath = await window.electronAPI.selectFile({
      title: t('plugins.selectFileTitle'),
      filters: [{ name: 'Plugin Archive', extensions: ['zip', 'rar'] }]
    })
    if (filePath) {
      selectedFilePath.value = filePath
    }
  } else {
    console.warn('electronAPI.selectFile not available')
  }
}

// 拖拽相关处理
const onDragOver = () => {
  isDragOver.value = true
}

const onDragLeave = () => {
  isDragOver.value = false
}

const onDrop = async (e: DragEvent) => {
  isDragOver.value = false
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  
  // 获取第一个文件
  const file = files[0]
  if (!file) return
  const ext = file.name.toLowerCase().split('.').pop()
  
  if (ext !== 'zip' && ext !== 'rar') {
    installError.value = t('plugins.installError.unsupported_format')
    isInstallDialogOpen.value = true
    return
  }
  
  // Electron 中获取文件路径
  const filePath = (file as File & { path?: string }).path
  if (!filePath) {
    console.warn('File path not available, file:', file)
    installError.value = t('plugins.installError.unknown')
    isInstallDialogOpen.value = true
    return
  }
  
  // 直接安装
  selectedFilePath.value = filePath
  installError.value = ''
  installing.value = true
  isInstallDialogOpen.value = true
  
  try {
    if (window.electronAPI?.installPlugin) {
      const result = await window.electronAPI.installPlugin(filePath)
      if (result.success) {
        closeInstallDialog()
        await loadPlugins()
      } else {
        installError.value = t('plugins.installError.' + (result.error || 'unknown'))
      }
    }
  } catch (err) {
    console.error('Drop install error:', err)
    installError.value = t('plugins.installError.unknown')
  } finally {
    installing.value = false
  }
}

const installPlugin = async () => {
  if (!selectedFilePath.value) return
  installing.value = true
  installError.value = ''
  
  try {
    // 使用 Electron 端的安装功能（内置解压支持）
    if (window.electronAPI?.installPlugin) {
      const result = await window.electronAPI.installPlugin(selectedFilePath.value)
      if (result.success) {
        closeInstallDialog()
        await loadPlugins()
      } else {
        installError.value = t('plugins.installError.' + (result.error || 'unknown'))
      }
    } else {
      // 降级：直接调用后端（需要系统有 unrar/7z）
      const res = await fetch('http://localhost:18080/api/plugins/install', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ filePath: selectedFilePath.value })
      })
      const data = await res.json()
      if (res.ok && data.success) {
        closeInstallDialog()
        await loadPlugins()
      } else {
        installError.value = t('plugins.installError.' + (data.error || 'unknown'))
      }
    }
  } catch (e) {
    console.error('Install plugin error:', e)
    installError.value = t('plugins.installError.unknown')
  } finally {
    installing.value = false
  }
}

const confirmDelete = (plugin: Plugin) => {
  deleteTarget.value = plugin
  isDeleteDialogOpen.value = true
}

const closeDeleteDialog = () => {
  isDeleteDialogOpen.value = false
  deleteTarget.value = null
}

const uninstallPlugin = async () => {
  if (!deleteTarget.value) return
  const deletedId = deleteTarget.value.id
  const name = getLocalizedText(deleteTarget.value.name)

  // 长效通知：正在删除插件及虚拟环境
  const notifId = notification.info(t('plugins.uninstallingWithVenv', { name }), { persistent: true })

  try {
    const res = await fetch(`http://localhost:18080/api/plugins?id=${encodeURIComponent(deletedId)}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      // 如果删除的是当前选中的插件，清空选中状态
      if (selectedPlugin.value?.id === deletedId) {
        selectedPlugin.value = null
      }
      closeDeleteDialog()
      await loadPlugins()

      // 更新通知为成功，3 秒后自动消失
      notification.update(notifId, {
        type: 'success',
        message: t('plugins.uninstallWithVenvSuccess', { name }),
        persistent: false
      })
    } else {
      notification.update(notifId, {
        type: 'error',
        message: t('plugins.uninstallFailed'),
        persistent: false
      })
    }
  } catch (e) {
    console.error('Uninstall plugin error:', e)
    notification.update(notifId, {
      type: 'error',
      message: t('plugins.uninstallFailed'),
      persistent: false
    })
  }
}

onMounted(() => {
  loadPlugins()
  document.addEventListener('click', handleDocumentClick)
})

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick)
})
</script>

<style scoped>
.plugins-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: row;
  box-sizing: border-box;
  position: relative;
  background: var(--color-bg-app);
}

.plugins-view--dragover {
  background: rgba(var(--color-accent-rgb, 59, 130, 246), 0.05);
}

.plugins-drop-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  z-index: 100;
  color: #fff;
  gap: 12px;
  border: 3px dashed var(--color-accent);
  border-radius: 8px;
  margin: 12px;
}

.plugins-drop-overlay p {
  font-size: 1.125rem;
  margin: 0;
}

/* ========== 左侧栏 ========== */
.plugins-sidebar {
  width: 300px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-sidebar);
  border-right: 1px solid var(--color-border-subtle);
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.sidebar-install-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 10px;
  height: 28px;
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
}

.sidebar-install-btn:hover {
  background: var(--color-accent-hover);
}

.sidebar-install-text {
  white-space: nowrap;
}

.sidebar-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-search {
  padding: 12px;
  border-bottom: 1px solid var(--color-border-subtle);
}

.search-input {
  width: 100%;
  box-sizing: border-box;
  background: var(--color-bg-app);
  border: 1px solid var(--color-border-subtle);
  border-radius: 4px;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--color-fg);
  outline: none;
}

.search-input:focus {
  border-color: var(--color-accent);
}

.search-input::placeholder {
  color: var(--color-fg-muted);
}

.sidebar-panels {
  flex: 1;
  overflow-y: auto;
}

/* 折叠面板 */
.collapse-panel {
  border-bottom: 1px solid var(--color-border-subtle);
}

.collapse-header {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 10px 12px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-fg);
  text-align: left;
}

.collapse-header:hover {
  background: var(--color-bg-sidebar-hover);
}

.collapse-count {
  margin-left: auto;
  font-size: 11px;
  color: var(--color-fg-muted);
  background: var(--color-bg-app);
  padding: 1px 6px;
  border-radius: 8px;
}

.collapse-content {
  padding: 4px 8px 8px;
}

.panel-loading, .panel-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  font-size: 12px;
  color: var(--color-fg-muted);
}

.plugin-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.plugin-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.plugin-item:hover {
  background: var(--color-bg-sidebar-hover);
}

.plugin-item--active {
  background: var(--color-accent-soft);
}

.plugin-item-icon {
  color: var(--color-fg-muted);
  flex-shrink: 0;
}

.plugin-item--active .plugin-item-icon {
  color: var(--color-accent);
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

/* ========== 右侧主内容区 ========== */
.plugins-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding: 24px;
}

.plugins-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-fg-muted);
}

.plugins-empty-icon {
  opacity: 0.4;
  margin-bottom: 12px;
}

/* 插件 Logo */
.plugin-item-logo {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  object-fit: contain;
  flex-shrink: 0;
}

/* 插件详情 */
.plugin-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.detail-icon {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-app);
  border-radius: 12px;
  border: 1px solid var(--color-border-subtle);
  color: var(--color-accent);
  flex-shrink: 0;
}

.detail-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-title-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
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

.detail-sub-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 12px;
  color: var(--color-fg-muted);
}

.detail-author {
  font-weight: 500;
}

.detail-dot {
  opacity: 0.7;
}

.detail-installed {
  font-size: 11px;
}

.detail-type {
  font-size: 11px;
  padding: 2px 8px;
  background: var(--color-bg-sidebar);
  border-radius: 4px;
}

.detail-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-primary-btn {
  padding: 6px 16px;
  border-radius: 6px;
  border: none;
  background: var(--color-accent);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}

.detail-primary-btn:hover {
  background: var(--color-accent-hover);
}

.detail-desc {
  font-size: 14px;
  color: var(--color-fg-muted);
  line-height: 1.6;
  margin: 0 0 20px;
}

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 24px;
  padding: 12px 16px;
  background: var(--color-bg-sidebar);
  border-radius: 8px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.meta-label {
  font-size: 12px;
  color: var(--color-fg-muted);
  min-width: 80px;
}

.meta-value {
  font-size: 12px;
  color: var(--color-fg);
}

/* 加载动画 */
.plugins-loading-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 对话框样式 */
.plugins-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.plugins-modal {
  background: var(--color-bg-sidebar);
  border-radius: 12px;
  padding: 24px;
  min-width: 400px;
  max-width: 500px;
}

.plugins-modal__title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.plugins-modal__desc {
  font-size: 0.875rem;
  color: var(--color-fg-muted);
  margin: 0 0 16px 0;
}

.plugins-modal__file-input {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.plugins-modal__input {
  flex: 1;
  padding: 8px 12px;
  background: var(--color-bg-app);
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  color: var(--color-fg);
  font-size: 0.875rem;
}

.plugins-modal__browse {
  padding: 8px 16px;
  background: var(--color-bg-app);
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  color: var(--color-fg);
  cursor: pointer;
}

.plugins-modal__browse:hover {
  background: var(--color-bg-sidebar-hover);
}

.plugins-modal__error {
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
  color: #ef4444;
  font-size: 0.875rem;
  margin-bottom: 16px;
}

.plugins-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.plugins-modal__btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  background: transparent;
  color: var(--color-fg);
  cursor: pointer;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  gap: 6px;
}

.plugins-modal__btn:hover {
  background: var(--color-bg-sidebar-hover);
}

.plugins-modal__btn--primary {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.plugins-modal__btn--primary:hover {
  background: var(--color-accent-hover);
}

.plugins-modal__btn--primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.plugins-modal__btn--danger {
  background: #ef4444;
  border-color: #ef4444;
  color: #fff;
}

.plugins-modal__btn--danger:hover {
  background: #dc2626;
}

/* 详情 Logo */
.detail-logo {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  object-fit: contain;
}

/* README 区域 */
.detail-readme {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-top: 20px;
  overflow: hidden;
}

.readme-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  color: var(--color-fg-muted);
}

.readme-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: var(--color-bg-sidebar);
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-fg-muted);
  /* 自定义滚动条 */
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-subtle) transparent;
}

.readme-content::-webkit-scrollbar {
  width: 6px;
}

.readme-content::-webkit-scrollbar-track {
  background: transparent;
}

.readme-content::-webkit-scrollbar-thumb {
  background: var(--color-border-subtle);
  border-radius: 3px;
}

.readme-content::-webkit-scrollbar-thumb:hover {
  background: var(--color-fg-muted);
}

.readme-content h2,
.readme-content h3,
.readme-content h4 {
  color: var(--color-fg);
  margin: 16px 0 8px;
}

.readme-content h2:first-child,
.readme-content h3:first-child,
.readme-content h4:first-child {
  margin-top: 0;
}

.readme-content code {
  background: var(--color-bg-app);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
}

.readme-content pre {
  background: var(--color-bg-app);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 12px 0;
}

.readme-content pre code {
  padding: 0;
  background: none;
}

.readme-content a {
  color: var(--color-accent);
}

.readme-content li {
  margin-left: 20px;
  list-style: disc;
}

.readme-empty {
  padding: 24px;
  text-align: center;
  color: var(--color-fg-muted);
  font-size: 13px;
}

/* 右键菜单 */
.context-menu {
  position: fixed;
  background: var(--color-bg-sidebar);
  border: 1px solid var(--color-border-subtle);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 4px;
  min-width: 160px;
  z-index: 9999;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: none;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  color: var(--color-fg);
  cursor: pointer;
  text-align: left;
}

.context-menu-item:hover {
  background: var(--color-bg-sidebar-hover);
}

.context-menu-item--danger {
  color: #ef4444;
}

.context-menu-item--danger:hover {
  background: rgba(239, 68, 68, 0.1);
}
</style>
