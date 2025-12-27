<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Minus, Plus, Maximize, Maximize2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

// ==================== 类型定义 ====================
export type CategoryType = 'bbox' | 'keypoint' | 'polygon' | 'category'

export type ProjectCategory = {
  id: number
  name: string
  type: CategoryType
  color: string
  sortOrder: number
  mate: string
}

export type KeypointMate = {
  keypoints: { id: number; name: string }[]
}

export type ProjectImageListItem = {
  id: number
  filename: string
  hasThumb: boolean
  isExternal: boolean
  thumbPath: string
  originalPath: string
  annotationStatus: 'none' | 'annotated' | 'negative'
}

// 归一化坐标的标注数据
// 关键点格式: [x, y, v] 其中 v: 0=不存在, 1=存在但不可见, 2=可见
export type BboxAnnotationData = {
  x: number; y: number; width: number; height: number; rotation?: number
  keypoints?: [number, number, number][] // 关键点作为 bbox 的扩展字段
  keypointCategoryId?: number // 关键点对应的类别ID
}

export type PolygonAnnotationData = {
  points: [number, number][]
}

export type AnnotationData = BboxAnnotationData | PolygonAnnotationData

export type Annotation = {
  id: string
  dbId?: number
  imageId: number
  categoryId: number
  type: CategoryType
  data: AnnotationData
  isInference?: boolean // 是否是推理标注（用虚线显示）
  confidence?: number // 推理置信度
}

// 推理结果标注类型
export type InferenceAnnotation = {
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

// ==================== Props & Emits ====================
const props = defineProps<{
  image: ProjectImageListItem | null
  imageUrl: string
  categories: ProjectCategory[]
  selectedCategory: ProjectCategory | null // 主类别（bbox 或 polygon）
  selectedKeypointCategory: ProjectCategory | null // 副类别（keypoint），可选
  annotations: Annotation[]
  classificationTags?: { id: number; name: string }[] // 分类标签列表
}>()

// 计算当前图片的分类标签（用于显示）
const displayClassificationTags = computed(() => props.classificationTags || [])

const emit = defineEmits<{
  (e: 'update:annotations', annotations: Annotation[]): void
  (e: 'save'): void
  (e: 'saveAsNegative'): void
  (e: 'prev'): void
  (e: 'next'): void
  (e: 'notify', payload: { type: 'info' | 'warning' | 'error'; message: string }): void
  (e: 'annotationComplete'): void  // 用户完成标注操作（创建/调整/删除）
  (e: 'promptClick', payload: { x: number; y: number; type: 'positive' | 'negative'; append?: boolean }): void  // SAM-2 提示点击
}>()

const { t } = useI18n()

// ==================== Refs ====================
const containerRef = ref<HTMLDivElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const imageRef = ref<HTMLImageElement | null>(null)
const ctx = ref<CanvasRenderingContext2D | null>(null)

// 图片状态
const imageNaturalWidth = ref(0)
const imageNaturalHeight = ref(0)
const imageLoaded = ref(false)

// 缩放与平移
const scale = ref(1)
const translate = ref({ x: 0, y: 0 })
const minScale = 0.1
const maxScale = 10
const hasUserZoomed = ref(false) // 用户是否手动缩放过，如果没有则窗口调整时自动适应

// 交互状态
const isSpacePressed = ref(false)
const isPanning = ref(false)
const panLastPoint = ref<{ x: number; y: number } | null>(null)
const isAltPressed = ref(false)
const isShiftPressed = ref(false)  // Shift+点击用于 SAM-2 提示
const isCtrlPressed = ref(false)

// 绘制状态
type DrawingMode = 'none' | 'bbox' | 'keypoint' | 'polygon'
const drawingMode = ref<DrawingMode>('none')
const isDrawing = ref(false)
const drawStartPoint = ref<{ x: number; y: number } | null>(null)
const drawCurrentPoint = ref<{ x: number; y: number } | null>(null)
const polygonTempPoints = ref<[number, number][]>([])
const keypointTempPoints = ref<[number, number, number][]>([])
const keypointDrawingBboxId = ref<string | null>(null)

// 选中状态
const selectedAnnotationId = ref<string | null>(null)
const selectedPointIndex = ref<number | null>(null)
const hoveredAnnotationId = ref<string | null>(null)

// 拖拽状态
const isDragging = ref(false)
const dragStartPoint = ref<{ x: number; y: number } | null>(null)
const dragStartAnnotationData = ref<AnnotationData | null>(null)

// 调整大小状态
type ResizeHandle = 'nw' | 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w' | 'rotate'
const isResizing = ref(false)
const resizeHandle = ref<ResizeHandle | null>(null)
const resizeStartPoint = ref<{ x: number; y: number } | null>(null)
const resizeStartData = ref<BboxAnnotationData | null>(null)

// 右键菜单
const isContextMenuOpen = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuAnnotationId = ref<string | null>(null)
const contextMenuPointIndex = ref<number | null>(null)

// 多边形边悬停状态（用于添加点）
const hoveredEdgeInfo = ref<{ annotationId: string; edgeIndex: number; point: { x: number; y: number } } | null>(null)

// 鼠标在画布内的位置（用于绘制十字准线）
const mouseInCanvas = ref(false)
const mousePosition = ref<{ x: number; y: number } | null>(null)

// ResizeObserver 用于监听容器大小变化
let resizeObserver: ResizeObserver | null = null

// ==================== 画布尺寸（响应式） ====================
const canvasWidth = ref(800)
const canvasHeight = ref(600)

const imageDisplayRect = computed(() => {
  if (!imageLoaded.value || !imageNaturalWidth.value || !imageNaturalHeight.value) {
    return { x: 0, y: 0, width: 0, height: 0 }
  }
  const cw = canvasWidth.value
  const ch = canvasHeight.value
  const iw = imageNaturalWidth.value
  const ih = imageNaturalHeight.value
  const fitScale = Math.min(cw / iw, ch / ih)
  const displayWidth = iw * fitScale * scale.value
  const displayHeight = ih * fitScale * scale.value
  const x = (cw - displayWidth) / 2 + translate.value.x
  const y = (ch - displayHeight) / 2 + translate.value.y
  return { x, y, width: displayWidth, height: displayHeight }
})

const getCategoryById = (id: number) => props.categories.find(c => c.id === id)

const parseKeypointMate = (mate: string): KeypointMate | null => {
  if (!mate) return null
  try {
    const obj = JSON.parse(mate) as KeypointMate
    return obj?.keypoints ? obj : null
  } catch { return null }
}

// ==================== 坐标转换 ====================
const canvasToNormalized = (cx: number, cy: number) => {
  const r = imageDisplayRect.value
  if (r.width === 0 || r.height === 0) return { x: 0, y: 0 }
  return { x: (cx - r.x) / r.width, y: (cy - r.y) / r.height }
}

const normalizedToCanvas = (nx: number, ny: number) => {
  const r = imageDisplayRect.value
  return { x: r.x + nx * r.width, y: r.y + ny * r.height }
}

const eventToCanvas = (e: MouseEvent) => {
  const canvas = canvasRef.value
  if (!canvas) return { x: 0, y: 0 }
  const rect = canvas.getBoundingClientRect()
  return { x: e.clientX - rect.left, y: e.clientY - rect.top }
}

// ==================== 生成唯一 ID ====================
const generateId = () => `ann_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`

// ==================== 标注操作 ====================
const updateAnnotations = (newAnnotations: Annotation[]) => {
  emit('update:annotations', newAnnotations)
}

const addAnnotation = (annotation: Annotation) => {
  updateAnnotations([...props.annotations, annotation])
}

const updateAnnotation = (id: string, data: Partial<AnnotationData>) => {
  const updated = props.annotations.map(a => a.id === id ? { ...a, data: { ...a.data, ...data } } : a)
  updateAnnotations(updated)
}

const deleteAnnotation = (id: string, triggerComplete = true) => {
  updateAnnotations(props.annotations.filter(a => a.id !== id))
  if (selectedAnnotationId.value === id) selectedAnnotationId.value = null
  if (triggerComplete) emit('annotationComplete')
}

const selectAnnotation = (id: string | null) => {
  if (selectedAnnotationId.value !== id) {
    selectedAnnotationId.value = id
    selectedPointIndex.value = null
    render()
  }
}

// ==================== 碰撞检测 ====================
const getAnnotationAtPoint = (p: { x: number; y: number }): Annotation | null => {
  const anns = props.annotations
  // 优先检测关键点（因为关键点是小圆点，需要优先检测）
  for (let i = anns.length - 1; i >= 0; i--) {
    const ann = anns[i]
    if (ann && ann.type === 'keypoint' && isPointInAnnotation(p, ann)) return ann
  }
  // 检测多边形顶点（顶点可能在多边形外部，所以需要单独检测）
  for (let i = anns.length - 1; i >= 0; i--) {
    const ann = anns[i]
    if (ann && ann.type === 'polygon') {
      if (isPointNearPolygonVertex(p, ann)) return ann
    }
  }
  // 然后检测其他类型（多边形内部、bbox等）
  for (let i = anns.length - 1; i >= 0; i--) {
    const ann = anns[i]
    if (ann && ann.type !== 'keypoint' && isPointInAnnotation(p, ann)) return ann
  }
  return null
}

// 检测点是否在多边形的任何顶点附近
const isPointNearPolygonVertex = (p: { x: number; y: number }, ann: Annotation): boolean => {
  if (ann.type !== 'polygon') return false
  const d = ann.data as PolygonAnnotationData
  for (const pt of d.points) {
    if (!pt) continue
    const cp = normalizedToCanvas(pt[0], pt[1])
    // 使用较大的检测半径，确保顶点外部也可以点击
    if (Math.hypot(p.x - cp.x, p.y - cp.y) <= POLYGON_VERTEX_RADIUS_SELECTED + 3) return true
  }
  return false
}

const isPointInAnnotation = (p: { x: number; y: number }, ann: Annotation): boolean => {
  if (ann.type === 'bbox') {
    const d = ann.data as BboxAnnotationData
    // 先检查关键点（优先级更高）
    if (d.keypoints) {
      for (const pt of d.keypoints) {
        if (!pt) continue
        const [nx, ny] = pt
        const cp = normalizedToCanvas(nx, ny)
        if (Math.hypot(p.x - cp.x, p.y - cp.y) < 12) return true
      }
    }
    // 再检查矩形框范围
    const tl = normalizedToCanvas(d.x, d.y)
    const br = normalizedToCanvas(d.x + d.width, d.y + d.height)
    return p.x >= tl.x && p.x <= br.x && p.y >= tl.y && p.y <= br.y
  }
  if (ann.type === 'polygon') {
    const d = ann.data as PolygonAnnotationData
    const pts = d.points.map(([nx, ny]) => normalizedToCanvas(nx, ny))
    return isPointInPolygon(p, pts)
  }
  return false
}

// 获取 bbox 关键点中被点击的具体点索引（包括v=0的不存在点）
const getKeypointPointAtPoint = (ann: Annotation, p: { x: number; y: number }): number | null => {
  if (ann.type !== 'bbox') return null
  const d = ann.data as BboxAnnotationData
  if (!d.keypoints) return null
  for (let i = 0; i < d.keypoints.length; i++) {
    const pt = d.keypoints[i]
    if (!pt) continue
    const [nx, ny] = pt
    const cp = normalizedToCanvas(nx, ny)
    if (Math.hypot(p.x - cp.x, p.y - cp.y) < 12) return i
  }
  return null
}

// 全局搜索：检测点击位置是否命中任何 bbox 的关键点（优先级高于标注框本身）
const getAnyKeypointAtPoint = (p: { x: number; y: number }): { annotation: Annotation; pointIndex: number } | null => {
  // 倒序遍历（后绘制的在上层，优先选中）
  for (let i = props.annotations.length - 1; i >= 0; i--) {
    const ann = props.annotations[i]
    if (!ann || ann.type !== 'bbox') continue
    const ki = getKeypointPointAtPoint(ann, p)
    if (ki !== null) return { annotation: ann, pointIndex: ki }
  }
  return null
}

const isPointInPolygon = (p: { x: number; y: number }, pts: { x: number; y: number }[]): boolean => {
  let inside = false
  for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) {
    const pi = pts[i], pj = pts[j]
    if (!pi || !pj) continue
    const xi = pi.x, yi = pi.y
    const xj = pj.x, yj = pj.y
    if (((yi > p.y) !== (yj > p.y)) && (p.x < (xj - xi) * (p.y - yi) / (yj - yi) + xi)) {
      inside = !inside
    }
  }
  return inside
}

