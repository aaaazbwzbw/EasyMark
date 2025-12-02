# 路由配置说明

## 已完成的配置

### 1. 创建的文件

- **`frontend/src/router/index.ts`** - 路由配置文件
- **`frontend/src/views/HomePage.vue`** - 主页面
- **`frontend/src/views/UiPage.vue`** - UI 页面（包含通知演示）
- **`frontend/src/RouterRoot.vue`** - 路由根组件

### 2. 路由配置（嵌套路由）

```typescript
// frontend/src/router/index.ts
const routes = [
  {
    path: '/',
    component: App,  // App.vue 作为布局组件
    children: [
      {
        path: '',
        name: 'Home',
        component: HomePage  // 主页面
      },
      {
        path: 'ui',
        name: 'UI',
        component: UiPage  // UI 页面
      }
    ]
  }
]
```

### 3. 更新的文件

- **`frontend/src/main.ts`** - 集成了 Vue Router
- **`frontend/src/App.vue`** - 改为布局组件，`app-main` 区域使用 `<router-view />`

## 使用方法

### 访问路由

1. **主页面**: `http://localhost:5173/`
   - 显示 HomePage.vue 内容（后端健康检查信息）
   - 保留 App.vue 的布局（标题栏、侧边栏、底部栏、通知面板）

2. **UI 页面**: `http://localhost:5173/ui`
   - 显示 UiPage.vue 内容（通知系统演示）
   - 保留 App.vue 的布局（标题栏、侧边栏、底部栏、通知面板）

### 布局结构

所有页面都在 App.vue 的 `app-main` 区域内挂载，共享相同的布局：
- 顶部标题栏（带窗口控制按钮）
- 左侧侧边栏（项目、数据集、训练、插件）
- 底部栏（通知按钮）
- 通知面板（右下角）

### 在代码中导航

```typescript
import { useRouter } from 'vue-router'

const router = useRouter()

// 跳转到 UI 页面
router.push('/ui')

// 跳转到主页
router.push('/')
```

### 在模板中导航

```vue
<template>
  <!-- 使用 router-link -->
  <router-link to="/ui">前往 UI 页面</router-link>
  <router-link to="/">返回主页</router-link>
</template>
```

## 测试

启动开发服务器后：

```bash
cd frontend
npm run dev
```

然后在浏览器中访问：
- `http://localhost:5173/` - 主页面
- `http://localhost:5173/ui` - 空白 UI 页面

## 下一步

现在 `/ui` 路由已经配置好了，你可以：

1. 在 `frontend/src/views/UiPage.vue` 中添加你需要的内容
2. 添加更多路由到 `frontend/src/router/index.ts`
3. 使用嵌套路由、路由守卫等高级功能

## 文件结构

```
frontend/src/
├── router/
│   └── index.ts           # 路由配置（嵌套路由）
├── views/
│   ├── HomePage.vue       # 主页面
│   └── UiPage.vue         # UI 页面（通知演示）
├── RouterRoot.vue         # 路由根组件
├── App.vue                # 布局组件（标题栏、侧边栏、底部栏）
└── main.ts                # 入口文件（已集成路由）
```

## 架构说明

```
RouterRoot.vue (根组件)
  └── <router-view />
      └── App.vue (布局组件)
          ├── 标题栏
          ├── 侧边栏
          ├── app-main
          │   └── <router-view /> (页面内容区域)
          │       ├── HomePage.vue (/)
          │       └── UiPage.vue (/ui)
          ├── 底部栏
          └── 通知面板
```

