import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  base: './', // 使用相对路径，支持 Electron 打包后加载
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  // 确保 md 文件被识别为静态资源（用于 ?raw 导入）
  assetsInclude: ['**/*.md'],
  build: {
    // 优化打包配置
    rollupOptions: {
      output: {
        // 分割代码块，优化加载
        manualChunks: {
          'vendor': ['vue', 'vue-router', 'vue-i18n'],
          'ui': ['lucide-vue-next'],
          'utils': ['marked']
        }
      }
    },
    // 提高构建性能
    target: 'esnext',
    // 内联小于 4kb 的资源
    assetsInlineLimit: 4096
  }
})