// 多边形顶点显示半径常量（与绘制保持一致）
const POLYGON_VERTEX_RADIUS = 5
const POLYGON_VERTEX_RADIUS_SELECTED = 7

const getPolygonPointAtPoint = (ann: Annotation, p: { x: number; y: number }): number | null => {
  if (ann.type !== 'polygon') return null
  const d = ann.data as PolygonAnnotationData
  for (let i = 0; i < d.points.length; i++) {
    const pt = d.points[i]
    if (!pt) continue
    const cp = normalizedToCanvas(pt[0], pt[1])
    // 使用较大的检测半径，确保顶点整个圆形区域都可以点击
    const detectRadius = POLYGON_VERTEX_RADIUS_SELECTED + 3
    if (Math.hypot(p.x - cp.x, p.y - cp.y) <= detectRadius) return i
  }
  return null
}

// 检测点到线段的距离，返回最近点
const pointToSegmentDistance = (p: { x: number; y: number }, a: { x: number; y: number }, b: { x: number; y: number }): { distance: number; closestPoint: { x: number; y: number } } => {
  const dx = b.x - a.x, dy = b.y - a.y
  const len2 = dx * dx + dy * dy
  if (len2 === 0) return { distance: Math.hypot(p.x - a.x, p.y - a.y), closestPoint: { x: a.x, y: a.y } }
  let t = ((p.x - a.x) * dx + (p.y - a.y) * dy) / len2
  t = Math.max(0, Math.min(1, t))
  const closestPoint = { x: a.x + t * dx, y: a.y + t * dy }
  return { distance: Math.hypot(p.x - closestPoint.x, p.y - closestPoint.y), closestPoint }
}

// 检测鼠标是否在多边形的边上（用于添加点）
const getPolygonEdgeAtPoint = (ann: Annotation, p: { x: number; y: number }): { edgeIndex: number; point: { x: number; y: number } } | null => {
  if (ann.type !== 'polygon') return null
  const d = ann.data as PolygonAnnotationData
  if (d.points.length < 3) return null
  const pts = d.points.map(([nx, ny]) => normalizedToCanvas(nx, ny))
  const threshold = 8
  for (let i = 0; i < pts.length; i++) {
    const a = pts[i], b = pts[(i + 1) % pts.length]
    if (!a || !b) continue
    const { distance, closestPoint } = pointToSegmentDistance(p, a, b)
    if (distance < threshold) {
      // 确保不是在顶点附近（使用顶点显示半径）
      if (Math.hypot(p.x - a.x, p.y - a.y) > POLYGON_VERTEX_RADIUS_SELECTED && Math.hypot(p.x - b.x, p.y - b.y) > POLYGON_VERTEX_RADIUS_SELECTED) {
        return { edgeIndex: i, point: closestPoint }
      }
    }
  }
  return null
}

const getResizeHandleAtPoint = (p: { x: number; y: number }): ResizeHandle | null => {
  if (!selectedAnnotationId.value) return null
  const ann = props.annotations.find(a => a.id === selectedAnnotationId.value)
  if (!ann || ann.type !== 'bbox') return null
  const d = ann.data as BboxAnnotationData
  const tl = normalizedToCanvas(d.x, d.y)
  const br = normalizedToCanvas(d.x + d.width, d.y + d.height)
  const w = br.x - tl.x, h = br.y - tl.y
  const handles: { x: number; y: number; type: ResizeHandle }[] = [
    { x: tl.x, y: tl.y, type: 'nw' },
    { x: tl.x + w / 2, y: tl.y, type: 'n' },
    { x: br.x, y: tl.y, type: 'ne' },
    { x: br.x, y: tl.y + h / 2, type: 'e' },
    { x: br.x, y: br.y, type: 'se' },
    { x: tl.x + w / 2, y: br.y, type: 's' },
    { x: tl.x, y: br.y, type: 'sw' },
    { x: tl.x, y: tl.y + h / 2, type: 'w' },
    { x: tl.x + w / 2, y: tl.y - 30, type: 'rotate' }
  ]
  for (const h of handles) {
    if (Math.hypot(p.x - h.x, p.y - h.y) < 8) return h.type
  }
  return null
}

// ==================== 图片加载 ====================
const onImageLoad = () => {
  const img = imageRef.value
  if (!img) return
  imageNaturalWidth.value = img.naturalWidth
  imageNaturalHeight.value = img.naturalHeight
  imageLoaded.value = true
  nextTick(() => { initCanvas(); render() })
}

const initCanvas = () => {
  const canvas = canvasRef.value
  const container = containerRef.value
  if (!canvas || !container) return
  // 更新响应式尺寸
  canvasWidth.value = container.clientWidth
  canvasHeight.value = container.clientHeight
  // 更新 canvas 实际尺寸
  canvas.width = canvasWidth.value
  canvas.height = canvasHeight.value
  ctx.value = canvas.getContext('2d')
}

// ==================== 绘制 ====================
const render = () => {
  const canvas = canvasRef.value
  const context = ctx.value
  const img = imageRef.value
  if (!canvas || !context || !img || !imageLoaded.value) return
  context.clearRect(0, 0, canvas.width, canvas.height)
  const rect = imageDisplayRect.value
  context.drawImage(img, rect.x, rect.y, rect.width, rect.height)
  // 绘制所有标注（包括推理标注），选中的标注最后绘制（置于顶层）
  const selectedId = selectedAnnotationId.value
  for (const ann of props.annotations) {
    if (ann.id !== selectedId) drawAnnotation(context, ann)
  }
  // 最后绘制选中的标注
  if (selectedId) {
    const selectedAnn = props.annotations.find(a => a.id === selectedId)
    if (selectedAnn) drawAnnotation(context, selectedAnn)
  }
  drawCurrentDrawing(context)
  drawCrosshair(context)
}

// 绘制鼠标十字准线
const drawCrosshair = (c: CanvasRenderingContext2D) => {
  if (!mouseInCanvas.value || !mousePosition.value) return
  const canvas = canvasRef.value
  if (!canvas) return
  const { x, y } = mousePosition.value
  c.save()
  c.setLineDash([4, 4])
  c.strokeStyle = 'rgba(255, 255, 255, 0.8)'
  c.lineWidth = 1
  // 垂直线
  c.beginPath()
  c.moveTo(x, 0)
  c.lineTo(x, canvas.height)
  c.stroke()
  // 水平线
  c.beginPath()
  c.moveTo(0, y)
  c.lineTo(canvas.width, y)
  c.stroke()
  c.restore()
}

