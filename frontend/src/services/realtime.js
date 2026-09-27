import { addNotification } from '@/data/notifications'
import { readonly, ref } from 'vue'

const status = ref('disconnected')
const listeners = new Map()
let socket = null
let reconnectTimer = null
let reconnectAttempt = 0
let reconnectEnabled = false
// last time we showed a "new message" popup for each chat.
// it has to live outside the message handler, if not it resets every time
// and the 30 second limit never works
const lastMessageNotifications = new Map()
const MESSAGE_POPUP_GAP = 30000

function realtimeURL() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}

function dispatch(type, event) {
  for (const listener of listeners.get(type) || []) {
    try {
      listener(event)
    } catch (error) {
      console.error('Realtime listener failed:', error)
    }
  }
}

function scheduleReconnect() {
  if (!reconnectEnabled || reconnectTimer) return
  const delay = Math.min(1000 * (2 ** reconnectAttempt), 15000)
  reconnectAttempt += 1
  status.value = 'reconnecting'
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = null
    connectRealtime()
  }, delay)
}

export function connectRealtime() {
  reconnectEnabled = true
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return

  status.value = reconnectAttempt ? 'reconnecting' : 'connecting'
  const connection = new WebSocket(realtimeURL())
  socket = connection

  connection.addEventListener('open', () => {
    if (socket !== connection) return
    reconnectAttempt = 0
    status.value = 'connected'
    dispatch('connection', { status: 'connected' })
  })

  connection.addEventListener('message', (rawEvent) => {
    if (socket !== connection) return
    try {
      const event = JSON.parse(rawEvent.data)
      if (!event?.type) return
      if (
        event.type === 'notification' &&
        event.notification.category === 'messages'
      ) {
        // relatedId is the chat id, so each chat gets its own 30 second limit
        const chatID = event.notification.relatedId
        const now = Date.now()
        const lastNotification = lastMessageNotifications.get(chatID) || 0

        if (now - lastNotification >= MESSAGE_POPUP_GAP) {
          lastMessageNotifications.set(chatID, now)
          addNotification(event.notification.message)
        }
      }
      dispatch(event.type, event)
    } catch {
      dispatch('error', {
        type: 'error',
        code: 'invalid_server_event',
        message: 'Received an invalid realtime update.',
      })
    }
  })

  connection.addEventListener('close', () => {
    if (socket !== connection) return
    socket = null
    status.value = 'disconnected'
    dispatch('connection', { status: 'disconnected' })
    scheduleReconnect()
  })

  connection.addEventListener('error', () => {
    if (socket !== connection) return
    // The close event owns reconnect scheduling so only one timer is created.
    status.value = 'disconnected'
  })
}

export function disconnectRealtime() {
  reconnectEnabled = false
  reconnectAttempt = 0
  if (reconnectTimer) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  const current = socket
  socket = null
  status.value = 'disconnected'
  listeners.clear()
  current?.close(1000, 'logout')
}

export function subscribeRealtime(type, listener) {
  if (!listeners.has(type)) listeners.set(type, new Set())
  listeners.get(type).add(listener)
  connectRealtime()
  return () => {
    const typeListeners = listeners.get(type)
    typeListeners?.delete(listener)
    if (typeListeners?.size === 0) listeners.delete(type)
  }
}

export function sendChatMessage(chatId, content) {
  const normalizedContent = content.trim()
  if (!Number.isInteger(Number(chatId)) || Number(chatId) <= 0) {
    throw new Error('Select a valid chat.')
  }
  if (!normalizedContent) {
    throw new Error('Message content is required.')
  }
  if ([...normalizedContent].length > 2000) {
    throw new Error('Message must be 2000 characters or fewer.')
  }
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    throw new Error('Chat connection is reconnecting. Please try again.')
  }

  socket.send(JSON.stringify({
    type: 'message',
    chat_id: Number(chatId),
    content: normalizedContent,
  }))
}

export const realtimeStatus = readonly(status)
