<template>
  <div class="dataset-view">
    <!-- 左侧：项目/版本/类别树 -->
    <aside class="dataset-tree">
      <header class="dataset-tree__header">
        <h2>{{ t('dataset.tree.projects') }}</h2>
        <button class="tree-header__btn" @click="openExportDir" :title="t('dataset.openExportDir')">
          <FolderOpen :size="16" />
        </button>
      </header>

      <div v-if="loading" class="dataset-tree__loading">
        <Loader2 :size="20" class="animate-spin" />
        {{ t('dataset.loading') }}
      </div>

      <div v-else-if="projects.length === 0" class="dataset-tree__empty">
        {{ t('dataset.tree.noProjects') }}
      </div>

      <div v-else class="dataset-tree__content">
        <div v-for="project in projects" :key="project.id" class="tree-node">
          <!-- 项目节点 -->
          <div 
            class="tree-node__row tree-node__row--project"
            :class="{ 'tree-node__row--selected': selectedNode?.type === 'project' && selectedNode?.id === project.id }"
            @click="selectProject(project)"
          >
            <button class="tree-node__toggle" @click.stop="toggleProject(project.id)">
              <component :is="expandedProjects.has(project.id) ? ChevronDown : ChevronRight" :size="18" />
            </button>
            <Folder :size="18" class="tree-node__icon" />
            <span class="tree-node__label">{{ project.name }}</span>
            <button class="tree-node__sync" @click.stop="openSyncModal(project)" :title="t('dataset.sync.button')">
              <RefreshCw :size="16" />
            </button>
          </div>

          <!-- 版本列表 -->
          <div v-if="expandedProjects.has(project.id)" class="tree-node__children">
            <div v-if="!projectVersions[project.id]?.length" class="tree-node__empty-hint">
              {{ t('dataset.project.noVersions') }}
            </div>
            <div v-for="version in projectVersions[project.id] || []" :key="`${project.id}-v${version.version}`" class="tree-node">
              <!-- 版本节点 -->
              <div 
                class="tree-node__row tree-node__row--version"
                :class="{ 'tree-node__row--selected': selectedNode?.type === 'version' && selectedNode?.projectId === project.id && selectedNode?.version === version.version }"
                @click="selectVersion(project, version)"
                @contextmenu.prevent="openVersionContextMenu($event, project, version)"
              >
                <button class="tree-node__toggle" @click.stop="toggleVersion(project.id, version.version)">
                  <component :is="expandedVersions.has(`${project.id}-v${version.version}`) ? ChevronDown : ChevronRight" :size="18" />
                </button>
                <Tag :size="16" class="tree-node__icon" />
                <span class="tree-node__label">v{{ version.version }}</span>
                <span class="tree-node__meta">{{ version.imageCount }} img</span>
              </div>

              <!-- 类别列表 -->
              <div v-if="expandedVersions.has(`${project.id}-v${version.version}`)" class="tree-node__children">
                <div 
                  v-for="cat in versionCategories[`${project.id}-v${version.version}`] || []" 
                  :key="`${project.id}-v${version.version}-${cat.id}`"
                  class="tree-node__row tree-node__row--category"
                  :class="{ 'tree-node__row--checked': isCategorySelected(project.id, version.version, cat.id) }"
                >
                  <input 
                    type="checkbox" 
                    class="tree-node__checkbox"
                    :checked="isCategorySelected(project.id, version.version, cat.id)"
                    @change="toggleCategorySelection(project.id, version.version, cat)"
                  />
                  <component :is="getCategoryIcon(cat.type)" :size="14" class="tree-node__icon" :style="{ color: cat.color }" />
                  <span class="tree-node__label">{{ cat.name }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- 右侧：详情区 -->
    <main class="dataset-detail">
      <!-- 导出工具栏 -->
      <div v-if="selectedCategories.length > 0" class="dataset-export-bar">
        <span>{{ t('dataset.export.selected', { count: selectedCategories.length }) }}</span>
        <button class="export-btn" @click="openExportWizard">
          <Download :size="14" />
          {{ t('dataset.export.button') }}
        </button>
      </div>

      <!-- 空状态 -->
      <div v-if="!selectedNode" class="detail-empty">
        <Database :size="48" />
        <p>{{ t('dataset.empty.desc') }}</p>
      </div>

      <!-- 项目概览 -->
      <div v-else-if="selectedNode.type === 'project'" class="detail-panel">
        <header class="detail-header">
          <div class="detail-header__title">
            <Folder :size="20" />
            <h2>{{ selectedProject?.name }}</h2>
          </div>
        </header>
        
        <div class="detail-body">
          <div class="stat-cards">
            <div class="stat-card">
              <Tags :size="24" class="stat-card__icon" />
              <div class="stat-card__content">
                <span class="stat-card__value">{{ projectVersions[selectedProject?.id || '']?.length || 0 }}</span>
                <span class="stat-card__label">{{ t('dataset.tree.versions') }}</span>
              </div>
            </div>
            <div class="stat-card">
              <Clock :size="24" class="stat-card__icon" />
              <div class="stat-card__content">
                <span class="stat-card__value stat-card__value--small">{{ formatDate(projectVersions[selectedProject?.id || '']?.[0]?.createdAt) || '-' }}</span>
                <span class="stat-card__label">{{ t('dataset.project.lastSync') }}</span>
              </div>
            </div>
          </div>
          
          <div v-if="!projectVersions[selectedProject?.id || '']?.length" class="detail-tip">
            <Lightbulb :size="18" />
            <span>{{ t('dataset.project.noVersionsHint') }}</span>
          </div>
        </div>
      </div>

      <!-- 版本详情仪表盘 -->
      <div v-else-if="selectedNode.type === 'version'" class="detail-panel">
        <header class="detail-header">
          <div class="detail-header__title">
            <Tag :size="20" />
            <h2>v{{ selectedVersion?.version }}</h2>
            <span class="detail-header__badge">{{ selectedProject?.name }}</span>
          </div>
          <div class="detail-header__meta">
            <span class="detail-header__time">
              <Clock :size="14" />
              {{ formatDate(selectedVersion?.createdAt) }}
            </span>
          </div>
        </header>
        
        <!-- 备注栏 -->
        <div class="detail-note-bar" v-if="!isEditingNote">
          <FileText :size="16" />
          <span class="detail-note-bar__text">{{ selectedVersion?.note || t('dataset.version.noNote') }}</span>
          <button class="detail-note-bar__edit" @click="startEditNote">
            <Pencil :size="14" />
          </button>
        </div>
        <div class="detail-note-bar detail-note-bar--editing" v-else>
          <input type="text" v-model="editNoteValue" @keyup.enter="saveNote" :placeholder="t('dataset.sync.notePlaceholder')" />
          <button class="detail-note-bar__btn" @click="saveNote"><Check :size="16" /></button>
          <button class="detail-note-bar__btn" @click="isEditingNote = false"><X :size="16" /></button>
        </div>

        <div class="detail-body">
          <!-- 统计卡片 -->
          <div class="stat-cards stat-cards--triple">
            <div class="stat-card stat-card--accent">
              <Images :size="28" class="stat-card__icon" />
              <div class="stat-card__content">
                <span class="stat-card__value">{{ selectedVersion?.imageCount || 0 }}</span>
                <span class="stat-card__label">{{ t('dataset.version.images') }}</span>
              </div>
            </div>
            <div class="stat-card stat-card--success">
              <Shapes :size="28" class="stat-card__icon" />
              <div class="stat-card__content">
                <span class="stat-card__value">{{ selectedVersion?.categoryCount || 0 }}</span>
                <span class="stat-card__label">{{ t('dataset.version.categories') }}</span>
              </div>
            </div>
            <div class="stat-card stat-card--warning">
              <Square :size="28" class="stat-card__icon" />
              <div class="stat-card__content">
                <span class="stat-card__value">{{ selectedVersion?.annotationCount || 0 }}</span>
                <span class="stat-card__label">{{ t('dataset.version.annotations') }}</span>
              </div>
            </div>
          </div>

          <!-- 数据健康度 -->
          <section class="detail-section">
            <h3>
              <HeartPulse :size="16" />
              {{ t('dataset.health.title') }}
            </h3>
            
            <div class="health-panel">
              <!-- 标注进度 -->
              <div class="health-item">
                <div class="health-item__header">
                  <span class="health-item__label">{{ t('dataset.health.progress') }}</span>
                  <span class="health-item__value">100%</span>
                </div>
                <div class="health-progress">
                  <div class="health-progress__bar" style="width: 100%"></div>
                </div>
                <p class="health-item__desc">{{ t('dataset.health.progressDesc', { count: selectedVersion?.imageCount || 0 }) }}</p>
              </div>

              <!-- 密度指标 -->
              <div class="health-grid">
                <div class="health-metric">
                  <span class="health-metric__value">
                    {{ selectedVersion?.imageCount ? (selectedVersion.annotationCount / selectedVersion.imageCount).toFixed(1) : '0' }}
                  </span>
                  <span class="health-metric__label">{{ t('dataset.health.avgAnnotations') }}</span>
                </div>
                <div class="health-check health-check--pass">
                  <CheckCircle :size="16" />
                  <span>{{ t('dataset.health.checkEmpty') }}</span>
                </div>
                <div class="health-check health-check--warning">
                  <AlertTriangle :size="16" />
                  <span>{{ t('dataset.health.checkBalance') }}</span>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </main>

    <!-- 版本右键菜单 -->
    <div 
      v-if="isVersionContextMenuOpen" 
      class="context-menu"
      :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }"
    >
      <button class="context-menu__item" @click="openRollbackModal">
        <History :size="16" />
        {{ t('dataset.version.rollback') }}
      </button>
      <button class="context-menu__item context-menu__item--danger" @click="openDeleteModal">
        <Trash2 :size="16" />
        {{ t('dataset.version.delete') }}
      </button>
    </div>

    <!-- 同步对话框 -->
    <div v-if="isSyncModalOpen" class="modal-overlay" @click.self="closeSyncModal">
      <div class="modal">
        <h2>{{ t('dataset.sync.title') }}</h2>
        <p class="modal__desc">{{ t('dataset.sync.desc') }}</p>
        <p class="modal__project">{{ syncTargetProject?.name }}</p>
        <div class="modal__field">
          <label>{{ t('dataset.sync.noteLabel') }}</label>
          <input type="text" v-model="syncNote" :placeholder="t('dataset.sync.notePlaceholder')" />
        </div>
        <div class="modal__actions">
          <button class="btn" @click="closeSyncModal">{{ t('dataset.sync.cancel') }}</button>
          <button class="btn btn--primary" @click="createVersion" :disabled="isSyncing">
            <Loader2 v-if="isSyncing" :size="16" class="animate-spin" />
            {{ isSyncing ? t('dataset.sync.creating') : t('dataset.sync.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 回溯确认对话框 -->
    <div v-if="isRollbackModalOpen" class="modal-overlay" @click.self="closeRollbackModal">
      <div class="modal modal--warning">
        <h2>{{ t('dataset.rollback.title', { version: selectedVersion?.version }) }}</h2>
        <p class="modal__warning">⚠️ {{ t('dataset.rollback.warning') }}</p>
        <label class="modal__confirm-check">
          <input type="checkbox" v-model="rollbackConfirmed" />
          {{ t('dataset.rollback.confirmLabel') }}
        </label>
        <div class="modal__actions">
          <button class="btn" @click="closeRollbackModal">{{ t('dataset.rollback.cancel') }}</button>
          <button class="btn btn--warning" @click="executeRollback" :disabled="!rollbackConfirmed || isRollingBack">
            <Loader2 v-if="isRollingBack" :size="16" class="animate-spin" />
            {{ t('dataset.rollback.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="isDeleteModalOpen" class="modal-overlay" @click.self="closeDeleteModal">
      <div class="modal modal--danger">
        <h2>{{ t('dataset.deleteVersion.title') }}</h2>
        <p>{{ t('dataset.deleteVersion.confirm', { version: selectedVersion?.version }) }}</p>
        <div class="modal__actions">
          <button class="btn" @click="closeDeleteModal">{{ t('dataset.rollback.cancel') }}</button>
          <button class="btn btn--danger" @click="deleteVersion" :disabled="isDeleting">
            <Loader2 v-if="isDeleting" :size="16" class="animate-spin" />
            {{ t('dataset.version.delete') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 导出面板 -->
    <div v-if="isExportModalOpen" class="modal-overlay" @click.self="closeExportModal">
      <div class="modal modal--export">
        <h2>
          <Download :size="20" />
          {{ t('dataset.exportWizard.title') }}
        </h2>

        <!-- 已选类别摘要 -->
        <section class="export-summary">
          <div class="export-summary__item">
            <Shapes :size="18" />
            <span>{{ t('dataset.exportWizard.categories', { count: selectedCategories.length }) }}</span>
          </div>
          <div class="export-summary__item">
            <Images :size="18" />
            <span>{{ t('dataset.exportWizard.images', { count: exportStats.imageCount }) }}</span>
          </div>
          <div class="export-summary__item">
            <Square :size="18" />
            <span>{{ t('dataset.exportWizard.annotations', { count: exportStats.annotationCount }) }}</span>
          </div>
        </section>

        <!-- 导出路径 -->
        <div class="modal__field">
          <label>{{ t('dataset.exportWizard.outputPath') }}</label>
          <div class="export-path-input">
            <input type="text" v-model="exportPath" :placeholder="t('dataset.exportWizard.pathPlaceholder')" />
            <button class="btn btn--small" @click="browseExportPath">
              <FolderOpen :size="16" />
            </button>
          </div>
        </div>

        <!-- 格式选择 -->
        <div class="export-row">
          <div class="modal__field export-field">
            <label>{{ t('dataset.exportWizard.format') }}</label>
            <select v-model="exportFormat" @change="onFormatChange">
              <option value="yolo">YOLO</option>
              <option value="coco">COCO</option>
              <option value="voc">Pascal VOC</option>
              <option value="custom">{{ t('dataset.exportWizard.customFormat') }}</option>
            </select>
          </div>
          <div class="modal__field export-field">
            <label>{{ t('dataset.exportWizard.plugin') }}</label>
            <select v-model="exportPlugin" :disabled="availableExportPlugins.length === 0">
              <option v-for="p in availableExportPlugins" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
            <p v-if="availableExportPlugins.length === 0" class="export-no-plugin">
              {{ t('dataset.exportWizard.noPlugin') }}
            </p>
          </div>
        </div>

        <!-- 数据集划分 -->
        <div class="modal__field">
          <label>{{ t('dataset.exportWizard.split') }}</label>
          <div class="export-split">
            <div class="export-split__item">
              <span class="export-split__label">{{ t('dataset.exportWizard.train') }}</span>
              <div class="export-split__row">
                <input type="number" v-model.number="exportSplit.train" min="0" max="100" @input="onSplitChange('train')" />
                <span class="export-split__unit">%</span>
              </div>
              <span class="export-split__count">≈ {{ Math.round(exportStats.imageCount * exportSplit.train / 100) }}</span>
            </div>
            <div class="export-split__item">
              <span class="export-split__label">{{ t('dataset.exportWizard.val') }}</span>
              <div class="export-split__row">
                <input type="number" v-model.number="exportSplit.val" min="0" max="100" @input="onSplitChange('val')" />
                <span class="export-split__unit">%</span>
              </div>
              <span class="export-split__count">≈ {{ Math.round(exportStats.imageCount * exportSplit.val / 100) }}</span>
            </div>
            <div class="export-split__item">
              <span class="export-split__label">{{ t('dataset.exportWizard.test') }}</span>
              <div class="export-split__row">
                <input type="number" v-model.number="exportSplit.test" min="0" max="100" @input="onSplitChange('test')" />
                <span class="export-split__unit">%</span>
              </div>
              <span class="export-split__count">≈ {{ Math.round(exportStats.imageCount * exportSplit.test / 100) }}</span>
            </div>
          </div>
          <p v-if="splitTotal !== 100" class="export-split__warning">
            <AlertTriangle :size="14" />
            {{ t('dataset.exportWizard.splitWarning', { total: splitTotal }) }}
          </p>
        </div>

        <div class="modal__actions">
          <button class="btn" @click="closeExportModal">{{ t('dataset.sync.cancel') }}</button>
          <button class="btn btn--primary" @click="executeExport" :disabled="isExporting || !exportPlugin || splitTotal !== 100">
            <Loader2 v-if="isExporting" :size="16" class="animate-spin" />
            {{ isExporting ? t('dataset.exportWizard.exporting') : t('dataset.exportWizard.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { 
  FolderOpen, Loader2, ChevronDown, ChevronRight, Folder, Tag, RefreshCw,
  Download, Database, Tags, Clock, Lightbulb, Pencil, Check, X, Images,
  Shapes, Square, HeartPulse, AlertTriangle, CheckCircle, FileText, History, Trash2
} from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'

const { t } = useI18n()
const notification = useNotification()
const { success: notifySuccess, error: notifyError } = notification

// Types
interface Project { id: string; name: string }
interface DatasetVersion { version: number; createdAt: string; note: string; imageCount: number; labeledImageCount: number; categoryCount: number; annotationCount: number }
interface Category { id: number; name: string; type: string; color: string }
interface SelectedCategory { projectId: string; version: number; categoryId: number; categoryName: string }
type SelectedNode = { type: 'project'; id: string } | { type: 'version'; projectId: string; version: number } | null

// State
const loading = ref(true)
const projects = ref<Project[]>([])
const projectVersions = ref<Record<string, DatasetVersion[]>>({})
const versionCategories = ref<Record<string, Category[]>>({})
const expandedProjects = ref<Set<string>>(new Set())
const expandedVersions = ref<Set<string>>(new Set())
const selectedNode = ref<SelectedNode>(null)
const selectedCategories = ref<SelectedCategory[]>([])

// Modals
const isSyncModalOpen = ref(false)
const syncTargetProject = ref<Project | null>(null)
const syncNote = ref('')
const isSyncing = ref(false)
const isRollbackModalOpen = ref(false)
const rollbackConfirmed = ref(false)
const isRollingBack = ref(false)
const isDeleteModalOpen = ref(false)
const isDeleting = ref(false)
const isEditingNote = ref(false)
const editNoteValue = ref('')

// 右键菜单
const isVersionContextMenuOpen = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuVersion = ref<{ projectId: string; version: number } | null>(null)

// 导出面板
interface ExportPlugin { id: string; name: string; formats: string[] }
const isExportModalOpen = ref(false)
const isExporting = ref(false)
const exportPath = ref('')
const exportFormat = ref('yolo')
const exportPlugin = ref('')
const exportSplit = ref({ train: 70, val: 20, test: 10 })
const allPlugins = ref<ExportPlugin[]>([])

// Computed
const selectedProject = computed(() => {
  if (!selectedNode.value) return null
  const id = selectedNode.value.type === 'project' ? selectedNode.value.id : selectedNode.value.projectId
  return projects.value.find(p => p.id === id) || null
})

const selectedVersion = computed(() => {
  const node = selectedNode.value
  if (!node || node.type !== 'version') return null
  return projectVersions.value[node.projectId]?.find(v => v.version === node.version) || null
})

// 导出相关计算属性
const exportStats = computed(() => {
  // 从选中的类别中计算已标注图片数和标注数
  let labeledImageCount = 0
  let annotationCount = 0
  const versionMap = new Map<string, number>()
  selectedCategories.value.forEach(c => {
    const key = `${c.projectId}-v${c.version}`
    if (!versionMap.has(key)) {
      const v = projectVersions.value[c.projectId]?.find(ver => ver.version === c.version)
      if (v) {
        // 使用已标注的图片数，如果字段不存在则回退到总图片数
        labeledImageCount += v.labeledImageCount ?? v.imageCount
        annotationCount += v.annotationCount
      }
      versionMap.set(key, 1)
    }
  })
  return { imageCount: labeledImageCount, annotationCount }
})

const availableExportPlugins = computed(() => {
  const format = exportFormat.value === 'custom' ? 'custom' : exportFormat.value
  return allPlugins.value.filter(p => p.formats.includes(format) || (format === 'custom' && p.formats.some(f => !['yolo', 'coco', 'voc'].includes(f))))
})

const splitTotal = computed(() => exportSplit.value.train + exportSplit.value.val + exportSplit.value.test)

// Helpers
function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// 打开导出目录
async function openExportDir() {
  try {
    const res = await fetch('http://localhost:18080/api/settings/paths')
    if (res.ok) {
      const data = await res.json()
      const exportDir = `${data.dataPath}\\dataset_out`
      await fetch('http://localhost:18080/api/shell/open-folder', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: exportDir })
      })
    }
  } catch (e) { console.error(e) }
}

// Methods
async function loadProjects() {
  try {
    const res = await fetch('http://localhost:18080/api/projects')
    if (res.ok) {
      const data = await res.json()
      // API 直接返回数组
      projects.value = Array.isArray(data) ? data : (data.projects || [])
    }
  } catch (e) { console.error(e) }
}

async function loadVersions(projectId: string) {
  try {
    const res = await fetch(`http://localhost:18080/api/dataset-versions?projectId=${projectId}`)
    if (res.ok) { projectVersions.value[projectId] = (await res.json()).versions || [] }
  } catch (e) { console.error(e) }
}

async function loadCategories(projectId: string, version: number) {
  const key = `${projectId}-v${version}`
  try {
    const res = await fetch(`http://localhost:18080/api/project-categories?projectId=${projectId}&version=${version}`)
    if (res.ok) {
      const data = await res.json()
      // API 返回 { items: [...] } 格式
      versionCategories.value[key] = Array.isArray(data) ? data : (data.items || data.categories || [])
    }
  } catch (e) { versionCategories.value[key] = [] }
}

function toggleProject(projectId: string) {
  if (expandedProjects.value.has(projectId)) { expandedProjects.value.delete(projectId) }
  else { expandedProjects.value.add(projectId); if (!projectVersions.value[projectId]) loadVersions(projectId) }
}

function toggleVersion(projectId: string, version: number) {
  const key = `${projectId}-v${version}`
  // 展开时同时选中该版本
  selectedNode.value = { type: 'version', projectId, version }
  if (expandedVersions.value.has(key)) { expandedVersions.value.delete(key) }
  else { expandedVersions.value.add(key); if (!versionCategories.value[key]) loadCategories(projectId, version) }
}

function selectProject(project: Project) {
  selectedNode.value = { type: 'project', id: project.id }
  if (!projectVersions.value[project.id]) loadVersions(project.id)
}

function selectVersion(project: Project, version: DatasetVersion) {
  selectedNode.value = { type: 'version', projectId: project.id, version: version.version }
}

function getCategoryIcon(type: string) {
  const icons: Record<string, typeof Square | typeof Tag> = { bbox: Square, keypoint: Tag, polygon: Shapes, mask: Shapes }
  return icons[type] || Tag
}

function isCategorySelected(projectId: string, version: number, categoryId: number): boolean {
  return selectedCategories.value.some(c => c.projectId === projectId && c.version === version && c.categoryId === categoryId)
}

function toggleCategorySelection(projectId: string, version: number, category: Category) {
  const idx = selectedCategories.value.findIndex(c => c.projectId === projectId && c.version === version && c.categoryId === category.id)
  if (idx >= 0) { selectedCategories.value.splice(idx, 1) }
  else {
    const conflict = selectedCategories.value.find(c => c.projectId === projectId && c.categoryName === category.name && c.version !== version)
    if (conflict) { notifyError(t('dataset.export.conflictWarning')); return }
    selectedCategories.value.push({ projectId, version, categoryId: category.id, categoryName: category.name })
  }
}

// Modal handlers
function openSyncModal(project: Project) { syncTargetProject.value = project; syncNote.value = ''; isSyncModalOpen.value = true }
function closeSyncModal() { isSyncModalOpen.value = false }

async function createVersion() {
  if (!syncTargetProject.value) return
  isSyncing.value = true
  try {
    const res = await fetch('http://localhost:18080/api/dataset-versions/create', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId: syncTargetProject.value.id, note: syncNote.value })
    })
    if (res.ok) { const data = await res.json(); await loadVersions(syncTargetProject.value.id); closeSyncModal(); notifySuccess(t('dataset.sync.success', { version: data.version })) }
    else { notifyError(t('dataset.sync.error')) }
  } catch { notifyError(t('dataset.sync.error')) }
  finally { isSyncing.value = false }
}

function startEditNote() { editNoteValue.value = selectedVersion.value?.note || ''; isEditingNote.value = true }

async function saveNote() {
  if (selectedNode.value?.type !== 'version') return
  try {
    await fetch('http://localhost:18080/api/dataset-versions/update', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId: selectedNode.value.projectId, version: selectedNode.value.version, note: editNoteValue.value })
    })
    await loadVersions(selectedNode.value.projectId); isEditingNote.value = false
  } catch (e) { console.error(e) }
}

// 右键菜单
function openVersionContextMenu(e: MouseEvent, project: Project, version: DatasetVersion) {
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuVersion.value = { projectId: project.id, version: version.version }
  selectedNode.value = { type: 'version', projectId: project.id, version: version.version }
  isVersionContextMenuOpen.value = true
  // 点击其他地方关闭菜单
  setTimeout(() => document.addEventListener('click', closeVersionContextMenu, { once: true }), 0)
}

function closeVersionContextMenu() {
  isVersionContextMenuOpen.value = false
}

function openRollbackModal() { closeVersionContextMenu(); rollbackConfirmed.value = false; isRollbackModalOpen.value = true }
function closeRollbackModal() { isRollbackModalOpen.value = false }

async function executeRollback() {
  const target = contextMenuVersion.value || (selectedNode.value?.type === 'version' ? { projectId: selectedNode.value.projectId, version: selectedNode.value.version } : null)
  if (!target) return
  isRollingBack.value = true
  try {
    const res = await fetch('http://localhost:18080/api/dataset-versions/rollback', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId: target.projectId, version: target.version })
    })
    if (res.ok) { const data = await res.json(); await loadVersions(target.projectId); closeRollbackModal(); notifySuccess(t('dataset.rollback.success', { version: target.version, backup: data.backupVersion })) }
    else { notifyError(t('dataset.rollback.error')) }
  } catch { notifyError(t('dataset.rollback.error')) }
  finally { isRollingBack.value = false }
}

