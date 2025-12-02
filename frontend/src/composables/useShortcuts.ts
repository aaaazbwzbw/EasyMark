import { ref, watch } from 'vue'

// 快捷键动作定义
export type ShortcutAction =
  | 'save'              // 保存标注
  | 'saveAsNegative'    // 保存为负样本
  | 'prevImage'         // 上一张图片
  | 'nextImage'         // 下一张图片
  | 'prevUnannotated'   // 上一张未标注图片
  | 'nextUnannotated'   // 下一张未标注图片
  | 'resetView'         // 重置视图
  | 'deleteSelected'    // 删除选中标注
  | 'toggleKeypointVisibility' // 切换关键点可见性

// 快捷键配置类型
export interface ShortcutConfig {
  key: string           // 主键（如 'S', 'ArrowLeft'）
  ctrl?: boolean        // 是否需要 Ctrl
  shift?: boolean       // 是否需要 Shift
  alt?: boolean         // 是否需要 Alt
}

export interface ShortcutSettings {
  [key: string]: ShortcutConfig  // action -> config
}

// 默认快捷键配置
const defaultShortcuts: ShortcutSettings = {
  save: { key: 'S', ctrl: true },
  saveAsNegative: { key: 'S', ctrl: true, shift: true },
  prevImage: { key: 'ArrowLeft' },
  nextImage: { key: 'ArrowRight' },
  prevUnannotated: { key: 'ArrowLeft', ctrl: true },
  nextUnannotated: { key: 'ArrowRight', ctrl: true },
  resetView: { key: '0', ctrl: true },
  deleteSelected: { key: 'Backspace' },
  toggleKeypointVisibility: { key: 'V' }
}

const STORAGE_KEY = 'easymark_shortcuts'

// 将快捷键配置转换为显示字符串
export function shortcutToString(config: ShortcutConfig): string {
  const parts: string[] = []
  if (config.ctrl) parts.push('Ctrl')
  if (config.shift) parts.push('Shift')
  if (config.alt) parts.push('Alt')
  
  // 特殊键名转换
  const keyDisplay: Record<string, string> = {
    'ArrowLeft': '←',
    'ArrowRight': '→',
    'ArrowUp': '↑',
    'ArrowDown': '↓',
    'Backspace': '⌫',
    'Delete': 'Del',
    'Escape': 'Esc',
    'Enter': '↵',
    'Space': '空格',
    ' ': '空格'
  }
  
  parts.push(keyDisplay[config.key] || config.key.toUpperCase())
  return parts.join(' + ')
}

// 检查键盘事件是否匹配快捷键配置
export function matchShortcut(e: KeyboardEvent, config: ShortcutConfig): boolean {
  const ctrlMatch = !!config.ctrl === (e.ctrlKey || e.metaKey)
  const shiftMatch = !!config.shift === e.shiftKey
  const altMatch = !!config.alt === e.altKey
  
  // 处理键值匹配
  let keyMatch = false
  if (config.key.length === 1) {
    // 单字符键（字母/数字）
    keyMatch = e.key.toUpperCase() === config.key.toUpperCase()
  } else {
    // 特殊键（如 ArrowLeft, Backspace）
    keyMatch = e.key === config.key || e.code === config.key
  }
  
  return ctrlMatch && shiftMatch && altMatch && keyMatch
}

// 加载保存的快捷键配置
function loadShortcuts(): ShortcutSettings {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaultShortcuts }
    const parsed = JSON.parse(raw) as Record<string, ShortcutConfig>
    // 合并：只覆盖有效的配置
    const result: ShortcutSettings = { ...defaultShortcuts }
    for (const key of Object.keys(parsed)) {
      if (parsed[key] && typeof parsed[key].key === 'string') {
        result[key] = parsed[key]
      }
    }
    return result
  } catch {
    return { ...defaultShortcuts }
  }
}

const shortcutsRef = ref<ShortcutSettings>(loadShortcuts())

// 监听变化并保存
watch(
  shortcutsRef,
  (value) => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    } catch {
      // ignore
    }
  },
  { deep: true }
)

export function useShortcuts() {
  // 更新单个快捷键
  const updateShortcut = (action: ShortcutAction, config: ShortcutConfig) => {
    shortcutsRef.value = {
      ...shortcutsRef.value,
      [action]: config
    }
  }

  // 重置为默认
  const resetToDefault = () => {
    shortcutsRef.value = { ...defaultShortcuts }
  }

  // 重置单个快捷键
  const resetShortcut = (action: ShortcutAction) => {
    if (defaultShortcuts[action]) {
      shortcutsRef.value = {
        ...shortcutsRef.value,
        [action]: { ...defaultShortcuts[action] }
      }
    }
  }

  // 获取指定动作的快捷键配置
  const getShortcut = (action: ShortcutAction): ShortcutConfig | undefined => {
    return shortcutsRef.value[action]
  }

  // 获取指定动作的快捷键显示字符串
  const getShortcutDisplay = (action: ShortcutAction): string => {
    const config = shortcutsRef.value[action]
    return config ? shortcutToString(config) : ''
  }

  // 检查键盘事件匹配哪个动作
  const matchAction = (e: KeyboardEvent): ShortcutAction | null => {
    // 按优先级排序：更复杂的组合键优先匹配
    const actions: ShortcutAction[] = [
      'saveAsNegative',    // Ctrl+Shift+S 优先于 Ctrl+S
      'save',
      'prevUnannotated',   // Ctrl+Left 优先于 Left
      'nextUnannotated',
      'prevImage',
      'nextImage',
      'resetView',
      'deleteSelected',
      'toggleKeypointVisibility'
    ]

    for (const action of actions) {
      const config = shortcutsRef.value[action]
      if (config && matchShortcut(e, config)) {
        return action
      }
    }
    return null
  }

  return {
    shortcuts: shortcutsRef,
    defaultShortcuts,
    updateShortcut,
    resetToDefault,
    resetShortcut,
    getShortcut,
    getShortcutDisplay,
    matchAction,
    shortcutToString
  }
}
