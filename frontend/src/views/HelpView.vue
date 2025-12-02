<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Mail, LayoutDashboard, Folder, Database, Brain, Bot } from 'lucide-vue-next'
import { marked } from 'marked'
import overviewZh from '../docs/help-zh-CN.md?raw'
import projectZh from '../docs/help-project-zh-CN.md?raw'
import datasetZh from '../docs/help-dataset-zh-CN.md?raw'
import trainingZh from '../docs/help-training-zh-CN.md?raw'
import inferenceZh from '../docs/help-inference-zh-CN.md?raw'
import overviewEn from '../docs/help-en-US.md?raw'
import projectEn from '../docs/help-project-en-US.md?raw'
import datasetEn from '../docs/help-dataset-en-US.md?raw'
import trainingEn from '../docs/help-training-en-US.md?raw'
import inferenceEn from '../docs/help-inference-en-US.md?raw'

const { t, locale } = useI18n()

type SectionId = 'overview' | 'project' | 'dataset' | 'training' | 'inference'

const sections = [
  { id: 'overview' as const, icon: LayoutDashboard, titleKey: 'help.nav.overview' },
  { id: 'project' as const, icon: Folder, titleKey: 'help.nav.project' },
  { id: 'dataset' as const, icon: Database, titleKey: 'help.nav.dataset' },
  { id: 'training' as const, icon: Brain, titleKey: 'help.nav.training' },
  { id: 'inference' as const, icon: Bot, titleKey: 'help.nav.inference' }
]

const activeSectionId = ref<SectionId>('overview')
const renderedHelp = ref('')

const getMarkdown = (id: SectionId) => {
  const isZh = locale.value === 'zh-CN'
  switch (id) {
    case 'overview':
      return isZh ? overviewZh : overviewEn
    case 'project':
      return isZh ? projectZh : projectEn
    case 'dataset':
      return isZh ? datasetZh : datasetEn
    case 'training':
      return isZh ? trainingZh : trainingEn
    case 'inference':
      return isZh ? inferenceZh : inferenceEn
  }
}

const loadHelp = () => {
  const md = getMarkdown(activeSectionId.value) || ''
  renderedHelp.value = marked.parse(md) as string
}

watch([locale, activeSectionId], () => {
  loadHelp()
}, { immediate: true })

// 版本号
const appVersion = ref('')

onMounted(async () => {
  appVersion.value = await window.electronAPI?.getAppVersion?.() || ''
})
</script>

<template>
  <div class="help-view">
    <div class="help-view__container">
      <!-- 顶部标题区 -->
      <header class="help-view__header">
        <h1 class="help-view__title">{{ t('help.title') }}</h1>
        <p class="help-view__desc">{{ t('help.description') }}</p>
      </header>

      <!-- 主体：左侧索引栏 + 右侧内容 -->
      <div class="help-view__body">
        <aside class="help-view__sidebar">
          <nav class="help-view__nav">
            <button
              v-for="section in sections"
              :key="section.id"
              type="button"
              class="help-view__nav-item"
              :class="{ 'help-view__nav-item--active': section.id === activeSectionId }"
              @click="activeSectionId = section.id"
            >
              <component :is="section.icon" :size="16" class="help-view__nav-icon" />
              <span class="help-view__nav-label">{{ t(section.titleKey) }}</span>
            </button>
          </nav>
          <!-- 联系卡片 -->
          <div class="help-card help-card--contact">
            <div class="help-card__contact">
              <Mail :size="24" class="help-card__contact-icon" />
              <h3 class="help-card__contact-title">{{ t('help.contactCard.title') }}</h3>
              <p class="help-card__contact-desc">{{ t('help.contactCard.desc') }}</p>
              <div class="help-card__contact-list">
                <div class="help-card__contact-item">
                  <span class="help-card__contact-label">Email:</span>
                  <a href="mailto:1526196180@qq.com" class="help-card__contact-value">1526196180@qq.com</a>
                </div>
                <div class="help-card__contact-item">
                  <span class="help-card__contact-label">QQ:</span>
                  <span class="help-card__contact-value">1526196180</span>
                </div>
              </div>
              <div v-if="appVersion" class="help-card__contact-version">
                v{{ appVersion }}
              </div>
            </div>
          </div>
        </aside>

        <main class="help-view__main">
          <!-- 详细使用说明：从 Markdown 渲染 -->
          <div class="help-card help-card--markdown">
            <div class="help-view__markdown" v-html="renderedHelp" />
          </div>
        </main>
      </div>
    </div>
  </div>
