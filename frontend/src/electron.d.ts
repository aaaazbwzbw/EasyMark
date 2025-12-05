// Electron API 类型定义

// 推理结果标注类型
interface InferenceAnnotation {
  type: 'bbox' | 'polygon'
  categoryName: string
  confidence: number
  data: {
    x?: number
    y?: number
    width?: number
    height?: number
    points?: [number, number][]
    keypoints?: [number, number, number][]
  }
}

declare global {
  interface Window {
    electronAPI?: {
      // 主窗口控制
      minimize: () => void
      toggleMaximize: () => void
      close: () => void
      
      // 文件/目录选择
      selectDirectory: (defaultPath?: string) => Promise<string | null>
      selectFiles: (options?: { defaultPath?: string; filters?: { name: string; extensions: string[] }[] }) => Promise<string[]>
      selectFile: (options?: { title?: string; filters?: { name: string; extensions: string[] }[] }) => Promise<string | null>
      
      // 插件安装
      installPlugin: (filePath: string) => Promise<{ success: boolean; error?: string; message?: string; plugin?: { id: string; name: string; version: string } }>
      
      // 打开路径
      openPath: (targetPath: string) => Promise<{ success: boolean; error?: string }>
      
      // 使用系统浏览器打开外部链接
      openExternal: (url: string) => Promise<{ success: boolean; error?: string }>
      
      // 模型推理窗口
      openInferenceWindow: (options?: { theme?: string; locale?: string; pluginId?: string }) => void
      minimizeInference: () => void
      closeInference: () => void
      
      // 推理相关
      getCurrentImagePath: () => Promise<string | null>
      sendInferenceResults: (results: InferenceAnnotation[]) => void
      onImageChanged: (callback: (imagePath: string) => void) => void
      onInferenceResults: (callback: (results: InferenceAnnotation[]) => void) => void
      notifyImageChanged: (imagePath: string) => void
      
      // 小窗日志转发
      forwardLog: (type: string, args: string[]) => void
      onForwardedLog: (callback: (data: { type: string; args: string[] }) => void) => void
      
      // 模型下载进度转发（主窗口 -> 推理小窗）
      forwardDownloadProgress: (data: { type: string; filename: string; progress?: number; error?: string }) => void
      onModelDownloadProgress: (callback: (data: { type: string; filename: string; progress?: number; error?: string }) => void) => void
      
      // 广播主题/语言变化到其他窗口
      broadcastThemeChange: (theme: string) => void
      broadcastLocaleChange: (locale: string) => void
      
      // 监听来自主进程的消息
      onThemeChanged: (callback: (theme: string) => void) => void
      onLocaleChanged: (callback: (locale: string) => void) => void
      onInferenceWindowClosed: (callback: () => void) => void
      
      // 推理服务（行协议，新接口）
      inferenceStartService: (pluginId?: string) => Promise<{ success: boolean }>
      inferenceStopService: () => Promise<{ success: boolean }>
      inferenceLoadModel: (modelPath: string) => Promise<{ success: boolean; error?: string; task?: string }>
      inferenceRun: (payload: { projectId: string; path: string; conf: number; iou: number; points?: Array<{x: number; y: number; type: 'positive' | 'negative'}>; multimask?: boolean; prompt?: string }) => Promise<{
        success: boolean
        error?: string
        annotations: Array<{
          type: string
          categoryName: string
          confidence: number
          data: {
            x: number
            y: number
            width: number
            height: number
            keypoints?: [number, number, number][]
          }
          polygon?: [number, number][]
        }>
        inferTimeMs?: number
      }>
      inferenceSetImage: (imagePath: string) => Promise<{ success: boolean; error?: string }>
      inferenceServiceStatus: () => Promise<{ running: boolean }>
      
      // 小窗通知主窗口参数变化
      notifyInferenceParamsChanged: (params: { conf?: number; iou?: number; continuousMode?: boolean }) => void
      onInferenceParamsChanged: (callback: (params: { conf?: number; iou?: number; continuousMode?: boolean }) => void) => void
      
      // 小窗通知主窗口模型变化
      notifyInferenceModelChanged: () => void
      onInferenceModelChanged: (callback: () => void) => void
      
      // 类别相关接口（供插件使用）
      getProjectCategories: () => Promise<{ success: boolean; categories: Array<{ id: number; name: string; type: string; color: string }> }>
      getSelectedCategory: () => Promise<{ success: boolean; category: { id: number; name: string; type: string; color: string } | null }>
      selectCategory: (categoryId: number) => Promise<{ success: boolean }>
      onCategoryChanged: (callback: (category: { id: number; name: string; type: string; color: string } | null) => void) => void
      
      // 显示通知
      showNotification: (type: 'success' | 'warning' | 'error', message: string) => void
      onShowNotification: (callback: (data: { type: string; message: string }) => void) => void
      
      // 日志发送到主进程
      sendLog: (level: string, ...args: string[]) => void
      
      // 获取应用版本
      getAppVersion: () => Promise<string>
    }
  }
}

export {}