function openDeleteModal() { closeVersionContextMenu(); isDeleteModalOpen.value = true }
function closeDeleteModal() { isDeleteModalOpen.value = false }

async function deleteVersion() {
  const target = contextMenuVersion.value || (selectedNode.value?.type === 'version' ? { projectId: selectedNode.value.projectId, version: selectedNode.value.version } : null)
  if (!target) return
  isDeleting.value = true
  try {
    const res = await fetch('http://localhost:18080/api/dataset-versions/delete', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId: target.projectId, version: target.version })
    })
    if (res.ok) { selectedNode.value = { type: 'project', id: target.projectId }; await loadVersions(target.projectId); closeDeleteModal(); notifySuccess(t('dataset.deleteVersion.success')) }
    else { notifyError(t('dataset.deleteVersion.error')) }
  } catch { notifyError(t('dataset.deleteVersion.error')) }
  finally { isDeleting.value = false }
}

// 导出功能
async function openExportWizard() {
  // 加载导出插件列表
  await loadExportPlugins()
  // 生成默认导出路径
  generateDefaultExportPath()
  isExportModalOpen.value = true
}

function closeExportModal() {
  isExportModalOpen.value = false
}

async function loadExportPlugins() {
  try {
    const res = await fetch('http://localhost:18080/api/plugins/export')
    if (res.ok) {
      const data = await res.json()
      allPlugins.value = data.plugins || []
      // 自动选择第一个可用插件
      const first = availableExportPlugins.value[0]
      if (first) exportPlugin.value = first.id
    }
  } catch (e) { console.error(e) }
}

