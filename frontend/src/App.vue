<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, watch, nextTick } from 'vue'
import { 
  Folder, Database, Bot, Puzzle, Palette, HelpCircle, Github,
  Settings, Minus, Square, X, Bell, Import, ImagePlus, ChevronDown,
  Trash2, ImageOff, FilterX, Save, Pencil, Loader2, Plus, List, Clock, ArrowUp
} from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import logo from './assets/logo.svg'
import NotificationItem from './components/NotificationItem.vue'
import SettingsView from './views/SettingsView.vue'
import AnnotationCanvas from './components/AnnotationCanvas.vue'
import InferencePluginBar from './components/InferencePluginBar.vue'
import type { Annotation } from './components/AnnotationCanvas.vue'
import { useNotification } from './composables/useNotification'
import { useShortcuts } from './composables/useShortcuts'
import { useGlobalWs } from './composables/useGlobalWs'
import { useInferencePlugins } from './composables/useInferencePlugins'

// 初始化全局 WebSocket
const { connect: connectWs, subscribe: subscribeWs } = useGlobalWs()

// 推理插件
const { activePlugin } = useInferencePlugins()

// 防止 Electron IPC 监听器重复注册（组件重建时会重复调用 onMounted）
let electronListenersRegistered = false

const router = useRouter()
const route = useRoute()
const isSettingsOpen = ref(false)
const isProjectMenuOpen = ref(false)
const isCreateProjectModalOpen = ref(false)
const newProjectName = ref('')
const isCreatingProject = ref(false)
const createProjectError = ref('')

// 项目右键菜单相关状态
const isProjectContextMenuOpen = ref(false)
const projectContextMenuX = ref(0)
const projectContextMenuY = ref(0)
const projectContextMenuTarget = ref<ProjectSummary | null>(null)
const isDeleteProjectModalOpen = ref(false)
const isDeletingProject = ref(false)
const isRenameProjectModalOpen = ref(false)
const renameProjectName = ref('')
const isRenamingProject = ref(false)
const renameProjectError = ref('')
const isImportImagesModalOpen = ref(false)
const importImagesError = ref('')
const isImportingImages = ref(false)
const importMode = ref<'copy' | 'link' | 'external'>('copy')

// 导入数据集相关状态
const isImportDatasetModalOpen = ref(false)
const importDatasetStep = ref<'select' | 'detecting' | 'configure' | 'importing'>('select')
const importDatasetError = ref('')
const importDatasetPath = ref('')
const detectedPlugins = ref<{ pluginId: string; score: number; reason: string }[]>([])
const selectedPluginId = ref('')
const importDatasetStats = ref<{ images: number; categories: number; annotations: number } | null>(null)
const importDatasetMode = ref<'copy' | 'link' | 'external'>('copy')

type ProjectSummary = {
  id: string
  name: string
  createdAt?: string
  imageCount?: number
}

// 检查是否存在依赖未安装的 Python 插件
const checkPythonPluginDepsIssue = async () => {
	try {
		const res = await fetch('http://127.0.0.1:18080/api/python/plugins-deps-summary')
		if (!res.ok) return
		const data = await res.json()
		hasPythonPluginDepsIssue.value = !!data.hasProblem
	} catch {
		// 网络 / 后端异常时，不打角标，避免干扰正常使用
		hasPythonPluginDepsIssue.value = false
	}
}

// 底部栏：项目统计信息
const footerProjectStats = computed(() => {
	if (!currentProject.value) return null
	return {
		total: projectStats.value.total,
		annotated: projectStats.value.annotatedCount,
		unannotated: projectStats.value.unannotatedCount,
		negative: projectStats.value.negativeCount
	}
})

// 底部栏：当前图片信息
const footerActiveImageInfo = computed(() => {
	if (!activeImage.value) return null
	const status = activeImage.value.annotationStatus
	return {
		status,
		annotationCount: currentAnnotations.value.length
	}
})

const projects = ref<ProjectSummary[]>([])
const currentProject = ref<ProjectSummary | null>(null)

// 是否存在依赖未安装的 Python 插件（用于在侧边栏 Python 按钮上显示橙色角标）
const hasPythonPluginDepsIssue = ref(false)

type ProjectImageListItem = {
  id: number
  filename: string
  hasThumb: boolean
  isExternal: boolean
  thumbPath: string
  originalPath: string
  annotationStatus: 'none' | 'annotated' | 'negative'
}

type ImageLoadStatus = 'loading' | 'loaded' | 'error'

// 推理结果标注类型
type InferenceAnnotation = {
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

const projectImages = ref<ProjectImageListItem[]>([])
const imageListRef = ref<HTMLDivElement | null>(null)
const imageListScrollTop = ref(0)
const imageListScrollTopPending = ref(0)
let imageListScrollTimer: number | null = null
const imageListViewportHeight = ref(0)
const imageRowHeight = ref(110)
const imageRowBuffer = 2
const imageLoadState = ref<Record<number, ImageLoadStatus>>({})
const isImageListExpanded = ref(false)
const selectedImageIds = ref<Set<number>>(new Set())
const lastSelectedIndex = ref<number | null>(null)
const isDragSelecting = ref(false)
const dragStartPoint = ref<{ x: number; y: number } | null>(null)
const dragCurrentPoint = ref<{ x: number; y: number } | null>(null)
const dragSelectRect = ref<{ left: number; top: number; width: number; height: number } | null>(null)
const dragStartContentY = ref<number | null>(null)
const dragCurrentContentY = ref<number | null>(null)
const imageCellRefs = ref<Map<number, HTMLElement>>(new Map())

const isImageContextMenuOpen = ref(false)
const imageContextMenuX = ref(0)
const imageContextMenuY = ref(0)
const imageContextMenuImageId = ref<number | null>(null)
const imageContextMenuUseSelection = ref(false)
const imageContextMenuSelectionIds = ref<number[]>([])
const isDeleteImagesModalOpen = ref(false)
const deleteTargetImageIds = ref<number[]>([])
const activeImage = ref<ProjectImageListItem | null>(null)

// AnnotationCanvas 组件引用（用于调用暴露的方法）
const annotationCanvasRef = ref<InstanceType<typeof AnnotationCanvas> | null>(null)

// 自动保存开关
const isAutoSaveEnabled = ref(false)
// 自动保存进行中的 Promise（用于等待保存完成）
let autoSavePromise: Promise<void> | null = null

// 快捷键管理
const { matchAction } = useShortcuts()

const hasExternalInDeleteTargets = computed(() => {
	if (!deleteTargetImageIds.value.length) return false
	const idSet = new Set(deleteTargetImageIds.value)
	return projectImages.value.some((img) => idSet.has(img.id) && img.isExternal)
})

// 切换图片时加载标注并自动推理
watch(activeImage, async (newImg) => {
	if (newImg) {
		await loadAnnotations(newImg.id)
		// 为 SAM-2 等交互式推理设置图像
		if (currentProject.value && newImg.originalPath) {
			// 外部图片使用绝对路径，内部图片使用相对路径
			const imagePath = newImg.isExternal 
				? newImg.originalPath 
				: `project_item/${currentProject.value.id}/${newImg.originalPath}`
			console.log('[App] Setting inference image:', imagePath, 'isExternal:', newImg.isExternal)
			// 等待 set_image 完成后再推理
			const setImageRes = await window.electronAPI?.inferenceSetImage?.(imagePath)
			console.log('[App] Set image result:', setImageRes)
		} else {
			console.log('[App] Skip set_image: project=', currentProject.value?.id, 'originalPath=', newImg.originalPath)
		}
		// 自动推理（如果开启，且不是交互式推理模式）
		// SAM-2 等 prompt 模式插件需要用户通过 Shift+点击 提供提示点
		// YOLOE 等 box 模式插件需要用户框选一个目标作为 visual prompt
		// text 模式插件（如 Grounding-DINO）自动使用类别名推理
		const mode = activePlugin.value?.interactionMode
		if (autoInference.value && mode !== 'prompt' && mode !== 'box') {
			runInference()
		}
	} else {
		currentAnnotations.value = []
	}
})

type CategoryTab = 'bbox' | 'keypoint' | 'polygon' | 'category'
const activeCategoryTab = ref<CategoryTab>('bbox')

type ImageFilterTab = 'all' | 'annotated' | 'unannotated' | 'negative'
const activeImageFilterTab = ref<ImageFilterTab>('all')

// 项目统计
const projectStats = ref<{
  total: number
  annotatedCount: number
  unannotatedCount: number
  negativeCount: number
  totalAnnotations: number
}>({ total: 0, annotatedCount: 0, unannotatedCount: 0, negativeCount: 0, totalAnnotations: 0 })

// 根据筛选条件过滤图片
const filteredProjectImages = computed(() => {
  if (activeImageFilterTab.value === 'all') return projectImages.value
  return projectImages.value.filter(img => {
    if (activeImageFilterTab.value === 'annotated') return img.annotationStatus === 'annotated'
    if (activeImageFilterTab.value === 'unannotated') return img.annotationStatus === 'none'
    if (activeImageFilterTab.value === 'negative') return img.annotationStatus === 'negative'
    return true
  })
})

// 类别面板：新建类别编辑状态
type CategoryType = 'bbox' | 'keypoint' | 'polygon' | 'category'
type ProjectCategory = {
	id: number
	name: string
	type: CategoryType
	color: string
	sortOrder: number
	mate: string
}

type KeypointMate = {
	keypoints: { id: number; name: string }[]
}

const parseKeypointMate = (mate: string): KeypointMate | null => {
	if (!mate) return null
	try {
		const obj = JSON.parse(mate) as KeypointMate
		if (obj && Array.isArray(obj.keypoints)) return obj
		return null
	} catch {
		return null
	}
}

const selectPresetEditCategoryColor = (color: string) => {
	if (!editCategoryTarget.value) return
	editCategoryColor.value = color
	// 颜色编辑时直接复用原名称，避免服务端 name_required 错误
	if (!editCategoryName.value) {
		editCategoryName.value = editCategoryTarget.value.name
	}
}

const handleEditNativeColorChange = (event: Event) => {
	const input = event.target as HTMLInputElement | null
	if (!input) return
	const value = input.value
	if (!value) return
	editCategoryColor.value = value
}

const getKeypointCount = (cat: ProjectCategory): number => {
	if (cat.type !== 'keypoint') return 0
	const parsed = parseKeypointMate(cat.mate)
	return parsed?.keypoints?.length ?? 0
}
const projectCategories = ref<ProjectCategory[]>([])
const selectedCategory = ref<ProjectCategory | null>(null)
const selectedKeypointCategory = ref<ProjectCategory | null>(null) // 关键点副类别
const currentAnnotations = ref<Annotation[]>([])

const filteredCategories = computed(() => {
	const type = activeCategoryTab.value
	return projectCategories.value
		.filter((cat) => cat.type === type)
		.sort((a, b) => {
			if (a.sortOrder !== b.sortOrder) return a.sortOrder - b.sortOrder
			return a.id - b.id
		})
})

const isCreatingCategory = ref(false)
const newCategoryName = ref('')
const newCategoryColor = ref('#f97316')
const isCategoryColorPickerOpen = ref(false)
const categoryEditorRef = ref<HTMLDivElement | null>(null)

// 获取当前项目中可用于绑定的矩形框类别列表（用于关键点配置面板）
const availableBboxCategories = computed(() => {
	return projectCategories.value.filter(c => c.type === 'bbox')
})
const categoryColorButtonRef = ref<HTMLButtonElement | null>(null)

const presetCategoryColors: string[] = [
	'#ff0000', // red
	'#00ff00', // green
	'#0000ff', // blue
	'#ffff00', // yellow
	'#ff00ff', // magenta
	'#00ffff', // cyan
	'#ff7f00', // orange
	'#00ff7f'  // spring green
]

let nextPresetCategoryColorIndex = 0

// 类别右键菜单状态
const isCategoryContextMenuOpen = ref(false)
const categoryContextMenuX = ref(0)
const categoryContextMenuY = ref(0)
const categoryContextMenuTarget = ref<ProjectCategory | null>(null)

// 关键点配置弹窗状态
const isKeypointConfigModalOpen = ref(false)
const keypointConfigTarget = ref<ProjectCategory | null>(null)
const keypointConfigList = ref<{ name: string }[]>([])
const keypointConfigError = ref('')
const isKeypointConfigSaving = ref(false)
const keypointConfigBindBboxId = ref<number | null>(null) // 关键点类别绑定的矩形框类别 ID

const isDeleteCategoryModalOpen = ref(false)
const deleteCategoryTarget = ref<ProjectCategory | null>(null)
const deleteCategoryError = ref('')
const isDeletingCategory = ref(false)
const editCategoryTarget = ref<ProjectCategory | null>(null)
const editCategoryName = ref('')
const editCategoryColor = ref('#ff0000')
const isEditingCategory = ref(false)
const isEditingCategoryName = ref(false)
const editCategoryError = ref('')
const isEditCategoryColorPickerOpen = ref(false)
const draggingCategoryId = ref<number | null>(null)
const dragOverCategoryId = ref<number | null>(null)

// 类别合并相关状态
const isMergeCategoryModalOpen = ref(false)
const mergeCategoryTargetName = ref('')
const mergeCategoryTargetId = ref<number | null>(null)
const isMergingCategory = ref(false)

const { t } = useI18n()

type ElectronWindowApi = {
  minimize: () => void
  toggleMaximize: () => void
  close: () => void
}

const invokeWindowApi = (method: keyof ElectronWindowApi) => {
  const anyWindow = window as any
  const api: ElectronWindowApi | undefined = anyWindow?.electronAPI
  if (api && typeof api[method] === 'function') {
    api[method]()
  }
}

type ImportTaskPhase = 'scanning' | 'copying' | 'indexing' | 'completed' | 'failed'

type ImportTaskStatus = {
  id: string
  projectId: string
  phase: ImportTaskPhase
  progress: number
  imported: number
  total: number
  error?: string
}

const imageListColumns = computed(() => (isImageListExpanded.value ? 5 : 2))

const totalImageRows = computed(() => {
  const cols = imageListColumns.value || 1
  return Math.ceil(filteredProjectImages.value.length / cols)
})

// 可见行范围：基于“即时”滚动位置，用于占位图和虚拟行布局，避免快速滚动时出现空白
const visibleImageRange = computed(() => {
  const vh = imageListViewportHeight.value || 0
  const total = totalImageRows.value
  if (!vh || total === 0) {
    return { start: 0, end: 0 }
  }
  const rowHeight = imageRowHeight.value || 1
  const firstRow = Math.floor(imageListScrollTopPending.value / rowHeight)
  const visibleCount = Math.ceil(vh / rowHeight)
  const start = Math.max(0, firstRow - imageRowBuffer)
  const end = Math.min(total, firstRow + visibleCount + imageRowBuffer)
  return { start, end }
})

// 图片实际加载范围：基于节流后的滚动位置，减少图片请求数量
const imageLoadRange = computed(() => {
  const vh = imageListViewportHeight.value || 0
  const total = totalImageRows.value
  if (!vh || total === 0) {
    return { start: 0, end: 0 }
  }
  const rowHeight = imageRowHeight.value || 1
  const firstRow = Math.floor(imageListScrollTop.value / rowHeight)
  const visibleCount = Math.ceil(vh / rowHeight)
  const start = Math.max(0, firstRow - imageRowBuffer)
  const end = Math.min(total, firstRow + visibleCount + imageRowBuffer)
  return { start, end }
})

const visibleImageRows = computed(() => {
  const { start, end } = visibleImageRange.value
  const rows: { index: number; items: ProjectImageListItem[] }[] = []
  const items = filteredProjectImages.value
  const cols = imageListColumns.value || 1
  for (let row = start; row < end; row++) {
    const base = row * cols
    if (base >= items.length) break
    const rowItems: ProjectImageListItem[] = []
    for (let i = 0; i < cols; i++) {
      const idx = base + i
      if (idx >= items.length) break
      const item = items[idx]!
      rowItems.push(item)
    }
    rows.push({ index: row, items: rowItems })
  }
  return rows
})

const imageListPaddingTop = computed(
  () => visibleImageRange.value.start * (imageRowHeight.value || 0)
)
const imageListPaddingBottom = computed(() => {
  const { end } = visibleImageRange.value
  const total = totalImageRows.value
  const remaining = total - end
  return remaining > 0 ? remaining * (imageRowHeight.value || 0) : 0
})

const shouldLoadImagesForRow = (rowIndex: number) => {
	const range = imageLoadRange.value
	return rowIndex >= range.start && rowIndex < range.end
}

const handleImageListScroll = (event: Event) => {
  const el = event.target as HTMLElement | null
  if (!el) return
  imageListScrollTopPending.value = el.scrollTop
  imageListViewportHeight.value = el.clientHeight

  if (imageListScrollTimer !== null) {
    window.clearTimeout(imageListScrollTimer)
  }
  imageListScrollTimer = window.setTimeout(() => {
    imageListScrollTop.value = imageListScrollTopPending.value
  }, 80)
}

const refreshImageListViewport = () => {
  const el = imageListRef.value
  if (!el) return
  imageListViewportHeight.value = el.clientHeight
  imageListScrollTop.value = el.scrollTop

  const firstRow = el.querySelector('.project-image-row') as HTMLElement | null
  if (firstRow) {
    const h = firstRow.offsetHeight
    if (h > 0) {
      imageRowHeight.value = h
    }
  }
}

const toggleImageListExpanded = () => {
  isImageListExpanded.value = !isImageListExpanded.value
  void nextTick(() => {
    refreshImageListViewport()
    // 确保当前选中的图片在可视范围内
    if (activeImage.value) {
      scrollToImage(activeImage.value.id)
    }
  })
}

const handleSelectAllImages = () => {
	const images = projectImages.value
	if (!images.length) {
		selectedImageIds.value = new Set()
		return
	}
	const allIds = images.map((img) => img.id)
	const current = selectedImageIds.value
	const allSelected = allIds.every((id) => current.has(id))
	selectedImageIds.value = allSelected ? new Set() : new Set(allIds)
}

let dragMouseMoveHandler: ((e: MouseEvent) => void) | null = null
let dragMouseUpHandler: ((e: MouseEvent) => void) | null = null
let dragSelectionMoved = false

// 计算拖拽选择框的样式（fixed 定位，裁剪到容器范围）
const dragSelectionRectStyle = computed(() => {
	if (!isDragSelecting.value || !imageListRef.value) return null
	if (dragStartContentY.value == null || dragCurrentContentY.value == null) return null
	if (!dragStartPoint.value || !dragCurrentPoint.value) return null
	
	const container = imageListRef.value
	const containerRect = container.getBoundingClientRect()
	
	// X 方向：使用屏幕坐标，裁剪到容器范围
	const startX = dragStartPoint.value.x
	const currentX = dragCurrentPoint.value.x
	const minX = Math.max(Math.min(startX, currentX), containerRect.left)
	const maxX = Math.min(Math.max(startX, currentX), containerRect.right)
	
	// Y 方向：使用内容坐标转换为屏幕坐标，裁剪到容器范围
	const startScreenY = containerRect.top + dragStartContentY.value - container.scrollTop
	const currentScreenY = dragCurrentPoint.value.y
	const minY = Math.max(Math.min(startScreenY, currentScreenY), containerRect.top)
	const maxY = Math.min(Math.max(startScreenY, currentScreenY), containerRect.bottom)
	
	if (maxX <= minX || maxY <= minY) return null
	
	return {
		left: minX + 'px',
		top: minY + 'px',
		width: (maxX - minX) + 'px',
		height: (maxY - minY) + 'px'
	}
})

const updateDragSelection = () => {
	const container = imageListRef.value
	if (!container || dragStartContentY.value == null || dragCurrentContentY.value == null) return
	
	// 基于内容坐标计算选中的行范围（支持虚拟滚动）
	const startY = dragStartContentY.value
	const currentY = dragCurrentContentY.value
	const minY = Math.min(startY, currentY)
	const maxY = Math.max(startY, currentY)
	
	const rowHeightPx = imageRowHeight.value || 1
	const cols = imageListColumns.value || 1
	const images = filteredProjectImages.value
	const totalRows = Math.ceil(images.length / cols)
	
	if (rowHeightPx <= 0 || totalRows <= 0) return
	
	// 计算选中的行范围
	let firstRow = Math.floor(minY / rowHeightPx)
	let lastRow = Math.floor(maxY / rowHeightPx)
	if (firstRow < 0) firstRow = 0
	if (lastRow >= totalRows) lastRow = totalRows - 1
	
	// 选中这些行中的所有图片
	const selected = new Set<number>()
	for (let r = firstRow; r <= lastRow; r++) {
		const base = r * cols
		for (let c = 0; c < cols; c++) {
			const idx = base + c
			if (idx >= images.length) break
			const img = images[idx]
			if (img) selected.add(img.id)
		}
	}
	selectedImageIds.value = selected
}

const handleImageListMouseDown = (event: MouseEvent) => {
	if (event.button !== 0) return
	const container = imageListRef.value
	if (!container) return
	const target = event.target as HTMLElement | null
	if (target && target.closest('.project-image-cell')) {
		return
	}
	event.preventDefault()
	isDragSelecting.value = true
	// 从空白区域开始框选时，认为是一次新的选择操作，清空之前的选中状态
	selectedImageIds.value = new Set()
	dragSelectionMoved = false
	const start = { x: event.clientX, y: event.clientY }
	dragStartPoint.value = start
	dragCurrentPoint.value = start
	const rect = container.getBoundingClientRect()
	const startContentY = start.y - rect.top + container.scrollTop
	dragStartContentY.value = startContentY
	dragCurrentContentY.value = startContentY
	dragSelectRect.value = { left: start.x, top: start.y, width: 0, height: 0 }

	const handleMove = (e: MouseEvent) => {
		if (!isDragSelecting.value || dragStartContentY.value == null) return
		const containerRect = container.getBoundingClientRect()
		const current = { x: e.clientX, y: e.clientY }
		dragCurrentPoint.value = current
		
		// 更新当前内容坐标
		const currentContentY = current.y - containerRect.top + container.scrollTop
		dragCurrentContentY.value = currentContentY
		
		// 计算选择框：起始点使用内容坐标转换回屏幕坐标
		const startScreenY = containerRect.top + dragStartContentY.value - container.scrollTop
		const left = Math.min(dragStartPoint.value?.x ?? current.x, current.x)
		const top = Math.min(startScreenY, current.y)
		const width = Math.abs((dragStartPoint.value?.x ?? current.x) - current.x)
		const height = Math.abs(startScreenY - current.y)
		dragSelectRect.value = { left, top, width, height }
		updateDragSelection()

		// 鼠标靠近容器上下边缘时，自动滚动以支持跨屏幕框选
		const edge = 40
		let delta = 0
		if (current.y < containerRect.top + edge) {
			delta = -Math.min(edge, containerRect.top + edge - current.y)
		} else if (current.y > containerRect.bottom - edge) {
			delta = Math.min(edge, current.y - (containerRect.bottom - edge))
		}
		if (delta !== 0) {
			container.scrollTop += delta
		}
		if (!dragSelectionMoved && (width > 3 || height > 3)) {
			dragSelectionMoved = true
		}
	}

	const handleUp = () => {
		// 最后一次更新选中状态（确保覆盖所有滚动过的区域）
		if (isDragSelecting.value && dragSelectionMoved) {
			updateDragSelection()
		}
		
		isDragSelecting.value = false
		dragStartPoint.value = null
		dragCurrentPoint.value = null
		dragSelectRect.value = null
		if (dragMouseMoveHandler) {
			window.removeEventListener('mousemove', dragMouseMoveHandler)
			dragMouseMoveHandler = null
		}
		if (dragMouseUpHandler) {
			window.removeEventListener('mouseup', dragMouseUpHandler)
			dragMouseUpHandler = null
		}
	}

	dragMouseMoveHandler = handleMove
	dragMouseUpHandler = handleUp
	window.addEventListener('mousemove', handleMove)
	window.addEventListener('mouseup', handleUp)
}

const handleImageListClick = (event: MouseEvent) => {
	// 如果刚刚完成过一次拖拽框选，不把这次 click 当作清空操作
	if (dragSelectionMoved) {
		dragSelectionMoved = false
		return
	}
	if (event.shiftKey) {
		return
	}
	const target = event.target as HTMLElement | null
	// 点击图片：清空所有选中
	if (target && target.closest('.project-image-cell')) {
		selectedImageIds.value = new Set()
		return
	}
	// 点击空白区域：清空所有选中
	selectedImageIds.value = new Set()
}

onBeforeUnmount(() => {
	if (dragMouseMoveHandler) {
		window.removeEventListener('mousemove', dragMouseMoveHandler)
	}
	if (dragMouseUpHandler) {
		window.removeEventListener('mouseup', dragMouseUpHandler)
	}
})

const markImageLoaded = (id: number) => {
  imageLoadState.value = {
    ...imageLoadState.value,
    [id]: 'loaded'
  }
}

const markImageError = (id: number) => {
  imageLoadState.value = {
    ...imageLoadState.value,
    [id]: 'error'
  }
}

const isImageLoaded = (id: number) => imageLoadState.value[id] === 'loaded'
const isImageError = (id: number) => imageLoadState.value[id] === 'error'

const getImageThumbUrl = (item: ProjectImageListItem) => {
	if (!currentProject.value) return ''
	const params = new URLSearchParams({
		projectId: currentProject.value.id,
		path: item.thumbPath
	})
	return `http://localhost:18080/api/project-image-file?${params.toString()}`
}

const getImageOriginalUrl = (item: ProjectImageListItem) => {
	if (!currentProject.value) return ''
	const params = new URLSearchParams({
		projectId: currentProject.value.id,
		imageId: String(item.id),
		kind: 'original'
	})
	return `http://localhost:18080/api/project-image?${params.toString()}`
}

// ==================== 标注相关函数 ====================
const loadAnnotations = async (imageId: number) => {
	if (!currentProject.value) return
	try {
		const params = new URLSearchParams({
			projectId: currentProject.value.id,
			imageId: String(imageId)
		})
		const res = await fetch(`http://localhost:18080/api/project-annotations?${params.toString()}`)
		if (!res.ok) {
			currentAnnotations.value = []
			return
		}
		const data = await res.json() as { annotations?: { id: number; imageId: number; categoryId: number; type: string; data: string }[] }
		if (!Array.isArray(data.annotations)) {
			currentAnnotations.value = []
			return
		}
		// 将后端数据转换为前端 Annotation 格式
		currentAnnotations.value = data.annotations.map(a => ({
			id: `db_${a.id}`,
			dbId: a.id,
			imageId: a.imageId,
			categoryId: a.categoryId,
			type: a.type as 'bbox' | 'keypoint' | 'polygon' | 'category',
			data: JSON.parse(a.data)
		}))
	} catch (e) {
		console.error('load annotations error:', e)
		currentAnnotations.value = []
	}
}

// 解析矩形框类别的 mate 字段，获取绑定的关键点类别 ID
const parseBboxMate = (mate: string): { keypointCategoryId?: number } | null => {
	if (!mate) return null
	try {
		return JSON.parse(mate) as { keypointCategoryId?: number }
	} catch {
		return null
	}
}

// 为推理标注创建不存在的类别（包括自动创建关键点类别并绑定）
const ensureInferenceCategoriesExist = async (): Promise<Map<string, number>> => {
	if (!currentProject.value) return new Map()
	
	const categoryNameToId = new Map<string, number>()
	for (const cat of projectCategories.value) {
		categoryNameToId.set(cat.name, cat.id)
	}
	
	// 获取所有推理标注
	const allInferenceAnns = currentAnnotations.value.filter(a => a.isInference)
	
	// 找出需要创建的矩形框类别（类别不存在的）
	const needCreateBbox = new Set<string>()
	// 记录每个类别名的关键点数量（用于自动创建关键点类别）- 包括已存在类别和新类别
	const categoryKeypointCount = new Map<string, number>()
	
	for (const ann of allInferenceAnns) {
		const catName = (ann as Annotation & { _categoryName?: string })._categoryName
		if (!catName) continue
		
		// 检查是否需要创建 bbox 类别
		if (ann.categoryId < 0 && !categoryNameToId.has(catName)) {
			needCreateBbox.add(catName)
		}
		
		// 记录关键点数量（无论类别是否已存在）
		if (ann.type === 'bbox') {
			const bboxData = ann.data as { keypoints?: [number, number, number][] }
			if (bboxData.keypoints?.length) {
				const existing = categoryKeypointCount.get(catName) || 0
				categoryKeypointCount.set(catName, Math.max(existing, bboxData.keypoints.length))
			}
		}
	}
	
	// 第一步：为带关键点的新类别先创建关键点类别
	const newKpCategoryIds = new Map<string, number>() // bboxName -> keypointCategoryId
	for (const [bboxName, kpCount] of categoryKeypointCount) {
		if (kpCount <= 0) continue
		
		// 检查该 bbox 类别是否已存在并已绑定关键点类别
		const existingBboxCat = projectCategories.value.find(c => c.name === bboxName && c.type === 'bbox')
		if (existingBboxCat) {
			const bboxMate = parseBboxMate(existingBboxCat.mate)
			if (bboxMate?.keypointCategoryId) continue // 已绑定，跳过
		}
		
		// 创建关键点类别
		const kpCatName = `${bboxName}_keypoints`
		if (!categoryNameToId.has(kpCatName)) {
			// 自动生成关键点语义（按 1,2,3... 命名）
			const keypoints = Array.from({ length: kpCount }, (_, i) => ({ id: i, name: `${i + 1}` }))
			const kpMate = JSON.stringify({ keypoints })
			
			try {
				const res = await fetch('http://localhost:18080/api/project-categories', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						projectId: currentProject.value.id,
						name: kpCatName,
						type: 'keypoint',
						color: '#10b981',
						mate: kpMate
					})
				})
				if (res.ok) {
					const data = await res.json() as { id?: number }
					if (data.id) {
						categoryNameToId.set(kpCatName, data.id)
						newKpCategoryIds.set(bboxName, data.id)
					}
				}
			} catch (e) {
				console.error('Failed to create keypoint category:', e)
			}
		}
	}
	
	// 第二步：创建矩形框类别（并在 mate 中绑定关键点类别）
	for (const name of needCreateBbox) {
		try {
			// 获取需要绑定的关键点类别 ID
			const keypointCategoryId = newKpCategoryIds.get(name)
			const mate = keypointCategoryId ? JSON.stringify({ keypointCategoryId }) : undefined
			
			const res = await fetch('http://localhost:18080/api/project-categories', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					projectId: currentProject.value.id,
					name,
					type: 'bbox',
					color: '#3b82f6',
					mate
				})
			})
			if (res.ok) {
				const data = await res.json() as { id?: number }
				if (data.id) {
					categoryNameToId.set(name, data.id)
				}
			}
		} catch (e) {
			console.error('Failed to create bbox category:', e)
		}
	}
	
	// 第三步：为已存在但未绑定关键点类别的 bbox 类别绑定关键点类别
	for (const [bboxName, kpCatId] of newKpCategoryIds) {
		// 检查是否是已存在的类别（不在 needCreateBbox 中）
		if (!needCreateBbox.has(bboxName)) {
			const existingBboxCat = projectCategories.value.find(c => c.name === bboxName && c.type === 'bbox')
			if (existingBboxCat) {
				try {
					const newBboxMate = JSON.stringify({ keypointCategoryId: kpCatId })
					await fetch('http://localhost:18080/api/project-categories', {
						method: 'PUT',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							projectId: currentProject.value.id,
							id: existingBboxCat.id,
							mate: newBboxMate
						})
					})
				} catch (e) {
					console.error('Failed to bind keypoint category to bbox:', e)
				}
			}
		}
	}
	
	if (needCreateBbox.size > 0 || newKpCategoryIds.size > 0) {
		await loadProjectCategories()
	}
	
	return categoryNameToId
}

