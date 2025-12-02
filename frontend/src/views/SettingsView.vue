<script setup lang="ts">
import { inject, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettings } from '../composables/useSettings'
import { useShortcuts, shortcutToString, type ShortcutAction, type ShortcutConfig } from '../composables/useShortcuts'

type SettingsTab = 'general' | 'paths' | 'shortcuts'

const activeTab = ref<SettingsTab>('general')

// 快捷键相关
const { shortcuts, updateShortcut, resetShortcut, resetToDefault } = useShortcuts()

// 快捷键录入状态
const recordingAction = ref<ShortcutAction | null>(null)
const recordedKeys = ref<ShortcutConfig | null>(null)

// 快捷键动作列表（用于显示）
const shortcutActions: { action: ShortcutAction; labelKey: string }[] = [
  { action: 'save', labelKey: 'settings.shortcuts.actions.save' },
  { action: 'saveAsNegative', labelKey: 'settings.shortcuts.actions.saveAsNegative' },
  { action: 'prevImage', labelKey: 'settings.shortcuts.actions.prevImage' },
  { action: 'nextImage', labelKey: 'settings.shortcuts.actions.nextImage' },
  { action: 'prevUnannotated', labelKey: 'settings.shortcuts.actions.prevUnannotated' },
  { action: 'nextUnannotated', labelKey: 'settings.shortcuts.actions.nextUnannotated' },
  { action: 'resetView', labelKey: 'settings.shortcuts.actions.resetView' },
  { action: 'deleteSelected', labelKey: 'settings.shortcuts.actions.deleteSelected' },
  { action: 'toggleKeypointVisibility', labelKey: 'settings.shortcuts.actions.toggleKeypointVisibility' }
]

// 开始录入快捷键
const startRecording = (action: ShortcutAction) => {
  recordingAction.value = action
  recordedKeys.value = null
}

// 处理录入时的按键
const handleRecordKeydown = (e: KeyboardEvent) => {
  if (!recordingAction.value) return
  
  e.preventDefault()
  e.stopPropagation()
  
  // Escape 取消录入
  if (e.key === 'Escape') {
    recordingAction.value = null
    recordedKeys.value = null
    return
  }
  
  // 忽略单独的修饰键
  if (['Control', 'Shift', 'Alt', 'Meta'].includes(e.key)) {
    return
  }
  
  // 构建快捷键配置
  const config: ShortcutConfig = {
    key: e.key.length === 1 ? e.key.toUpperCase() : e.key,
    ctrl: e.ctrlKey || e.metaKey,
    shift: e.shiftKey,
    alt: e.altKey
  }
  
  // 保存快捷键
  updateShortcut(recordingAction.value, config)
  recordingAction.value = null
  recordedKeys.value = null
}

// 获取快捷键显示文本
const getShortcutText = (action: ShortcutAction): string => {
  const config = shortcuts.value[action]
  return config ? shortcutToString(config) : ''
}

// 重置单个快捷键
const handleResetShortcut = (action: ShortcutAction) => {
  resetShortcut(action)
}

// 重置所有快捷键
const handleResetAll = () => {
  resetToDefault()
}

const switchTab = (tab: SettingsTab) => {
  activeTab.value = tab
}

const { settings, setTheme, setLanguage, updatePaths } = useSettings()
const { t, locale } = useI18n()
const applyTheme = inject<() => void>('applyTheme')

const handleThemeChange = (event: Event) => {
  const value = (event.target as HTMLSelectElement).value as 'dark' | 'light'
  setTheme(value)
  applyTheme && applyTheme()
}

const handleLanguageChange = (event: Event) => {
  const value = (event.target as HTMLSelectElement).value as 'zh-CN' | 'en-US'
  setLanguage(value)
  locale.value = value
  // 广播语言变化到其他窗口
  window.electronAPI?.broadcastLocaleChange?.(value)
}

const normalizePath = (raw: string, previous: string) => {
  const next = raw.replace(/\//g, '\\').trim()
  return next || previous
}

const handleDataPathChange = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  updatePaths({ dataPath: normalizePath(value, settings.value.dataPath) })
}

const handleDatasetExportPathChange = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  updatePaths({ datasetExportPath: normalizePath(value, settings.value.datasetExportPath) })
}