async function generateDefaultExportPath() {
  // 获取数据目录
  try {
    const res = await fetch('http://localhost:18080/api/settings/paths')
    if (res.ok) {
      const data = await res.json()
      const proj = selectedCategories.value[0]
      if (proj && data.dataPath) {
        const project = projects.value.find(p => p.id === proj.projectId)
        exportPath.value = `${data.dataPath}\\dataset_out\\${project?.name || proj.projectId}_v${proj.version}_${exportFormat.value}`
      }
    }
  } catch (e) { console.error(e) }
}

function onFormatChange() {
  // 格式变更时更新插件选择和路径
  const first = availableExportPlugins.value[0]
  exportPlugin.value = first ? first.id : ''
  // 更新路径中的格式部分
  if (exportPath.value) {
    exportPath.value = exportPath.value.replace(/_[^_]+$/, `_${exportFormat.value}`)
  }
}

// 自动调整划分比例
function onSplitChange(changed: 'train' | 'val' | 'test') {
  const val = exportSplit.value[changed]
  if (val < 0) exportSplit.value[changed] = 0
  if (val > 100) exportSplit.value[changed] = 100

  const remaining = 100 - exportSplit.value[changed]
  
  if (changed === 'train') {
    // 按原比例分配剩余给 val 和 test
    const oldSum = exportSplit.value.val + exportSplit.value.test
    if (oldSum > 0) {
      const ratio = exportSplit.value.val / oldSum
      exportSplit.value.val = Math.round(remaining * ratio)
      exportSplit.value.test = remaining - exportSplit.value.val
    } else {
      exportSplit.value.val = Math.round(remaining / 2)
      exportSplit.value.test = remaining - exportSplit.value.val
    }
  } else if (changed === 'val') {
    // 按原比例分配剩余给 train 和 test
    const oldSum = exportSplit.value.train + exportSplit.value.test
    if (oldSum > 0) {
      const ratio = exportSplit.value.train / oldSum
      exportSplit.value.train = Math.round(remaining * ratio)
      exportSplit.value.test = remaining - exportSplit.value.train
    } else {
      exportSplit.value.train = Math.round(remaining / 2)
      exportSplit.value.test = remaining - exportSplit.value.train
    }
  } else {
    // 按原比例分配剩余给 train 和 val
    const oldSum = exportSplit.value.train + exportSplit.value.val
    if (oldSum > 0) {
      const ratio = exportSplit.value.train / oldSum
      exportSplit.value.train = Math.round(remaining * ratio)
      exportSplit.value.val = remaining - exportSplit.value.train
    } else {
      exportSplit.value.train = Math.round(remaining / 2)
      exportSplit.value.val = remaining - exportSplit.value.train
    }
  }
  
  // 确保非负
  if (exportSplit.value.train < 0) exportSplit.value.train = 0
  if (exportSplit.value.val < 0) exportSplit.value.val = 0
  if (exportSplit.value.test < 0) exportSplit.value.test = 0
}