const drawAnnotation = (c: CanvasRenderingContext2D, ann: Annotation) => {
  // 推理标注可能没有对应的类别，使用蓝色作为默认
  let cat = getCategoryById(ann.categoryId)
  const isInf = ann.isInference === true
  if (!cat) {
    if (isInf) {
      // 为推理标注创建临时类别对象
      const infAnn = ann as Annotation & { _categoryName?: string }
      cat = { id: -1, name: infAnn._categoryName || '未知', type: ann.type, color: '#3b82f6' } as ProjectCategory
    } else {
      return
    }
  }
  const sel = selectedAnnotationId.value === ann.id
  const hov = hoveredAnnotationId.value === ann.id
  if (ann.type === 'bbox') drawBbox(c, ann, cat, sel, hov, isInf)
  else if (ann.type === 'polygon') drawPolygon(c, ann, cat, sel, hov, isInf)
}

const drawBbox = (c: CanvasRenderingContext2D, ann: Annotation, cat: ProjectCategory, sel: boolean, hov: boolean, isInf = false) => {
  const d = ann.data as BboxAnnotationData
  const tl = normalizedToCanvas(d.x, d.y)
  const br = normalizedToCanvas(d.x + d.width, d.y + d.height)
  const w = br.x - tl.x, h = br.y - tl.y
  c.save()
  if (d.rotation) { c.translate(tl.x + w/2, tl.y + h/2); c.rotate(d.rotation); c.translate(-(tl.x + w/2), -(tl.y + h/2)) }
  // 推理标注使用虚线
  if (isInf) c.setLineDash([6, 4])
  c.strokeStyle = cat.color; c.lineWidth = sel ? 3 : hov ? 2.5 : 2
  c.strokeRect(tl.x, tl.y, w, h)
  c.fillStyle = cat.color + '20'; c.fillRect(tl.x, tl.y, w, h)
  c.setLineDash([])
  // 推理标注显示置信度
  const label = isInf && ann.confidence !== undefined ? `${cat.name} ${(ann.confidence * 100).toFixed(0)}%` : cat.name
  drawLabel(c, label, cat.color, tl.x, tl.y, w, h)
  if (sel) drawHandles(c, tl.x, tl.y, w, h, cat.color)
  c.restore()
  // 绘制关键点（如果有）
  drawBboxKeypoints(c, ann, sel)
}

const drawLabel = (c: CanvasRenderingContext2D, name: string, color: string, x: number, y: number, w: number, h: number) => {
  c.font = '12px sans-serif'
  const tw = c.measureText(name).width + 8, th = 18
  let lx = x, ly = y + h - th
  if (ly + th > canvasHeight.value || ly < 0) ly = y
  if (lx < 0) lx = x + w - tw
  c.fillStyle = color; c.fillRect(lx, ly, tw, th)
  c.fillStyle = '#fff'; c.textBaseline = 'middle'; c.fillText(name, lx + 4, ly + th/2)
}

// 多边形标签绘制：优先左下，其次左上，然后右下，最后右上
const drawPolygonLabel = (c: CanvasRenderingContext2D, name: string, color: string, pts: { x: number; y: number }[]) => {
  if (pts.length === 0) return
  c.font = '12px sans-serif'
  const tw = c.measureText(name).width + 8, th = 18
  // 找到各个角落的顶点
  let bottomLeft = pts[0], topLeft = pts[0], bottomRight = pts[0], topRight = pts[0]
  for (const p of pts) {
    if (!p) continue
    // 左下：x最小且y最大
    if (p.x < bottomLeft!.x || (p.x === bottomLeft!.x && p.y > bottomLeft!.y)) bottomLeft = p
    // 左上：x最小且y最小
    if (p.x < topLeft!.x || (p.x === topLeft!.x && p.y < topLeft!.y)) topLeft = p
    // 右下：x最大且y最大
    if (p.x > bottomRight!.x || (p.x === bottomRight!.x && p.y > bottomRight!.y)) bottomRight = p
    // 右上：x最大且y最小
    if (p.x > topRight!.x || (p.x === topRight!.x && p.y < topRight!.y)) topRight = p
  }
  // 按优先级尝试放置标签
  const candidates = [
    { anchor: bottomLeft, lx: bottomLeft!.x, ly: bottomLeft!.y }, // 左下
    { anchor: topLeft, lx: topLeft!.x, ly: topLeft!.y - th },      // 左上
    { anchor: bottomRight, lx: bottomRight!.x - tw, ly: bottomRight!.y }, // 右下
    { anchor: topRight, lx: topRight!.x - tw, ly: topRight!.y - th }  // 右上
  ]
  let lx = candidates[0]!.lx, ly = candidates[0]!.ly
  for (const cand of candidates) {
    const testLx = cand.lx, testLy = cand.ly
    // 检查是否在画布范围内
    if (testLx >= 0 && testLx + tw <= canvasWidth.value && testLy >= 0 && testLy + th <= canvasHeight.value) {
      lx = testLx; ly = testLy; break
    }
  }
  c.fillStyle = color; c.fillRect(lx, ly, tw, th)
  c.fillStyle = '#fff'; c.textBaseline = 'middle'; c.fillText(name, lx + 4, ly + th/2)
}

const drawHandles = (c: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, color: string) => {
  const sz = 8
  const pts: [number,number][] = [[x,y],[x+w/2,y],[x+w,y],[x+w,y+h/2],[x+w,y+h],[x+w/2,y+h],[x,y+h],[x,y+h/2]]
  c.beginPath(); c.moveTo(x+w/2,y); c.lineTo(x+w/2,y-30); c.strokeStyle = color; c.lineWidth = 1; c.stroke()
  c.beginPath(); c.arc(x+w/2,y-30,sz/2,0,Math.PI*2); c.fillStyle='#fff'; c.fill(); c.strokeStyle=color; c.lineWidth=2; c.stroke()
  for (const pt of pts) { c.beginPath(); c.rect(pt[0]-sz/2,pt[1]-sz/2,sz,sz); c.fillStyle='#fff'; c.fill(); c.strokeStyle=color; c.lineWidth=2; c.stroke() }
}

const drawPolygon = (c: CanvasRenderingContext2D, ann: Annotation, cat: ProjectCategory, sel: boolean, hov: boolean, isInf = false) => {
  const d = ann.data as PolygonAnnotationData
  if (d.points.length < 3) return
  const pts = d.points.map(([nx,ny]) => normalizedToCanvas(nx,ny))
  const first = pts[0]; if (!first) return
  c.save()
  // 推理标注使用虚线
  if (isInf) c.setLineDash([6, 4])
  c.beginPath(); c.moveTo(first.x, first.y)
  for (let i=1;i<pts.length;i++) { const p = pts[i]; if (p) c.lineTo(p.x, p.y) }
  c.closePath(); c.strokeStyle = cat.color; c.lineWidth = sel?3:hov?2.5:2; c.stroke()
  c.fillStyle = cat.color+'20'; c.fill()
  c.setLineDash([])
  // 推理标注显示置信度
  const label = isInf && ann.confidence !== undefined ? `${cat.name} ${(ann.confidence * 100).toFixed(0)}%` : cat.name
  drawPolygonLabel(c, label, cat.color, pts)
  c.restore()
  if (sel) {
    // 绘制顶点（使用常量保持一致）
    for (let i=0;i<pts.length;i++) { const p = pts[i]; if (!p) continue; const r=selectedPointIndex.value===i?POLYGON_VERTEX_RADIUS_SELECTED:POLYGON_VERTEX_RADIUS; c.beginPath(); c.arc(p.x,p.y,r,0,Math.PI*2); c.fillStyle='#fff'; c.fill(); c.strokeStyle=cat.color; c.lineWidth=2; c.stroke() }
    // 绘制边上的添加点指示器
    if (hoveredEdgeInfo.value && hoveredEdgeInfo.value.annotationId === ann.id) {
      const ep = hoveredEdgeInfo.value.point
      c.beginPath(); c.arc(ep.x, ep.y, 5, 0, Math.PI*2); c.fillStyle = cat.color; c.fill(); c.strokeStyle = '#fff'; c.lineWidth = 2; c.stroke()
      // 绘制加号
      c.strokeStyle = '#fff'; c.lineWidth = 1.5
      c.beginPath(); c.moveTo(ep.x - 3, ep.y); c.lineTo(ep.x + 3, ep.y); c.stroke()
      c.beginPath(); c.moveTo(ep.x, ep.y - 3); c.lineTo(ep.x, ep.y + 3); c.stroke()
    }
  }
}

// 解析矩形框类别的 mate，获取绑定的关键点类别 ID
const parseBboxMate = (mate: string): { keypointCategoryId?: number } | null => {
  if (!mate) return null
  try { return JSON.parse(mate) as { keypointCategoryId?: number } }
  catch { return null }
}

