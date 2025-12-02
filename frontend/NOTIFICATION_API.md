# 通知系统 API 文档

## 概述

EasyMark 提供了一个完整的通知系统，支持多种通知类型、持久通知、进度更新等功能。

## 通知类型

### 1. 成功通知 (Success)
- **图标**: 绿色的勾选图标
- **用途**: 显示操作成功的消息
- **默认行为**: 3秒后自动消失

```typescript
notification.success('操作成功！')
```

### 2. 普通通知 (Info)
- **图标**: 蓝色的信息图标
- **用途**: 显示一般提示信息
- **默认行为**: 3秒后自动消失

```typescript
notification.info('这是一条提示信息')
```

### 3. 警告通知 (Warning)
- **图标**: 黄色的警告图标
- **用途**: 显示警告信息
- **默认行为**: 3秒后自动消失
- **可选**: 支持添加操作按钮

```typescript
notification.warning('请注意检查输入内容')

// 带按钮的警告
notification.warning('发现新版本', {
  button: {
    text: '更新',
    onClick: () => console.log('开始更新')
  }
})
```

### 4. 错误通知 (Error)
- **图标**: 红色的错误图标
- **用途**: 显示错误信息
- **默认行为**: 3秒后自动消失
- **可选**: 支持添加操作按钮

```typescript
notification.error('操作失败，请重试')

// 带按钮的错误
notification.error('连接失败', {
  button: {
    text: '重试',
    onClick: () => console.log('重新连接')
  }
})
```

## API 方法

### notification.success(message, options?)
显示成功通知

### notification.info(message, options?)
显示普通通知

### notification.warning(message, options?)
显示警告通知

### notification.error(message, options?)
显示错误通知

### notification.update(id, updates)
更新已存在的通知

### notification.remove(id)
手动移除指定通知

### notification.clear()
清除所有通知

## 选项参数 (Options)

```typescript
interface NotificationOptions {
  persistent?: boolean        // 是否持久显示（不自动消失）
  button?: {                 // 可选的操作按钮
    text: string             // 按钮文本
    onClick: () => void      // 点击回调
  }
}
```

## 使用示例

### 基础用法

```typescript
import { notification } from '@/utils/notification'

// 显示成功消息
notification.success('保存成功！')

// 显示错误消息
notification.error('保存失败，请重试')
```

### 持久通知

```typescript
// 创建持久通知（不会自动消失）
const id = notification.info('正在处理...', { persistent: true })

// 稍后更新通知
notification.update(id, {
  type: 'success',
  message: '处理完成！',
  persistent: false  // 转为短效通知，3秒后消失
})
```

### 进度通知

```typescript
// 显示进度
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
      message: '导入成功！',
      persistent: false
    })
    clearInterval(interval)
  }
}, 500)
```

### 带按钮的通知

```typescript
notification.warning('发现新版本', {
  button: {
    text: '立即更新',
    onClick: () => {
      // 执行更新逻辑
      startUpdate()
    }
  }
})
```

## 全局访问

通知API已挂载到全局，可以通过以下方式访问：

```typescript
// 在 Vue 组件中
this.$notification.success('成功！')

// 在任何地方（包括非Vue上下文）
window.$notification.success('成功！')

// 推荐：通过导入使用
import { notification } from '@/utils/notification'
notification.success('成功！')
```

## 特性

- ✅ 4种通知类型（成功、信息、警告、错误）
- ✅ 自动消失（默认3秒）
- ✅ 持久通知（不自动消失）
- ✅ 实时更新通知内容和类型
- ✅ 可选的操作按钮
- ✅ 持久通知的角标提示
- ✅ 平滑的动画效果
- ✅ 全局API访问