const handleModelOutputPathChange = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  updatePaths({ modelOutputPath: normalizePath(value, settings.value.modelOutputPath) })
}

const selectDirectory = async (currentPath: string) => {
  const anyWindow = window as any
  const api = anyWindow?.electronAPI
  if (!api || typeof api.selectDirectory !== 'function') {
    return null
  }

  try {
    const selected = (await api.selectDirectory(currentPath)) as string | null | undefined
    if (!selected) return null
    return normalizePath(String(selected), currentPath)
  } catch {
    return null
  }
}

const handleBrowseDataPath = async () => {
  const next = await selectDirectory(settings.value.dataPath)
  if (next) {
    updatePaths({ dataPath: next })
  }
}

const handleBrowseDatasetExportPath = async () => {
  const next = await selectDirectory(settings.value.datasetExportPath)
  if (next) {
    updatePaths({ datasetExportPath: next })
  }
}

const handleBrowseModelOutputPath = async () => {
  const next = await selectDirectory(settings.value.modelOutputPath)
  if (next) {
    updatePaths({ modelOutputPath: next })
  }
}
</script>

<template>
  <div class="settings-overlay">
    <div class="settings-layout">
      <aside class="settings-sidebar">
        <div class="settings-sidebar__header">
          <h2 class="settings-title">设置</h2>
        </div>
        
        <nav class="settings-nav">
          <div
            class="settings-nav__item"
            :class="{ active: activeTab === 'general' }"
            @click="switchTab('general')"
          >
            {{ t('settings.general') }}
          </div>
          <div
            class="settings-nav__item"
            :class="{ active: activeTab === 'paths' }"
            @click="switchTab('paths')"
          >
            {{ t('settings.paths') }}
          </div>
          <div
            class="settings-nav__item"
            :class="{ active: activeTab === 'shortcuts' }"
            @click="switchTab('shortcuts')"
          >
            {{ t('settings.shortcuts.title') }}
          </div>
        </nav>
      </aside>
      
      <main class="settings-content" @keydown="handleRecordKeydown">
        <div class="settings-content__header">
          <h1 v-if="activeTab === 'general'">{{ t('settings.general') }}</h1>
          <h1 v-else-if="activeTab === 'paths'">{{ t('settings.paths') }}</h1>
          <h1 v-else>{{ t('settings.shortcuts.title') }}</h1>
        </div>
        <div class="settings-content__body">
          <!-- 通用设置 Tab -->
          <template v-if="activeTab === 'general'">
            <section class="settings-section">
              <h2 class="settings-section__title">{{ t('settings.sections.language.title') }}</h2>
              <p class="settings-section__description">{{ t('settings.sections.language.description') }}</p>
              <div class="settings-field-group">
                <div class="settings-select-field">
                  <label class="settings-select-field__label" for="settings-language-select">{{ t('settings.language') }}</label>
                  <div class="settings-select-field__control">
                    <select
                      id="settings-language-select"
                      name="language"
                      :value="settings.language"
                      @change="handleLanguageChange"
                    >
                      <option value="zh-CN">简体中文 (zh-CN)</option>
                      <option value="en-US">English (en-US)</option>
                    </select>
                  </div>
                </div>
              </div>
            </section>

            <section class="settings-section">
              <h2 class="settings-section__title">{{ t('settings.sections.theme.title') }}</h2>
              <p class="settings-section__description">{{ t('settings.sections.theme.description') }}</p>
              <div class="settings-field-group">
                <div class="settings-select-field">
                  <label class="settings-select-field__label" for="settings-theme-select">主题模式</label>
                  <div class="settings-select-field__control">
                    <select
                      id="settings-theme-select"
                      name="theme"
                      :value="settings.theme"
                      @change="handleThemeChange"
                    >
                      <option value="dark">深色主题 (Dark)</option>
                      <option value="light">浅色主题 (Light)</option>
                    </select>
                  </div>
                </div>
              </div>
            </section>
          </template>

          <!-- 路径设置 Tab -->
          <template v-else-if="activeTab === 'paths'">
            <section class="settings-section">
              <h2 class="settings-section__title">{{ t('settings.sections.paths.title') }}</h2>
              <p class="settings-section__description">
                {{ t('settings.sections.paths.description') }}
              </p>
              
              <!-- 项目数据路径 -->
              <h3 class="settings-subsection__title">{{ t('settings.sections.paths.dataPathLabel') }}</h3>
              <div class="settings-field-group settings-field-group--inline">
                <div class="settings-text-field">
                  <label class="settings-text-field__label">{{ t('settings.sections.paths.currentPath') }}</label>
                  <div class="settings-text-field__input" title="示例路径">
                    <input
                      type="text"
                      class="settings-text-field__input-el"
                      :value="settings.dataPath"
                      @change="handleDataPathChange"
                    />
                  </div>
                </div>
                <button type="button" class="settings-button" @click="handleBrowseDataPath">
                  {{ t('settings.sections.paths.browse') }}
                </button>
              </div>
              
              <!-- 数据集导出路径 -->
              <h3 class="settings-subsection__title">{{ t('settings.sections.paths.datasetExportLabel') }}</h3>
              <div class="settings-field-group settings-field-group--inline">
                <div class="settings-text-field">
                  <label class="settings-text-field__label">{{ t('settings.sections.paths.currentPath') }}</label>
                  <div class="settings-text-field__input" title="示例路径">
                    <input
                      type="text"
                      class="settings-text-field__input-el"
                      :value="settings.datasetExportPath"
                      @change="handleDatasetExportPathChange"
                    />
                  </div>
                </div>
                <button type="button" class="settings-button" @click="handleBrowseDatasetExportPath">
                  {{ t('settings.sections.paths.browse') }}
                </button>
              </div>

              <!-- 模型训练输出路径 -->
              <h3 class="settings-subsection__title">{{ t('settings.sections.paths.modelOutputLabel') }}</h3>
              <div class="settings-field-group settings-field-group--inline">
                <div class="settings-text-field">
                  <label class="settings-text-field__label">{{ t('settings.sections.paths.currentPath') }}</label>
                  <div class="settings-text-field__input" title="示例路径">
                    <input
                      type="text"
                      class="settings-text-field__input-el"
                      :value="settings.modelOutputPath"
                      @change="handleModelOutputPathChange"
                    />
                  </div>
                </div>
                <button type="button" class="settings-button" @click="handleBrowseModelOutputPath">
                  {{ t('settings.sections.paths.browse') }}
                </button>
              </div>
            </section>
          </template>

          <!-- 快捷键设置 Tab -->
          <template v-else-if="activeTab === 'shortcuts'">
            <section class="settings-section">
              <h2 class="settings-section__title">{{ t('settings.shortcuts.sectionTitle') }}</h2>
              <p class="settings-section__description">{{ t('settings.shortcuts.description') }}</p>
              
              <div class="shortcuts-list">
                <div 
                  v-for="item in shortcutActions" 
                  :key="item.action"
                  class="shortcut-item"
                >
                  <span class="shortcut-item__label">{{ t(item.labelKey) }}</span>
                  <div class="shortcut-item__controls">
                    <button
                      type="button"
                      class="shortcut-item__key"
                      :class="{ 'shortcut-item__key--recording': recordingAction === item.action }"
                      @click="startRecording(item.action)"
                    >
                      {{ recordingAction === item.action ? t('settings.shortcuts.pressKey') : getShortcutText(item.action) }}
                    </button>
                    <button
                      type="button"
                      class="shortcut-item__reset"
                      :title="t('settings.shortcuts.reset')"
                      @click="handleResetShortcut(item.action)"
                    >
                      ↺
                    </button>
                  </div>
                </div>
              </div>
              
              <div class="shortcuts-actions">
                <button type="button" class="settings-button" @click="handleResetAll">
                  {{ t('settings.shortcuts.resetAll') }}
                </button>
              </div>
            </section>
          </template>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.settings-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: transparent;
  z-index: 100;
  display: flex;
  flex-direction: column;
  text-align: left;
  color: var(--color-fg);
}