// 绘制 bbox 上的关键点
const drawBboxKeypoints = (c: CanvasRenderingContext2D, ann: Annotation, sel: boolean) => {
  if (ann.type !== 'bbox') return
  const d = ann.data as BboxAnnotationData
  if (!d.keypoints?.length) return
  
  // 获取关键点类别信息
  let kptCat: ProjectCategory | null = null
  
  // 方式1：直接使用标注数据中的 keypointCategoryId
  if (d.keypointCategoryId) {
    kptCat = props.categories.find(cat => cat.id === d.keypointCategoryId) || null
  }
  
  // 方式2：根据 bbox 类别查找绑定的关键点类别（适用于所有有关键点的 bbox 标注）
  if (!kptCat) {
    // 先找到 bbox 对应的类别
    let bboxCat: ProjectCategory | undefined
    if (ann.categoryId > 0) {
      bboxCat = props.categories.find(cat => cat.id === ann.categoryId)
    } else if (ann.isInference) {
      // 推理标注：使用 _categoryName 查找
      const infAnn = ann as Annotation & { _categoryName?: string }
      if (infAnn._categoryName) {
        bboxCat = props.categories.find(cat => cat.name === infAnn._categoryName && cat.type === 'bbox')
      }
    }
    // 从 bbox 类别的 mate 中获取绑定的关键点类别 ID
    if (bboxCat) {
      const bboxMate = parseBboxMate(bboxCat.mate)
      if (bboxMate?.keypointCategoryId) {
        kptCat = props.categories.find(cat => cat.id === bboxMate.keypointCategoryId) || null
      }
    }
  }
  
  const mate = kptCat ? parseKeypointMate(kptCat.mate) : null
  // 推理标注使用蓝色作为默认关键点颜色
  const color = kptCat?.color || '#3b82f6'
  
  for (let i = 0; i < d.keypoints.length; i++) {
    const pt = d.keypoints[i]; if (!pt) continue
    const [nx, ny, v] = pt
    const {x, y} = normalizedToCanvas(nx, ny)
    const r = sel && selectedPointIndex.value === i ? 8 : 6
    c.beginPath(); c.arc(x, y, r, 0, Math.PI * 2)
    // v=2 可见(实心), v=1 不可见/遮挡(空心实线), v=0 不存在(空心虚线)
    if (v === 2) {
      c.fillStyle = color; c.fill()
      c.strokeStyle = '#fff'; c.lineWidth = 2; c.setLineDash([]); c.stroke()
    } else if (v === 1) {
      // 不可见：空心实线
      c.strokeStyle = '#fff'; c.lineWidth = 2; c.setLineDash([]); c.stroke()
      c.beginPath(); c.arc(x, y, r - 2, 0, Math.PI * 2); c.strokeStyle = color; c.lineWidth = 2; c.stroke()
    } else {
      // 不存在(v=0)：空心虚线
      c.strokeStyle = '#fff'; c.lineWidth = 2; c.setLineDash([3, 3]); c.stroke()
      c.beginPath(); c.arc(x, y, r - 2, 0, Math.PI * 2); c.strokeStyle = color; c.lineWidth = 2; c.setLineDash([3, 3]); c.stroke()
      c.setLineDash([])
    }
    const name = mate?.keypoints[i]?.name || `${i + 1}`
    c.font = '10px sans-serif'; const tw = c.measureText(name).width + 4
    c.fillStyle = color + 'cc'; c.fillRect(x + 8, y - 18, tw, 14)
    c.fillStyle = '#fff'; c.textBaseline = 'middle'; c.fillText(name, x + 10, y - 11)
  }
}

const drawCurrentDrawing = (c: CanvasRenderingContext2D) => {
  if (!isDrawing.value) return
  
  // 绘制关键点时使用 selectedKeypointCategory 的颜色
  if (drawingMode.value === 'keypoint' && props.selectedKeypointCategory) {
    const color = props.selectedKeypointCategory.color
    const mate = parseKeypointMate(props.selectedKeypointCategory.mate)
    for(let i=0;i<keypointTempPoints.value.length;i++){
      const kpt = keypointTempPoints.value[i]; if (!kpt) continue
      const [nx,ny]=kpt; const {x,y}=normalizedToCanvas(nx,ny)
      c.beginPath();c.arc(x,y,6,0,Math.PI*2);c.fillStyle=color;c.fill();c.strokeStyle='#fff';c.lineWidth=2;c.stroke()
      const name=mate?.keypoints[i]?.name||`${i+1}`; c.font='10px sans-serif'; const tw=c.measureText(name).width+4
      c.fillStyle=color+'cc';c.fillRect(x+8,y-18,tw,14);c.fillStyle='#fff';c.textBaseline='middle';c.fillText(name,x+10,y-11)
    }
    return
  }
  
  // 绘制 bbox 和 polygon 时使用 selectedCategory 的颜色
  if (!props.selectedCategory) return
  const color = props.selectedCategory.color
  if (drawingMode.value === 'bbox' && drawStartPoint.value && drawCurrentPoint.value) {
    const x=Math.min(drawStartPoint.value.x,drawCurrentPoint.value.x), y=Math.min(drawStartPoint.value.y,drawCurrentPoint.value.y)
    const w=Math.abs(drawCurrentPoint.value.x-drawStartPoint.value.x), h=Math.abs(drawCurrentPoint.value.y-drawStartPoint.value.y)
    c.strokeStyle=color; c.lineWidth=2; c.setLineDash([5,5]); c.strokeRect(x,y,w,h); c.setLineDash([]); c.fillStyle=color+'20'; c.fillRect(x,y,w,h)
  } else if (drawingMode.value === 'polygon' && polygonTempPoints.value.length > 0 && drawCurrentPoint.value) {
    const pts = polygonTempPoints.value.map(([nx,ny])=>normalizedToCanvas(nx,ny))
    const first = pts[0]; if (!first) return
    c.beginPath(); c.moveTo(first.x,first.y); for(let i=1;i<pts.length;i++) { const p = pts[i]; if (p) c.lineTo(p.x,p.y) }
    c.lineTo(drawCurrentPoint.value.x,drawCurrentPoint.value.y); c.strokeStyle=color; c.lineWidth=2; c.setLineDash([5,5]); c.stroke(); c.setLineDash([])
    for(const p of pts){c.beginPath();c.arc(p.x,p.y,5,0,Math.PI*2);c.fillStyle='#fff';c.fill();c.strokeStyle=color;c.lineWidth=2;c.stroke()}
  }
}

// 检测是否点击在临时关键点上（绘制过程中）
const getTempKeypointAtPoint = (p: { x: number; y: number }): number | null => {
  if (!isDrawing.value || drawingMode.value !== 'keypoint') return null
  for (let i = 0; i < keypointTempPoints.value.length; i++) {
    const kpt = keypointTempPoints.value[i]
    if (!kpt) continue
    const cp = normalizedToCanvas(kpt[0], kpt[1])
    if (Math.hypot(p.x - cp.x, p.y - cp.y) < 12) return i
  }
  return null
}

// 拖拽临时关键点的状态
const draggingTempKeypointIndex = ref<number | null>(null)
const tempKeypointDragStart = ref<{ x: number; y: number } | null>(null)

// ==================== 事件处理 ====================
const handleMouseDown = (e: MouseEvent) => {
  closeContextMenu()
  if (e.button === 2) return
  if (e.button !== 0) return
  const p = eventToCanvas(e)
  if (isSpacePressed.value) { isPanning.value = true; panLastPoint.value = {x:e.clientX,y:e.clientY}; return }
  
  // 检查是否点击在临时关键点上（绘制过程中可以调整）
  if (isDrawing.value && drawingMode.value === 'keypoint') {
    const tempIdx = getTempKeypointAtPoint(p)
    if (tempIdx !== null) {
      draggingTempKeypointIndex.value = tempIdx
      tempKeypointDragStart.value = p
      return
    }
  }
  
  // ★ Shift+点击发送 SAM-2 提示点（左键正向，右键负向）
  if (isShiftPressed.value) {
    const normalized = canvasToNormalized(p.x, p.y)
    emit('promptClick', { x: normalized.x, y: normalized.y, type: 'positive', append: isCtrlPressed.value })
    return
  }
  
  // ★ Alt+点击创建关键点（使用 selectedKeypointCategory）
  if (isAltPressed.value && props.selectedKeypointCategory) {
    handleKeypointClick(p)
    return
  }
  // ★ Alt+点击创建多边形
  if (isAltPressed.value && props.selectedCategory?.type === 'polygon') {
    handlePolygonClick(p)
    return
  }
  
  // ★ 正在绘制多边形或关键点时，禁止选中其他标注
  if (isDrawing.value && (drawingMode.value === 'polygon' || drawingMode.value === 'keypoint')) {
    return
  }
  if (selectedAnnotationId.value) { const h = getResizeHandleAtPoint(p); if (h) { startResize(h, p); return } }
  
  // 检查是否点击在多边形边上（添加点并立即开始拖拽）
  if (hoveredEdgeInfo.value) {
    addPointToPolygonEdgeAndStartDrag(hoveredEdgeInfo.value.annotationId, hoveredEdgeInfo.value.edgeIndex, p)
    return
  }
  
  // ★ 多边形：如果已选中某个顶点，检查是否点击了顶点本身
  if (selectedAnnotationId.value && selectedPointIndex.value !== null) {
    const ann = props.annotations.find(a => a.id === selectedAnnotationId.value)
    if (ann && ann.type === 'polygon') {
      const pi = getPolygonPointAtPoint(ann, p)
      if (pi !== null) {
        // 点击了某个顶点
        if (pi === selectedPointIndex.value) {
          // 点击的是当前选中的顶点，开始拖拽
          startDragPoint(p)
          return
        } else {
          // 点击的是其他顶点，切换选中
          selectedPointIndex.value = pi
          startDragPoint(p)
          return
        }
      } else {
        // 点击在多边形内部但不在任何顶点上，切换到选中多边形本身
        if (isPointInAnnotation(p, ann)) {
          selectedPointIndex.value = null
          startDrag(ann, p)
          return
        }
      }
    }
  }
  
  // ★ 智能选择：优先检测关键点（即使被其他标注框覆盖）
  const keypointHit = getAnyKeypointAtPoint(p)
  if (keypointHit) {
    selectAnnotation(keypointHit.annotation.id)
    selectedPointIndex.value = keypointHit.pointIndex
    startDragPoint(p)
    return
  }
  
  const clicked = getAnnotationAtPoint(p)
  if (clicked) {
    selectAnnotation(clicked.id)
    // 多边形：选中具体的点并开始拖拽
    if (clicked.type === 'polygon') {
      const pi = getPolygonPointAtPoint(clicked, p)
      if (pi !== null) { selectedPointIndex.value = pi; startDragPoint(p); return }
    }
    startDrag(clicked, p)
    return
  }
  
  // ★ 点击空白处：立即取消选中任何标注
  selectAnnotation(null)
  
  // 如果有选中的类别是 bbox，且点击在图片范围内，开始绘制
  if (props.selectedCategory?.type === 'bbox') {
    const rect = imageDisplayRect.value
    if (p.x >= rect.x && p.x <= rect.x + rect.width && p.y >= rect.y && p.y <= rect.y + rect.height) {
      startBboxDraw(p)
    }
    return
  }
}