const handleSaveAnnotations = async () => {
	if (!currentProject.value || !activeImage.value) return
	try {
		// 为推理标注创建不存在的类别
		const categoryNameToId = await ensureInferenceCategoriesExist()
		
		const annotations = currentAnnotations.value.map(a => {
			let categoryId = a.categoryId
			// 如果是推理标注且类别ID无效，尝试从映射中获取
			if (a.isInference && categoryId < 0) {
				const catName = (a as Annotation & { _categoryName?: string })._categoryName
				if (catName) {
					categoryId = categoryNameToId.get(catName) || categoryId
				}
			}
			return {
				id: a.id,
				categoryId,
				type: a.type,
				data: JSON.stringify(a.data)
			}
		}).filter(a => a.categoryId > 0) // 过滤掉无效类别的标注
		
		const res = await fetch('http://localhost:18080/api/project-annotations/save', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				imageId: activeImage.value.id,
				isNegative: false,
				annotations
			})
		})
		if (res.ok) {
			const data = await res.json() as { status?: string }
			updateImageAnnotationStatus(activeImage.value.id, data.status as 'none' | 'annotated' | 'negative' || 'none')
			// 重新加载标注以获取服务端生成的ID（会清除推理标记）
			await loadAnnotations(activeImage.value.id)
			notifyInfo(t('annotation.save.success'))
		} else {
			notifyError(t('annotation.save.error'))
		}
	} catch (e) {
		console.error('save annotations error:', e)
		notifyError(t('annotation.save.error'))
	}
}

const handleSaveAsNegative = async () => {
	if (!currentProject.value || !activeImage.value) return
	try {
		// 保存为负样本：清空标注并标记为负样本
		const res = await fetch('http://localhost:18080/api/project-annotations/save', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				imageId: activeImage.value.id,
				isNegative: true,
				annotations: []
			})
		})
		if (res.ok) {
			currentAnnotations.value = []
			// 更新本地图片状态
			updateImageAnnotationStatus(activeImage.value.id, 'negative')
			notifyInfo(t('annotation.saveAsNegative.success'))
		} else {
			notifyError(t('annotation.saveAsNegative.error'))
		}
	} catch (e) {
		console.error('save as negative error:', e)
		notifyError(t('annotation.saveAsNegative.error'))
	}
}

// 更新本地图片标注状态
const updateImageAnnotationStatus = (imageId: number, status: 'none' | 'annotated' | 'negative') => {
	const img = projectImages.value.find(i => i.id === imageId)
	if (img) {
		const oldStatus = img.annotationStatus
		img.annotationStatus = status
		// 更新统计
		if (oldStatus === 'annotated') projectStats.value.annotatedCount--
		else if (oldStatus === 'negative') projectStats.value.negativeCount--
		else projectStats.value.unannotatedCount--
		if (status === 'annotated') projectStats.value.annotatedCount++
		else if (status === 'negative') projectStats.value.negativeCount++
		else projectStats.value.unannotatedCount++
	}
}

// 切换图片（可选是否保存当前标注）
const handlePrevImage = async (autoSave = true) => {
	if (!activeImage.value) return
	// 等待自动保存完成，避免数据库锁定
	if (autoSavePromise) await autoSavePromise
	// 通过按钮切换时保存，快捷键切换时不保存（避免保存推理结果）
	if (autoSave) {
		await handleSaveAnnotations()
	}
	const idx = projectImages.value.findIndex(img => img.id === activeImage.value?.id)
	if (idx > 0) {
		const prevImg = projectImages.value[idx - 1]
		if (prevImg) {
			activeImage.value = prevImg
			await loadAnnotations(prevImg.id)
		}
	}
}

const handleNextImage = async (autoSave = true) => {
	if (!activeImage.value) return
	// 等待自动保存完成，避免数据库锁定
	if (autoSavePromise) await autoSavePromise
	// 通过按钮切换时保存，快捷键切换时不保存（避免保存推理结果）
	if (autoSave) {
		await handleSaveAnnotations()
	}
	const idx = projectImages.value.findIndex(img => img.id === activeImage.value?.id)
	if (idx < projectImages.value.length - 1) {
		const nextImg = projectImages.value[idx + 1]
		if (nextImg) {
			activeImage.value = nextImg
			await loadAnnotations(nextImg.id)
		}
	}
}

// 切换到上一张未标注的图片（快捷键调用，不自动保存）
const handlePrevUnannotated = async () => {
	if (!activeImage.value) return
	if (autoSavePromise) await autoSavePromise
	const currentIdx = projectImages.value.findIndex(img => img.id === activeImage.value?.id)
	// 从当前位置向前查找未标注的图片
	for (let i = currentIdx - 1; i >= 0; i--) {
		const img = projectImages.value[i]
		if (img && img.annotationStatus === 'none') {
			activeImage.value = img
			await loadAnnotations(img.id)
			return
		}
	}
	notifyInfo(t('shortcut.noMoreUnannotated'))
}

// 切换到下一张未标注的图片（快捷键调用，不自动保存）
const handleNextUnannotated = async () => {
	if (!activeImage.value) return
	if (autoSavePromise) await autoSavePromise
	const currentIdx = projectImages.value.findIndex(img => img.id === activeImage.value?.id)
	// 从当前位置向后查找未标注的图片
	for (let i = currentIdx + 1; i < projectImages.value.length; i++) {
		const img = projectImages.value[i]
		if (img && img.annotationStatus === 'none') {
			activeImage.value = img
			await loadAnnotations(img.id)
			return
		}
	}
	notifyInfo(t('shortcut.noMoreUnannotated'))
}

// 重置标注画布视图
const handleResetView = () => {
	annotationCanvasRef.value?.resetView()
}

// 删除选中的标注
const handleDeleteSelected = () => {
	annotationCanvasRef.value?.deleteSelectedAnnotation()
}

const handleSelectCategory = (cat: ProjectCategory) => {
	// 允许同时选中 bbox/polygon 类别和 keypoint 类别
	if (cat.type === 'keypoint') {
		// 点击关键点类别：切换 keypoint 选中
		if (selectedKeypointCategory.value?.id === cat.id) {
			selectedKeypointCategory.value = null // 取消选中
		} else {
			selectedKeypointCategory.value = cat
		}
	} else {
		// 点击 bbox/polygon 类别
		if (selectedCategory.value?.id === cat.id) {
			selectedCategory.value = null // 取消选中
		} else {
			selectedCategory.value = cat
			// 选中 polygon 时，清除 keypoint 选中（polygon 不支持关键点）
			if (cat.type === 'polygon') {
				selectedKeypointCategory.value = null
			}
		}
	}
}

const handleAnnotationNotify = (payload: { type: 'info' | 'warning' | 'error'; message: string }) => {
	if (payload.type === 'warning') {
		notifyInfo(payload.message) // 使用 info 样式显示警告
	} else if (payload.type === 'error') {
		notifyError(payload.message)
	} else {
		notifyInfo(payload.message)
	}
}