async function browseExportPath() {
  try {
    // 调用 Electron 文件对话框
    const res = await fetch('http://localhost:18080/api/dialog/select-folder', { method: 'POST' })
    if (res.ok) {
      const data = await res.json()
      if (data.path) {
        exportPath.value = data.path
      }
    }
  } catch (e) { console.error(e) }
}

async function executeExport() {
  if (!exportPlugin.value || splitTotal.value !== 100) return
  
  // 保存当前导出路径（关闭面板后可能被清空）
  const currentExportPath = exportPath.value
  
  // 立即关闭面板
  closeExportModal()
  
  // 显示导出进度通知（持久显示直到导出完成）
  const notifyKey = notification.info(t('dataset.exportWizard.exporting') + ' 0%', { persistent: true })
  
  try {
    // 发起导出请求，获取任务 ID
    const res = await fetch('http://localhost:18080/api/dataset/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        categories: selectedCategories.value,
        outputPath: currentExportPath,
        format: exportFormat.value,
        pluginId: exportPlugin.value,
        split: exportSplit.value
      })
    })
    
    console.log('[Export] Response status:', res.status, res.ok)
    const data = await res.json()
    console.log('[Export] Response data:', data)
    
    if (!res.ok) {
      notification.remove(notifyKey)
      notifyError(t('dataset.exportWizard.error', { msg: data.error || 'unknown' }))
      return
    }
    
    const taskId = data.taskId
    console.log('[Export] Task ID:', taskId)
    if (!taskId) {
      notification.remove(notifyKey)
      notifyError(t('dataset.exportWizard.error', { msg: 'no_task_id' }))
      return
    }
    
    console.log('[Export] Task started, will poll in 500ms')
    
    // 轮询任务状态
    const pollStatus = async () => {
      try {
        const statusRes = await fetch(`http://localhost:18080/api/dataset/export-status?taskId=${taskId}`)
        const task = await statusRes.json() as { 
          phase: string
          progress: number
          error?: string
          outputPath?: string
        }
        
        console.log('[Export] Task status:', task)
        
        if (!statusRes.ok) {
          notification.remove(notifyKey)
          notifyError(t('dataset.exportWizard.error', { msg: task.error || 'status_query_failed' }))
          return
        }
        
        if (task.phase === 'completed') {
          notification.remove(notifyKey)
          notifySuccess(t('dataset.exportWizard.success', { path: task.outputPath || currentExportPath }))
        } else if (task.phase === 'failed') {
          notification.remove(notifyKey)
          notifyError(t('dataset.exportWizard.error', { msg: task.error || 'unknown' }))
        } else {
          // 更新进度通知
          notification.update(notifyKey, {
            message: t('dataset.exportWizard.exporting') + ` ${task.progress}%`
          })
          // 继续轮询
          setTimeout(pollStatus, 500)
        }
      } catch (e) {
        console.error('[Export] Poll error:', e)
        notification.remove(notifyKey)
        notifyError(t('dataset.exportWizard.error', { msg: 'network_error' }))
      }
    }
    
    // 开始轮询
    setTimeout(pollStatus, 500)
  } catch (e) {
    notification.remove(notifyKey)
    notifyError(t('dataset.exportWizard.error', { msg: String(e) }))
  }
}

