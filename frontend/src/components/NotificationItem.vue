<script setup lang="ts">
import { CheckCircle, Info, AlertTriangle, XCircle } from 'lucide-vue-next'
import type { Notification } from '../composables/useNotification'

defineProps<{
  notification: Notification
}>()

const getIconConfig = (type: string) => {
  switch (type) {
    case 'success':
      return { icon: CheckCircle, color: '#10b981' }
    case 'info':
      return { icon: Info, color: '#3b82f6' }
    case 'warning':
      return { icon: AlertTriangle, color: '#f59e0b' }
    case 'error':
      return { icon: XCircle, color: '#ef4444' }
    default:
      return { icon: Info, color: '#3b82f6' }
  }
}
</script>

<template>
  <div class="notification-item">
    <div class="notification-item__icon">
      <component
        :is="getIconConfig(notification.type).icon"
        :size="20"
        :style="{ color: getIconConfig(notification.type).color }"
      />
    </div>
    <div class="notification-item__message">
      {{ notification.message }}
    </div>
    <button
      v-if="notification.button"
      type="button"
      class="notification-item__button"
      @click="notification.button.onClick"
    >
      {{ notification.button.text }}
    </button>
  </div>
</template>

<style scoped>
.notification-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background-color: var(--color-notification-header-bg);
  border-bottom: 1px solid var(--color-notification-border);
  min-height: 48px;
  overflow: hidden;
}

.notification-item:last-child {
  border-bottom: none;
}

.notification-item__icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.notification-item__message {
  flex: 1;
  font-size: 0.8rem;
  color: var(--color-fg);
  line-height: 1.4;
  word-break: break-word;
  overflow-wrap: break-word;
  min-width: 0;
}

.notification-item__button {
  flex-shrink: 0;
  padding: 4px 12px;
  font-size: 0.75rem;
  border: 1px solid var(--color-border-subtle);
  background-color: var(--color-accent-soft);
  color: var(--color-fg);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.notification-item__button:hover {
  background-color: var(--color-accent);
}
</style>