// 自动保存：当用户完成标注操作时触发（不包括推理结果）
const handleAnnotationComplete = () => {
	// ★ Box 模式推理：如果当前是 box 模式插件，用最后创建的 bbox 作为 visual prompt
	if (activePlugin.value?.interactionMode === 'box') {
		const userBboxes = currentAnnotations.value.filter(a => !a.isInference && a.type === 'bbox')
		const lastBbox = userBboxes[userBboxes.length - 1]
		if (lastBbox) {
			const boxData = lastBbox.data as { x: number; y: number; width: number; height: number }
			console.log('[App] Box mode: using last bbox as visual prompt:', boxData)
			// 删除这个临时的 bbox，然后发起推理
			currentAnnotations.value = currentAnnotations.value.filter(a => a.id !== lastBbox.id)
			runBoxInference(boxData)
		}
		return  // Box 模式不触发自动保存
	}
	
	if (!isAutoSaveEnabled.value) return
	if (!currentProject.value || !activeImage.value) return
	
	// 过滤掉推理结果，只保存用户创建的标注
	const userAnnotations = currentAnnotations.value.filter(a => !a.isInference)
	if (userAnnotations.length === 0) return
	
	const projectId = currentProject.value.id
	const imageId = activeImage.value.id
	const annotations = userAnnotations.map(a => ({
		id: a.id,
		categoryId: a.categoryId,
		type: a.type,
		data: JSON.stringify(a.data)
	})).filter(a => a.categoryId > 0)
	
	// 使用 Promise 追踪保存状态，供切换图片时等待
	autoSavePromise = (async () => {
		try {
			await fetch('http://localhost:18080/api/project-annotations/save', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ projectId, imageId, isNegative: false, annotations })
			})
			// 自动保存不显示通知，静默保存
			updateImageAnnotationStatus(imageId, annotations.length > 0 ? 'annotated' : 'none')
		} catch (e) {
			console.error('auto save error:', e)
		} finally {
			autoSavePromise = null
		}
	})()
}

const isImageSelected = (id: number) => selectedImageIds.value.has(id)

const registerImageCell = (el: HTMLElement | null, id: number) => {
	const map = imageCellRefs.value
	if (el) {
		map.set(id, el)
	} else {
		map.delete(id)
	}
}

// 滚动图片列表使指定图片可见
const scrollToImage = (imageId: number) => {
	const container = imageListRef.value
	if (!container) return
	
	// 先等待 DOM 更新（虚拟滚动可能需要渲染）
	nextTick(() => {
		const cell = imageCellRefs.value.get(imageId)
		if (!cell) {
			// 图片不在可视范围，需要先滚动到大概位置
			const idx = filteredProjectImages.value.findIndex(img => img.id === imageId)
			if (idx < 0) return
			
			// 计算目标行和滚动位置
			const imagesPerRow = isImageListExpanded.value ? 4 : 2
			const rowIndex = Math.floor(idx / imagesPerRow)
			const targetScrollTop = rowIndex * imageRowHeight.value - container.clientHeight / 2 + imageRowHeight.value / 2
			
			container.scrollTop = Math.max(0, targetScrollTop)
			
			// 滚动后再次尝试
			setTimeout(() => {
				const cellAfterScroll = imageCellRefs.value.get(imageId)
				if (cellAfterScroll) {
					scrollElementIntoView(container, cellAfterScroll)
				}
			}, 50)
		} else {
			scrollElementIntoView(container, cell)
		}
	})
}

// 确保元素在容器可视范围内
const scrollElementIntoView = (container: HTMLElement, element: HTMLElement) => {
	const containerRect = container.getBoundingClientRect()
	const elementRect = element.getBoundingClientRect()
	
	// 计算元素相对于容器的位置
	const elementTop = elementRect.top - containerRect.top + container.scrollTop
	const elementBottom = elementTop + elementRect.height
	
	const viewTop = container.scrollTop
	const viewBottom = viewTop + container.clientHeight
	
	// 如果元素完全在可视范围内，不滚动
	if (elementTop >= viewTop && elementBottom <= viewBottom) {
		return
	}
	
	// 元素在上方，滚动到顶部对齐（留一点边距）
	if (elementTop < viewTop) {
		container.scrollTop = elementTop - 8
	}
	// 元素在下方，滚动到底部对齐（留一点边距）
	else if (elementBottom > viewBottom) {
		container.scrollTop = elementBottom - container.clientHeight + 8
	}
}

const openImageContextMenu = (event: MouseEvent, item: ProjectImageListItem) => {
	const current = selectedImageIds.value
	const wasSelected = current.has(item.id)
	if (!wasSelected) {
		const next = new Set<number>([item.id])
		selectedImageIds.value = next
		imageContextMenuUseSelection.value = false
		imageContextMenuSelectionIds.value = [item.id]
	} else {
		imageContextMenuUseSelection.value = true
		imageContextMenuSelectionIds.value = Array.from(current)
	}
	imageContextMenuImageId.value = item.id
	imageContextMenuX.value = event.clientX
	imageContextMenuY.value = event.clientY
	isImageContextMenuOpen.value = true
}

const closeImageContextMenu = () => {
	isImageContextMenuOpen.value = false
}

const openDeleteImagesDialogFromContextMenu = () => {
	const ids = imageContextMenuSelectionIds.value.slice()
	if (!ids.length) {
		closeImageContextMenu()
		return
	}
	deleteTargetImageIds.value = ids
	closeImageContextMenu()
	isDeleteImagesModalOpen.value = true
}

const closeDeleteImagesDialog = () => {
	isDeleteImagesModalOpen.value = false
}

	type DeleteImageResult = {
		id: number
		deleted: boolean
		fileDeleted: boolean
		isExternal: boolean
		error?: string
	}

const confirmDeleteImages = async () => {
	if (!currentProject.value || deleteTargetImageIds.value.length === 0) {
		isDeleteImagesModalOpen.value = false
		return
	}
	const ids = deleteTargetImageIds.value.slice()
	isDeleteImagesModalOpen.value = false
	try {
		const res = await fetch('http://localhost:18080/api/project-images/delete', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				projectId: currentProject.value.id,
				imageIds: ids
			})
		})
		
		// Check for conflict (another I/O task is running)
		if (res.status === 409) {
			notifyError(t('project.ioTaskBusy'))
			return
		}
		
		if (!res.ok) {
			throw new Error('network')
		}
		
		const data = await res.json()
		
		// New async task mode
		if (data.taskId) {
			pollDeleteImagesTask(data.taskId, ids)
			return
		}
		
		// Legacy sync mode fallback
		const results = Array.isArray(data.results) ? data.results : []
		const deletedCount = results.filter((r: DeleteImageResult) => r.deleted).length
		if (deletedCount > 0) {
			notifyInfo(t('project.images.deleteSuccess', { count: deletedCount }))
			selectedImageIds.value = new Set()
			await loadProjectImages()
		} else {
			notifyError(t('project.images.deleteError'))
		}
	} catch {
		notifyError(t('project.images.deleteError'))
	}
}

const pollDeleteImagesTask = (taskId: string, deletedImageIds: number[]) => {
	const notificationId = notifyInfo(
		t('project.deleteProgress.deleting', { progress: 0 }),
		{ persistent: true }
	)

	const poll = async () => {
		try {
			const res = await fetch(`http://localhost:18080/api/import-tasks?id=${taskId}`)
			if (!res.ok) {
				updateNotification(notificationId, {
					type: 'error',
					message: t('project.images.deleteError'),
					persistent: false
				})
				return
			}
			const { phase, progress, imported, total } = await res.json()

			if (phase === 'deleting') {
				updateNotification(notificationId, {
					type: 'info',
					message: t('project.deleteProgress.deleting', { progress }),
					persistent: true
				})
				setTimeout(poll, 300)
			} else if (phase === 'completed') {
				updateNotification(notificationId, {
					type: 'success',
					message: t('project.deleteProgress.completed', { count: imported || total }),
					persistent: false
				})
				
				// Clear selection
				selectedImageIds.value = new Set()
				
				// Check if active image was deleted
				if (activeImage.value && deletedImageIds.includes(activeImage.value.id)) {
					activeImage.value = null
					currentAnnotations.value = []
				}
				
				// Refresh image list
				await loadProjectImages()
			} else if (phase === 'failed') {
				updateNotification(notificationId, {
					type: 'error',
					message: t('project.images.deleteError'),
					persistent: false
				})
			} else {
				setTimeout(poll, 300)
			}
		} catch {
			updateNotification(notificationId, {
				type: 'error',
				message: t('project.images.deleteError'),
				persistent: false
			})
		}
	}

	poll()
}

const handleImageCellClick = (item: ProjectImageListItem, event: MouseEvent) => {
	const images = projectImages.value
	const idx = images.findIndex((img) => img.id === item.id)
	if (idx === -1) return

	if (event.shiftKey) {
		// Shift + 点击：在当前选中集合中进行“添加/移除”
		const current = new Set(selectedImageIds.value)
		if (current.has(item.id)) {
			current.delete(item.id)
		} else {
			current.add(item.id)
		}
		selectedImageIds.value = current
		lastSelectedIndex.value = idx
		return
	}

	// 普通点击图片：打开该图片，并更新锚点
	lastSelectedIndex.value = idx
	selectedImageIds.value = new Set()
	activeImage.value = item
}

const loadProjectImages = async () => {
  if (!currentProject.value) {
    projectImages.value = []
    projectStats.value = { total: 0, annotatedCount: 0, unannotatedCount: 0, negativeCount: 0, totalAnnotations: 0 }
    return
  }
  try {
    const res = await fetch(
      `http://localhost:18080/api/project-images?projectId=${encodeURIComponent(currentProject.value.id)}`
    )
    if (!res.ok) {
      projectImages.value = []
      return
    }
    const data = (await res.json()) as { 
      items?: ProjectImageListItem[]
      total?: number
      annotatedCount?: number
      unannotatedCount?: number
      negativeCount?: number
    }
    projectImages.value = Array.isArray(data.items) ? data.items : []
    // 更新统计
    projectStats.value = {
      total: data.total || 0,
      annotatedCount: data.annotatedCount || 0,
      unannotatedCount: data.unannotatedCount || 0,
      negativeCount: data.negativeCount || 0,
      totalAnnotations: 0 // 稍后可添加统计总标注数的API
    }
    // 延迟刷新 viewport，确保 DOM 已完全渲染
    await nextTick()
    refreshImageListViewport()
    // 再延迟一次，处理首次加载时 DOM 可能还没完全准备好的情况
    setTimeout(() => {
      refreshImageListViewport()
    }, 100)
  } catch {
    projectImages.value = []
  }
}

const loadProjectCategories = async () => {
	if (!currentProject.value) {
		projectCategories.value = []
		return
	}
	try {
		const res = await fetch(
			`http://127.0.0.1:18080/api/project-categories?projectId=${encodeURIComponent(currentProject.value.id)}`
		)
		if (!res.ok) {
			projectCategories.value = []
			return
		}
		const data = (await res.json()) as { items?: ProjectCategory[] }
		projectCategories.value = Array.isArray(data.items) ? data.items : []
	} catch {
		projectCategories.value = []
	}
}

const pollImportTask = (taskId: string) => {
  const notificationId = notifyInfo(
    t('project.importProgress.scanning'),
    { persistent: true }
  )

  isImportingImages.value = true

  const poll = async () => {
    try {
      const res = await fetch(
        `http://localhost:18080/api/import-tasks?id=${encodeURIComponent(taskId)}`
      )
      if (!res.ok) {
        updateNotification(notificationId, {
          type: 'error',
          message: t('project.importProgress.failed'),
          persistent: false
        })
        isImportingImages.value = false
        return
      }
      const status = (await res.json()) as ImportTaskStatus
      const { phase, progress, imported, total: serverTotal } = status

      let message = ''
      if (phase === 'scanning') {
        message = t('project.importProgress.scanning')
      } else if (phase === 'copying') {
        message = t('project.importProgress.copying', {
          imported,
          total: serverTotal,
          progress
        })
      } else if (phase === 'indexing') {
        message = t('project.importProgress.indexing', {
          imported,
          total: serverTotal,
          progress
        })
      } else if (phase === 'completed') {
        message = t('project.importProgress.completed', {
          count: serverTotal
        })
        updateNotification(notificationId, {
          type: 'success',
          message,
          persistent: false
        })
        isImportingImages.value = false
        void loadProjectImages()
        return
      } else if (phase === 'failed') {
        updateNotification(notificationId, {
          type: 'error',
          message: t('project.importProgress.failed'),
          persistent: false
        })
        isImportingImages.value = false
        return
      }

      updateNotification(notificationId, {
        type: 'info',
        message,
        persistent: true
      })
      setTimeout(poll, 600)
    } catch {
      updateNotification(notificationId, {
        type: 'error',
        message: t('project.importProgress.failed'),
        persistent: false
      })
      isImportingImages.value = false
    }
  }

  void poll()
}

const startImportImages = async (mode: 'directory' | 'files', paths: string[]) => {
  if (!currentProject.value) return

  importImagesError.value = ''

  try {
    const response = await fetch('http://localhost:18080/api/import-images', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        projectId: currentProject.value.id,
        mode,
        importMode: importMode.value,
        paths
      })
    })

    if (!response.ok) {
      let errorKey = 'project.importImagesModal.errorGeneric'
      if (response.status === 409) {
        // Another I/O task is running
        errorKey = 'project.ioTaskBusy'
      } else if (response.status === 400) {
        try {
          const data = await response.json()
          if (data && typeof data.error === 'string') {
            if (data.error === 'directory_invalid') {
              errorKey = 'project.importImagesModal.errorDirectoryInvalid'
            } else if (data.error === 'no_images') {
              errorKey = 'project.importImagesModal.errorNoImages'
            } else if (data.error === 'import_mode_invalid') {
              errorKey = 'project.importImagesModal.errorImportModeInvalid'
            }
          }
        } catch {
          // ignore parse error
        }
      }
      importImagesError.value = t(errorKey)
      notifyError(t(errorKey))
      return
    }

    const data = (await response.json()) as { taskId: string; total: number }
    isImportImagesModalOpen.value = false
    pollImportTask(data.taskId)
  } catch {
    const msg = t('project.importImagesModal.errorNetwork')
    importImagesError.value = msg
    notifyError(msg)
  }
}

const isDriveRootPath = (p: string) => {
  const trimmed = p.trim()
  if (!trimmed) return false
  // Windows 盘符根目录，例如 C:\ 或 D:\
  if (/^[A-Za-z]:\\?$/.test(trimmed)) return true
  // POSIX 根目录
  if (trimmed === '/') return true
  return false
}

const handleImportFromDirectory = async () => {
  const anyWindow = window as any
  const api = anyWindow?.electronAPI
  if (!api || typeof api.selectDirectory !== 'function') {
    const msg = t('project.importImagesModal.errorGeneric')
    importImagesError.value = msg
    notifyError(msg)
    return
  }

  try {
    const selected = (await api.selectDirectory(undefined)) as string | null | undefined
    if (!selected) return
    if (isDriveRootPath(String(selected))) {
      const msg = t('project.importImagesModal.errorRootDirectory')
      importImagesError.value = msg
      notifyError(msg)
      return
    }
    await startImportImages('directory', [String(selected)])
  } catch {
    const msg = t('project.importImagesModal.errorNetwork')
    importImagesError.value = msg
    notifyError(msg)
  }
}

const handleImportFromFiles = async () => {
  const anyWindow = window as any
  const api = anyWindow?.electronAPI
  if (!api || typeof api.selectFiles !== 'function') {
    const msg = t('project.importImagesModal.errorGeneric')
    importImagesError.value = msg
    notifyError(msg)
    return
  }

  try {
    const selected = (await api.selectFiles({})) as string[] | null | undefined
    if (!selected || selected.length === 0) return
    await startImportImages('files', selected.map(String))
  } catch {
    const msg = t('project.importImagesModal.errorNetwork')
    importImagesError.value = msg
    notifyError(msg)
  }
}

const closeImportImagesModal = () => {
  if (isImportingImages.value) return
  isImportImagesModalOpen.value = false
}

const handleMinimize = () => invokeWindowApi('minimize')
const handleToggleMaximize = () => invokeWindowApi('toggleMaximize')
const handleClose = () => invokeWindowApi('close')

const handleOpenSettings = () => {
  isSettingsOpen.value = true
}

const handleCloseSettings = () => {
  isSettingsOpen.value = false
}

const handleImportImagesClick = () => {
  if (!currentProject.value) return
  isProjectMenuOpen.value = false
  importImagesError.value = ''
  importMode.value = 'copy'
  isImportImagesModalOpen.value = true
}

// ==================== 导入数据集 ====================

const handleImportDatasetClick = () => {
  if (!currentProject.value) return
  isProjectMenuOpen.value = false
  importDatasetStep.value = 'select'
  importDatasetError.value = ''
  importDatasetPath.value = ''
  detectedPlugins.value = []
  selectedPluginId.value = ''
  importDatasetStats.value = null
  importDatasetMode.value = 'copy'
  isImportDatasetModalOpen.value = true
}

const handleSelectDatasetPath = async () => {
  const anyWindow = window as Window & { electronAPI?: { selectDirectory?: (defaultPath?: string) => Promise<string | null> } }
  const api = anyWindow?.electronAPI
  if (!api || typeof api.selectDirectory !== 'function') {
    importDatasetError.value = t('project.importDataset.errorGeneric')
    return
  }
  try {
    const selected = await api.selectDirectory()
    if (!selected) return
    importDatasetPath.value = selected
    // 自动开始探测
    await detectDatasetFormat()
  } catch {
    importDatasetError.value = t('project.importDataset.errorGeneric')
  }
}

const detectDatasetFormat = async () => {
  if (!importDatasetPath.value) return
  importDatasetStep.value = 'detecting'
  importDatasetError.value = ''
  
  try {
    const res = await fetch('http://localhost:18080/api/dataset/detect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rootPath: importDatasetPath.value })
    })
    const data = await res.json()
    
    if (res.ok && data.results && data.results.length > 0) {
      detectedPlugins.value = data.results
      selectedPluginId.value = data.results[0].pluginId
      importDatasetStep.value = 'configure'
    } else {
      importDatasetError.value = t('project.importDataset.noPluginDetected')
      importDatasetStep.value = 'select'
    }
  } catch {
    importDatasetError.value = t('project.importDataset.errorNetwork')
    importDatasetStep.value = 'select'
  }
}

const executeDatasetImport = async () => {
  if (!currentProject.value || !selectedPluginId.value || !importDatasetPath.value) return
  
  // 立即关闭对话框
  isImportDatasetModalOpen.value = false
  
  try {
    const res = await fetch('http://localhost:18080/api/dataset/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        projectId: currentProject.value.id,
        pluginId: selectedPluginId.value,
        rootPath: importDatasetPath.value,
        importMode: importDatasetMode.value,
        params: {}
      })
    })
    
    // Check for conflict (another I/O task is running)
    if (res.status === 409) {
      notifyError(t('project.ioTaskBusy'))
      return
    }
    
    const data = await res.json()
    
    if (res.ok && data.taskId) {
      // 开始轮询任务状态
      pollDatasetImportTask(data.taskId)
    } else {
      notifyError(t('project.importDataset.errorImport'))
    }
  } catch {
    notifyError(t('project.importDataset.errorNetwork'))
  }
}

const pollDatasetImportTask = (taskId: string) => {
  const notificationId = notifyInfo(
    t('project.importProgress.scanning'),
    { persistent: true }
  )

  const poll = async () => {
    try {
      const res = await fetch(`http://localhost:18080/api/import-tasks?id=${taskId}`)
      if (!res.ok) {
        updateNotification(notificationId, {
          type: 'error',
          message: t('project.importDataset.errorImport'),
          persistent: false
        })
        return
      }
      const { phase, progress, imported, total } = await res.json()

      let message = ''
      if (phase === 'scanning') {
        message = t('project.importProgress.scanning')
      } else if (phase === 'copying') {
        message = t('project.importProgress.copying', {
          imported,
          total,
          progress
        })
      } else if (phase === 'indexing') {
        message = t('project.importProgress.indexing', {
          imported,
          total,
          progress
        })
      } else if (phase === 'completed') {
        message = t('project.importProgress.completed', {
          count: total
        })
        updateNotification(notificationId, {
          type: 'success',
          message,
          persistent: false
        })
        // 刷新数据
        await loadProjectImages()
        await loadProjectCategories()
        if (activeImage.value) {
          await loadAnnotations(activeImage.value.id)
        }
        return
      } else if (phase === 'failed') {
        updateNotification(notificationId, {
          type: 'error',
          message: t('project.importDataset.errorImport'),
          persistent: false
        })
        return
      }

      updateNotification(notificationId, {
        type: 'info',
        message,
        persistent: true
      })
      setTimeout(poll, 500)
    } catch {
      updateNotification(notificationId, {
        type: 'error',
        message: t('project.importDataset.errorNetwork'),
        persistent: false
      })
    }
  }

  poll()
}