onMounted(async () => { await loadProjects(); loading.value = false })
</script>

<style scoped>
.dataset-view { display: flex; width: 100%; height: 100%; background: var(--color-bg-app); position: absolute; inset: 0; }

/* 左侧树 */
.dataset-tree { width: 220px; min-width: 180px; border-right: 2px solid var(--color-border-subtle); display: flex; flex-direction: column; background: var(--color-bg-sidebar); }
.dataset-tree__header { padding: 12px 16px; border-bottom: 1px solid var(--color-border-subtle); text-align: left; display: flex; align-items: center; justify-content: space-between; }
.dataset-tree__header h2 { margin: 0; font-size: 13px; font-weight: 600; color: var(--color-fg); text-align: left; }
.tree-header__btn { background: none; border: none; padding: 4px; cursor: pointer; color: var(--color-fg-muted); display: flex; align-items: center; border-radius: 4px; }
.tree-header__btn:hover { background: var(--color-bg-hover); color: var(--color-fg); }
.dataset-tree__loading, .dataset-tree__empty { padding: 24px 16px; text-align: center; color: var(--color-fg-muted); font-size: 12px; }
.dataset-tree__content { flex: 1; overflow-y: auto; padding: 4px 0; }

/* 树节点 */
.tree-node { user-select: none; }
.tree-node__row { display: flex; align-items: center; padding: 5px 8px; cursor: pointer; gap: 2px; transition: background 0.15s; }
.tree-node__row:hover { background: var(--color-bg-sidebar-hover); }
.tree-node__row--selected { background: var(--color-accent-soft) !important; }
.tree-node__row--project { padding-left: 4px; }
.tree-node__row--version { padding-left: 20px; }
.tree-node__row--category { padding-left: 36px; font-size: 12px; }
.tree-node__row--checked { background: var(--color-accent-soft); }
.tree-node__toggle { background: none; border: none; padding: 2px; cursor: pointer; color: var(--color-fg-muted); display: flex; align-items: center; }
.tree-node__icon { color: var(--color-fg-muted); flex-shrink: 0; }
.tree-node__label { flex: 1; font-size: 12px; color: var(--color-fg); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tree-node__meta { font-size: 10px; color: var(--color-fg-muted); margin-left: 4px; }
.tree-node__sync { background: none; border: none; padding: 2px; cursor: pointer; color: var(--color-fg-muted); opacity: 0; transition: opacity 0.15s; display: flex; }
.tree-node__row:hover .tree-node__sync { opacity: 1; }
.tree-node__sync:hover { color: var(--color-accent); }
.tree-node__checkbox { margin-right: 4px; }
.tree-node__empty-hint { padding: 6px 8px 6px 36px; font-size: 11px; color: var(--color-fg-muted); font-style: italic; }

/* 右侧详情区 */
.dataset-detail { flex: 1; display: flex; flex-direction: column; overflow: hidden; background: var(--color-bg-app); }
.dataset-export-bar { display: flex; align-items: center; justify-content: space-between; padding: 6px 16px; background: var(--color-bg-sidebar); border-bottom: 1px solid var(--color-border-subtle); font-size: 12px; color: var(--color-fg); }
.export-btn { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; border: 1px solid var(--color-accent); border-radius: 0; background: var(--color-accent); color: white; font-size: 11px; cursor: pointer; }
.export-btn:hover { background: var(--color-accent-hover); }

/* 空状态 */
.detail-empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; color: var(--color-fg-muted); gap: 8px; }
.detail-empty p { margin: 0; font-size: 13px; }

