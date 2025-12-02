<script setup lang="ts">
import { notification } from '../utils/notification'

// 1. 成功通知
const showSuccess = () => {
  notification.success('操作成功！')
}

// 2. 普通通知
const showInfo = () => {
  notification.info('这是一条提示信息')
}

// 3. 警告通知
const showWarning = () => {
  notification.warning('请注意检查输入内容')
}

// 4. 错误通知
const showError = () => {
  notification.error('操作失败，请重试')
}

// 5. 带按钮的警告
const showWarningWithButton = () => {
  notification.warning('发现新版本可用', {
    button: {
      text: '立即更新',
      onClick: () => {
        notification.info('正在下载更新...')
      }
    }
  })
}

// 6. 带按钮的错误
const showErrorWithButton = () => {
  notification.error('连接服务器失败', {
    button: {
      text: '重试',
      onClick: () => {
        notification.info('正在重新连接...')
      }
    }
  })
}

// 7. 持久通知
const showPersistent = () => {
  notification.info('这是一条持久通知，不会自动消失', {
    persistent: true
  })
}

// 8. 进度通知
const showProgress = () => {
  const id = notification.info('正在导入数据 0%', { persistent: true })
  
  let progress = 0
  const interval = setInterval(() => {
    progress += 10
    
    if (progress < 100) {
      notification.update(id, {
        message: `正在导入数据 ${progress}%`
      })
    } else {
      notification.update(id, {
        type: 'success',
        message: '数据导入成功！',
        persistent: false
      })
      clearInterval(interval)
    }
  }, 500)
}

// 9. 异步操作
const showAsync = async () => {
  const id = notification.info('正在处理请求...', { persistent: true })
  
  await new Promise(resolve => setTimeout(resolve, 2000))
  
  notification.update(id, {
    type: 'success',
    message: '请求处理成功！',
    persistent: false
  })
}

// 10. 长文本示例
const showLongText = () => {
  const longText = `这是一条非常长的通知消息，用于测试通知系统对长文本的处理能力。\n\n` +
    `在计算机科学中，通知系统是一种用于向用户提供信息的机制，通常用于显示应用程序的状态更新、错误信息或其他重要通知。\n\n` +
    `良好的通知系统应该能够：\n` +
    `• 自动换行长文本\n` +
    `• 支持滚动查看完整内容\n` +
    `• 保持通知界面的整洁和易读性\n` +
    `• 提供适当的视觉反馈\n\n` +
    `这个示例展示了如何处理包含多段文本的通知消息，确保用户能够轻松阅读完整内容。`
  
  notification.info(longText, {
    persistent: true
  })
}
</script>

<template>
  <div class="demo-container">
    <h2>通知系统演示</h2>
    
    <div class="demo-section">
      <h3>基础通知（3秒后自动消失）</h3>
      <div class="demo-buttons">
        <button @click="showSuccess">成功通知</button>
        <button @click="showInfo">普通通知</button>
        <button @click="showWarning">警告通知</button>
        <button @click="showError">错误通知</button>
      </div>
    </div>

    <div class="demo-section">
      <h3>带按钮的通知</h3>
      <div class="demo-buttons">
        <button @click="showWarningWithButton">警告 + 按钮</button>
        <button @click="showErrorWithButton">错误 + 按钮</button>
      </div>
    </div>

    <div class="demo-section">
      <h3>持久通知</h3>
      <div class="demo-buttons">
        <button @click="showPersistent">持久通知</button>
        <button @click="showProgress">进度通知</button>
        <button @click="showAsync">异步操作</button>
        <button @click="showLongText">长文本示例</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.demo-container {
  padding: 2rem;
  max-width: 800px;
  margin: 0 auto;
}

h2 {
  margin-bottom: 2rem;
  color: var(--color-fg);
}

.demo-section {
  margin-bottom: 2rem;
}

.demo-section h3 {
  margin-bottom: 1rem;
  font-size: 1rem;
  color: var(--color-fg-muted);
}

.demo-buttons {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.demo-buttons button {
  padding: 8px 16px;
  font-size: 0.875rem;
  border: 1px solid var(--color-accent);
  background-color: var(--color-accent);
  color: #ffffff;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.demo-buttons button:hover {
  background-color: var(--color-accent-hover);
}
</style>

