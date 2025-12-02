/**
 * 全局 WebSocket 管理器
 * 所有页面共用一个 WebSocket 连接
 */
import { ref, shallowRef } from 'vue'

export interface WsMessage {
  type: string
  taskId?: string
  pluginId?: string
  message?: string
  data?: any
  success?: boolean
}

type MessageHandler = (msg: WsMessage) => void

// 全局状态（模块级别，所有组件共享）
const ws = shallowRef<WebSocket | null>(null)
const connected = ref(false)
const reconnectTimer = ref<number | null>(null)

// 消息处理器映射：type -> handlers[]
const handlers = new Map<string, Set<MessageHandler>>()

// 通用处理器（接收所有消息）
const globalHandlers = new Set<MessageHandler>()

// WebSocket URL
const WS_URL = 'ws://localhost:18080/api/ws'

// 重连间隔（毫秒）
const RECONNECT_INTERVAL = 3000

/**
 * 连接 WebSocket
 */
function connect() {
  if (ws.value?.readyState === WebSocket.OPEN) return
  
  try {
    ws.value = new WebSocket(WS_URL)
    
    ws.value.onopen = () => {
      console.log('[WS] Connected')
      connected.value = true
      if (reconnectTimer.value) {
        clearTimeout(reconnectTimer.value)
        reconnectTimer.value = null
      }
    }
    
    ws.value.onmessage = (event) => {
      try {
        const msg: WsMessage = JSON.parse(event.data)
        
        // 调用类型特定的处理器
        const typeHandlers = handlers.get(msg.type)
        if (typeHandlers) {
          typeHandlers.forEach(handler => handler(msg))
        }
        
        // 调用全局处理器
        globalHandlers.forEach(handler => handler(msg))
      } catch (e) {
        console.error('[WS] Parse error:', e)
      }
    }
    
    ws.value.onclose = () => {
      console.log('[WS] Disconnected')
      connected.value = false
      scheduleReconnect()
    }
    
    ws.value.onerror = (error) => {
      console.error('[WS] Error:', error)
    }
  } catch (e) {
    console.error('[WS] Connect error:', e)
    scheduleReconnect()
  }
}

/**
 * 计划重连
 */
function scheduleReconnect() {
  if (reconnectTimer.value) return
  reconnectTimer.value = window.setTimeout(() => {
    reconnectTimer.value = null
    connect()
  }, RECONNECT_INTERVAL)
}

/**
 * 断开连接
 */
function disconnect() {
  if (reconnectTimer.value) {
    clearTimeout(reconnectTimer.value)
    reconnectTimer.value = null
  }
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }
  connected.value = false
}

/**
 * 发送消息
 */
function send(msg: WsMessage) {
  if (ws.value?.readyState === WebSocket.OPEN) {
    ws.value.send(JSON.stringify(msg))
  } else {
    console.warn('[WS] Not connected, message dropped:', msg)
  }
}

/**
 * 订阅特定类型的消息
 * @param type 消息类型
 * @param handler 处理函数
 * @returns 取消订阅函数
 */
function subscribe(type: string, handler: MessageHandler): () => void {
  if (!handlers.has(type)) {
    handlers.set(type, new Set())
  }
  handlers.get(type)!.add(handler)
  
  return () => {
    handlers.get(type)?.delete(handler)
  }
}

/**
 * 订阅所有消息
 * @param handler 处理函数
 * @returns 取消订阅函数
 */
function subscribeAll(handler: MessageHandler): () => void {
  globalHandlers.add(handler)
  return () => {
    globalHandlers.delete(handler)
  }
}

/**
 * 全局 WebSocket composable
 */
export function useGlobalWs() {
  return {
    connected,
    connect,
    disconnect,
    send,
    subscribe,
    subscribeAll
  }
}

// 默认导出
export default useGlobalWs