/* 详情面板 */
.detail-panel { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.detail-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 20px; background: var(--color-bg-sidebar); border-bottom: 1px solid var(--color-border-subtle); }
.detail-header__title { display: flex; align-items: center; gap: 8px; color: var(--color-fg); }
.detail-header__title h2 { margin: 0; font-size: 15px; font-weight: 600; }
.detail-header__badge { padding: 2px 8px; background: var(--color-accent-soft); color: var(--color-fg-muted); font-size: 11px; border-radius: 2px; }
.detail-header__meta { display: flex; align-items: center; gap: 12px; }
.detail-header__time { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--color-fg-muted); }

/* 备注栏 */
.detail-note-bar { display: flex; align-items: center; gap: 8px; padding: 8px 20px; background: var(--color-bg-header); border-bottom: 1px solid var(--color-border-subtle); font-size: 12px; color: var(--color-fg-muted); }
.detail-note-bar__text { flex: 1; }
.detail-note-bar__edit { background: none; border: none; padding: 2px; cursor: pointer; color: var(--color-fg-muted); opacity: 0.6; display: flex; }
.detail-note-bar:hover .detail-note-bar__edit { opacity: 1; }
.detail-note-bar__edit:hover { color: var(--color-accent); }
.detail-note-bar--editing { background: var(--color-bg-sidebar); }
.detail-note-bar--editing input { flex: 1; padding: 4px 8px; border: 1px solid var(--color-border-subtle); background: var(--color-bg-app); color: var(--color-fg); font-size: 12px; }
.detail-note-bar__btn { background: none; border: none; padding: 4px; cursor: pointer; color: var(--color-fg-muted); display: flex; }
.detail-note-bar__btn:hover { color: var(--color-accent); }