const closeImportDatasetModal = () => {
  isImportDatasetModalOpen.value = false
}

const loadProjects = async () => {
  try {
    const response = await fetch('http://localhost:18080/api/projects')
    if (!response.ok) {
      return
    }
    const data = await response.json()
    projects.value = Array.isArray(data) ? data : []
    // 加载完成后尝试恢复上次打开的项目
    tryRestoreLastProject()
  } catch {
  }
}

watch(currentProject, () => {
  projectImages.value = []
  projectCategories.value = []
  if (!currentProject.value) return
  void loadProjectImages()
  void loadProjectCategories()
})

const toggleProjectMenu = () => {
  isProjectMenuOpen.value = !isProjectMenuOpen.value
}

const handleNewProjectClick = () => {
  isProjectMenuOpen.value = false
  newProjectName.value = ''
  createProjectError.value = ''
  isCreateProjectModalOpen.value = true
}

const handleProjectClick = (project: ProjectSummary) => {
  // 切换项目时清除推理结果和选中的类别（避免跨项目使用错误类别）
  currentAnnotations.value = currentAnnotations.value.filter(a => !a.isInference)
  selectedCategory.value = null
  selectedKeypointCategory.value = null
  currentProject.value = project
  isProjectMenuOpen.value = false
  if (route.path !== '/project') {
    router.push('/project')
  }
}

// 项目右键菜单
const handleProjectContextMenu = (event: MouseEvent, project: ProjectSummary) => {
  event.preventDefault()
  event.stopPropagation()
  projectContextMenuTarget.value = project
  projectContextMenuX.value = event.clientX
  projectContextMenuY.value = event.clientY
  isProjectContextMenuOpen.value = true
}

const closeProjectContextMenu = () => {
  isProjectContextMenuOpen.value = false
}

// 打开删除项目确认弹窗
const openDeleteProjectModal = () => {
  if (!projectContextMenuTarget.value) return
  isProjectContextMenuOpen.value = false
  isDeleteProjectModalOpen.value = true
}

const closeDeleteProjectModal = () => {
  if (isDeletingProject.value) return
  isDeleteProjectModalOpen.value = false
}

// 确认删除项目
const submitDeleteProject = async () => {
  if (!projectContextMenuTarget.value || isDeletingProject.value) return
  
  isDeletingProject.value = true
  try {
    const res = await fetch(`http://localhost:18080/api/projects?id=${projectContextMenuTarget.value.id}`, {
      method: 'DELETE'
    })
    if (!res.ok) {
      const err = await res.json()
      notifyError(t('project.deleteModal.error', { msg: err.error || 'unknown' }))
      return
    }
    // 如果删除的是当前项目，清除当前项目
    if (currentProject.value?.id === projectContextMenuTarget.value.id) {
      currentProject.value = null
      projectImages.value = []
      projectCategories.value = []
      activeImage.value = null
    }
    // 刷新项目列表
    await loadProjects()
    notifySuccess(t('project.deleteModal.success'))
    isDeleteProjectModalOpen.value = false
  } catch (e) {
    notifyError(t('project.deleteModal.error', { msg: String(e) }))
  } finally {
    isDeletingProject.value = false
  }
}

// 打开重命名项目弹窗
const openRenameProjectModal = () => {
  if (!projectContextMenuTarget.value) return
  isProjectContextMenuOpen.value = false
  renameProjectName.value = projectContextMenuTarget.value.name
  renameProjectError.value = ''
  isRenameProjectModalOpen.value = true
}

const closeRenameProjectModal = () => {
  if (isRenamingProject.value) return
  isRenameProjectModalOpen.value = false
}

// 确认重命名项目
const submitRenameProject = async () => {
  if (!projectContextMenuTarget.value || isRenamingProject.value) return
  
  const name = renameProjectName.value.trim()
  if (!name) {
    renameProjectError.value = t('project.createModal.nameRequired')
    return
  }
  if (/[<>:"/\\|?*]/.test(name)) {
    renameProjectError.value = t('project.createModal.nameInvalid')
    return
  }
  if (name === projectContextMenuTarget.value.name) {
    isRenameProjectModalOpen.value = false
    return
  }
  
  isRenamingProject.value = true
  renameProjectError.value = ''
  
  try {
    const res = await fetch('http://localhost:18080/api/projects', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: projectContextMenuTarget.value.id,
        name
      })
    })
    if (!res.ok) {
      const err = await res.json()
      renameProjectError.value = err.error || 'unknown'
      return
    }
    // 更新项目列表和当前项目
    if (currentProject.value?.id === projectContextMenuTarget.value.id) {
      currentProject.value = { ...currentProject.value, name }
    }
    await loadProjects()
    notifySuccess(t('project.renameModal.success'))
    isRenameProjectModalOpen.value = false
  } catch (e) {
    renameProjectError.value = String(e)
  } finally {
    isRenamingProject.value = false
  }
}

const closeCreateProjectModal = () => {
  if (isCreatingProject.value) return
  isCreateProjectModalOpen.value = false
}

const submitCreateProject = async () => {
  if (isCreatingProject.value) return

  const name = newProjectName.value.trim()
  if (!name) {
    createProjectError.value = t('project.createModal.nameRequired')
    return
  }
  if (/[<>:"/\\|?*]/.test(name)) {
    createProjectError.value = t('project.createModal.nameInvalid')
    return
  }

  isCreatingProject.value = true
  createProjectError.value = ''

  try {
    const response = await fetch('http://localhost:18080/api/projects', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ name })
    })

    if (!response.ok) {
      let errorKey = 'project.createModal.errorGeneric'
      if (response.status === 409) {
        errorKey = 'project.createModal.errorExists'
      }
      createProjectError.value = t(errorKey)
      return
    }

    const created = (await response.json()) as ProjectSummary
    projects.value = [...projects.value, created]
    currentProject.value = created
    isCreateProjectModalOpen.value = false
    newProjectName.value = ''
  } catch {
    createProjectError.value = t('project.createModal.errorNetwork')
  } finally {
    isCreatingProject.value = false
  }
}

type SidebarSection = 'project' | 'dataset' | 'training' | 'plugins' | 'python-env' | 'help'

const handleSidebarClick = (section: SidebarSection) => {
  handleCloseSettings()
  const pathMap: Record<SidebarSection, string> = {
    project: '/project',
    dataset: '/dataset',
    training: '/training',
    plugins: '/plugins',
    'python-env': '/python-env',
    help: '/help'
  }
  router.push(pathMap[section])
}

const handleNavigateToUi = () => {
  handleCloseSettings()
  router.push('/ui')
}

const handleOpenGitHub = () => {
  window.electronAPI?.openExternal?.('https://github.com/aaaazbwzbw/EasyMark')
}

// 版本更新检查
const hasNewVersion = ref(false)
const latestVersion = ref('')
const currentVersion = ref('')

const checkForUpdates = async () => {
  try {
    // 获取本地版本
    const localVersion = await window.electronAPI?.getAppVersion?.()
    if (!localVersion) return
    currentVersion.value = localVersion
    
    // 从 GitHub API 获取最新版本
    const response = await fetch('https://api.github.com/repos/aaaazbwzbw/EasyMark/releases/latest', {
      headers: { 'Accept': 'application/vnd.github.v3+json' }
    })
    if (!response.ok) return
    
    const data = await response.json()
    const remoteVersion = data.tag_name?.replace(/^v/, '') || ''
    if (!remoteVersion) return
    
    latestVersion.value = remoteVersion
    
    // 简单版本比较（假设格式为 x.y.z）
    const local = localVersion.split('.').map(Number)
    const remote = remoteVersion.split('.').map(Number)
    
    for (let i = 0; i < 3; i++) {
      if ((remote[i] || 0) > (local[i] || 0)) {
        hasNewVersion.value = true
        return
      } else if ((remote[i] || 0) < (local[i] || 0)) {
        return
      }
    }
  } catch (e) {
    console.log('[Version] Check failed:', e)
  }
}

// 使用通知系统
const {
  notifications,
  isNotificationPanelOpen,
  hasPersistentNotifications,
  toggleNotificationPanel,
  clearAllNotifications,
  success: notifySuccess,
  warning: notifyWarning,
  info: notifyInfo,
  error: notifyError,
  update: updateNotification
} = useNotification()

// ========== 推理相关 ==========
const inferenceConf = ref(0.3)
const inferenceIou = ref(0.5)
const autoInference = ref(true)  // 切图时自动推理
const isInferring = ref(false)

// 处理推理结果（NMS + 过滤 + 合并）
const processInferenceResults = (results: InferenceAnnotation[]) => {
  if (!activeImage.value) return
  
  // 移除现有的推理标注（无论新结果是否为空）
  currentAnnotations.value = currentAnnotations.value.filter(a => !a.isInference)
  
  // 如果没有新结果，直接返回
  if (!results?.length) return
  
  // IoU 计算
  const calcIoU = (b1: {x:number,y:number,width:number,height:number}, b2: {x:number,y:number,width:number,height:number}) => {
    const x1 = Math.max(b1.x, b2.x), y1 = Math.max(b1.y, b2.y)
    const x2 = Math.min(b1.x + b1.width, b2.x + b2.width), y2 = Math.min(b1.y + b1.height, b2.y + b2.height)
    if (x2 <= x1 || y2 <= y1) return 0
    const inter = (x2 - x1) * (y2 - y1)
    return inter / (b1.width * b1.height + b2.width * b2.height - inter)
  }
  
  const getBbox = (ann: InferenceAnnotation | Annotation) => {
    const d = ann.data as {x?:number,y?:number,width?:number,height?:number,points?:[number,number][]}
    if (d.x !== undefined && d.y !== undefined && d.width !== undefined && d.height !== undefined) {
      return { x: d.x, y: d.y, width: d.width, height: d.height }
    }
    if (d.points?.length) {
      const xs = d.points.map(p => p[0]), ys = d.points.map(p => p[1])
      return { x: Math.min(...xs), y: Math.min(...ys), width: Math.max(...xs) - Math.min(...xs), height: Math.max(...ys) - Math.min(...ys) }
    }
    return null
  }
  
  const iouThreshold = 0.5
  
  // NMS
  const sorted = [...results].sort((a, b) => b.confidence - a.confidence)
  const afterNms: InferenceAnnotation[] = []
  for (const inf of sorted) {
    const box1 = getBbox(inf)
    if (!box1) continue
    let suppressed = false
    for (const kept of afterNms) {
      const box2 = getBbox(kept)
      if (box2 && calcIoU(box1, box2) >= iouThreshold) { suppressed = true; break }
    }
    if (!suppressed) afterNms.push(inf)
  }
  
  // 过滤与已有标注重叠的
  const existingAnns = currentAnnotations.value.filter(a => !a.isInference)
  const filtered = afterNms.filter(inf => {
    const box1 = getBbox(inf)
    if (!box1) return true
    for (const ann of existingAnns) {
      const box2 = getBbox(ann)
      if (box2 && calcIoU(box1, box2) >= iouThreshold) return false
    }
    return true
  })
  
  // 转换并合并
  const newAnns: Annotation[] = filtered.map((inf, idx) => {
    const cat = projectCategories.value.find(c => c.name === inf.categoryName)
    const categoryId = cat?.id ?? -1
    let data: Annotation['data']
    if (inf.type === 'bbox') {
      data = { x: inf.data.x!, y: inf.data.y!, width: inf.data.width!, height: inf.data.height! }
      if (inf.data.keypoints?.length) {
        (data as {keypoints?: [number,number,number][]}).keypoints = inf.data.keypoints
      }
    } else {
      data = { points: inf.data.points as [number, number][] }
    }
    return {
      id: `inference-${idx}-${Date.now()}`,
      imageId: activeImage.value!.id,
      categoryId,
      type: inf.type as 'bbox' | 'polygon',
      data,
      isInference: true,
      confidence: inf.confidence,
      _categoryName: inf.categoryName
    } as Annotation & { _categoryName: string }
  })
  
  // 先移除旧的推理结果，再添加新的
  console.log('[App] Adding inference annotations:', newAnns.length, 'total:', currentAnnotations.value.filter(a => !a.isInference).length + newAnns.length)
  if (newAnns.length > 0) {
    console.log('[App] First annotation:', JSON.stringify(newAnns[0]).slice(0, 200))
  }
  currentAnnotations.value = [...currentAnnotations.value.filter(a => !a.isInference), ...newAnns]
}

// SAM-2 等交互式推理：带提示点
const runPromptInference = async (points: Array<{x: number; y: number; type: 'positive' | 'negative'}>) => {
  if (!activeImage.value || !currentProject.value || isInferring.value) return
  
  const imageId = activeImage.value.id
  
  // 根据选中的类别类型决定输出类型（bbox -> rect，其他用 polygon）
  const outputType = selectedCategory.value?.type === 'bbox' ? 'rect' : 'polygon'
  
  isInferring.value = true
  try {
    const t0 = performance.now()
    const result = await window.electronAPI?.inferenceRun?.({
      projectId: currentProject.value.id,
      path: activeImage.value.originalPath || '',
      conf: inferenceConf.value,
      iou: inferenceIou.value,
      points,
      multimask: false,
      outputType
    } as any)
    const t1 = performance.now()
    
    console.log('[App] Prompt inference timing(ms):', {
      total: (t1 - t0).toFixed(1),
      pythonTime: result?.inferTimeMs?.toFixed(1) || 'N/A'
    })
    
    if (activeImage.value?.id !== imageId) {
      isInferring.value = false
      return
    }
    
    if (result?.success && result.annotations?.length) {
      // SAM-2 结果直接使用选中的类别
      const categoryId = selectedCategory.value?.id ?? -1
      const categoryType = selectedCategory.value?.type || 'polygon'
      const newAnns: Annotation[] = (result.annotations as any[]).map((inf, idx: number) => {
        // 根据返回类型和选中类别构建标注
        if (inf.type === 'rect' && categoryType === 'bbox') {
          return {
            id: `inference-${idx}-${Date.now()}`,
            imageId: activeImage.value!.id,
            categoryId,
            type: 'bbox' as const,
            data: { x: inf.data.x, y: inf.data.y, width: inf.data.width, height: inf.data.height },
            isInference: true,
            confidence: inf.confidence
          }
        } else {
          return {
            id: `inference-${idx}-${Date.now()}`,
            imageId: activeImage.value!.id,
            categoryId,
            type: 'polygon' as const,
            data: { points: inf.data.points as [number, number][] },
            isInference: true,
            confidence: inf.confidence
          }
        }
      })
      // 移除旧的推理结果，添加新的
      currentAnnotations.value = [...currentAnnotations.value.filter(a => !a.isInference), ...newAnns]
      console.log('[App] SAM-2 inference added:', newAnns.length, 'annotations with category:', categoryId, 'type:', categoryType)
    } else if (!result?.success) {
      console.warn('[App] Prompt inference failed:', result?.error)
    }
  } catch (e) {
    console.error('[App] Prompt inference error:', e)
  } finally {
    isInferring.value = false
  }
}

// SAM-2 提示点点击处理
const handlePromptClick = (payload: { x: number; y: number; type: 'positive' | 'negative' }) => {
  // 检查是否选中了多边形或矩形框类别
  if (!selectedCategory.value || (selectedCategory.value.type !== 'polygon' && selectedCategory.value.type !== 'bbox')) {
    notifyWarning(t('inference.selectPolygonCategory'))
    return
  }
  
  console.log('[App] Prompt click:', payload, 'category:', selectedCategory.value.name, 'type:', selectedCategory.value.type)
  // 直接使用单个点进行推理（可以扩展为累积多个点）
  runPromptInference([payload])
}

// YOLOE 等 box 模式推理：用户框选的 bbox 作为 visual prompt
const runBoxInference = async (box: { x: number; y: number; width: number; height: number }) => {
  if (!activeImage.value || !currentProject.value || isInferring.value) return
  
  const imageId = activeImage.value.id
  const categoryId = selectedCategory.value?.id ?? -1
  
  isInferring.value = true
  try {
    const t0 = performance.now()
    // 转换为普通对象，避免 IPC 序列化错误
    const plainBox = { x: box.x, y: box.y, width: box.width, height: box.height }
    const result = await window.electronAPI?.inferenceRun?.({
      projectId: currentProject.value.id,
      path: activeImage.value.originalPath || '',
      conf: inferenceConf.value,
      iou: inferenceIou.value,
      box: plainBox,  // 传递框选区域作为 visual prompt
      outputType: 'rect'  // YOLOE 输出矩形框
    } as any)
    const t1 = performance.now()
    
    console.log('[App] Box inference timing(ms):', {
      total: (t1 - t0).toFixed(1),
      pythonTime: result?.inferTimeMs?.toFixed(1) || 'N/A'
    })
    
    if (activeImage.value?.id !== imageId) {
      isInferring.value = false
      return
    }
    
    if (result?.success && result.annotations?.length) {
      // 过滤低置信度结果
      const confThreshold = inferenceConf.value
      const filtered = (result.annotations as any[]).filter(inf => inf.confidence >= confThreshold)
      
      const newAnns: Annotation[] = filtered.map((inf, idx: number) => ({
        id: `inference-${idx}-${Date.now()}`,
        imageId: activeImage.value!.id,
        categoryId,
        type: 'bbox' as const,
        data: { x: inf.data.x, y: inf.data.y, width: inf.data.width, height: inf.data.height },
        isInference: true,
        confidence: inf.confidence
      }))
      // 移除旧的推理结果和触发推理的临时 bbox，添加新的
      currentAnnotations.value = [...currentAnnotations.value.filter(a => !a.isInference), ...newAnns]
      console.log('[App] Box inference added:', newAnns.length, 'annotations (filtered from', result.annotations.length, 'by conf >=', confThreshold, ')')
    } else if (!result?.success) {
      console.warn('[App] Box inference failed:', result?.error)
    }
  } catch (e) {
    console.error('[App] Box inference error:', e)
  } finally {
    isInferring.value = false
  }
}

// 工作台直接发起推理（通过 IPC 调用 Electron 主进程，由主进程管理 Python 子进程）
const runInference = async () => {
  if (!activeImage.value || !currentProject.value || isInferring.value) return
  
  const projectId = currentProject.value.id
  const imageId = activeImage.value.id  // 记录发起推理时的图片ID
  const relPath = activeImage.value.originalPath
  if (!relPath) return
  
  // text 模式插件需要类别名作为 prompt
  const isTextMode = activePlugin.value?.interactionMode === 'text'
  if (isTextMode && !selectedCategory.value) {
    console.log('[App] Text mode inference skipped: no category selected')
    return
  }
  
  isInferring.value = true
  try {
    const t0 = performance.now()
    const result = await window.electronAPI?.inferenceRun?.({
      projectId,
      path: relPath,
      conf: inferenceConf.value,
      iou: inferenceIou.value,
      // text 模式：传递类别名作为 prompt
      prompt: isTextMode ? selectedCategory.value?.name : undefined
    })
    const t1 = performance.now()
    
    console.log('[App] Inference timing(ms):', {
      total: (t1 - t0).toFixed(1),
      pythonTime: result?.inferTimeMs?.toFixed(1) || 'N/A'
    })
    
    // 校验：结果返回时，当前图片是否仍是发起请求时的图片
    if (activeImage.value?.id !== imageId) {
      console.log('[App] Inference result discarded: image changed')
      // 为当前图片重新发起推理
      isInferring.value = false
      if (autoInference.value && activeImage.value) {
        runInference()
      }
      return
    }
    
    console.log('[App] Inference result:', result)
    if (result?.success) {
      console.log('[App] Inference annotations count:', result.annotations?.length || 0)
      processInferenceResults(result.annotations as InferenceAnnotation[] || [])
    } else {
      console.warn('[App] Inference failed:', result?.error)
    }
  } catch (e) {
    console.error('[App] Inference error:', e)
  } finally {
    isInferring.value = false
  }
}