const handleMouseMove = (e: MouseEvent) => {
  const p = eventToCanvas(e)
  // 更新鼠标位置（用于十字准线）
  mousePosition.value = p
  if (isPanning.value && panLastPoint.value) {
    translate.value = {x:translate.value.x+e.clientX-panLastPoint.value.x, y:translate.value.y+e.clientY-panLastPoint.value.y}
    panLastPoint.value = {x:e.clientX,y:e.clientY}
    hasUserZoomed.value = true // 用户平移了视图
    render(); return
  }
  // 拖拽临时关键点（需要限制在父矩形框内）
  if (draggingTempKeypointIndex.value !== null) {
    let norm = canvasToNormalized(p.x, p.y)
    // 限制在图片范围内
    norm.x = Math.max(0, Math.min(1, norm.x))
    norm.y = Math.max(0, Math.min(1, norm.y))
    // 限制在父矩形框内
    if (keypointDrawingBboxId.value) {
      const parentAnn = props.annotations.find(a => a.id === keypointDrawingBboxId.value)
      if (parentAnn && parentAnn.type === 'bbox') {
        const bbox = parentAnn.data as BboxAnnotationData
        norm.x = Math.max(bbox.x, Math.min(bbox.x + bbox.width, norm.x))
        norm.y = Math.max(bbox.y, Math.min(bbox.y + bbox.height, norm.y))
      }
    }
    keypointTempPoints.value[draggingTempKeypointIndex.value] = [norm.x, norm.y, 1]
    render()
    return
  }
  if (isResizing.value) { handleResizeMove(p); return }
  if (isDragging.value) { handleDragMove(p); return }
  if (isDrawing.value) { drawCurrentPoint.value = p; render(); return }
  
  // 悬停检测
  const newHoveredId = getAnnotationAtPoint(p)?.id || null
  if (hoveredAnnotationId.value !== newHoveredId) {
    hoveredAnnotationId.value = newHoveredId
  }
  
  // 检测多边形边悬停（用于添加点）
  let newEdgeInfo: typeof hoveredEdgeInfo.value = null
  if (selectedAnnotationId.value) {
    const ann = props.annotations.find(a => a.id === selectedAnnotationId.value)
    if (ann && ann.type === 'polygon') {
      const edge = getPolygonEdgeAtPoint(ann, p)
      if (edge) {
        newEdgeInfo = { annotationId: ann.id, edgeIndex: edge.edgeIndex, point: edge.point }
      }
    }
  }
  // 更新边悬停状态
  const edgeChanged = (hoveredEdgeInfo.value?.edgeIndex !== newEdgeInfo?.edgeIndex) || 
                      (hoveredEdgeInfo.value?.annotationId !== newEdgeInfo?.annotationId)
  if (edgeChanged) {
    hoveredEdgeInfo.value = newEdgeInfo
  }
  
  updateCursor(p)
  // 十字准线需要实时更新，所以每次移动都渲染
  render()
}

// 鼠标进入/离开画布
const handleMouseEnter = () => {
  mouseInCanvas.value = true
}

const handleMouseLeave = () => {
  mouseInCanvas.value = false
  mousePosition.value = null
  render()
}

const handleMouseUp = (e: MouseEvent) => {
  if (e.button !== 0) return
  // 结束临时关键点拖拽
  if (draggingTempKeypointIndex.value !== null) {
    draggingTempKeypointIndex.value = null
    tempKeypointDragStart.value = null
    return
  }
  if (isPanning.value) { isPanning.value = false; panLastPoint.value = null; return }
  if (isResizing.value) { endResize(); return }
  if (isDragging.value) { endDrag(); return }
  if (isDrawing.value && drawingMode.value === 'bbox') { finishBboxDraw(); return }
}

const handleWheel = (e: WheelEvent) => {
  e.preventDefault()
  const factor = e.deltaY < 0 ? 1.1 : 0.9
  const newScale = Math.max(minScale, Math.min(maxScale, scale.value * factor))
  if (newScale === scale.value) return
  const canvas = canvasRef.value; if (!canvas) return
  const rect = canvas.getBoundingClientRect()
  const mx = e.clientX - rect.left, my = e.clientY - rect.top
  const ratio = newScale / scale.value
  // 缩放以鼠标位置为中心：使用相对于画布中心的坐标
  const centerX = canvas.width / 2, centerY = canvas.height / 2
  const mcx = mx - centerX, mcy = my - centerY
  translate.value = {
    x: mcx - ratio * (mcx - translate.value.x),
    y: mcy - ratio * (mcy - translate.value.y)
  }
  scale.value = newScale
  hasUserZoomed.value = true // 用户手动缩放了
  render()
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.code === 'Space' && !e.repeat) { e.preventDefault(); isSpacePressed.value = true }
  if (e.code === 'AltLeft' || e.code === 'AltRight') { e.preventDefault(); isAltPressed.value = true }
  if (e.code === 'ShiftLeft' || e.code === 'ShiftRight') { isShiftPressed.value = true }
  if (e.code === 'ControlLeft' || e.code === 'ControlRight') { isCtrlPressed.value = true }
  if (e.code === 'Escape') cancelCurrentDrawing()
  if (e.code === 'Delete' && selectedAnnotationId.value) deleteAnnotation(selectedAnnotationId.value)
  // V 键切换关键点可见性已移至全局快捷键系统
}

const handleKeyUp = (e: KeyboardEvent) => {
  if (e.code === 'Space') { isSpacePressed.value = false; if (isPanning.value) { isPanning.value = false; panLastPoint.value = null } }
  if (e.code === 'AltLeft' || e.code === 'AltRight') isAltPressed.value = false
  if (e.code === 'ShiftLeft' || e.code === 'ShiftRight') isShiftPressed.value = false
  if (e.code === 'ControlLeft' || e.code === 'ControlRight') isCtrlPressed.value = false
}

const handleBlur = () => { isSpacePressed.value = false; isAltPressed.value = false; isShiftPressed.value = false; isCtrlPressed.value = false; isPanning.value = false; panLastPoint.value = null }

const handleContextMenuEvent = (e: MouseEvent) => {
  e.preventDefault()
  const p = eventToCanvas(e)
  
  // ★ Shift+右键发送 SAM-2 负向提示点
  if (isShiftPressed.value) {
    const normalized = canvasToNormalized(p.x, p.y)
    emit('promptClick', { x: normalized.x, y: normalized.y, type: 'negative', append: isCtrlPressed.value })
    return
  }
  
  if (isDrawing.value && (drawingMode.value === 'polygon' || drawingMode.value === 'keypoint')) { cancelCurrentDrawing(); return }
  const clicked = getAnnotationAtPoint(p)
  if (clicked) {
    contextMenuAnnotationId.value = clicked.id
    // 多边形检测具体的点（用于删除单个点）
    if (clicked.type === 'polygon') {
      contextMenuPointIndex.value = getPolygonPointAtPoint(clicked, p)
    } else if (clicked.type === 'bbox') {
      // bbox 关键点检测具体的点（用于切换可见性）
      contextMenuPointIndex.value = getKeypointPointAtPoint(clicked, p)
    } else {
      contextMenuPointIndex.value = null
    }
    const cr = containerRef.value?.getBoundingClientRect()
    contextMenuX.value = e.clientX - (cr?.left || 0); contextMenuY.value = e.clientY - (cr?.top || 0)
    isContextMenuOpen.value = true
  }
}

const closeContextMenu = () => { isContextMenuOpen.value = false }

// 切换关键点可见性 (v=2 可见, v=1 不可见/遮挡) - 现在从bbox的keypoints字段读取
const toggleKeypointVisibility = (annId: string, pointIndex: number) => {
  const ann = props.annotations.find(a => a.id === annId)
  if (!ann || ann.type !== 'bbox') return
  const d = ann.data as BboxAnnotationData
  if (!d.keypoints) return
  const pt = d.keypoints[pointIndex]
  if (!pt) return
  const [nx, ny, v] = pt
  // 切换: 2 -> 1 -> 2
  const newV = v === 2 ? 1 : 2
  const newPts = d.keypoints.map((p: [number, number, number], i: number) => i === pointIndex ? [nx, ny, newV] as [number, number, number] : p)
  updateAnnotation(annId, { keypoints: newPts })
  render()
}

// 设置关键点为不存在 (v=0)
const setKeypointNotExist = (annId: string, pointIndex: number) => {
  const ann = props.annotations.find(a => a.id === annId)
  if (!ann || ann.type !== 'bbox') return
  const d = ann.data as BboxAnnotationData
  if (!d.keypoints) return
  const pt = d.keypoints[pointIndex]
  if (!pt) return
  const [nx, ny] = pt
  const newPts = d.keypoints.map((p: [number, number, number], i: number) => i === pointIndex ? [nx, ny, 0] as [number, number, number] : p)
  updateAnnotation(annId, { keypoints: newPts })
  selectedPointIndex.value = null // 取消选中
  render()
}

