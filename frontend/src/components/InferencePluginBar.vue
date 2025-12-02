<script setup lang="ts">
/**
 * 推理插件栏组件
 * 显示在底部栏，点击图标可激活推理插件并打开小窗
 */
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Brain, Box, Scan, Wand2, Sparkles, Cpu, Zap, Loader2 } from 'lucide-vue-next'
import { useInferencePlugins } from '../composables/useInferencePlugins'
import { useNotification } from '../composables/useNotification'
import type { InferencePlugin } from '../types/inference-plugin'

const { locale, t } = useI18n()
const { warning: notifyWarning } = useNotification()
const {
  inferencePlugins,
  activePluginId,
  isLoading,
  loadPlugins,
  activatePlugin,
  getPluginName
} = useInferencePlugins()

// 检查插件 Python 依赖是否就绪
const checkPluginDepsReady = async (pluginId: string): Promise<boolean> => {
  try {
    const res = await fetch(`http://localhost:18080/api/python/env-status?pluginId=${encodeURIComponent(pluginId)}`)
    if (res.ok) {
      const data = await res.json()
      // 检查是否有虚拟环境且所有依赖都已安装
      if (!data.hasVenv) return false
      const deps = data.dependencies || []
      const missingDeps = deps.filter((d: any) => d.required && !d.installed)
      return missingDeps.length === 0
    }
  } catch (e) {
    console.error('Check plugin deps error:', e)
  }
  return false
}

// 小窗状态
const isWindowOpen = ref(false)

// 内置图标映射
const iconComponents: Record<string, any> = {
  brain: Brain,
  box: Box,
  scan: Scan,
  wand: Wand2,
  sparkles: Sparkles,
  cpu: Cpu,
  zap: Zap
}

// 获取插件图标组件
const getIconComponent = (plugin: InferencePlugin) => {
  const iconName = plugin.defaultIcon || 'brain'
  return iconComponents[iconName] || Brain
}

// 获取插件 logo URL
const getPluginLogoUrl = (plugin: InferencePlugin) => {
  if (plugin.icon) {
    return `http://localhost:18080/api/plugins/${plugin.id}/logo`
  }
  return null
}

// 点击插件图标
const handlePluginClick = async (plugin: InferencePlugin) => {
  if (activePluginId.value === plugin.id) {
    // 已激活，切换小窗显示
    if (isWindowOpen.value) {
      // 关闭小窗
      window.electronAPI?.closeInference?.()
      isWindowOpen.value = false
    } else {
      // 重新打开小窗
      isWindowOpen.value = true
      window.electronAPI?.openInferenceWindow?.({
        theme: document.documentElement.getAttribute('data-theme') || 'dark',
        locale: locale.value,
        pluginId: plugin.id
      })
    }
    return
  } else {
    // 检查依赖是否就绪
    const depsReady = await checkPluginDepsReady(plugin.id)
    if (!depsReady) {
      notifyWarning(t('inference.pluginDepsNotReady', { name: getPluginName(plugin, locale.value) }))
      return
    }
    
    // 激活新插件
    const success = await activatePlugin(plugin.id)
    if (success) {
      isWindowOpen.value = true
      // 打开推理小窗，传递插件 ID
      window.electronAPI?.openInferenceWindow?.({
        theme: document.documentElement.getAttribute('data-theme') || 'dark',
        locale: locale.value,
        pluginId: plugin.id
      })
    }
  }
}

// 关闭小窗
const handleWindowClose = () => {
  isWindowOpen.value = false
}

// 是否有推理插件
const hasPlugins = computed(() => inferencePlugins.value.length > 0)

// 加载插件列表
onMounted(async () => {
  await loadPlugins()
  
  // 监听小窗关闭事件，同步更新状态
  window.electronAPI?.onInferenceWindowClosed?.(() => {
    isWindowOpen.value = false
  })
})

// 暴露给父组件
defineExpose({
  isWindowOpen,
  handleWindowClose,
  hasPlugins
})
</script>

<template>
  <!-- 没有插件时不显示任何内容 -->
  <div v-if="isLoading || hasPlugins" class="inference-plugin-bar">
    <!-- 加载中 -->
    <div v-if="isLoading" class="inference-plugin-bar__loading">
      <Loader2 :size="14" class="spin" />
    </div>
    
    <!-- 插件图标列表 -->
    <template v-else-if="hasPlugins">
      <button
        v-for="plugin in inferencePlugins"
        :key="plugin.id"
        type="button"
        class="inference-plugin-bar__btn"
        :class="{ 
          'inference-plugin-bar__btn--active': activePluginId === plugin.id,
          'inference-plugin-bar__btn--open': activePluginId === plugin.id && isWindowOpen
        }"
        :title="getPluginName(plugin, locale)"
        @click="handlePluginClick(plugin)"
      >
        <img 
          v-if="getPluginLogoUrl(plugin)" 
          :src="getPluginLogoUrl(plugin)!" 
          class="inference-plugin-bar__logo"
          :alt="getPluginName(plugin, locale)"
        />
        <component v-else :is="getIconComponent(plugin)" :size="14" />
      </button>
    </template>
  </div>
</template>

<style scoped>
.inference-plugin-bar {
  display: flex;
  align-items: center;
  gap: 2px;
}

.inference-plugin-bar__loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8px;
}

.inference-plugin-bar__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-fg-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.inference-plugin-bar__btn:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-fg-primary);
}

.inference-plugin-bar__btn--active {
  color: var(--color-accent);
}

.inference-plugin-bar__btn--open {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
}

.inference-plugin-bar__btn--open:hover {
  background: rgba(59, 130, 246, 0.3);
}

.inference-plugin-bar__logo {
  width: 14px;
  height: 14px;
  object-fit: contain;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.spin {
  animation: spin 1s linear infinite;
}
</style>