onMounted(() => {
  // 初始化全局 WebSocket 连接
  connectWs()
  
  // 订阅模型下载进度消息
  subscribeWs('model_download_progress', (msg) => {
    const data = msg.data as { filename: string; progress: number; downloaded: number; total: number }
    // 转发给推理小窗口
    window.electronAPI?.forwardDownloadProgress?.({ type: 'progress', ...data })
    // 更新通知（使用持久通知）
    updateNotification(`download-${data.filename}`, {
      type: 'info',
      message: t('inference.downloading', { name: data.filename, progress: data.progress }),
      persistent: true
    })
  })
  subscribeWs('model_download_complete', (msg) => {
    const data = msg.data as { filename: string }
    // 转发给推理小窗口
    window.electronAPI?.forwardDownloadProgress?.({ type: 'complete', ...data })
    // 更新之前的进度通知为完成状态（而不是创建新通知）
    updateNotification(`download-${data.filename}`, {
      type: 'success',
      message: t('inference.downloadComplete', { name: data.filename }),
      persistent: false  // 完成后自动消失
    })
  })
  subscribeWs('model_download_error', (msg) => {
    const data = msg.data as { filename: string }
    // 转发给推理小窗口
    window.electronAPI?.forwardDownloadProgress?.({ type: 'error', ...data, error: msg.message })
    // 更新之前的进度通知为错误状态
    updateNotification(`download-${data.filename}`, {
      type: 'error',
      message: t('inference.downloadFailed', { name: data.filename, error: msg.message }),
      persistent: false
    })
  })
  
  // 确保路由准备好后再加载项目
  router.isReady().then(() => {
    loadProjects()
    // 检查是否有依赖未安装的 Python 插件，用于 Python 环境角标
    void checkPythonPluginDepsIssue()
  })
  
  // 检查是否有新版本
  checkForUpdates()
  
  // 只注册一次 Electron IPC 监听器，防止组件重建时重复注册
  if (!electronListenersRegistered) {
    electronListenersRegistered = true
    
    // 监听推理结果（兼容推理小窗发来的结果）
    window.electronAPI?.onInferenceResults?.((results: InferenceAnnotation[]) => {
      processInferenceResults(results)
    })
    
    // 监听小窗转发的日志
    window.electronAPI?.onForwardedLog?.((data) => {
      const prefix = '[InferenceWindow]'
      if (data.type === 'error') {
        console.error(prefix, ...data.args)
      } else if (data.type === 'warn') {
        console.warn(prefix, ...data.args)
      } else {
        console.log(prefix, ...data.args)
      }
    })
    
    // 监听小窗参数变化，更新本地参数并重新推理
    window.electronAPI?.onInferenceParamsChanged?.((params) => {
      console.log('[App] Inference params changed from window:', params)
      // 支持部分更新
      if (params.conf !== undefined) inferenceConf.value = params.conf
      if (params.iou !== undefined) inferenceIou.value = params.iou
      // 用新参数重新推理当前图片（box 模式不自动推理）
      if (activePlugin.value?.interactionMode !== 'box') {
        runInference()
      }
    })
    
    // 监听小窗模型变化，重新设置图片
    window.electronAPI?.onInferenceModelChanged?.(async () => {
      console.log('[App] Inference model changed from window')
      // 模型加载后需要重新设置图片（SAM-2 需要）
      if (activeImage.value && currentProject.value) {
        const imagePath = activeImage.value.isExternal 
          ? activeImage.value.originalPath 
          : `project_item/${currentProject.value.id}/${activeImage.value.originalPath}`
        if (imagePath) {
          await window.electronAPI?.inferenceSetImage?.(imagePath)
        }
        // 重新推理（prompt 模式插件不自动推理）
        if (activePlugin.value?.interactionMode !== 'prompt') {
          runInference()
        }
      }
    })
    
    // 监听显示通知请求（来自插件小窗）
    window.electronAPI?.onShowNotification?.((data: { type: string; message: string }) => {
      if (data.type === 'success') {
        notifySuccess(data.message)
      } else if (data.type === 'warning') {
        notifyWarning(data.message)
      } else if (data.type === 'error') {
        notifyError(data.message)
      }
    })
    
    // 暴露全局函数供 Electron 主进程调用（类别相关）
    ;(window as any).__easymark_getCategories = () => {
      return {
        success: true,
        categories: projectCategories.value.map(c => ({
          id: c.id,
          name: c.name,
          type: c.type,
          color: c.color
        }))
      }
    }
    
    ;(window as any).__easymark_getSelectedCategory = () => {
      const cat = selectedCategory.value
      return {
        success: true,
        category: cat ? { id: cat.id, name: cat.name, type: cat.type, color: cat.color } : null
      }
    }
    
    ;(window as any).__easymark_selectCategory = (categoryId: number) => {
      const cat = projectCategories.value.find(c => c.id === categoryId)
      if (cat) {
        selectedCategory.value = cat
        return { success: true }
      }
      return { success: false }
    }
  }

  const handleGlobalMousedown = (event: MouseEvent) => {
    const target = event.target as HTMLElement | null

    // 新建类别编辑区域外点击，关闭新建卡片
    if (isCreatingCategory.value) {
      const editor = categoryEditorRef.value
      if (editor && target && (editor === target || editor.contains(target))) {
        // 点击在新建卡片内部，不处理
      } else {
        isCreatingCategory.value = false
        isCategoryColorPickerOpen.value = false
        newCategoryName.value = ''
      }
    }

    // 已有类别调色盘：仅在“只改颜色”模式下，点击空白处时提交
    if (isEditCategoryColorPickerOpen.value && !isEditingCategoryName.value && editCategoryTarget.value) {
      if (target) {
        const palettes = document.querySelectorAll('.category-color-palette')
        for (const el of Array.from(palettes)) {
          if (el.contains(target)) {
            // 点击在调色盘内部，不提交
            return
          }
        }
      }
      // 点击到了调色盘以外的位置，提交当前颜色修改
      void submitEditCategory()
    }

    // 编辑类别名时，点击输入框外部保存
    if (isEditingCategoryName.value && editCategoryTarget.value) {
      const inputs = document.querySelectorAll('.category-editor-card__input')
      let clickedInInput = false
      for (const el of Array.from(inputs)) {
        if (el === target || el.contains(target)) {
          clickedInInput = true
          break
        }
      }
      if (!clickedInInput) {
        void submitEditCategory()
      }
    }
  }
  window.addEventListener('mousedown', handleGlobalMousedown, true)
  
  // 全局快捷键监听
  const handleGlobalKeydown = (e: KeyboardEvent) => {
    // 如果焦点在输入框中，不处理快捷键（除了 Escape）
    const target = e.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
      return
    }
    
    // 如果设置面板打开，不处理快捷键
    if (isSettingsOpen.value) return
    
    const action = matchAction(e)
    if (!action) return
    
    // 阻止默认行为
    e.preventDefault()
    e.stopPropagation()
    
    switch (action) {
      case 'save':
        handleSaveAnnotations()
        break
      case 'saveAsNegative':
        handleSaveAsNegative()
        break
      case 'prevImage':
        handlePrevImage(false).then(() => {
          if (activeImage.value) scrollToImage(activeImage.value.id)
        })
        break
      case 'nextImage':
        handleNextImage(false).then(() => {
          if (activeImage.value) scrollToImage(activeImage.value.id)
        })
        break
      case 'prevUnannotated':
        handlePrevUnannotated().then(() => {
          if (activeImage.value) scrollToImage(activeImage.value.id)
        })
        break
      case 'nextUnannotated':
        handleNextUnannotated().then(() => {
          if (activeImage.value) scrollToImage(activeImage.value.id)
        })
        break
      case 'resetView':
        handleResetView()
        break
      case 'deleteSelected':
        handleDeleteSelected()
        break
      case 'toggleKeypointVisibility':
        annotationCanvasRef.value?.toggleSelectedKeypointVisibility()
        break
    }
  }
  window.addEventListener('keydown', handleGlobalKeydown, true)
  
  onBeforeUnmount(() => {
    window.removeEventListener('mousedown', handleGlobalMousedown, true)
    window.removeEventListener('keydown', handleGlobalKeydown, true)
  })
})

// 记住上次打开的项目ID（使用 sessionStorage 防止组件重建时丢失）
const getLastProjectId = () => sessionStorage.getItem('lastProjectId')
const setLastProjectId = (id: string | null) => {
  if (id) {
    sessionStorage.setItem('lastProjectId', id)
  } else {
    sessionStorage.removeItem('lastProjectId')
  }
}

// 监听当前项目变化，记住打开的项目
watch(currentProject, (project) => {
  if (project) {
    setLastProjectId(project.id)
  }
})

// 尝试恢复上次打开的项目（只设置项目，不跳转页面）
const tryRestoreLastProject = () => {
  const lastId = getLastProjectId()
  if (!lastId) return
  const lastProject = projects.value.find(p => p.id === lastId)
  if (lastProject && !currentProject.value) {
    console.log('[App] Restoring last project:', lastProject.name)
    // 只设置当前项目，不调用 handleProjectClick（它会跳转到项目页）
    currentProject.value = lastProject
  }
}

// 项目列表加载完成后尝试恢复（loadProjects 中已调用，这里作为备份）
watch(projects, (projectList) => {
  if (projectList.length > 0) {
    tryRestoreLastProject()
  }
})

const openCreateCategory = () => {
	if (!currentProject.value) return
	const total = presetCategoryColors.length
	if (total > 0) {
		if (nextPresetCategoryColorIndex >= total) {
			nextPresetCategoryColorIndex = 0
		}
		const color =
			presetCategoryColors[nextPresetCategoryColorIndex] ??
			presetCategoryColors[0] ??
			newCategoryColor.value
		newCategoryColor.value = color
		nextPresetCategoryColorIndex = (nextPresetCategoryColorIndex + 1) % total
	}
	isCreatingCategory.value = true
	isCategoryColorPickerOpen.value = false
	newCategoryName.value = ''
}

const handleCategoryColorClick = () => {
	if (!isCreatingCategory.value) return
	isCategoryColorPickerOpen.value = !isCategoryColorPickerOpen.value
}

const selectPresetCategoryColor = (color: string) => {
	newCategoryColor.value = color
	isCategoryColorPickerOpen.value = false
}

const handleNativeColorChange = (event: Event) => {
	const input = event.target as HTMLInputElement | null
	if (!input) return
	const value = input.value
	if (!value) return
	newCategoryColor.value = value
}

const submitCreateCategory = async () => {
	if (!currentProject.value) return
	const name = newCategoryName.value.trim()
	if (!name) {
		notifyError(t('project.categoryPanel.createErrorNameRequired'))
		return
	}
	const type: CategoryType = activeCategoryTab.value
	try {
		const resp = await fetch('http://127.0.0.1:18080/api/project-categories', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				name,
				type,
				color: newCategoryColor.value
			})
		})
		if (!resp.ok) {
			const errText = await resp.text().catch(() => '')
			console.error('create category failed', resp.status, errText)
			notifyError(t('project.categoryPanel.createErrorGeneric'))
			return
		}
		isCreatingCategory.value = false
		isCategoryColorPickerOpen.value = false
		newCategoryName.value = ''
		await loadProjectCategories()
		notifyInfo(t('project.categoryPanel.createSuccess'))
	} catch (e) {
		console.error('create category error', e)
		notifyError(t('project.categoryPanel.createErrorNetwork'))
	}
}

const openCategoryContextMenu = (event: MouseEvent, cat: ProjectCategory) => {
	event.preventDefault()
	// 预估菜单尺寸，避免菜单超出窗口边界
	const menuWidth = 180
	const menuHeight = 40
	const vw = window.innerWidth || 0
	const vh = window.innerHeight || 0
	const margin = 4
	let x = event.clientX
	let y = event.clientY
	if (x + menuWidth > vw - margin) {
		x = Math.max(margin, vw - margin - menuWidth)
	}
	if (y + menuHeight > vh - margin) {
		y = Math.max(margin, vh - margin - menuHeight)
	}
	if (x < margin) x = margin
	if (y < margin) y = margin
	categoryContextMenuX.value = x
	categoryContextMenuY.value = y
	categoryContextMenuTarget.value = cat
	isCategoryContextMenuOpen.value = true
}

const closeCategoryContextMenu = () => {
	isCategoryContextMenuOpen.value = false
	categoryContextMenuTarget.value = null
}

const openEditCategoryModal = (cat: ProjectCategory) => {
	if (!currentProject.value) return
	closeCategoryContextMenu()
	isCreatingCategory.value = false
	isCategoryColorPickerOpen.value = false
	editCategoryTarget.value = cat
	editCategoryName.value = cat.name
	editCategoryColor.value = cat.color || '#ff0000'
	editCategoryError.value = ''
	isEditCategoryColorPickerOpen.value = false
	isEditingCategoryName.value = true
}

const openEditCategoryFromContextMenu = () => {
	if (!currentProject.value) return
	const cat = categoryContextMenuTarget.value
	if (!cat) return
	openEditCategoryModal(cat)
}

const openEditCategoryFromColor = (cat: ProjectCategory) => {
	if (!currentProject.value) return
	// 仅编辑颜色：不进入名称编辑模式，只在该行下展开调色盘
	editCategoryTarget.value = cat
	editCategoryName.value = cat.name
	editCategoryColor.value = cat.color || '#ff0000'
	isEditingCategoryName.value = false
	isCategoryColorPickerOpen.value = false
	isEditCategoryColorPickerOpen.value =
		editCategoryTarget.value && editCategoryTarget.value.id === cat.id
			? !isEditCategoryColorPickerOpen.value
			: true
}

const openKeypointConfigModal = () => {
	const cat = categoryContextMenuTarget.value
	if (!cat || cat.type !== 'keypoint') return
	closeCategoryContextMenu()
	keypointConfigTarget.value = cat
	keypointConfigError.value = ''
	// 解析已有的关键点配置
	const parsed = parseKeypointMate(cat.mate)
	if (parsed && parsed.keypoints.length > 0) {
		keypointConfigList.value = parsed.keypoints.map((kp) => ({ name: kp.name }))
	} else {
		keypointConfigList.value = [{ name: '' }]
	}
	// 查找已绑定的矩形框类别（从矩形框类别的 mate 中查找）
	keypointConfigBindBboxId.value = null
	for (const bbox of availableBboxCategories.value) {
		const bboxMate = parseBboxMate(bbox.mate)
		if (bboxMate?.keypointCategoryId === cat.id) {
			keypointConfigBindBboxId.value = bbox.id
			break
		}
	}
	isKeypointConfigModalOpen.value = true
}

const closeKeypointConfigModal = () => {
	if (isKeypointConfigSaving.value) return
	isKeypointConfigModalOpen.value = false
	keypointConfigTarget.value = null
	keypointConfigList.value = []
	keypointConfigError.value = ''
	keypointConfigBindBboxId.value = null
}

const addKeypointConfigItem = () => {
	if (keypointConfigList.value.length >= 64) {
		keypointConfigError.value = t('project.categoryPanel.keypointConfig.errorTooMany')
		return
	}
	keypointConfigList.value.push({ name: '' })
}

const removeKeypointConfigItem = (index: number) => {
	if (keypointConfigList.value.length <= 1) return
	keypointConfigList.value.splice(index, 1)
}

const submitKeypointConfig = async () => {
	if (!currentProject.value || !keypointConfigTarget.value) return
	const validKeypoints = keypointConfigList.value
		.map((kp) => ({ name: kp.name.trim() }))
		.filter((kp) => kp.name !== '')
	if (validKeypoints.length === 0) {
		keypointConfigError.value = t('project.categoryPanel.keypointConfig.errorEmpty')
		return
	}
	isKeypointConfigSaving.value = true
	keypointConfigError.value = ''
	try {
		// 保存关键点配置
		const resp = await fetch('http://127.0.0.1:18080/api/project-categories', {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				categoryId: keypointConfigTarget.value.id,
				keypoints: validKeypoints
			})
		})
		if (!resp.ok) {
			const data = await resp.json().catch(() => ({})) as { error?: string }
			if (data.error === 'mate_unsupported') {
				keypointConfigError.value = t('project.categoryPanel.keypointConfig.errorUnsupported')
			} else if (data.error === 'keypoints_empty') {
				keypointConfigError.value = t('project.categoryPanel.keypointConfig.errorEmpty')
			} else {
				keypointConfigError.value = t('project.categoryPanel.keypointConfig.errorGeneric')
			}
			return
		}
		
		// 更新矩形框类别的绑定关系
		const kpCatId = keypointConfigTarget.value.id
		const newBindBboxId = keypointConfigBindBboxId.value
		
		// 查找当前绑定这个关键点类别的矩形框
		let oldBindBboxId: number | null = null
		for (const bbox of availableBboxCategories.value) {
			const bboxMate = parseBboxMate(bbox.mate)
			if (bboxMate?.keypointCategoryId === kpCatId) {
				oldBindBboxId = bbox.id
				break
			}
		}
		
		// 如果绑定关系有变化
		if (oldBindBboxId !== newBindBboxId) {
			// 清除旧的绑定
			if (oldBindBboxId && currentProject.value) {
				await fetch('http://127.0.0.1:18080/api/project-categories', {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ projectId: currentProject.value.id, id: oldBindBboxId, mate: '' })
				})
			}
			// 设置新的绑定
			if (newBindBboxId && currentProject.value) {
				const newBboxMate = JSON.stringify({ keypointCategoryId: kpCatId })
				await fetch('http://127.0.0.1:18080/api/project-categories', {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ projectId: currentProject.value.id, id: newBindBboxId, mate: newBboxMate })
				})
			}
		}
		
		await loadProjectCategories()
		notifyInfo(t('project.categoryPanel.keypointConfig.saveSuccess'))
		// 保存成功后关闭面板
		isKeypointConfigModalOpen.value = false
		keypointConfigTarget.value = null
		keypointConfigList.value = []
		keypointConfigError.value = ''
		keypointConfigBindBboxId.value = null
	} catch (e) {
		console.error('save keypoint config error', e)
		keypointConfigError.value = t('project.categoryPanel.keypointConfig.errorNetwork')
	} finally {
		isKeypointConfigSaving.value = false
	}
}

// 类别拖拽排序处理（同一类型内）
const handleCategoryDragStart = (event: DragEvent, cat: ProjectCategory) => {
	if (!currentProject.value) return
	draggingCategoryId.value = cat.id
	dragOverCategoryId.value = null
	// 提升拖拽兼容性
	if (event.dataTransfer) {
		event.dataTransfer.effectAllowed = 'move'
		event.dataTransfer.setData('text/plain', String(cat.id))
	}
}

const handleCategoryDragOver = (event: DragEvent, cat: ProjectCategory) => {
	if (!draggingCategoryId.value) return
	if (!currentProject.value) return
	// 只允许同一类型下排序
	if (cat.type !== activeCategoryTab.value) return
	if (event.preventDefault) event.preventDefault()
	dragOverCategoryId.value = cat.id
}