.settings-layout {
  display: flex;
  flex: 1;
  height: 100%;
  overflow: hidden;
}

.settings-sidebar {
  width: 180px;
  background-color: var(--color-bg-sidebar);
  border-right: 1px solid var(--color-border-subtle);
  display: flex;
  flex-direction: column;
}

.settings-sidebar__header {
  height: 60px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid var(--color-border-subtle);
  gap: 12px;
}

.settings-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: var(--color-fg);
}

.settings-nav {
  flex: 1;
  padding: 0 0 16px 0;
  overflow-y: auto;
}

.settings-nav__item {
  padding: 12px 24px;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 15px;
  color: #9ca3af;
}

.settings-nav__item:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.settings-nav__item.active {
  background-color: var(--color-accent-soft);
  color: var(--color-accent);
  font-weight: 500;
  border-right: 3px solid var(--color-accent);
}

.settings-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: var(--color-bg-app);
}

.settings-content__header {
  height: 60px;
  display: flex;
  align-items: center;
  padding: 0 32px;
  border-bottom: 1px solid var(--color-border-subtle);
}

.settings-content__header h1 {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
  color: var(--color-fg);
}

.settings-content__body {
  flex: 1;
  padding: 32px;
  overflow-y: auto;
}
.settings-section {
  margin-bottom: 32px;
}

