# 贡献指南

感谢你对 EasyMark 的关注！本文档为贡献者提供指南和信息。

## 目录

- [行为准则](#行为准则)
- [开始之前](#开始之前)
- [开发环境](#开发环境)
- [提交更改](#提交更改)
- [Pull Request 流程](#pull-request-流程)
- [代码规范](#代码规范)

## 行为准则

请在所有交流中保持尊重和建设性。我们致力于为每个人提供友好、包容的环境。

## 开始之前

### 环境要求

- Node.js 18+
- Go 1.21+
- Python 3.10+（插件开发需要）
- Git

### Fork 与克隆

1. 在 GitHub 上 Fork 本仓库
2. 克隆你的 Fork：
   ```bash
   git clone https://github.com/你的用户名/EasyMark.git
   cd EasyMark
   ```
3. 添加上游仓库：
   ```bash
   git remote add upstream https://github.com/aaaazbwzbw/EasyMark.git
   ```

## 开发环境

### 安装依赖

```bash
# 前端
cd frontend && npm install

# Electron
cd ../host-electron && npm install

# 后端
cd ../backend-go && go mod download
```

### 运行开发环境

```bash
# 终端 1：后端
cd backend-go && go run .

# 终端 2：前端
cd frontend && npm run dev

# 终端 3：Electron
cd host-electron && npm run dev
```

## 提交更改

### 分支命名

- `feature/` - 新功能
- `fix/` - Bug 修复
- `docs/` - 文档更新
- `refactor/` - 代码重构

示例：`feature/add-video-annotation`

### Commit 信息

遵循约定式提交格式：

```
类型(范围): 描述

[可选的正文]

[可选的脚注]
```

类型：
- `feat` - 新功能
- `fix` - Bug 修复
- `docs` - 文档
- `style` - 格式化
- `refactor` - 重构
- `test` - 测试
- `chore` - 维护

示例：
```
feat(annotation): 添加多边形平滑工具

- 使用贝塞尔曲线平滑多边形顶点
- 在工具栏添加开关按钮
```

## Pull Request 流程

1. **同步上游**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **创建功能分支**
   ```bash
   git checkout -b feature/your-feature
   ```

3. **提交更改**
   - 编写清晰、有注释的代码
   - 如适用，添加测试
   - 如需要，更新文档

4. **测试更改**
   ```bash
   # 前端测试
   cd frontend && npm run test

   # 后端测试
   cd backend-go && go test ./...
   ```

5. **提交 Pull Request**
   - 填写 PR 模板
   - 关联相关 Issue
   - 请求维护者审核

## 代码规范

### TypeScript/Vue

- 使用 TypeScript 严格模式
- 遵循 Vue 3 Composition API 模式
- 使用 `<script setup>` 语法
- 使用 Prettier 格式化

### Go

- 遵循 Go 标准规范
- 使用 `gofmt` 格式化
- 编写描述性的错误信息

### Python（插件）

- 遵循 PEP 8
- 使用类型注解
- 为公共函数编写文档

### 通用原则

- 保持函数小而专注
- 编写自解释的代码
- 为复杂逻辑添加注释
- **所有用户可见文本必须国际化**

## 有问题？

欢迎在 Issue 中提问或讨论！

---

感谢你为 EasyMark 做出贡献！