const handleCategoryDrop = async (event: DragEvent, cat: ProjectCategory) => {
	if (!draggingCategoryId.value) return
	if (!currentProject.value) return
	if (event.preventDefault) event.preventDefault()
	const fromId = draggingCategoryId.value
	const toId = cat.id
	if (fromId === toId) {
		draggingCategoryId.value = null
		dragOverCategoryId.value = null
		return
	}
	const type: CategoryType = activeCategoryTab.value
	// 取出当前类型下的类别，按现有顺序排序
	const sameType = projectCategories.value
		.filter((c) => c.type === type)
		.sort((a, b) => {
			if (a.sortOrder !== b.sortOrder) return a.sortOrder - b.sortOrder
			return a.id - b.id
		})
	const fromIndex = sameType.findIndex((c) => c.id === fromId)
	const toIndex = sameType.findIndex((c) => c.id === toId)
	if (fromIndex === -1 || toIndex === -1) {
		draggingCategoryId.value = null
		dragOverCategoryId.value = null
		return
	}
	const removed = sameType.splice(fromIndex, 1)
	const moved = removed[0]
	if (!moved) {
		draggingCategoryId.value = null
		dragOverCategoryId.value = null
		return
	}
	sameType.splice(toIndex, 0, moved)
	// 本地更新 sortOrder
	const newIds: number[] = []
	sameType.forEach((c, idx) => {
		c.sortOrder = idx + 1
		newIds.push(c.id)
	})
	// 同步到 projectCategories 数组
	projectCategories.value = projectCategories.value.map((c) => {
		const found = sameType.find((it) => it.id === c.id)
		return found ? { ...c, sortOrder: found.sortOrder } : c
	})
	// 调用后端保存排序
	try {
		await saveCategorySortOrder(type, newIds)
	} finally {
		draggingCategoryId.value = null
		dragOverCategoryId.value = null
	}
}

const handleCategoryDragEnd = () => {
	draggingCategoryId.value = null
	dragOverCategoryId.value = null
}

const saveCategorySortOrder = async (type: CategoryType, ids: number[]) => {
	if (!currentProject.value) return
	if (!ids.length) return
	try {
		const resp = await fetch('http://127.0.0.1:18080/api/project-categories/sort', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				type,
				categoryIds: ids
			})
		})
		if (!resp.ok) {
			console.error('save category sort failed', resp.status)
			notifyError(t('project.categoryPanel.sort.errorGeneric'))
			return
		}
	} catch (e) {
		console.error('save category sort error', e)
		notifyError(t('project.categoryPanel.sort.errorNetwork'))
	}
}

const submitEditCategory = async (forceMerge = false) => {
	if (!currentProject.value || !editCategoryTarget.value) return
	if (isEditingCategory.value) return
	const name = editCategoryName.value.trim()
	if (!name) {
		editCategoryError.value = t('project.categoryPanel.editCategory.errorNameRequired')
		return
	}
	const color = editCategoryColor.value || editCategoryTarget.value.color
	
	// 检测是否存在同名同类型的类别（仅在非强制合并时）
	if (!forceMerge) {
		const existingCategory = projectCategories.value.find(
			c => c.id !== editCategoryTarget.value!.id && 
			     c.name === name && 
			     c.type === editCategoryTarget.value!.type
		)
		if (existingCategory) {
			// 弹出合并确认对话框
			mergeCategoryTargetName.value = name
			mergeCategoryTargetId.value = existingCategory.id
			isMergeCategoryModalOpen.value = true
			return
		}
	}
	
	isEditingCategory.value = true
	editCategoryError.value = ''
	try {
		const resp = await fetch('http://127.0.0.1:18080/api/project-categories/edit', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				categoryId: editCategoryTarget.value.id,
				name,
				color,
				merge: forceMerge,
				mergeTargetId: forceMerge ? mergeCategoryTargetId.value : undefined
			})
		})
		if (!resp.ok) {
			const data = (await resp.json().catch(() => ({}))) as { error?: string }
			if (data.error === 'name_required') {
				editCategoryError.value = t('project.categoryPanel.editCategory.errorNameRequired')
			} else if (data.error === 'color_invalid' || data.error === 'color_required') {
				editCategoryError.value = t('project.categoryPanel.editCategory.errorGeneric')
			} else if (data.error === 'category_exists') {
				editCategoryError.value = t('project.categoryPanel.editCategory.errorGeneric')
			} else if (data.error === 'category_not_found') {
				editCategoryError.value = t('project.categoryPanel.deleteCategory.errorNotFound')
			} else {
				editCategoryError.value = t('project.categoryPanel.editCategory.errorGeneric')
			}
			return
		}
		await loadProjectCategories()
		// 合并后同步更新当前画布中的标注
		if (forceMerge && mergeCategoryTargetId.value && editCategoryTarget.value) {
			currentAnnotations.value = currentAnnotations.value.map(a => 
				a.categoryId === editCategoryTarget.value!.id 
					? { ...a, categoryId: mergeCategoryTargetId.value! }
					: a
			)
		}
		editCategoryTarget.value = null
		isEditingCategoryName.value = false
		isEditCategoryColorPickerOpen.value = false
		isMergeCategoryModalOpen.value = false
		mergeCategoryTargetId.value = null
		notifyInfo(forceMerge 
			? t('project.categoryPanel.mergeCategory.success')
			: t('project.categoryPanel.editCategory.success')
		)
	} catch (e) {
		console.error('edit category error', e)
		editCategoryError.value = t('project.categoryPanel.editCategory.errorNetwork')
	} finally {
		isEditingCategory.value = false
		isMergingCategory.value = false
	}
}

const confirmMergeCategory = () => {
	isMergingCategory.value = true
	submitEditCategory(true)
}

const cancelMergeCategory = () => {
	isMergeCategoryModalOpen.value = false
	mergeCategoryTargetId.value = null
	mergeCategoryTargetName.value = ''
}

const openDeleteCategoryDialog = () => {
	if (!currentProject.value) return
	const cat = categoryContextMenuTarget.value
	if (!cat) return
	closeCategoryContextMenu()
	deleteCategoryTarget.value = cat
	deleteCategoryError.value = ''
	isDeleteCategoryModalOpen.value = true
}

const closeDeleteCategoryModal = () => {
	if (isDeletingCategory.value) return
	isDeleteCategoryModalOpen.value = false
	deleteCategoryTarget.value = null
	deleteCategoryError.value = ''
}

const submitDeleteCategory = async () => {
	if (!currentProject.value || !deleteCategoryTarget.value) return
	isDeletingCategory.value = true
	deleteCategoryError.value = ''
	const deletedCategoryId = deleteCategoryTarget.value.id
	try {
		const resp = await fetch('http://127.0.0.1:18080/api/project-categories', {
			method: 'DELETE',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				projectId: currentProject.value.id,
				categoryId: deletedCategoryId
			})
		})
		if (!resp.ok) {
			const data = await resp.json().catch(() => ({})) as { error?: string }
			if (data.error === 'category_not_found') {
				deleteCategoryError.value = t('project.categoryPanel.deleteCategory.errorNotFound')
			} else {
				deleteCategoryError.value = t('project.categoryPanel.deleteCategory.errorGeneric')
			}
			return
		}
		await loadProjectCategories()
		// 刷新图片列表以更新标注状态
		await loadProjectImages()
		// 删除类别后，同步删除当前画布中该类别的标注
		currentAnnotations.value = currentAnnotations.value.filter(a => a.categoryId !== deletedCategoryId)
		// 如果删除的是当前选中的类别，清空选中
		if (selectedCategory.value?.id === deletedCategoryId) {
			selectedCategory.value = null
		}
		isDeleteCategoryModalOpen.value = false
		deleteCategoryTarget.value = null
		deleteCategoryError.value = ''
		notifyInfo(t('project.categoryPanel.deleteCategory.success'))
	} catch (e) {
		console.error('delete category error', e)
		deleteCategoryError.value = t('project.categoryPanel.deleteCategory.errorNetwork')
	} finally {
		isDeletingCategory.value = false
	}
}
</script>