</template>

<style scoped>
.help-view {
  width: 100%;
  height: 100%;
  overflow: hidden;
  background-color: var(--color-bg-app);
}

.help-view__container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  padding: 2rem;
  overflow: hidden;
}

.help-view__header {
  text-align: center;
  margin-bottom: 2rem;
}

.help-view__title {
  margin: 0 0 0.5rem;
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--color-fg);
}

.help-view__desc {
  margin: 0;
  font-size: 0.95rem;
  color: var(--color-fg-muted);
}

.help-view__body {
  flex: 1;
  display: flex;
  gap: 1.25rem;
  min-height: 0;
}

.help-view__sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
}

.help-view__main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  min-height: 0;
}

.help-card {
  background-color: var(--color-bg-panel);
  border: 1px solid var(--color-border-subtle);
  border-radius: 8px;
  padding: 1.25rem;
}

.help-card--contact {
  background: linear-gradient(135deg, var(--color-accent-soft), var(--color-bg-panel));
  border-color: var(--color-accent);
}

.help-card--markdown {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.help-card--markdown::-webkit-scrollbar {
  display: none;
}

.help-card__contact {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.help-card__contact-icon {
  color: var(--color-accent);
  margin-bottom: 0.5rem;
}

.help-card__contact-title {
  margin: 0 0 0.25rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-fg);
}

.help-card__contact-desc {
  margin: 0 0 0.75rem;
  font-size: 0.8rem;
  color: var(--color-fg-muted);
}

.help-card__contact-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  width: 100%;
}

.help-card__contact-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 0.85rem;
}

.help-card__contact-label {
  color: var(--color-fg-muted);
  font-weight: 500;
}

.help-card__contact-value {
  color: var(--color-accent);
  text-decoration: none;
}

.help-card__contact-value:hover {
  text-decoration: underline;
}

.help-card__contact-version {
  margin-top: 0.75rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--color-border-subtle);
  font-size: 0.75rem;
  color: var(--color-fg-muted);
  text-align: center;
}

.help-view__nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.help-view__nav-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 6px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 0.85rem;
  color: var(--color-fg-muted);
  text-align: left;
}

.help-view__nav-item--active {
  background-color: var(--color-bg-panel);
  color: var(--color-fg);
}

.help-view__nav-icon {
  flex-shrink: 0;
}

.help-view__nav-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.help-view__markdown {
  font-size: 0.85rem;
  line-height: 1.6;
  color: var(--color-fg-muted);
}

.help-view__markdown h1,
.help-view__markdown h2,
.help-view__markdown h3 {
  color: var(--color-fg);
  margin: 1.2rem 0 0.5rem;
}

.help-view__markdown p {
  margin: 0.4rem 0;
}

.help-view__markdown ul {
  padding-left: 1.2rem;
  margin: 0.4rem 0;
}

.help-card__title {
  margin: 0 0 1rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-fg);
}

.help-card__list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.help-card__item {
  position: relative;
  padding-left: 1rem;
  margin-bottom: 0.6rem;
  font-size: 0.85rem;
  color: var(--color-fg-muted);
  line-height: 1.5;
}

.help-card__item::before {
  content: '•';
  position: absolute;
  left: 0;
  color: var(--color-accent);
}
.help-card__item:last-child {
  margin-bottom: 0;
}
</style>