/* 内容主体 */
.detail-body { flex: 1; overflow-y: auto; padding: 20px; }

/* 提示 */
.detail-tip { display: flex; align-items: center; gap: 8px; padding: 12px 16px; background: var(--color-accent-soft); border-left: 3px solid var(--color-accent); font-size: 12px; color: var(--color-fg-muted); margin-top: 16px; }

/* 统计卡片 */
.stat-cards { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
.stat-cards--triple { grid-template-columns: repeat(3, 1fr); }
.stat-card { display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--color-bg-sidebar); border: 1px solid var(--color-border-subtle); }
.stat-card__icon { color: var(--color-fg-muted); }
.stat-card__content { display: flex; flex-direction: column; }
.stat-card__value { font-size: 22px; font-weight: 600; color: var(--color-fg); line-height: 1.2; }
.stat-card__value--small { font-size: 13px; font-weight: 500; }
.stat-card__label { font-size: 11px; color: var(--color-fg-muted); margin-top: 2px; }
.stat-card--accent .stat-card__icon { color: #3b82f6; }
.stat-card--success .stat-card__icon { color: #10b981; }
.stat-card--warning .stat-card__icon { color: #f59e0b; }

/* 健康度面板 */
.health-panel { background: var(--color-bg-sidebar); border: 1px solid var(--color-border-subtle); padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.health-item__header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.health-item__label { font-size: 12px; color: var(--color-fg); font-weight: 500; }
.health-item__value { font-size: 12px; color: var(--color-accent); font-weight: 600; }
.health-progress { height: 6px; background: var(--color-bg-app); border-radius: 3px; overflow: hidden; }
.health-progress__bar { height: 100%; background: var(--color-accent); border-radius: 3px; }
.health-item__desc { margin: 6px 0 0; font-size: 11px; color: var(--color-fg-muted); }

.health-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 12px; padding-top: 12px; border-top: 1px solid var(--color-border-subtle); }
.health-metric { display: flex; flex-direction: column; align-items: center; justify-content: center; }
.health-metric__value { font-size: 18px; font-weight: 600; color: var(--color-fg); }
.health-metric__label { font-size: 10px; color: var(--color-fg-muted); margin-top: 2px; }
.health-check { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; text-align: center; }
.health-check span { font-size: 10px; color: var(--color-fg-muted); }
.health-check--pass { color: #10b981; }
.health-check--warning { color: #f59e0b; }
.health-check--error { color: #ef4444; }

/* 操作按钮 */
.detail-actions { display: flex; gap: 12px; margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--color-border-subtle); }
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 16px; border: 1px solid var(--color-border-subtle); border-radius: 6px; background: var(--color-bg-sidebar); color: var(--color-fg); font-size: 13px; cursor: pointer; transition: all 0.15s; }
.btn:hover { background: var(--color-bg-sidebar-hover); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn--primary { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.btn--primary:hover { background: var(--color-accent-hover); }
.btn--warning { background: #f59e0b; border-color: #f59e0b; color: white; }
.btn--warning:hover { opacity: 0.9; }
.btn--danger { background: #ef4444; border-color: #ef4444; color: white; }
.btn--danger:hover { opacity: 0.9; }

/* 弹窗 */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: var(--color-bg-header); border-radius: 12px; padding: 24px; width: 420px; max-width: 90vw; border: 1px solid var(--color-border-subtle); }
.modal h2 { margin: 0 0 16px; font-size: 18px; color: var(--color-fg); }
.modal__desc { margin: 0 0 8px; font-size: 13px; color: var(--color-fg-muted); }
.modal__project { margin: 0 0 16px; font-size: 14px; font-weight: 600; color: var(--color-fg); }
.modal__warning { color: #f59e0b; font-size: 13px; margin: 0 0 12px; }
.modal__field { margin-bottom: 16px; }
.modal__field label { display: block; font-size: 13px; color: var(--color-fg-muted); margin-bottom: 6px; }
.modal__field input { width: 100%; padding: 8px 12px; border: 1px solid var(--color-border-subtle); border-radius: 6px; font-size: 13px; background: var(--color-bg-sidebar); color: var(--color-fg); box-sizing: border-box; }
.modal__confirm-check { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--color-fg); margin: 16px 0; cursor: pointer; }
.modal__actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 24px; }

/* 右键菜单 */
.context-menu { position: fixed; z-index: 2000; background: var(--color-bg-header); border: 1px solid var(--color-border-subtle); box-shadow: 0 4px 12px rgba(0,0,0,0.3); min-width: 140px; }
.context-menu__item { display: flex; align-items: center; gap: 8px; width: 100%; padding: 8px 12px; border: none; background: none; color: var(--color-fg); font-size: 12px; cursor: pointer; text-align: left; }
.context-menu__item:hover { background: var(--color-bg-sidebar-hover); }
.context-menu__item--danger { color: #ef4444; }
.context-menu__item--danger:hover { background: rgba(239,68,68,0.1); }

/* 导出面板 */
.modal--export { width: 520px; }
.modal--export h2 { display: flex; align-items: center; gap: 8px; }
.export-summary { display: flex; gap: 16px; padding: 12px 16px; background: var(--color-bg-sidebar); border: 1px solid var(--color-border-subtle); margin-bottom: 16px; }
.export-summary__item { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-fg); }
.export-path-input { display: flex; gap: 8px; }
.export-path-input input { flex: 1; }
.export-row { display: flex; gap: 16px; }
.export-field { flex: 1; }
.modal__field select { width: 100%; padding: 8px 12px; border: 1px solid var(--color-border-subtle); border-radius: 6px; font-size: 13px; background: var(--color-bg-sidebar); color: var(--color-fg); }
.export-no-plugin { margin: 4px 0 0; font-size: 11px; color: #f59e0b; }
.btn--small { padding: 6px 10px; }
.export-split { display: flex; gap: 12px; }
.export-split__item { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 6px; background: var(--color-bg-sidebar); border: 1px solid var(--color-border-subtle); padding: 10px 8px; }
.export-split__label { font-size: 11px; color: var(--color-fg-muted); }
.export-split__row { display: flex; align-items: center; gap: 4px; }
.export-split__item input { width: 50px; padding: 4px 6px; border: 1px solid var(--color-border-subtle); border-radius: 4px; background: var(--color-bg-app); color: var(--color-fg); font-size: 13px; text-align: center; }
.export-split__unit { font-size: 12px; color: var(--color-fg-muted); }
.export-split__count { font-size: 11px; color: var(--color-accent); }
.export-split__warning { display: flex; align-items: center; gap: 4px; margin: 8px 0 0; font-size: 11px; color: #f59e0b; }

/* 导出历史 */
.export-history { display: flex; flex-direction: column; gap: 6px; }
.export-history__item { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; background: var(--color-bg-sidebar); border: 1px solid var(--color-border-subtle); }
.export-history__info { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
.export-history__format { padding: 2px 6px; background: var(--color-accent-soft); color: var(--color-accent); font-size: 10px; font-weight: 600; }
.export-history__path { flex: 1; font-size: 12px; color: var(--color-fg); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.export-history__time { font-size: 11px; color: var(--color-fg-muted); }
.export-history__open { background: none; border: none; padding: 4px; cursor: pointer; color: var(--color-fg-muted); display: flex; }
.export-history__open:hover { color: var(--color-accent); }

/* 动画 */
.spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
