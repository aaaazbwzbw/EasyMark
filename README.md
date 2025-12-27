<div align="center">

# EasyMark

<img src="docs/assets/logo.png" alt="EasyMark Logo" width="120">

**专业的计算机视觉标注工具**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-blue.svg)]()
[![Electron](https://img.shields.io/badge/Electron-28-47848F.svg?logo=electron)](https://www.electronjs.org/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D.svg?logo=vue.js)](https://vuejs.org/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?logo=go)](https://golang.org/)
[![Latest Release](https://img.shields.io/github/v/release/aaaazbwzbw/EasyMark)](https://github.com/aaaazbwzbw/EasyMark/releases/latest)
[![Total Downloads](https://img.shields.io/github/downloads/aaaazbwzbw/EasyMark/total)](https://github.com/aaaazbwzbw/EasyMark/releases)

[简体中文](README.md) | [English](README_en-US.md)

⭐ **大家的 Star 就是我更新的动力！** ⭐

---

## 下载

从发布页面获取 EasyMark 的最新版本：

<a href="https://github.com/aaaazbwzbw/EasyMark/releases/latest"><img src="https://img.shields.io/badge/下载最新版本-4FC08D?style=for-the-badge&logo=windows" alt="下载最新版本"></a>


</div>

---

## 项目简介

EasyMark 是一款现代化、高性能的计算机视觉标注工具，面向检测/分割/关键点等任务，提供从数据集导入、标注、版本管理、导出到训练、推理的一体化工作流。

它采用 **Electron + Vue 3 + Go** 架构：

- **前端（Vue 3）**：负责 UI 与交互
- **后端（Go）**：负责文件读写、数据集导入导出、任务编排（大数据集性能更稳定）
- **插件系统**：推理/训练/格式转换均可插件化扩展

<div align="center">

<img src="docs/assets/demo1.gif" alt="EasyMark AI Segmentation Demo" width="100%">
<br>
<img src="docs/assets/demo2.gif" alt="EasyMark Auto Detection Demo" width="100%">

</div>

## 功能特性

### 标注类型

| 类型 | 应用场景 | 操作方式 |
|------|----------|-----------|
| **矩形框** | 目标检测 | 鼠标拖拽 |
| **多边形** | 实例分割 / 语义分割 | Alt + 点击 |
| **关键点** | 姿态估计 | Alt + 点击（需绑定骨架） |

### AI 辅助标注

内置 AI 插件，大幅提升标注效率：

- **SAM2** - 交互式分割（提示点点击即分割）
- **YOLO** - 切换图片自动检测

SAM2 交互补充：

- **Shift + 左键**：正向提示点
- **Shift + 右键**：负向提示点（排除区域）
- **Ctrl + Shift + 左键/右键**：连续追加提示点，基于当前分割结果持续细化
- **基于SAM2分割结果的后处理**: 可通过简单的点选实现矩形框的快速标注

### 数据集管理

- **格式支持**：YOLO、VOC、COCO 导入导出
- **版本控制**：快照、回溯、多版本管理
- **灵活导出**：跨项目合并，自定义训练集/验证集/测试集划分

### 模型训练

- 内置 **Ultralytics YOLO** 训练流程
- 训练完成自动部署到推理插件
- 训练历史与指标可视化

### 插件系统

可通过插件扩展以下能力：

- 数据集导入导出格式
- 训练框架
- 推理后端

你可以在应用内“插件”页面管理插件，也可以按文档自行开发插件。

## 技术栈

| 模块 | 技术 |
|-----------|------------|
| 前端 | Vue 3 + TypeScript + Vite + TailwindCSS |
| 桌面端 | Electron |
| 后端 | Go 1.21+ |
| 插件 | Python 3.10+ |

## 快速开始

### 环境要求

- **Node.js** 18+
- **Go** 1.21+
- **Python** 3.10+（AI 插件需要）

### 安装

```bash
# 克隆仓库
git clone https://github.com/aaaazbwzbw/EasyMark.git
cd easymark

# 安装前端依赖
cd frontend && npm install

# 安装 Electron 依赖
cd ../host-electron && npm install

# 编译后端
cd ../backend-go && go build
```

### 开发调试

```bash
# 终端 1：启动后端
cd backend-go && ./backend-go

# 终端 2：启动前端开发服务器
cd frontend && npm run dev

# 终端 3：启动 Electron
cd host-electron && npm run dev
```

### 打包构建

```bash
# 生产环境打包
cd host-electron && npm run build
```

## 项目结构

```
easymark/
├── frontend/              # Vue 3 frontend application
│   └── src/docs/          # User documentation
├── host-electron/         # Electron main process
├── backend-go/            # Go backend service
├── host-plugins/          # Built-in plugins
│   ├── infer-plugins/     # Inference plugins (SAM2, YOLO)
│   ├── train_python/      # Training plugin
│   └── dataset-common/    # Dataset format converters
└── docs/                  # Development documentation
```

## 文档

- **使用指南**：应用内帮助页面
- **插件开发**：参见 `docs/plugin-development-guide.md`
- **API 参考**：参见 `docs/plugin-api-reference.md`

## 参与贡献

欢迎贡献代码！请参阅 [贡献指南](CONTRIBUTING_zh-CN.md) 了解详情。

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 发起 Pull Request

## 开发路线

- [x] 视频标注
- [ ] 插件市场
- [ ] 云端同步
- [ ] 团队协作
- [ ] 更多 AI 模型支持

## 开源协议

本项目采用 MIT 协议开源 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [Ultralytics](https://github.com/ultralytics/ultralytics) - YOLO
- [Segment Anything](https://github.com/facebookresearch/segment-anything-2) - SAM2
- [Electron](https://www.electronjs.org/)
- [Vue.js](https://vuejs.org/)

---

<div align="center">

**如果觉得 EasyMark 对你有帮助，欢迎给个 Star！**

</div>
