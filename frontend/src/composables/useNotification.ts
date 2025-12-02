import { ref, computed } from 'vue'

export type NotificationType = 'success' | 'info' | 'warning' | 'error'

export interface NotificationButton {
  text: string
  onClick: () => void
}

export interface Notification {
  id: string
  type: NotificationType
  message: string
  persistent: boolean
  button?: NotificationButton
  createdAt: number
}

export interface NotificationOptions {
  persistent?: boolean
  button?: NotificationButton
}

// 全局通知状态
const notifications = ref<Notification[]>([])
const isNotificationPanelOpen = ref(false)
let notificationIdCounter = 0

// 生成唯一ID
function generateId(): string {
  return `notification-${Date.now()}-${notificationIdCounter++}`
}

// 计算是否有持久通知
const hasPersistentNotifications = computed(() => {
  return notifications.value.some(n => n.persistent)
})

// 添加通知
function addNotification(
  type: NotificationType,
  message: string,
  options: NotificationOptions = {}
): string {
  const id = generateId()
  const notification: Notification = {
    id,
    type,
    message,
    persistent: options.persistent || false,
    button: options.button,
    createdAt: Date.now()
  }

  notifications.value.push(notification)
  
  // 展开通知面板
  isNotificationPanelOpen.value = true

  // 如果不是持久通知，3秒后自动移除
  if (!notification.persistent) {
    setTimeout(() => {
      removeNotification(id)
      // 如果没有通知了，自动折叠面板
      if (notifications.value.length === 0) {
        isNotificationPanelOpen.value = false
      }
    }, 3000)
  }

  return id
}

// 移除通知
function removeNotification(id: string) {
  const index = notifications.value.findIndex(n => n.id === id)
  if (index !== -1) {
    notifications.value.splice(index, 1)
  }
}

// 更新或创建通知
function updateNotification(
  id: string,
  updates: {
    type?: NotificationType
    message?: string
    persistent?: boolean
    button?: NotificationButton
  }
) {
  let notification = notifications.value.find(n => n.id === id)
  
  // 如果通知不存在，创建新的
  if (!notification) {
    notification = {
      id,
      type: updates.type || 'info',
      message: updates.message || '',
      persistent: updates.persistent ?? false,
      button: updates.button,
      createdAt: Date.now()
    }
    notifications.value.push(notification)
    isNotificationPanelOpen.value = true
    
    // 如果不是持久通知，设置自动移除
    if (!notification.persistent) {
      setTimeout(() => {
        removeNotification(id)
        if (notifications.value.length === 0) {
          isNotificationPanelOpen.value = false
        }
      }, 3000)
    }
    return
  }
  
  // 更新已有通知
  if (updates.type !== undefined) notification.type = updates.type
  if (updates.message !== undefined) notification.message = updates.message
  if (updates.button !== undefined) notification.button = updates.button
  
  // 如果从持久改为非持久，启动自动移除
  if (updates.persistent === false && notification.persistent === true) {
    notification.persistent = false
    setTimeout(() => {
      removeNotification(id)
      if (notifications.value.length === 0) {
        isNotificationPanelOpen.value = false
      }
    }, 3000)
  } else if (updates.persistent !== undefined) {
    notification.persistent = updates.persistent
  }
}

// 清除所有通知
function clearAllNotifications() {
  notifications.value = []
}

// 切换面板显示
function toggleNotificationPanel() {
  isNotificationPanelOpen.value = !isNotificationPanelOpen.value
}

// 公开的API方法
export const notificationAPI = {
  success: (message: string, options?: NotificationOptions) => 
    addNotification('success', message, options),
  
  info: (message: string, options?: NotificationOptions) => 
    addNotification('info', message, options),
  
  warning: (message: string, options?: NotificationOptions) => 
    addNotification('warning', message, options),
  
  error: (message: string, options?: NotificationOptions) => 
    addNotification('error', message, options),
  
  update: updateNotification,
  
  remove: removeNotification,
  
  clear: clearAllNotifications
}

// 导出组合式函数
export function useNotification() {
  return {
    notifications,
    isNotificationPanelOpen,
    hasPersistentNotifications,
    toggleNotificationPanel,
    removeNotification,
    clearAllNotifications,
    ...notificationAPI
  }
}

