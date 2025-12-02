<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import logo from '../assets/logo.svg'

const { t } = useI18n()
</script>

<template>
  <div class="home-page">
    <img :src="logo" alt="EasyMark logo" class="home-page__logo" />
    <p class="home-page__title">{{ t('home.welcome') }}</p>
  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rem;
  width: 100%;
  height: 100%;
}

.home-page__logo {
  height: 200px;
  /* 深色主题下使用纯白 logo（SVG 本身即为白色） */
  opacity: 1;
}

.home-page__title {
  margin: 0;
  font-size: 2.6rem;
  font-weight: 600;
  color: #ffffff;
  opacity: 1;
}

[data-theme='light'] .home-page__title {
  /* 浅色主题下仍保持偏浅的灰度效果 */
  filter: brightness(0) saturate(0);
  opacity: 0.35;
}

[data-theme='dark'] .home-page__logo {
  /* 深色主题下强制灰色：先反转成白色，再降低亮度 */
  filter: brightness(0) invert(1) brightness(0.3);
}

[data-theme='dark'] .home-page__title {
  color: #4b5563;
}

[data-theme='light'] .home-page__logo {
  filter: brightness(0) saturate(0);
  opacity: 0.35;
}
</style>