<template>
  <div class="app">
    <!-- 拖拽选择框（fixed 定位，显示在最上层） -->
    <div
      v-if="dragSelectionRectStyle"
      class="project-image-selection-rect"
      :style="dragSelectionRectStyle"
    ></div>
    <header class="app-header">
      <div class="app-header__left">
        <img :src="logo" alt="EasyMark logo" class="app-header__logo" />
        <span class="app-header__title">{{ t('app.title') }}</span>
        <div
          v-if="route.path === '/project'"
          class="app-header__project-status-wrapper"
        >
          <button
            type="button"
            class="app-header__project-status"
            @click="toggleProjectMenu"
          >
            <Folder :size="14" class="app-header__project-status-icon" />
            <span>{{ currentProject ? currentProject.name : t('project.status.none') }}</span>
            <ChevronDown :size="14" class="app-header__project-status-chevron" />
          </button>
          <div v-if="isProjectMenuOpen" class="app-header__project-menu">
            <button
              type="button"
              class="app-header__project-menu-item"
              @click="handleNewProjectClick"
            >
              <Plus :size="14" class="app-header__project-menu-item-icon" />
              <span>{{ t('project.actions.new') }}</span>
            </button>
            <button
              type="button"
              class="app-header__project-menu-item"
              :disabled="!currentProject"
              @click="handleImportDatasetClick"
            >
              <Import :size="14" class="app-header__project-menu-item-icon" />
              <span>{{ t('project.actions.importDataset') }}</span>
            </button>
            <button
              type="button"
              class="app-header__project-menu-item"
              :disabled="!currentProject"
              @click="handleImportImagesClick"
            >
              <ImagePlus :size="14" class="app-header__project-menu-item-icon" />
              <span>{{ t('project.actions.importImages') }}</span>
            </button>
            <div
              v-if="projects.length"
              class="app-header__project-menu-separator"
            ></div>
            <button
              v-for="project in projects"
              :key="project.id"
              type="button"
              class="app-header__project-menu-item"
              @click="handleProjectClick(project)"
              @contextmenu="handleProjectContextMenu($event, project)"
            >
              <Folder :size="14" class="app-header__project-menu-item-icon" />
              <span
                class="app-header__project-menu-item-label"
                :title="project.name"
              >
                {{ project.name }}
              </span>
            </button>
          </div>
        </div>
      </div>
      <div class="app-header__right">
        <button
          type="button"
          class="window-control"
          aria-label="Settings"
          @click="handleOpenSettings"
        >
          <Settings :size="16" />
        </button>
        <button type="button" class="window-control" aria-label="Minimize" @click="handleMinimize">
          <Minus :size="16" />
        </button>
        <button
          type="button"
          class="window-control"
          aria-label="Maximize"
          @click="handleToggleMaximize"
        >
          <Square :size="16" />
        </button>
        <button
          type="button"
          class="window-control window-control--close"
          aria-label="Close"
          @click="handleClose"
        >
          <X :size="18" />
        </button>
      </div>
    </header>

    <div class="app-body">
      <aside class="app-sidebar">
        <button
          type="button"
          class="app-sidebar__item"
          :class="{ 'app-sidebar__item--active': route.path === '/project' }"
          :aria-label="t('sidebar.project')"
          :title="t('sidebar.project')"
          @click="handleSidebarClick('project')"
        >
          <Folder :size="22" />
        </button>
        <button
          type="button"
          class="app-sidebar__item"
          :class="{ 'app-sidebar__item--active': route.path === '/dataset' }"
          :aria-label="t('sidebar.dataset')"
          :title="t('sidebar.dataset')"
          @click="handleSidebarClick('dataset')"
        >
          <Database :size="22" />
        </button>
        <button
          type="button"
          class="app-sidebar__item"
          :class="{ 'app-sidebar__item--active': route.path === '/training' }"
          :aria-label="t('sidebar.training')"
          :title="t('sidebar.training')"
          @click="handleSidebarClick('training')"
        >
          <Bot :size="22" />
        </button>
        <button
          type="button"
          class="app-sidebar__item"
          :class="{ 'app-sidebar__item--active': route.path === '/plugins' }"
          :aria-label="t('sidebar.plugins')"
          :title="t('sidebar.plugins')"
          @click="handleSidebarClick('plugins')"
        >
          <Puzzle :size="22" />
        </button>
        <button
          type="button"
          class="app-sidebar__item app-sidebar__item--python"
          :class="{ 'app-sidebar__item--active': route.path === '/python-env' }"
          :aria-label="t('sidebar.pythonEnv')"
          :title="t('sidebar.pythonEnv')"
          @click="handleSidebarClick('python-env')"
        >
          <span class="app-sidebar__icon-wrapper">
            <!-- Python Logo (标准版) -->
            <svg width="22" height="22" viewBox="0 0 256 255" fill="currentColor">
              <path d="M126.916.072c-64.832 0-60.784 28.115-60.784 28.115l.072 29.128h61.868v8.745H41.631S.145 61.355.145 126.77c0 65.417 36.21 63.097 36.21 63.097h21.61v-30.356s-1.165-36.21 35.632-36.21h61.362s34.475.557 34.475-33.319V33.97S194.67.072 126.916.072zM92.802 19.66a11.12 11.12 0 0 1 11.13 11.13 11.12 11.12 0 0 1-11.13 11.13 11.12 11.12 0 0 1-11.13-11.13 11.12 11.12 0 0 1 11.13-11.13z" opacity=".85"/>
              <path d="M128.757 254.126c64.832 0 60.784-28.115 60.784-28.115l-.072-29.127H127.6v-8.745h86.441s41.486 4.705 41.486-60.712c0-65.416-36.21-63.096-36.21-63.096h-21.61v30.355s1.165 36.21-35.632 36.21h-61.362s-34.475-.557-34.475 33.32v56.013s-5.235 33.897 62.518 33.897zm34.114-19.586a11.12 11.12 0 0 1-11.13-11.13 11.12 11.12 0 0 1 11.13-11.131 11.12 11.12 0 0 1 11.13 11.13 11.12 11.12 0 0 1-11.13 11.13z" opacity=".85"/>
            </svg>
            <span
              v-if="hasPythonPluginDepsIssue"
              class="app-sidebar__badge app-sidebar__badge--warning"
            ></span>
          </span>
        </button>
        <!-- UI 按钮已隐藏 -->
        <!-- 底部图标占位 -->
        <div class="app-sidebar__spacer"></div>
        <!-- 帮助 -->
        <button
          type="button"
          class="app-sidebar__item"
          :class="{ 'app-sidebar__item--active': route.path === '/help' }"
          :aria-label="t('sidebar.help')"
          :title="t('sidebar.help')"
          @click="handleSidebarClick('help')"
        >
          <HelpCircle :size="22" />
        </button>
        <!-- GitHub -->
        <button
          type="button"
          class="app-sidebar__item"
          :class="{ 'app-sidebar__item--has-update': hasNewVersion }"
          aria-label="GitHub"
          :title="hasNewVersion ? t('sidebar.newVersionAvailable', { version: latestVersion }) : 'GitHub'"
          @click="handleOpenGitHub"
        >
          <Github :size="22" />
          <span v-if="hasNewVersion" class="app-sidebar__update-badge">
            <ArrowUp :size="10" />
          </span>
        </button>
      </aside>
      <main class="app-main">
        <div
          v-if="route.path === '/project' && currentProject"
          class="project-layout"
        >
          <div
            class="project-layout__left"
            :class="{ 'project-layout__left--expanded': isImageListExpanded }"
          >
            <div class="project-layout__left-top">
              <div class="project-layout__left-top-row">
                <span class="project-layout__section-title">
                  {{ t('project.sidebar.imageList') }}
                </span>
                <button
                  type="button"
                  class="project-layout__left-select-all-btn"
                  :disabled="!projectImages.length"
                  :title="t('project.sidebar.selectAll')"
                  @click="handleSelectAllImages"
                >
                  {{ t('project.sidebar.selectAll') }}
                </button>
              </div>
              <div class="project-image-filter-tabs">
                <button
                  type="button"
                  class="project-image-filter-tab"
                  :class="{ 'project-image-filter-tab--active': activeImageFilterTab === 'all' }"
                  @click="activeImageFilterTab = 'all'"
                >
                  {{ t('project.images.filters.all') }}
                </button>
                <button
                  type="button"
                  class="project-image-filter-tab"
                  :class="{ 'project-image-filter-tab--active': activeImageFilterTab === 'annotated' }"
                  @click="activeImageFilterTab = 'annotated'"
                >
                  {{ t('project.images.filters.annotated') }}
                </button>
                <button
                  type="button"
                  class="project-image-filter-tab"
                  :class="{ 'project-image-filter-tab--active': activeImageFilterTab === 'unannotated' }"
                  @click="activeImageFilterTab = 'unannotated'"
                >
                  {{ t('project.images.filters.unannotated') }}
                </button>
                <button
                  type="button"
                  class="project-image-filter-tab"
                  :class="{ 'project-image-filter-tab--active': activeImageFilterTab === 'negative' }"
                  @click="activeImageFilterTab = 'negative'"
                >
                  {{ t('project.images.filters.negative') }}
                </button>
              </div>
            </div>
            <div
              ref="imageListRef"
              class="project-layout__left-bottom project-image-list"
              :class="{ 'project-image-list--expanded': isImageListExpanded }"
              @scroll="handleImageListScroll"
              @mousedown="handleImageListMouseDown"
              @click="handleImageListClick"
            >
              <div v-if="isImportingImages && projectImages.length === 0" class="project-loading">
                <div class="project-loading__spinner">
                  <div class="project-loading__ring project-loading__ring--cw"></div>
                  <div class="project-loading__ring project-loading__ring--ccw"></div>
                </div>
              </div>
              <div v-else-if="projectImages.length === 0" class="project-empty">
                <ImageOff :size="22" class="project-empty__icon" />
                <p class="project-empty__text">
                  {{ t('project.sidebar.noImages') }}
                </p>
              </div>
              <!-- 筛选后无结果 -->
              <div v-else-if="filteredProjectImages.length === 0" class="project-empty">
                <FilterX :size="22" class="project-empty__icon" />
                <p class="project-empty__text">
                  {{ t('project.sidebar.noFilteredImages') }}
                </p>
              </div>
              <div v-else class="project-image-virtual">
                <div
                  class="project-image-virtual__spacer"
                  :style="{
                    paddingTop: imageListPaddingTop + 'px',
                    paddingBottom: imageListPaddingBottom + 'px'
                  }"
                >
                  <div
                    v-for="row in visibleImageRows"
                    :key="row.index"
                    class="project-image-row"
                    :class="{ 'project-image-row--expanded': isImageListExpanded }"
                  >
                    <div
                      v-for="item in row.items"
                      :key="item.id"
                      class="project-image-cell"
                      :class="{ 
                        'project-image-cell--selected': isImageSelected(item.id),
                        'project-image-cell--active': activeImage?.id === item.id
                      }"
                      :ref="(el) => registerImageCell(el as HTMLElement | null, item.id)"
                      @click.stop="(event) => handleImageCellClick(item, event as MouseEvent)"
                      @contextmenu.prevent.stop="(event) => openImageContextMenu(event as MouseEvent, item)"
                    >
                      <div
                        class="project-image-placeholder"
                        :class="{
                          'project-image-placeholder--hidden':
                            isImageLoaded(item.id) || isImageError(item.id)
                        }"
                      ></div>
                      <img
                        v-if="shouldLoadImagesForRow(row.index)"
                        :src="getImageThumbUrl(item)"
                        class="project-image-thumb"
                        loading="lazy"
                        decoding="async"
                        draggable="false"
                        @load="markImageLoaded(item.id)"
                        @error="markImageError(item.id)"
                      />
                      <!-- 标注状态徽章 -->
                      <span 
                        class="project-image-badge"
                        :class="{
                          'project-image-badge--annotated': item.annotationStatus === 'annotated',
                          'project-image-badge--unannotated': item.annotationStatus === 'none',
                          'project-image-badge--negative': item.annotationStatus === 'negative'
                        }"
                      >{{ t('project.images.badge.' + item.annotationStatus) }}</span>
                    </div>
                  </div>
                </div>
              </div>
                            <div
                v-if="isImageContextMenuOpen"
                class="project-image-context-menu-backdrop"
                @click="closeImageContextMenu"
              >
                <div
                  class="project-image-context-menu"
                  :style="{ left: imageContextMenuX + 'px', top: imageContextMenuY + 'px' }"
                  @click.stop
                >
                  <button
                    type="button"
                    class="project-image-context-menu__item"
                    @click="openDeleteImagesDialogFromContextMenu"
                  >
                    <span v-if="imageContextMenuUseSelection">
                      {{
                        t('project.images.contextMenu.deleteMultiple', {
                          count: imageContextMenuSelectionIds.length
                        })
                      }}
                    </span>
                    <span v-else>
                      {{ t('project.images.contextMenu.deleteSingle') }}
                    </span>
                  </button>
                </div>
              </div>
            </div>
            <button
              type="button"
              class="project-image-expand-toggle"
              :aria-label="
                isImageListExpanded
                  ? t('project.sidebar.collapseImageList')
                  : t('project.sidebar.expandImageList')
              "
              :title="
                isImageListExpanded
                  ? t('project.sidebar.collapseImageList')
                  : t('project.sidebar.expandImageList')
              "
              @click="toggleImageListExpanded"
            >
              <span class="project-image-expand-toggle__icon">
                {{ isImageListExpanded ? '<' : '>' }}
              </span>
            </button>
          </div>
          <div class="project-layout__right">
            <div class="project-layout__right-main">
              <AnnotationCanvas
                ref="annotationCanvasRef"
                v-if="activeImage"
                :image="activeImage"
                :image-url="getImageOriginalUrl(activeImage)"
                :categories="projectCategories"
                :selected-category="selectedCategory"
                :selected-keypoint-category="selectedKeypointCategory"
                :annotations="currentAnnotations"
                @update:annotations="currentAnnotations = $event"
                @save="handleSaveAnnotations"
                @save-as-negative="handleSaveAsNegative"
                @prev="handlePrevImage"
                @next="handleNextImage"
                @notify="handleAnnotationNotify"
                @annotation-complete="handleAnnotationComplete"
                @prompt-click="handlePromptClick"
              />
              <div v-else class="project-right-logo">
                <img :src="logo" alt="" class="project-right-logo__img" />
              </div>
            </div>
            <aside class="project-layout__right-panel">
              <div class="category-panel">
                <div class="category-panel__header">
		          <div class="category-panel__tabs">
		            <button
		              type="button"
		              class="category-panel__tab"
		              :class="{ 'category-panel__tab--active': activeCategoryTab === 'bbox' }"
		              @click="activeCategoryTab = 'bbox'"
		            >
		              {{ t('project.categoryPanel.tabs.bbox') }}
		            </button>
		            <button
		              type="button"
		              class="category-panel__tab"
		              :class="{ 'category-panel__tab--active': activeCategoryTab === 'keypoint' }"
		              @click="activeCategoryTab = 'keypoint'"
		            >
		              {{ t('project.categoryPanel.tabs.keypoint') }}
		            </button>
		            <button
		              type="button"
		              class="category-panel__tab"
		              :class="{ 'category-panel__tab--active': activeCategoryTab === 'polygon' }"
		              @click="activeCategoryTab = 'polygon'"
		            >
		              {{ t('project.categoryPanel.tabs.polygon') }}
		            </button>
		            <button
		              type="button"
		              class="category-panel__tab"
		              :class="{ 'category-panel__tab--active': activeCategoryTab === 'category' }"
		              @click="activeCategoryTab = 'category'"
		            >
		              {{ t('project.categoryPanel.tabs.category') }}
		            </button>
		          </div>
                </div>
                <div class="category-panel__body">
                  <!-- 分类类别：占位图 -->
                  <div v-if="activeCategoryTab === 'category'" class="category-panel__coming-soon">
                    <Clock :size="36" class="category-panel__coming-soon-icon" />
                    <p class="category-panel__coming-soon-title">{{ t('project.categoryPanel.comingSoon') }}</p>
                    <p class="category-panel__coming-soon-desc">{{ t('project.categoryPanel.comingSoonDesc') }}</p>
                  </div>
                  <!-- 边界框/多边形类别 -->
                  <template v-else>
				  <div class="category-panel__body-header">
					<button
					  type="button"
					  class="category-panel__add-btn"
					  :disabled="!currentProject"
					  @click.stop="openCreateCategory"
					>
					  {{ t('project.categoryPanel.addCategory') }}
					</button>
				  </div>
				  <div
					v-if="isCreatingCategory"
					ref="categoryEditorRef"
					class="category-editor-card"
				  >
					<button
					  ref="categoryColorButtonRef"
					  type="button"
					  class="category-editor-card__color"
					  :style="{ backgroundColor: newCategoryColor }"
					  @click.stop="handleCategoryColorClick"
					>
					</button>
					<div v-if="isCategoryColorPickerOpen" class="category-color-palette">
					  <input
					    type="color"
					    class="category-color-palette__native"
					    :value="newCategoryColor"
					    @input.stop="handleNativeColorChange"
					  />
					  <div class="category-color-palette__swatches">
					    <button
					      v-for="color in presetCategoryColors"
					      :key="color"
					      type="button"
					      class="category-color-palette__color"
					      :style="{ backgroundColor: color }"
					      @click.stop="selectPresetCategoryColor(color)"
					    ></button>
					  </div>
					</div>
					<input
					  v-model="newCategoryName"
					  type="text"
					  class="category-editor-card__input"
					  :placeholder="t('project.categoryPanel.namePlaceholder')"
					  @keyup.enter.stop="submitCreateCategory"
					/>
					<button
					  type="button"
					  class="category-editor-card__confirm"
					  @click.stop="submitCreateCategory"
					>
					  {{ t('project.categoryPanel.confirm') }}
					</button>
				  </div>
				  <p v-if="editCategoryError" class="project-modal__error">
					{{ editCategoryError }}
				  </p>
	              <div v-if="filteredCategories.length === 0" class="category-panel__empty">
	                <List :size="22" class="category-panel__empty-icon" />
	                <p class="category-panel__empty-text">
	                  {{ t('project.categoryPanel.empty') }}
	                </p>
	              </div>
				  <ul v-else class="category-list">
					<li
					  v-for="cat in filteredCategories"
					  :key="cat.id"
					  :class="[
					    'category-list-item',
					    {
					      'category-list-item--selected': (cat.type === 'keypoint' ? selectedKeypointCategory?.id === cat.id : selectedCategory?.id === cat.id),
					      'category-list-item--context':
					        isCategoryContextMenuOpen &&
					        categoryContextMenuTarget &&
					        categoryContextMenuTarget.id === cat.id
					    },
					    {
					      'category-list-item--dragging': draggingCategoryId === cat.id,
					      'category-list-item--dragover': dragOverCategoryId === cat.id && draggingCategoryId !== null
					    }
					  ]"
					  draggable="true"
					  @click="handleSelectCategory(cat)"
					  @dragstart="(e) => handleCategoryDragStart(e as DragEvent, cat)"
					  @dragover="(e) => handleCategoryDragOver(e as DragEvent, cat)"
					  @drop="(e) => handleCategoryDrop(e as DragEvent, cat)"
					  @dragend="handleCategoryDragEnd"
					  @contextmenu="(e) => openCategoryContextMenu(e as MouseEvent, cat)"
					>
					  <span
					    class="category-list-item__color"
					    :style="{ backgroundColor: cat.color }"
					    @click.stop="openEditCategoryFromColor(cat)"
					  ></span>
					  <div
					    v-if="editCategoryTarget && editCategoryTarget.id === cat.id && isEditCategoryColorPickerOpen"
					    class="category-color-palette"
					  >
					    <input
					      type="color"
					      class="category-color-palette__native"
					      :value="editCategoryColor"
					      @input.stop="handleEditNativeColorChange"
					    />
					    <div class="category-color-palette__swatches">
					      <button
					        v-for="color in presetCategoryColors"
					        :key="color"
					        type="button"
					        class="category-color-palette__color"
					        :style="{ backgroundColor: color }"
					        @click.stop="selectPresetEditCategoryColor(color)"
					      ></button>
					    </div>
					  </div>
					  <span
					    v-if="!(editCategoryTarget && editCategoryTarget.id === cat.id && isEditingCategoryName)"
					    class="category-list-item__name"
					    :title="cat.name"
					  >
					    {{ cat.name }}
					  </span>
					  <input
					    v-else
					    v-model="editCategoryName"
					    type="text"
					    class="category-editor-card__input"
					    :placeholder="t('project.categoryPanel.editCategory.namePlaceholder')"
					    @keyup.enter.stop="() => submitEditCategory()"
					  />
					  <span
					    v-if="cat.type === 'keypoint'"
					    class="category-list-item__badge"
					    :class="{ 'category-list-item__badge--empty': getKeypointCount(cat) === 0 }"
					  >
					    {{ getKeypointCount(cat) > 0
					      ? t('project.categoryPanel.keypointBadge', { count: getKeypointCount(cat) })
					      : t('project.categoryPanel.keypointBadgeEmpty')
					    }}
					  </span>
					</li>
				  </ul>
				  <!-- 类别右键菜单 -->
				  <div
				    v-if="isCategoryContextMenuOpen"
				    class="category-context-menu-backdrop"
				    @click="closeCategoryContextMenu"
				  >
				    <div
				      class="category-context-menu"
				      :style="{ left: categoryContextMenuX + 'px', top: categoryContextMenuY + 'px' }"
				      @click.stop
				    >
				      <button
				        v-if="categoryContextMenuTarget && categoryContextMenuTarget.type === 'keypoint'"
				        type="button"
				        class="category-context-menu__item"
				        @click="openKeypointConfigModal"
				      >
				        {{ t('project.categoryPanel.contextMenu.configureKeypoints') }}
				      </button>
				      <button
				        type="button"
				        class="category-context-menu__item"
				        @click="openEditCategoryFromContextMenu"
				      >
				        {{ t('project.categoryPanel.contextMenu.editCategory') }}
				      </button>
				      <button
				        type="button"
				        class="category-context-menu__item category-context-menu__item--danger"
				        @click="openDeleteCategoryDialog"
				      >
				        {{ t('project.categoryPanel.contextMenu.deleteCategory') }}
				      </button>
				    </div>
				  </div>
                  </template>
                </div>
              </div>
            </aside>
          </div>
        </div>
        <div v-else class="app-content">
          <router-view :key="$route.fullPath" />
        </div>
        <SettingsView v-if="isSettingsOpen" />
      </main>
    </div>
	    <footer class="app-footer">
	      <!-- 左侧统计信息：仅在项目页且打开项目时显示 -->
	      <div class="app-footer__left">
			<template v-if="route.path === '/project' && currentProject && footerProjectStats">
			  <span class="app-footer__stat">{{ t('footer.totalImages') }}: {{ footerProjectStats.total }}</span>
			  <span class="app-footer__stat">{{ t('footer.annotated') }}: {{ footerProjectStats.annotated }}</span>
			  <span class="app-footer__stat">{{ t('footer.unannotated') }}: {{ footerProjectStats.unannotated }}</span>
			  <span class="app-footer__stat">{{ t('footer.negative') }}: {{ footerProjectStats.negative }}</span>
			</template>
			<template v-if="route.path === '/project' && currentProject && footerActiveImageInfo">
			  <span class="app-footer__divider">|</span>
			  <span class="app-footer__stat">{{ t('footer.imageStatus') }}: {{ t('footer.status.' + footerActiveImageInfo.status) }}</span>
			  <span class="app-footer__stat">{{ t('footer.annotationCount') }}: {{ footerActiveImageInfo.annotationCount }}</span>
			</template>
		  </div>
      <div class="app-footer__right">
        <!-- 自动保存按钮：仅在项目页且打开项目时显示 -->
        <button
          v-if="route.path === '/project' && currentProject"
          type="button"
          class="app-footer__btn"
          :class="{ 'app-footer__btn--active': isAutoSaveEnabled }"
          :title="t('footer.autoSave')"
          @click="isAutoSaveEnabled = !isAutoSaveEnabled"
        >
          <Save :size="14" />
        </button>
        <!-- 推理插件栏：仅在项目页且打开项目时显示 -->
        <InferencePluginBar v-if="route.path === '/project' && currentProject" />
        <!-- 通知按钮：始终显示 -->
        <button
          type="button"
          class="app-footer__notify"
          :class="{ 'app-footer__notify--active': isNotificationPanelOpen }"
          aria-label="Notifications"
          @click="toggleNotificationPanel"
        >
          <Bell :size="14" />
          <span v-if="hasPersistentNotifications && !isNotificationPanelOpen" class="app-footer__notify-badge"></span>
        </button>
      </div>
    </footer>

	<div v-if="isDeleteImagesModalOpen" class="project-modal-overlay">
		<div class="project-modal">
			<h2 class="project-modal__title">
				{{ t('project.images.deleteConfirmTitle') }}
			</h2>
			<p class="project-modal__desc">
				{{
					t('project.images.deleteConfirmMessage', {
						count: deleteTargetImageIds.length
					})
				}}
			</p>
			<p v-if="hasExternalInDeleteTargets" class="project-modal__desc project-modal__desc--warning">
				{{ t('project.images.deleteExternalNote') }}
			</p>
			<div class="project-modal__actions">
				<button
					type="button"
					class="project-modal__button"
					@click="closeDeleteImagesDialog"
				>
					{{ t('project.images.deleteCancelButton') }}
				</button>
				<button
					type="button"
					class="project-modal__button project-modal__button--primary"
					@click="confirmDeleteImages"
				>
					{{ t('project.images.deleteConfirmButton') }}
				</button>
			</div>
		</div>
	</div>

    <!-- 通知面板 -->
    <Transition name="notification-panel">
      <div v-if="isNotificationPanelOpen" class="notification-panel">
        <div class="notification-panel__header">
          <h3 class="notification-panel__title">{{ t('notifications.panelTitle') }} {{ notifications.length > 0 ? `(${notifications.length})` : '' }}</h3>
          <div class="notification-panel__actions">
            <button
              type="button"
              class="notification-panel__action"
              :aria-label="t('notifications.clear')"
              :title="t('notifications.clear')"
              :disabled="notifications.length === 0"
              @click="clearAllNotifications"
            >
              <Trash2 :size="14" />
            </button>
            <button
              type="button"
              class="notification-panel__action"
              :aria-label="t('notifications.collapse')"
              :title="t('notifications.collapse')"
              @click="toggleNotificationPanel"
            >
              <ChevronDown :size="14" />
            </button>
          </div>
        </div>
        <div v-if="notifications.length > 0" class="notification-panel__content">
          <TransitionGroup name="notification-list">
            <NotificationItem
              v-for="notification in notifications"
              :key="notification.id"
              :notification="notification"
            />
          </TransitionGroup>
        </div>
      </div>
    </Transition>

    <div v-if="isCreateProjectModalOpen" class="project-modal-overlay">
      <div class="project-modal">
        <h2 class="project-modal__title">{{ t('project.createModal.title') }}</h2>
        <label class="project-modal__field">
          <span class="project-modal__label">{{ t('project.createModal.nameLabel') }}</span>
          <input
            v-model="newProjectName"
            type="text"
            class="project-modal__input"
            :placeholder="t('project.createModal.namePlaceholder')"
            @keyup.enter="submitCreateProject"
          />
        </label>
        <p v-if="createProjectError" class="project-modal__error">{{ createProjectError }}</p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button project-modal__button--primary"
            :disabled="isCreatingProject"
            @click="submitCreateProject"
          >
            {{ t('project.createModal.create') }}
          </button>
          <button
            type="button"
            class="project-modal__button"
            :disabled="isCreatingProject"
            @click="closeCreateProjectModal"
          >
            {{ t('project.createModal.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="isImportImagesModalOpen" class="project-modal-overlay">
      <div class="import-modal">
        <h2 class="project-modal__title">
          {{ t('project.importImagesModal.title') }}
        </h2>
        <p class="import-modal__desc">
          {{ t('project.importImagesModal.description') }}
        </p>
        <div class="import-modal__mode-group">
          <div class="import-modal__mode-label">
            {{ t('project.importImagesModal.modeLabel') }}
          </div>
          <p class="import-modal__mode-hint">
            {{ t('project.importImagesModal.modeHint') }}
          </p>
          <div class="import-modal__mode-options">
            <label
              class="import-modal__mode-option"
              :class="{ 'import-modal__mode-option--active': importMode === 'copy' }"
            >
              <input
                v-model="importMode"
                type="radio"
                value="copy"
              />
              <span class="import-modal__mode-option-text">
                {{ t('project.importImagesModal.modeCopy') }}
              </span>
            </label>
            <label
              class="import-modal__mode-option"
              :class="{ 'import-modal__mode-option--active': importMode === 'link' }"
            >
              <input
                v-model="importMode"
                type="radio"
                value="link"
              />
              <span class="import-modal__mode-option-text">
                {{ t('project.importImagesModal.modeLink') }}
              </span>
            </label>
            <label
              class="import-modal__mode-option"
              :class="{ 'import-modal__mode-option--active': importMode === 'external' }"
            >
              <input
                v-model="importMode"
                type="radio"
                value="external"
              />
              <span class="import-modal__mode-option-text">
                {{ t('project.importImagesModal.modeExternal') }}
              </span>
            </label>
          </div>
          <p v-if="importMode === 'external'" class="import-modal__mode-warning">
            {{ t('project.importImagesModal.modeExternalWarning') }}
          </p>
        </div>
        <div class="import-modal__actions">
          <button
            type="button"
            class="import-modal__button"
            @click="handleImportFromDirectory"
          >
            {{ t('project.importImagesModal.byDirectory') }}
          </button>
          <button
            type="button"
            class="import-modal__button"
            @click="handleImportFromFiles"
          >
            {{ t('project.importImagesModal.byFiles') }}
          </button>
        </div>
        <p v-if="importImagesError" class="project-modal__error">
          {{ importImagesError }}
        </p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button"
            @click="closeImportImagesModal"
          >
            {{ t('project.importImagesModal.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 导入数据集弹窗 -->
    <div v-if="isImportDatasetModalOpen" class="project-modal-overlay">
      <div class="import-modal">
        <h2 class="project-modal__title">
          {{ t('project.importDataset.title') }}
        </h2>
        
        <!-- 步骤1: 选择数据集目录 -->
        <template v-if="importDatasetStep === 'select'">
          <p class="project-modal__desc">{{ t('project.importDataset.selectDesc') }}</p>
          <div class="import-modal__path-row">
            <input
              type="text"
              class="import-modal__path-input"
              :value="importDatasetPath"
              readonly
              :placeholder="t('project.importDataset.pathPlaceholder')"
            />
            <button
              type="button"
              class="project-modal__button project-modal__button--primary"
              @click="handleSelectDatasetPath"
            >
              {{ t('project.importDataset.browse') }}
            </button>
          </div>
        </template>

        <!-- 步骤2: 正在探测 -->
        <template v-else-if="importDatasetStep === 'detecting'">
          <div class="import-modal__detecting">
            <Loader2 :size="24" class="import-modal__loading-icon animate-spin" />
            <p>{{ t('project.importDataset.detecting') }}</p>
          </div>
        </template>

        <!-- 步骤3: 配置导入 -->
        <template v-else-if="importDatasetStep === 'configure'">
          <p class="project-modal__desc">{{ t('project.importDataset.configureDesc') }}</p>
          <div class="import-modal__path-display">
            <Folder :size="16" />
            <span>{{ importDatasetPath }}</span>
          </div>
          <div class="import-modal__plugin-list">
            <label
              v-for="plugin in detectedPlugins"
              :key="plugin.pluginId"
              class="import-modal__plugin-item"
              :class="{ 'import-modal__plugin-item--selected': selectedPluginId === plugin.pluginId }"
            >
              <input
                v-model="selectedPluginId"
                type="radio"
                :value="plugin.pluginId"
                class="import-modal__plugin-radio"
              />
              <div class="import-modal__plugin-info">
                <span class="import-modal__plugin-name">{{ plugin.pluginId }}</span>
                <span class="import-modal__plugin-reason">{{ plugin.reason }}</span>
              </div>
            </label>
          </div>
          
          <!-- 图片导入模式选择 -->
          <div class="import-modal__mode-group">
            <div class="import-modal__mode-label">{{ t('project.importImagesModal.modeLabel') }}</div>
            <p class="import-modal__mode-hint">{{ t('project.importImagesModal.modeHint') }}</p>
            <div class="import-modal__mode-options">
              <label
                class="import-modal__mode-option"
                :class="{ 'import-modal__mode-option--active': importDatasetMode === 'copy' }"
              >
                <input v-model="importDatasetMode" type="radio" value="copy" />
                <span class="import-modal__mode-option-text">{{ t('project.importImagesModal.modeCopy') }}</span>
              </label>
              <label
                class="import-modal__mode-option"
                :class="{ 'import-modal__mode-option--active': importDatasetMode === 'link' }"
              >
                <input v-model="importDatasetMode" type="radio" value="link" />
                <span class="import-modal__mode-option-text">{{ t('project.importImagesModal.modeLink') }}</span>
              </label>
              <label
                class="import-modal__mode-option"
                :class="{ 'import-modal__mode-option--active': importDatasetMode === 'external' }"
              >
                <input v-model="importDatasetMode" type="radio" value="external" />
                <span class="import-modal__mode-option-text">{{ t('project.importImagesModal.modeExternal') }}</span>
              </label>
            </div>
            <p v-if="importDatasetMode === 'external'" class="import-modal__mode-warning">
              {{ t('project.importImagesModal.modeExternalWarning') }}
            </p>
          </div>

          <div class="import-modal__actions">
            <button
              type="button"
              class="project-modal__button"
              @click="closeImportDatasetModal"
            >
              {{ t('project.importDataset.cancel') }}
            </button>
            <button
              type="button"
              class="project-modal__button project-modal__button--primary"
              :disabled="!selectedPluginId"
              @click="executeDatasetImport"
            >
              {{ t('project.importDataset.import') }}
            </button>
          </div>
        </template>

        <!-- 步骤4: 正在导入 -->
        <template v-else-if="importDatasetStep === 'importing'">
          <div class="import-modal__detecting">
            <Loader2 :size="24" class="import-modal__loading-icon animate-spin" />
            <p>{{ t('project.importDataset.importing') }}</p>
          </div>
        </template>

        <p v-if="importDatasetError" class="project-modal__error">
          {{ importDatasetError }}
        </p>

        <div v-if="importDatasetStep === 'select'" class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button"
            @click="closeImportDatasetModal"
          >
            {{ t('project.importDataset.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 关键点配置弹窗 -->
    <div v-if="isKeypointConfigModalOpen" class="project-modal-overlay">
      <div class="keypoint-config-modal">
        <h2 class="project-modal__title">
          {{ keypointConfigTarget
            ? t('project.categoryPanel.keypointConfig.titleWithName', { name: keypointConfigTarget.name })
            : t('project.categoryPanel.keypointConfig.title')
          }}
        </h2>
        <p class="keypoint-config-modal__desc">
          {{ t('project.categoryPanel.keypointConfig.description') }}
        </p>
        <div class="keypoint-config-modal__list">
          <div
            v-for="(kp, index) in keypointConfigList"
            :key="index"
            class="keypoint-config-modal__item"
          >
            <span class="keypoint-config-modal__index">{{ index + 1 }}</span>
            <input
              v-model="kp.name"
              type="text"
              class="keypoint-config-modal__input"
              :placeholder="t('project.categoryPanel.keypointConfig.namePlaceholder')"
            />
            <button
              type="button"
              class="keypoint-config-modal__remove"
              :disabled="keypointConfigList.length <= 1"
              @click="removeKeypointConfigItem(index)"
            >
              <X :size="14" />
            </button>
          </div>
        </div>
        <button
          type="button"
          class="keypoint-config-modal__add"
          :disabled="keypointConfigList.length >= 64"
          @click="addKeypointConfigItem"
        >
          <Plus :size="14" />
          {{ t('project.categoryPanel.keypointConfig.addKeypoint') }}
        </button>
        <!-- 绑定矩形框类别 -->
        <div class="keypoint-config-modal__bind">
          <label class="keypoint-config-modal__bind-label">
            {{ t('project.categoryPanel.keypointConfig.bindBbox') }}
          </label>
          <select v-model="keypointConfigBindBboxId" class="keypoint-config-modal__bind-select">
            <option :value="null">{{ t('project.categoryPanel.keypointConfig.noBind') }}</option>
            <option v-for="bbox in availableBboxCategories" :key="bbox.id" :value="bbox.id">
              {{ bbox.name }}
            </option>
          </select>
        </div>
        <p v-if="keypointConfigError" class="project-modal__error">{{ keypointConfigError }}</p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button project-modal__button--primary"
            :disabled="isKeypointConfigSaving"
            @click="submitKeypointConfig"
          >
            {{ t('project.categoryPanel.keypointConfig.save') }}
          </button>
          <button
            type="button"
            class="project-modal__button"
            :disabled="isKeypointConfigSaving"
            @click="closeKeypointConfigModal"
          >
            {{ t('project.categoryPanel.keypointConfig.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除类别确认弹窗 -->
    <div v-if="isDeleteCategoryModalOpen" class="project-modal-overlay">
      <div class="project-modal">
        <h2 class="project-modal__title">
          {{ t('project.categoryPanel.deleteCategory.title') }}
        </h2>
        <p class="project-modal__desc">
          {{
            t('project.categoryPanel.deleteCategory.message', {
              name: deleteCategoryTarget ? deleteCategoryTarget.name : ''
            })
          }}
        </p>
        <p class="project-modal__desc project-modal__desc--warning">
          {{ t('project.categoryPanel.deleteCategory.warningMessage') }}
        </p>
        <p v-if="deleteCategoryError" class="project-modal__error">{{ deleteCategoryError }}</p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button project-modal__button--primary"
            :disabled="isDeletingCategory"
            @click="submitDeleteCategory"
          >
            {{ t('project.categoryPanel.deleteCategory.confirm') }}
          </button>
          <button
            type="button"
            class="project-modal__button"
            :disabled="isDeletingCategory"
            @click="closeDeleteCategoryModal"
          >
            {{ t('project.categoryPanel.deleteCategory.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 类别合并确认弹窗 -->
    <div v-if="isMergeCategoryModalOpen" class="project-modal-overlay">
      <div class="project-modal">
        <h2 class="project-modal__title">
          {{ t('project.categoryPanel.mergeCategory.title') }}
        </h2>
        <p class="project-modal__desc project-modal__desc--warning">
          {{ t('project.categoryPanel.mergeCategory.message', { name: mergeCategoryTargetName }) }}
        </p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button project-modal__button--primary"
            :disabled="isMergingCategory"
            @click="confirmMergeCategory"
          >
            {{ t('project.categoryPanel.mergeCategory.confirm') }}
          </button>
          <button
            type="button"
            class="project-modal__button"
            :disabled="isMergingCategory"
            @click="cancelMergeCategory"
          >
            {{ t('project.categoryPanel.mergeCategory.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 项目右键菜单 -->
    <div
      v-if="isProjectContextMenuOpen"
      class="project-context-menu-backdrop"
      @click="closeProjectContextMenu"
    >
      <div
        class="project-context-menu"
        :style="{ left: projectContextMenuX + 'px', top: projectContextMenuY + 'px' }"
        @click.stop
      >
        <button type="button" class="project-context-menu__item" @click="openRenameProjectModal">
          <Pencil :size="14" />
          <span>{{ t('project.contextMenu.rename') }}</span>
        </button>
        <button type="button" class="project-context-menu__item project-context-menu__item--danger" @click="openDeleteProjectModal">
          <Trash2 :size="14" />
          <span>{{ t('project.contextMenu.delete') }}</span>
        </button>
      </div>
    </div>

    <!-- 删除项目确认弹窗 -->
    <div v-if="isDeleteProjectModalOpen" class="project-modal-overlay">
      <div class="project-modal">
        <h2 class="project-modal__title">{{ t('project.deleteModal.title') }}</h2>
        <p class="project-modal__desc project-modal__desc--warning">
          {{ t('project.deleteModal.warning') }}
        </p>
        <p class="project-modal__desc">
          {{ t('project.deleteModal.message', { name: projectContextMenuTarget?.name || '' }) }}
        </p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button project-modal__button--danger"
            :disabled="isDeletingProject"
            @click="submitDeleteProject"
          >
            {{ t('project.deleteModal.confirm') }}
          </button>
          <button
            type="button"
            class="project-modal__button"
            :disabled="isDeletingProject"
            @click="closeDeleteProjectModal"
          >
            {{ t('project.deleteModal.cancel') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 重命名项目弹窗 -->
    <div v-if="isRenameProjectModalOpen" class="project-modal-overlay">
      <div class="project-modal">
        <h2 class="project-modal__title">{{ t('project.renameModal.title') }}</h2>
        <label class="project-modal__field">
          <span>{{ t('project.renameModal.nameLabel') }}</span>
          <input
            v-model="renameProjectName"
            type="text"
            :placeholder="t('project.renameModal.namePlaceholder')"
            @keyup.enter="submitRenameProject"
          />
        </label>
        <p v-if="renameProjectError" class="project-modal__error">{{ renameProjectError }}</p>
        <div class="project-modal__actions">
          <button
            type="button"
            class="project-modal__button project-modal__button--primary"
            :disabled="isRenamingProject"
            @click="submitRenameProject"
          >
            {{ t('project.renameModal.confirm') }}
          </button>
          <button
            type="button"
            class="project-modal__button"
            :disabled="isRenamingProject"
            @click="closeRenameProjectModal"
          >
            {{ t('project.renameModal.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.project-modal-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.project-modal {
  width: 360px;
  max-width: calc(100vw - 40px);
  padding: 16px 18px 14px;
  border-radius: 6px;
  background-color: var(--color-bg-header);
  border: 1px solid var(--color-border-subtle);
  box-shadow: 0 18px 45px rgba(0, 0, 0, 0.45);
}

.project-modal__title {
  margin: 0 0 12px;
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--color-fg);
}

.project-modal__field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.project-modal__label {
  font-size: 0.8rem;
  color: var(--color-fg-muted);
}

.project-modal__input {
  padding: 6px 8px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background-color: var(--color-bg-app);
  color: var(--color-fg);
  font-size: 0.8rem;
}

.project-modal__input:focus-visible {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 1px var(--color-accent-soft);
}

.project-modal__error {
  margin: 6px 0 0;
  font-size: 0.75rem;
  color: #f97373;
}

.project-modal__actions {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.project-modal__button {
  min-width: 64px;
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle);
  background-color: var(--color-bg-sidebar);
  color: var(--color-fg);
  font-size: 0.8rem;
  cursor: pointer;
}

.project-modal__button--primary {
  border-color: var(--color-accent);
  background-color: var(--color-accent);
  color: #ffffff;
}

.project-modal__button:disabled {
  opacity: 0.5;
  cursor: default;
}

.project-modal__button--danger {
  border-color: #dc3545;
  background-color: #dc3545;
  color: #ffffff;
}

.project-modal__desc--warning {
  color: #f97373;
  font-weight: 500;
}

/* 项目右键菜单 */
.project-context-menu-backdrop {
  position: fixed;
  inset: 0;
  z-index: 100;
}

.project-context-menu {
  position: fixed;
  min-width: 120px;
  padding: 4px 0;
  background-color: var(--color-bg-header);
  border: 1px solid var(--color-border-subtle);
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 101;
}

.project-context-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 12px;
  border: none;
  background: transparent;
  color: var(--color-fg);
  font-size: 0.8rem;
  text-align: left;
  cursor: pointer;
}

.project-context-menu__item:hover {
  background-color: var(--color-bg-sidebar-hover);
}

.project-context-menu__item--danger {
  color: #f97373;
}

.project-context-menu__item--danger:hover {
  background-color: rgba(249, 115, 115, 0.15);
}

.import-modal {
  width: 360px;
  max-width: calc(100vw - 40px);
  padding: 16px 18px 14px;
  border-radius: 0;
  background-color: var(--color-bg-header);
  border: 1px solid var(--color-border-subtle);
  box-shadow: 0 18px 45px rgba(0, 0, 0, 0.45);
}

.import-modal__desc {
  margin: 4px 0 10px;
  font-size: 0.8rem;
  color: var(--color-fg-muted);
}

.import-modal__actions {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.import-modal__button {
  width: 100%;
  padding: 4px 10px;
  border-radius: 0;
  border: 1px solid var(--color-border-subtle);
  background-color: var(--color-bg-sidebar);
  color: var(--color-fg);
  font-size: 0.8rem;
  text-align: left;
  cursor: pointer;
}

.import-modal__button:hover {
  background-color: var(--color-bg-sidebar-hover);
}

.import-modal__mode-group {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: 4px;
  background-color: var(--color-bg-app);
  border: 1px solid var(--color-border-subtle);
}

.import-modal__mode-label {
  font-size: 0.78rem;
  color: var(--color-fg-muted);
  margin-bottom: 4px;
}

.import-modal__mode-hint {
  margin: 0 0 6px;
  font-size: 0.75rem;
  color: var(--color-fg-muted);
}

.import-modal__mode-options {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.import-modal__mode-option {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid transparent;
  background-color: transparent;
  font-size: 0.8rem;
  color: var(--color-fg);
  cursor: pointer;
}

.import-modal__mode-option input[type='radio'] {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.import-modal__mode-option-text {
  flex: 1;
}

.import-modal__mode-option--active {
  border-color: var(--color-accent);
  background-color: var(--color-accent-soft);
}

.import-modal__mode-option--active .import-modal__mode-option-text {
  font-weight: 500;
}

.import-modal__mode-warning {
  margin: 6px 0 0;
  font-size: 0.75rem;
  color: #f97373;
}

.import-modal__path-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.import-modal__path-input {
  flex: 1;
  padding: 8px 12px;
  background: var(--color-bg-app);
  border: 1px solid var(--color-border-subtle);
  border-radius: 4px;
  color: var(--color-fg);
  font-size: 0.85rem;
}

.import-modal__detecting {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px;
  gap: 12px;
  color: var(--color-fg-muted);
}

.import-modal__loading-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.import-modal__path-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--color-bg-app);
  border-radius: 4px;
  color: var(--color-fg-muted);
  font-size: 0.85rem;
  margin-bottom: 12px;
}

.import-modal__plugin-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.import-modal__plugin-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  background: var(--color-bg-app);
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 0.15s, background-color 0.15s;
}

.import-modal__plugin-item:hover {
  background: var(--color-bg-sidebar-hover);
}

.import-modal__plugin-item--selected {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}

.import-modal__plugin-radio {
  margin-top: 2px;
}

.import-modal__plugin-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.import-modal__plugin-name {
  font-weight: 500;
  font-size: 0.9rem;
}

.import-modal__plugin-reason {
  font-size: 0.8rem;
  color: var(--color-fg-muted);
}

.project-image-list {
  position: relative;
}

.project-image-virtual {
  height: 100%;
}

.project-image-virtual__spacer {
  position: relative;
}

.project-image-row {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 6px;
	padding: 4px 2px;
	box-sizing: border-box;
}

.project-image-row--expanded {
	grid-template-columns: repeat(5, minmax(0, 1fr));
}

.project-image-cell {
  flex: 1 1 50%;
  aspect-ratio: 4 / 3;
  position: relative;
  overflow: hidden;
  border-radius: 2px;
}

.project-image-cell--selected::after {
	content: '';
	position: absolute;
	inset: 0;
	border: 3px solid #3b82f6;
	background-color: rgba(59, 130, 246, 0.15);
	box-sizing: border-box;
	pointer-events: none;
}

.project-image-expand-toggle {
	position: absolute;
	top: 50%;
	right: -10px;
	transform: translateY(-50%);
	width: 18px;
	height: 44px;
	border-radius: 3px;
	border: 1px solid var(--color-border-subtle);
	background-color: var(--color-bg-sidebar);
	color: var(--color-fg-muted);
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	z-index: 5;
}

.project-image-expand-toggle:hover {
	background-color: var(--color-bg-sidebar-hover);
}

.project-image-selection-rect {
	position: fixed;
	border: 1px solid rgba(0, 120, 215, 0.9);
	background-color: rgba(0, 120, 215, 0.18);
	pointer-events: none;
	z-index: 1000;
}

.project-image-placeholder {
	position: absolute;
	inset: 0;
	border-radius: 2px;
	background: linear-gradient(135deg, #4b5563, #374151);
	opacity: 0.85;
}

.project-image-thumb {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	object-fit: cover;
	display: block;
}

/* 图片当前打开状态（区别于多选，使用更粗的边框） */
.project-image-cell--active::after {
	content: '';
	position: absolute;
	inset: 0;
	border: 4px solid #3b82f6;
	background-color: rgba(59, 130, 246, 0.2);
	box-sizing: border-box;
	pointer-events: none;
	z-index: 2;
}

/* 图片状态徽章 */
.project-image-badge {
	position: absolute;
	top: 3px;
	right: 3px;
	padding: 1px 4px;
	border-radius: 2px;
	font-size: 0.6rem;
	font-weight: 500;
	color: #fff;
	z-index: 2;
	white-space: nowrap;
}

.project-image-badge--annotated {
	background-color: #10b981;
}

.project-image-badge--unannotated {
	background-color: #f59e0b;
}

.project-image-badge--negative {
	background-color: #ef4444;
}

/* 侧边栏 Python 环境角标 */
.app-sidebar__icon-wrapper {
	position: relative;
	display: inline-flex;
	align-items: center;
	justify-content: center;
}

.app-sidebar__badge {
	position: absolute;
	top: -2px;
	right: -2px;
	width: 8px;
	height: 8px;
	border-radius: 999px;
	box-shadow: 0 0 0 2px var(--color-bg-sidebar, #111827);
}

.app-sidebar__badge--warning {
	background: var(--color-warning, #f59e0b);
}

/* 底部栏统计样式 */
.app-footer__stat {
	font-size: 0.75rem;
	color: var(--color-fg-muted);
	margin-right: 12px;
}

.app-footer__divider {
	color: var(--color-border-subtle);
	margin: 0 8px;
}

/* 筛选标签间距调整 */
.project-image-filter-tabs {
	gap: 2px !important;
}

/* Footer 右侧按钮组 */
.app-footer__right {
	display: flex;
	align-items: center;
	gap: 4px;
}

.app-footer__btn {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 24px;
	height: 24px;
	padding: 0;
	border: none;
	background: transparent;
	color: var(--color-fg-muted);
	cursor: pointer;
	border-radius: 4px;
	transition: background 0.15s, color 0.15s;
	position: relative;
}

.app-footer__btn:hover {
	background: var(--color-bg-tertiary);
	color: var(--color-fg-primary);
}

.app-footer__btn--active {
	background: rgba(59, 130, 246, 0.2);
	color: #3b82f6;
}

.app-footer__btn--active:hover {
	background: rgba(59, 130, 246, 0.3);
	color: #3b82f6;
}
</style>