// 从右键菜单设置关键点不存在
const handleSetKeypointNotExistFromContextMenu = () => {
  if (!contextMenuAnnotationId.value || contextMenuPointIndex.value === null) return
  setKeypointNotExist(contextMenuAnnotationId.value, contextMenuPointIndex.value)
  closeContextMenu()
}

// 从右键菜单切换关键点可见性
const handleToggleKeypointVisibilityFromContextMenu = () => {
  if (!contextMenuAnnotationId.value || contextMenuPointIndex.value === null) return
  toggleKeypointVisibility(contextMenuAnnotationId.value, contextMenuPointIndex.value)
  closeContextMenu()
}

// 获取当前右键菜单关键点的可见性状态
const getContextMenuKeypointVisibility = (): number | null => {
  if (!contextMenuAnnotationId.value || contextMenuPointIndex.value === null) return null
  const ann = props.annotations.find(a => a.id === contextMenuAnnotationId.value)
  if (!ann || ann.type !== 'bbox') return null
  const d = ann.data as BboxAnnotationData
  if (!d.keypoints) return null
  const pt = d.keypoints[contextMenuPointIndex.value]
  if (!pt) return null
  return pt[2]
}

const handleDeleteFromContextMenu = () => {
  if (contextMenuAnnotationId.value) deleteAnnotation(contextMenuAnnotationId.value)
  closeContextMenu()
}

// 删除多边形中的单个点
const handleDeletePointFromContextMenu = () => {
  if (!contextMenuAnnotationId.value || contextMenuPointIndex.value === null) return
  const ann = props.annotations.find(a => a.id === contextMenuAnnotationId.value)
  if (!ann || ann.type !== 'polygon') { closeContextMenu(); return }
  
  const d = ann.data as PolygonAnnotationData
  if (d.points.length <= 3) {
    // 多边形至少需要3个点，删除后不足则删除整个标注
    deleteAnnotation(ann.id)
  } else {
    const newPts = d.points.filter((_, i) => i !== contextMenuPointIndex.value)
    updateAnnotation(ann.id, { points: newPts })
  }
  closeContextMenu()
}


// ==================== 矩形框绘制 ====================
// document 级别的事件处理（用于鼠标移出画布后继续绘制）
const handleDocumentMouseMove = (e: MouseEvent) => {
  if (!isDrawing.value || drawingMode.value !== 'bbox') return
  const canvas = canvasRef.value
  if (!canvas) return
  const rect = canvas.getBoundingClientRect()
  // 计算相对于画布的坐标
  let x = e.clientX - rect.left
  let y = e.clientY - rect.top
  // 限制在画布范围内
  x = Math.max(0, Math.min(canvas.width, x))
  y = Math.max(0, Math.min(canvas.height, y))
  // 进一步限制在图片范围内
  const imgRect = imageDisplayRect.value
  x = Math.max(imgRect.x, Math.min(imgRect.x + imgRect.width, x))
  y = Math.max(imgRect.y, Math.min(imgRect.y + imgRect.height, y))
  drawCurrentPoint.value = { x, y }
  render()
}

const handleDocumentMouseUp = (e: MouseEvent) => {
  if (e.button !== 0) return
  if (isDrawing.value && drawingMode.value === 'bbox') {
    finishBboxDraw()
  }
}

const addDocumentListeners = () => {
  document.addEventListener('mousemove', handleDocumentMouseMove)
  document.addEventListener('mouseup', handleDocumentMouseUp)
}

const removeDocumentListeners = () => {
  document.removeEventListener('mousemove', handleDocumentMouseMove)
  document.removeEventListener('mouseup', handleDocumentMouseUp)
}

const startBboxDraw = (p: {x:number;y:number}) => {
  drawingMode.value = 'bbox'
  isDrawing.value = true
  drawStartPoint.value = p
  drawCurrentPoint.value = p
  // 添加 document 级别监听，支持鼠标移出画布后继续绘制
  addDocumentListeners()
}

const finishBboxDraw = () => {
  if (!drawStartPoint.value || !drawCurrentPoint.value || !props.selectedCategory || !props.image) return
  const x1=Math.min(drawStartPoint.value.x,drawCurrentPoint.value.x), y1=Math.min(drawStartPoint.value.y,drawCurrentPoint.value.y)
  const x2=Math.max(drawStartPoint.value.x,drawCurrentPoint.value.x), y2=Math.max(drawStartPoint.value.y,drawCurrentPoint.value.y)
  if (x2-x1<5||y2-y1<5) { cancelCurrentDrawing(); return }
  const n1=canvasToNormalized(x1,y1), n2=canvasToNormalized(x2,y2)
  const ann: Annotation = { id: generateId(), imageId: props.image.id, categoryId: props.selectedCategory.id, type: 'bbox', data: {x:n1.x,y:n1.y,width:n2.x-n1.x,height:n2.y-n1.y} }
  addAnnotation(ann); cancelCurrentDrawing(); selectAnnotation(ann.id); emit('annotationComplete')
}

// ==================== 多边形绘制 ====================
const handlePolygonClick = (p: {x:number;y:number}) => {
  if (!props.selectedCategory || !props.image) return
  const norm = canvasToNormalized(p.x, p.y)
  if (!isDrawing.value) { drawingMode.value = 'polygon'; isDrawing.value = true; polygonTempPoints.value = [[norm.x,norm.y]]; drawCurrentPoint.value = p; return }
  if (polygonTempPoints.value.length >= 3) {
    const firstPt = polygonTempPoints.value[0]
    if (!firstPt) return
    const first = normalizedToCanvas(firstPt[0], firstPt[1])
    if (Math.hypot(p.x-first.x, p.y-first.y) < 10) {
      const ann: Annotation = { id: generateId(), imageId: props.image.id, categoryId: props.selectedCategory.id, type: 'polygon', data: { points: [...polygonTempPoints.value] } }
      addAnnotation(ann); cancelCurrentDrawing(); selectAnnotation(ann.id); emit('annotationComplete'); return
    }
  }
  polygonTempPoints.value.push([norm.x, norm.y])
}

// ==================== 关键点绘制（新逻辑：关键点作为 bbox 的扩展字段）====================
// 按住 Alt 点击选中的 bbox 时，添加关键点到该 bbox
const handleKeypointClick = (p: {x:number;y:number}) => {
  // 必须有选中的 keypoint 类别
  const kptCategory = props.selectedKeypointCategory
  if (!kptCategory || !props.image) return
  
  const mate = parseKeypointMate(kptCategory.mate)
  if (!mate || !mate.keypoints.length) {
    emit('notify', { type: 'warning', message: t('annotation.tips.keypointNeedConfig') })
    return
  }
  
  // 必须先选中一个 bbox
  let bbox: Annotation | undefined
  if (isDrawing.value && keypointDrawingBboxId.value) {
    bbox = props.annotations.find(a => a.id === keypointDrawingBboxId.value)
  } else {
    if (!selectedAnnotationId.value) {
      emit('notify', { type: 'warning', message: t('annotation.tips.keypointSelectBboxFirst') })
      return
    }
    bbox = props.annotations.find(a => a.id === selectedAnnotationId.value && a.type === 'bbox')
    if (!bbox) {
      emit('notify', { type: 'warning', message: t('annotation.tips.keypointSelectBboxFirst') })
      return
    }
  }
  
  if (!bbox) return
  const bboxData = bbox.data as BboxAnnotationData
  
  // 检查点击位置是否在矩形框内
  if (!isPointInAnnotation(p, bbox)) {
    emit('notify', { type: 'warning', message: t('annotation.tips.keypointNeedBbox') })
    return
  }
  
  // 检查该 bbox 是否已有关键点
  if (bboxData.keypoints && bboxData.keypoints.length > 0 && !isDrawing.value) {
    emit('notify', { type: 'warning', message: t('annotation.tips.keypointExists') })
    return
  }
  
  const norm = canvasToNormalized(p.x, p.y)
  if (!isDrawing.value) { 
    drawingMode.value = 'keypoint'
    isDrawing.value = true
    keypointDrawingBboxId.value = bbox.id
    keypointTempPoints.value = [[norm.x, norm.y, 2]] // v=2 表示可见
    drawCurrentPoint.value = p 
  } else {
    keypointTempPoints.value.push([norm.x, norm.y, 2])
  }
  
  // 当关键点数量达到配置要求时，更新 bbox 的 keypoints 字段
  if (keypointTempPoints.value.length >= mate.keypoints.length) {
    updateAnnotation(bbox.id, { 
      keypoints: [...keypointTempPoints.value],
      keypointCategoryId: kptCategory.id
    })
    cancelCurrentDrawing()
    emit('annotationComplete')
  }
}

const cancelCurrentDrawing = () => {
  // 移除 document 事件监听（bbox 绘制时添加的）
  removeDocumentListeners()
  isDrawing.value = false
  drawingMode.value = 'none'
  drawStartPoint.value = null
  drawCurrentPoint.value = null
  polygonTempPoints.value = []
  keypointTempPoints.value = []
  keypointDrawingBboxId.value = null
  render()
}

// ==================== 多边形边添加点并开始拖拽 ====================
const addPointToPolygonEdgeAndStartDrag = (annotationId: string, edgeIndex: number, p: { x: number; y: number }) => {
  const ann = props.annotations.find(a => a.id === annotationId)
  if (!ann || ann.type !== 'polygon') return
  const d = ann.data as PolygonAnnotationData
  const norm = canvasToNormalized(p.x, p.y)
  // 在 edgeIndex 后面插入新点
  const newPts: [number, number][] = [...d.points]
  newPts.splice(edgeIndex + 1, 0, [norm.x, norm.y])
  // 先设置拖拽状态（使用新数据）
  selectedAnnotationId.value = ann.id
  selectedPointIndex.value = edgeIndex + 1
  hoveredEdgeInfo.value = null
  isDragging.value = true
  dragStartPoint.value = p
  dragStartAnnotationData.value = { points: newPts } // 直接使用新数据
  // 然后更新标注
  updateAnnotation(ann.id, { points: newPts })
  render()
}

