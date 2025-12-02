/// <reference types="vite/client" />

// Markdown 文件作为原始字符串导入
declare module '*.md?raw' {
  const content: string
  export default content
}

// Markdown 文件作为 URL 导入（如果需要）
declare module '*.md?url' {
  const url: string
  export default url
}

// 图片资源
declare module '*.png' {
  const src: string
  export default src
}

declare module '*.jpg' {
  const src: string
  export default src
}

declare module '*.svg' {
  const src: string
  export default src
}