.settings-section__title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: var(--color-fg);
}

.settings-section__description {
  margin: 0 0 12px 0;
  font-size: 13px;
  color: var(--color-fg-muted);
}

.settings-field-group {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 24px;
}

.settings-field-group--inline {
  align-items: flex-end;
}

.settings-radio {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #e5e7eb;
}

.settings-radio input[type='radio'] {
  accent-color: #60a5fa;
}

.settings-radio__label {
  cursor: pointer;
}

.settings-select-field {
  min-width: 220px;
}

.settings-select-field__label {
  display: block;
  font-size: 13px;
  color: var(--color-fg-muted);
  margin-bottom: 6px;
}

.settings-select-field__control select {
  width: 100%;
  background-color: var(--color-bg-sidebar);
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  padding: 6px 10px;
  color: var(--color-fg);
  font-size: 13px;
  outline: none;
}

.settings-select-field__control select:focus-visible {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 40%, transparent);
}

.settings-select-field__control select:disabled {
  opacity: 0.8;
  cursor: not-allowed;
}

.settings-text-field {
  flex: 1;
  min-width: 0;
}

.settings-text-field__label {
  display: block;
  font-size: 13px;
  color: var(--color-fg-muted);
  margin-bottom: 6px;
}

.settings-text-field__input {
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  padding: 8px 10px;
  background-color: var(--color-bg-sidebar);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.settings-text-field__value {
  font-size: 13px;
  color: var(--color-fg);
  word-break: break-all;
}

.settings-text-field__input-el {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-fg);
  font-size: 13px;
  padding: 0;
}

.settings-text-field__hint {
  font-size: 12px;
  color: var(--color-fg-muted);
}

.settings-button {
  padding: 8px 14px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background-color: var(--color-bg-sidebar);
  color: var(--color-fg);
  font-size: 13px;
  cursor: pointer;
  opacity: 1;
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.settings-button:hover {
  background-color: var(--color-bg-sidebar-hover);
  border-color: var(--color-accent);
}

.settings-section__hint {
  margin: 8px 0 0 0;
  font-size: 12px;
  color: #6b7280;
}

/* 快捷键设置样式 */
.shortcuts-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 24px;
}

.shortcut-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background-color: var(--color-bg-sidebar);
  border-radius: 6px;
  border: 1px solid var(--color-border-subtle);
}

.shortcut-item__label {
  font-size: 14px;
  color: var(--color-fg);
}

.shortcut-item__controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.shortcut-item__key {
  min-width: 120px;
  padding: 6px 12px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background-color: var(--color-bg-app);
  color: var(--color-fg);
  font-size: 13px;
  font-family: monospace;
  cursor: pointer;
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.shortcut-item__key:hover {
  border-color: var(--color-accent);
}

.shortcut-item__key--recording {
  border-color: var(--color-accent);
  background-color: var(--color-accent-soft);
  color: var(--color-accent);
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.shortcut-item__reset {
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background-color: transparent;
  color: var(--color-fg-muted);
  font-size: 14px;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.shortcut-item__reset:hover {
  color: var(--color-accent);
  border-color: var(--color-accent);
}

.shortcuts-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