// ==================== 拖拽 ====================
const startDrag = (ann: Annotation, p: {x:number;y:number}) => { isDragging.value = true; dragStartPoint.value = p; dragStartAnnotationData.value = JSON.parse(JSON.stringify(ann.data)) }
const startDragPoint = (p: {x:number;y:number}) => { isDragging.value = true; dragStartPoint.value = p; const ann = props.annotations.find(a=>a.id===selectedAnnotationId.value); if(ann) dragStartAnnotationData.value = JSON.parse(JSON.stringify(ann.data)) }

// 辅助函数：限制值在图片范围内 (0-1)
const clampToImage = (v: number) => Math.max(0, Math.min(1, v))

// 辅助函数：限制关键点在 bbox 范围内
const clampKeypointToBbox = (x: number, y: number, bbox: BboxAnnotationData): [number, number] => {
  const minX = bbox.x, maxX = bbox.x + bbox.width
  const minY = bbox.y, maxY = bbox.y + bbox.height
  return [Math.max(minX, Math.min(maxX, x)), Math.max(minY, Math.min(maxY, y))]
}

const handleDragMove = (p: {x:number;y:number}) => {
  if (!dragStartPoint.value || !dragStartAnnotationData.value || !selectedAnnotationId.value) return
  const ann = props.annotations.find(a => a.id === selectedAnnotationId.value); if (!ann) return
  const rect = imageDisplayRect.value
  const dnx = (p.x - dragStartPoint.value.x) / rect.width, dny = (p.y - dragStartPoint.value.y) / rect.height
  
  // bbox 关键点：拖拽单个点（限制在 bbox 范围内）
  if (selectedPointIndex.value !== null && ann.type === 'bbox') {
    const orig = dragStartAnnotationData.value as BboxAnnotationData
    if (orig.keypoints) {
      const newPts = orig.keypoints.map((pt: [number, number, number], i: number) => {
        if (i === selectedPointIndex.value) {
          const rawX = pt[0] + dnx, rawY = pt[1] + dny
          // 限制在 bbox 范围内
          const [newX, newY] = clampKeypointToBbox(rawX, rawY, orig)
          return [newX, newY, pt[2]] as [number, number, number]
        }
        return pt
      })
      updateAnnotation(ann.id, { keypoints: newPts })
      render()
      return
    }
  }
  // 多边形：拖拽单个点
  else if (selectedPointIndex.value !== null && ann.type === 'polygon') {
    const orig = dragStartAnnotationData.value as PolygonAnnotationData
    const newPts = orig.points.map((pt,i) => {
      if (i === selectedPointIndex.value) {
        return [clampToImage(pt[0]+dnx), clampToImage(pt[1]+dny)] as [number,number]
      }
      return pt
    })
    updateAnnotation(ann.id, { points: newPts })
  } 
  // 矩形框：整体拖拽（限制在图片范围内）
  else if (ann.type === 'bbox') {
    const orig = dragStartAnnotationData.value as BboxAnnotationData
    let newX = orig.x + dnx, newY = orig.y + dny
    // 限制矩形框不超出图片边界
    if (newX < 0) newX = 0
    if (newY < 0) newY = 0
    if (newX + orig.width > 1) newX = 1 - orig.width
    if (newY + orig.height > 1) newY = 1 - orig.height
    updateAnnotation(ann.id, { ...orig, x: newX, y: newY })
  } 
  // 多边形：整体拖拽（限制在图片范围内）
  else if (ann.type === 'polygon') {
    const orig = dragStartAnnotationData.value as PolygonAnnotationData
    // 计算多边形的边界
    const minX = Math.min(...orig.points.map(pt => pt[0]))
    const maxX = Math.max(...orig.points.map(pt => pt[0]))
    const minY = Math.min(...orig.points.map(pt => pt[1]))
    const maxY = Math.max(...orig.points.map(pt => pt[1]))
    // 限制偏移量使多边形不超出图片边界
    let clampedDnx = dnx, clampedDny = dny
    if (minX + dnx < 0) clampedDnx = -minX
    if (maxX + dnx > 1) clampedDnx = 1 - maxX
    if (minY + dny < 0) clampedDny = -minY
    if (maxY + dny > 1) clampedDny = 1 - maxY
    updateAnnotation(ann.id, { points: orig.points.map(([x,y])=>[x+clampedDnx,y+clampedDny] as [number,number]) })
  }
  render()
}

const endDrag = () => { isDragging.value = false; dragStartPoint.value = null; dragStartAnnotationData.value = null; emit('annotationComplete') }

// ==================== 调整大小 ====================
const startResize = (handle: ResizeHandle, p: {x:number;y:number}) => {
  const ann = props.annotations.find(a => a.id === selectedAnnotationId.value); if (!ann || ann.type !== 'bbox') return
  isResizing.value = true; resizeHandle.value = handle; resizeStartPoint.value = p; resizeStartData.value = JSON.parse(JSON.stringify(ann.data))
}

const handleResizeMove = (p: {x:number;y:number}) => {
  if (!resizeStartPoint.value || !resizeStartData.value || !resizeHandle.value || !selectedAnnotationId.value) return
  const rect = imageDisplayRect.value, orig = resizeStartData.value
  const dx = (p.x - resizeStartPoint.value.x) / rect.width, dy = (p.y - resizeStartPoint.value.y) / rect.height
  let {x,y,width,height,rotation} = orig
  if (resizeHandle.value === 'rotate') {
    const cx = rect.x + (x+width/2)*rect.width, cy = rect.y + (y+height/2)*rect.height
    rotation = Math.atan2(p.y-cy, p.x-cx) + Math.PI/2
    updateAnnotation(selectedAnnotationId.value, {x,y,width,height,rotation}); render(); return
  }
  switch(resizeHandle.value) {
    case 'nw': x+=dx;y+=dy;width-=dx;height-=dy;break; case 'n': y+=dy;height-=dy;break
    case 'ne': y+=dy;width+=dx;height-=dy;break; case 'e': width+=dx;break
    case 'se': width+=dx;height+=dy;break; case 's': height+=dy;break
    case 'sw': x+=dx;width-=dx;height+=dy;break; case 'w': x+=dx;width-=dx;break
  }
  if(width<0.01)width=0.01; if(height<0.01)height=0.01
  // 限制在图片范围内
  if (x < 0) { width += x; x = 0 }
  if (y < 0) { height += y; y = 0 }
  if (x + width > 1) width = 1 - x
  if (y + height > 1) height = 1 - y
  if (width < 0.01) width = 0.01
  if (height < 0.01) height = 0.01
  updateAnnotation(selectedAnnotationId.value, {x,y,width,height,rotation}); render()
}

const endResize = () => { isResizing.value = false; resizeHandle.value = null; resizeStartPoint.value = null; resizeStartData.value = null; emit('annotationComplete') }

// ==================== 光标 ====================
const updateCursor = (p: {x:number;y:number}) => {
  const canvas = canvasRef.value; if (!canvas) return
  if (isSpacePressed.value) { canvas.style.cursor = isPanning.value ? 'grabbing' : 'grab'; return }
  if (selectedAnnotationId.value) {
    const h = getResizeHandleAtPoint(p)
    if (h) { const c: Record<ResizeHandle,string> = {nw:'nwse-resize',n:'ns-resize',ne:'nesw-resize',e:'ew-resize',se:'nwse-resize',s:'ns-resize',sw:'nesw-resize',w:'ew-resize',rotate:'crosshair'}; canvas.style.cursor = c[h]; return }
    // 悬停在多边形边上时显示添加光标
    if (hoveredEdgeInfo.value) { canvas.style.cursor = 'copy'; return }
  }
  canvas.style.cursor = getAnnotationAtPoint(p) ? 'pointer' : 'crosshair'
}

// ==================== 缩放控制 ====================
const zoomIn = () => { scale.value = Math.min(maxScale, scale.value * 1.2); hasUserZoomed.value = true; render() }
const zoomOut = () => { scale.value = Math.max(minScale, scale.value / 1.2); hasUserZoomed.value = true; render() }
const resetView = () => {
  // 重新初始化画布尺寸，确保使用最新的容器大小
  initCanvas()
  scale.value = 1
  translate.value = { x: 0, y: 0 }
  hasUserZoomed.value = false
  render()
}
const toggleFullscreen = () => {
  // 全屏整个 app-main 区域
  const el = document.querySelector('.app-main') as HTMLElement
  if (!el) return
  if (document.fullscreenElement) document.exitFullscreen(); else el.requestFullscreen()
}

