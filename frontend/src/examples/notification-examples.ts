/**
 * 通知系统使用示例
 * 
 * 这个文件展示了如何使用通知系统的各种功能
 */

import { notification } from '../utils/notification'

// ============================================
// 1. 成功通知（3秒后自动消失）
// ============================================
export function showSuccessNotification() {
  notification.success('操作成功！')
}

// ============================================
// 2. 普通通知（3秒后自动消失）
// ============================================
export function showInfoNotification() {
  notification.info('这是一条提示信息')
}

// ============================================
// 3. 警告通知（3秒后自动消失）
// ============================================
export function showWarningNotification() {
  notification.warning('请注意检查输入内容')
}

// ============================================
// 4. 错误通知（3秒后自动消失）
// ============================================
export function showErrorNotification() {
  notification.error('操作失败，请重试')
}

// ============================================
// 5. 带按钮的警告通知
// ============================================
export function showWarningWithButton() {
  notification.warning('发现新版本可用', {
    button: {
      text: '立即更新',
      onClick: () => {
        console.log('开始更新...')
        notification.info('正在下载更新...')
      }
    }
  })
}

// ============================================
// 6. 带按钮的错误通知
// ============================================
export function showErrorWithButton() {
  notification.error('连接服务器失败', {
    button: {
      text: '重试',
      onClick: () => {
        console.log('重新连接...')
        notification.info('正在重新连接...')
      }
    }
  })
}

// ============================================
// 7. 持久通知（不会自动消失）
// ============================================
export function showPersistentNotification() {
  const id = notification.info('这是一条持久通知，不会自动消失', {
    persistent: true
  })
  return id
}

// ============================================
// 8. 进度通知示例（持久通知 + 实时更新）
// ============================================
export function showProgressNotification() {
  // 创建一个持久的进度通知
  const id = notification.info('正在导入数据 0%', { persistent: true })
  
  let progress = 0
  const interval = setInterval(() => {
    progress += 10
    
    if (progress < 100) {
      // 更新进度
      notification.update(id, {
        message: `正在导入数据 ${progress}%`
      })
    } else {
      // 完成后改为成功通知，并转为短效通知
      notification.update(id, {
        type: 'success',
        message: '数据导入成功！',
        persistent: false
      })
      clearInterval(interval)
    }
  }, 500)
  
  return id
}

// ============================================
// 9. 复杂的异步操作示例
// ============================================
export async function showAsyncOperationNotification() {
  // 开始时显示持久通知
  const id = notification.info('正在处理请求...', { persistent: true })
  
  try {
    // 模拟异步操作
    await new Promise(resolve => setTimeout(resolve, 2000))
    
    // 成功后更新为成功通知
    notification.update(id, {
      type: 'success',
      message: '请求处理成功！',
      persistent: false
    })
  } catch (error) {
    // 失败时更新为错误通知，并添加重试按钮
    notification.update(id, {
      type: 'error',
      message: '请求处理失败',
      persistent: false,
      button: {
        text: '重试',
        onClick: () => showAsyncOperationNotification()
      }
    })
  }
}

// ============================================
// 10. 手动移除通知
// ============================================
export function showRemovableNotification() {
  const id = notification.info('这条通知5秒后会被手动移除', { persistent: true })
  
  setTimeout(() => {
    notification.remove(id)
  }, 5000)
}

// ============================================
// 11. 清除所有通知
// ============================================
export function clearAllNotifications() {
  notification.clear()
}

