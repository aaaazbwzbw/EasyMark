import { createApp } from 'vue'
import './style.css'
import { notification } from './utils/notification'
import router from './router'
import RouterRoot from './RouterRoot.vue'
import { useSettings } from './composables/useSettings'
import { setupI18n } from './i18n'

// 重定向 console 到主进程（写入日志文件）
const electronAPI = window.electronAPI
if (electronAPI && 'sendLog' in electronAPI) {
  const sendLog = electronAPI.sendLog as (level: string, ...args: string[]) => void
  const originalLog = console.log
  const originalError = console.error
  const originalWarn = console.warn

  console.log = (...args: unknown[]) => {
    originalLog.apply(console, args)
    sendLog('info', ...args.map(a => typeof a === 'object' ? JSON.stringify(a) : String(a)))
  }

  console.error = (...args: unknown[]) => {
    originalError.apply(console, args)
    sendLog('error', ...args.map(a => typeof a === 'object' ? JSON.stringify(a) : String(a)))
  }

  console.warn = (...args: unknown[]) => {
    originalWarn.apply(console, args)
    sendLog('warn', ...args.map(a => typeof a === 'object' ? JSON.stringify(a) : String(a)))
  }
}

const app = createApp(RouterRoot)

// 初始化全局设置（例如主题与语言）
const { settings } = useSettings()
const i18n = setupI18n(app, settings.value.language)
const applyTheme = () => {
  const root = document.documentElement
  root.dataset.theme = settings.value.theme
  // 广播主题变化到其他窗口（如模型推理窗口）
  window.electronAPI?.broadcastThemeChange?.(settings.value.theme)
}
applyTheme()

app.provide('settings', settings)
app.provide('applyTheme', applyTheme)
app.provide('i18n', i18n)

// 使用路由
app.use(router)

// 将通知API挂载到全局属性，方便在组件中使用
app.config.globalProperties.$notification = notification

// 也挂载到window对象，方便在非Vue上下文中使用
declare global {
  interface Window {
    $notification: typeof notification
  }
}
window.$notification = notification

app.mount('#app')
