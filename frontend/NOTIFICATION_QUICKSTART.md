# 通知系统快速开始

## 立即使用

通知系统已经完全集成到应用中，可以立即使用！

### 在任何 Vue 组件中使用

```vue
<script setup lang="ts">
import { notification } from '@/utils/notification'

const handleSave = () => {
  // 显示成功通知
  notification.success('保存成功！')
}

const handleError = () => {
  // 显示错误通知
  notification.error('保存失败，请重试')
}
</script>

<template>
  <button @click="handleSave">保存</button>
</template>
```

### 在普通 TypeScript 文件中使用

```typescript
import { notification } from '@/utils/notification'

export async function fetchData() {
  const id = notification.info('正在加载数据...', { persistent: true })
  
  try {
    const data = await api.getData()
    notification.update(id, {
      type: 'success',
      message: '数据加载成功！',
      persistent: false
    })
    return data
  } catch (error) {
    notification.update(id, {
      type: 'error',
      message: '数据加载失败',
      persistent: false
    })
    throw error
  }
}
```

### 通过全局对象使用

```typescript
// 在浏览器控制台或任何地方
window.$notification.success('测试通知')
```

## 常用场景

### 1. 表单提交
```typescript
const submitForm = async () => {
  try {
    await api.submit(formData)
    notification.success('提交成功！')
  } catch (error) {
    notification.error('提交失败，请重试')
  }
}
```

### 2. 文件上传进度
```typescript
const uploadFile = (file: File) => {
  const id = notification.info('正在上传 0%', { persistent: true })
  
  const xhr = new XMLHttpRequest()
  xhr.upload.onprogress = (e) => {
    const percent = Math.round((e.loaded / e.total) * 100)
    notification.update(id, {
      message: `正在上传 ${percent}%`
    })
  }
  
  xhr.onload = () => {
    notification.update(id, {
      type: 'success',
      message: '上传成功！',
      persistent: false
    })
  }
  
  xhr.onerror = () => {
    notification.update(id, {
      type: 'error',
      message: '上传失败',
      persistent: false
    })
  }
  
  xhr.open('POST', '/upload')
  xhr.send(file)
}
```

### 3. 需要用户确认的操作
```typescript
const checkUpdate = () => {
  notification.warning('发现新版本 v2.0.0', {
    persistent: true,
    button: {
      text: '立即更新',
      onClick: () => {
        startUpdate()
      }
    }
  })
}
```

### 4. 网络连接状态
```typescript
window.addEventListener('offline', () => {
  notification.error('网络连接已断开', {
    persistent: true,
    button: {
      text: '重试',
      onClick: () => location.reload()
    }
  })
})

window.addEventListener('online', () => {
  notification.success('网络连接已恢复')
})
```

## 测试通知系统

打开浏览器控制台，输入：

```javascript
// 测试成功通知
window.$notification.success('这是成功通知')

// 测试持久通知
const id = window.$notification.info('持久通知', { persistent: true })

// 更新通知
window.$notification.update(id, { 
  type: 'success', 
  message: '更新后的通知',
  persistent: false 
})

// 清除所有通知
window.$notification.clear()
```

## 下一步

- 查看 `NOTIFICATION_API.md` 了解完整API文档
- 查看 `src/examples/notification-examples.ts` 了解更多示例
- 运行应用查看演示组件（主界面）

## 提示

- 短效通知会在3秒后自动消失
- 持久通知需要手动清除或更新为短效通知
- 持久通知在面板折叠时会显示黄色角标
- 可以同时显示多个通知
- 通知会按照创建时间从上到下排列