// ==================== 生命周期 ====================
watch(() => props.annotations, () => render(), { deep: true })
watch(() => props.categories, () => render(), { deep: true })
watch(() => props.imageUrl, () => { 
  imageLoaded.value = false
  // 加载新图片时重置缩放状态
  hasUserZoomed.value = false
  scale.value = 1
  translate.value = { x: 0, y: 0 }
})

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  window.addEventListener('keyup', handleKeyUp)
  window.addEventListener('blur', handleBlur)
  const canvas = canvasRef.value
  if (canvas) {
    canvas.addEventListener('mousedown', handleMouseDown)
    canvas.addEventListener('mousemove', handleMouseMove)
    canvas.addEventListener('mouseup', handleMouseUp)
    canvas.addEventListener('mouseenter', handleMouseEnter)
    canvas.addEventListener('mouseleave', handleMouseLeave)
    canvas.addEventListener('wheel', handleWheel, { passive: false })
    canvas.addEventListener('contextmenu', handleContextMenuEvent)
  }
  // 使用 ResizeObserver 监听容器大小变化（比 window resize 更精确）
  const container = containerRef.value
  if (container) {
    resizeObserver = new ResizeObserver(() => {
      initCanvas()
      // 如果用户没有手动缩放过，自动适应新尺寸
      if (!hasUserZoomed.value) {
        scale.value = 1
        translate.value = { x: 0, y: 0 }
      }
      render()
    })
    resizeObserver.observe(container)
  }
  nextTick(() => { initCanvas(); if (imageLoaded.value) render() })
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('keyup', handleKeyUp)
  window.removeEventListener('blur', handleBlur)
  // 清理 document 事件监听
  removeDocumentListeners()
  // 清理 ResizeObserver
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

// 暴露方法给父组件（用于快捷键调用）
defineExpose({
  resetView,
  deleteSelectedAnnotation: () => {
    // 智能删除逻辑：关键点优先于矩形框，顶点优先于多边形
    if (!selectedAnnotationId.value) return false
    const ann = props.annotations.find(a => a.id === selectedAnnotationId.value)
    if (!ann) return false
    
    // 如果是 bbox 且选中了关键点，删除整组关键点
    if (ann.type === 'bbox' && selectedPointIndex.value !== null) {
      const d = ann.data as BboxAnnotationData
      if (d.keypoints?.length) {
        // 删除整组关键点（清空 keypoints 数组）
        updateAnnotation(ann.id, { keypoints: undefined, keypointCategoryId: undefined })
        selectedPointIndex.value = null
        render()
        emit('annotationComplete')
        return true
      }
    }
    
    // 如果是多边形且选中了顶点，优先删除顶点
    if (ann.type === 'polygon' && selectedPointIndex.value !== null) {
      const d = ann.data as PolygonAnnotationData
      if (d.points.length <= 3) {
        // 删除整个多边形
        deleteAnnotation(ann.id)
      } else {
        // 删除单个顶点
        const newPts = d.points.filter((_, i) => i !== selectedPointIndex.value)
        updateAnnotation(ann.id, { points: newPts })
        selectedPointIndex.value = null
        render()
        emit('annotationComplete')
      }
      return true
    }
    
    // 否则删除整个标注
    deleteAnnotation(ann.id)
    return true
  },
  toggleSelectedKeypointVisibility: () => {
    // 切换选中关键点的可见性
    if (!selectedAnnotationId.value || selectedPointIndex.value === null) return false
    const ann = props.annotations.find(a => a.id === selectedAnnotationId.value)
    if (!ann || ann.type !== 'bbox') return false
    const d = ann.data as BboxAnnotationData
    if (!d.keypoints?.length) return false
    toggleKeypointVisibility(selectedAnnotationId.value, selectedPointIndex.value)
    emit('annotationComplete')
    return true
  },
  getSelectedAnnotationId: () => selectedAnnotationId.value,
  getSelectedPointIndex: () => selectedPointIndex.value
})
</script>

<template>
  <div class="annotation-canvas-wrapper">
    <!-- 顶部栏 -->
    <div class="annotation-toolbar annotation-toolbar--top">
      <div class="annotation-toolbar__left">
        <span class="annotation-toolbar__filename">{{ image?.filename || '' }}</span>
      </div>
      <div class="annotation-toolbar__right">
        <button type="button" class="annotation-toolbar__btn" @click="emit('saveAsNegative')" :title="t('annotation.toolbar.saveAsNegative')">
          {{ t('annotation.toolbar.saveAsNegative') }}
        </button>
        <button type="button" class="annotation-toolbar__btn annotation-toolbar__btn--primary" @click="emit('save')" :title="t('annotation.toolbar.save')">
          {{ t('annotation.toolbar.save') }}
        </button>
        <button type="button" class="annotation-toolbar__btn" @click="emit('prev')" :title="t('annotation.toolbar.prev')">
          {{ t('annotation.toolbar.prev') }}
        </button>
        <button type="button" class="annotation-toolbar__btn" @click="emit('next')" :title="t('annotation.toolbar.next')">
          {{ t('annotation.toolbar.next') }}
        </button>
      </div>
    </div>

    <!-- 画布容器 -->
    <div ref="containerRef" class="annotation-canvas-container">
      <canvas ref="canvasRef" class="annotation-canvas"></canvas>
      <img ref="imageRef" :src="imageUrl" style="display: none" @load="onImageLoad" />
      
      <!-- 分类标签显示（右上角） -->
      <div v-if="displayClassificationTags.length > 0" class="classification-tags">
        <span
          v-for="tag in displayClassificationTags"
          :key="tag.id"
          class="classification-tags__item"
        >{{ tag.name }}</span>
      </div>
      
      <!-- 右键菜单 -->
      <div v-if="isContextMenuOpen" class="annotation-context-menu" :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }">
        <button type="button" class="annotation-context-menu__item" @click="handleDeleteFromContextMenu">
          {{ t('annotation.contextMenu.delete') }}
        </button>
        <!-- 多边形：删除单个点 -->
        <button v-if="contextMenuPointIndex !== null && props.annotations.find(a => a.id === contextMenuAnnotationId)?.type === 'polygon'" type="button" class="annotation-context-menu__item" @click="handleDeletePointFromContextMenu">
          {{ t('annotation.contextMenu.deletePoint') }}
        </button>
        <!-- bbox 关键点：切换可见性 -->
        <button v-if="contextMenuPointIndex !== null && (() => { const a = props.annotations.find(x => x.id === contextMenuAnnotationId); return a?.type === 'bbox' && (a.data as BboxAnnotationData).keypoints?.length })()" type="button" class="annotation-context-menu__item" @click="handleToggleKeypointVisibilityFromContextMenu">
          {{ getContextMenuKeypointVisibility() === 2 ? t('annotation.contextMenu.setInvisible') : t('annotation.contextMenu.setVisible') }}
        </button>
        <!-- bbox 关键点：设为不存在 -->
        <button v-if="contextMenuPointIndex !== null && (() => { const a = props.annotations.find(x => x.id === contextMenuAnnotationId); return a?.type === 'bbox' && (a.data as BboxAnnotationData).keypoints?.length })()" type="button" class="annotation-context-menu__item annotation-context-menu__item--danger" @click="handleSetKeypointNotExistFromContextMenu">
          {{ t('annotation.contextMenu.setNotExist') }}
        </button>
      </div>
    </div>

    <!-- 底部栏 -->
    <div class="annotation-toolbar annotation-toolbar--bottom">
      <div class="annotation-toolbar__left"></div>
      <div class="annotation-toolbar__right">
        <button type="button" class="annotation-toolbar__icon-btn" @click="zoomOut" :title="t('annotation.toolbar.zoomOut')">
          <Minus :size="18" />
        </button>
        <button type="button" class="annotation-toolbar__icon-btn" @click="zoomIn" :title="t('annotation.toolbar.zoomIn')">
          <Plus :size="18" />
        </button>
        <button type="button" class="annotation-toolbar__icon-btn" @click="resetView" :title="t('annotation.toolbar.reset')">
          <Maximize :size="18" />
        </button>
        <button type="button" class="annotation-toolbar__icon-btn" @click="toggleFullscreen" :title="t('annotation.toolbar.fullscreen')">
          <Maximize2 :size="18" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.annotation-canvas-wrapper {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background: var(--color-bg-app);
}

.annotation-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.25rem 0.75rem;
  background: var(--color-bg-sidebar);
  border-bottom: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}

.annotation-toolbar--bottom {
  border-bottom: none;
  border-top: 1px solid var(--color-border-subtle);
}

.annotation-toolbar__left,
.annotation-toolbar__right {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.annotation-toolbar__filename {
  font-size: 0.75rem;
  color: var(--color-fg-muted);
}

.annotation-toolbar__btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  border: 1px solid var(--color-border-subtle);
  border-radius: 4px;
  background: var(--color-bg-sidebar);
  color: var(--color-fg);
  cursor: pointer;
  transition: background 0.15s;
}

.annotation-toolbar__btn:hover {
  background: var(--color-bg-sidebar-hover);
}

.annotation-toolbar__btn--primary {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.annotation-toolbar__btn--primary:hover {
  background: var(--color-accent-emphasis);
}

.annotation-toolbar__icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--color-border-subtle);
  border-radius: 4px;
  background: var(--color-bg-sidebar);
  color: var(--color-fg);
  cursor: pointer;
  transition: background 0.15s;
}

.annotation-toolbar__icon-btn:hover {
  background: var(--color-bg-sidebar-hover);
}

.annotation-canvas-container {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.annotation-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.annotation-context-menu {
  position: absolute;
  min-width: 100px;
  background: var(--color-bg-sidebar);
  border: 1px solid var(--color-border-subtle);
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 100;
}

.annotation-context-menu__item {
  display: block;
  width: 100%;
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  text-align: left;
  border: none;
  background: transparent;
  color: var(--color-fg);
  cursor: pointer;
}

.annotation-context-menu__item:hover {
  background: var(--color-bg-sidebar-hover);
}

/* 分类标签显示（右上角） */
.classification-tags {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  max-width: 50%;
  z-index: 10;
}

.classification-tags__item {
  padding: 2px 8px;
  background: var(--color-accent);
  color: #fff;
  font-size: 0.7rem;
  border-radius: 4px;
  white-space: nowrap;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}
</style>
