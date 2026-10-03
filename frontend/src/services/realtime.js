import { addNotification } from '@/data/notifications'
import { notificationLink, notificationTitle } from '@/helpers/notificationDisplay.js'
import { readonly, ref } from 'vue'

const status = ref('disconnected')
const listeners = new Map()
let socket = null
let reconnectTimer = null
let reconnectAttempt = 0
let reconnectEnabled = false
// the chat that is open on screen right now. new messages for it are already
// visible, so they should not pop up as a notification as well
let openChatID = null

export function setOpenChat(chatId) {
  openChatID = chatId ? Number(chatId) : null
}
const TYPING_IDLE_DELAY = 1600
const TYPING_REFRESH_DELAY = 2000
const outgoingTyping = new Map()

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

// every live notification pops up at the top of the screen.
// private messages come as a "messages" notification, group chat messages
// only come as a message event, so those get their own popup here
function showPopup(event) {
  if (event.type === 'notification' && event.notification) {
    const notification = event.notification
    if (notification.isRead) return

    if (notification.category === 'messages') {
      if (Number(notification.relatedId) === openChatID) return
      addNotification(notification.message, 'message', {
        title: notificationTitle(notification),
        to: notificationLink(notification),
        key: `chat-${notification.relatedId}`,
      })
      return
    }

    addNotification(notification.message, 'alert', {
      title: notificationTitle(notification),
      to: notificationLink(notification),
    })
    return
  }

  if (event.type === 'message' && event.message?.chatType === 'group') {
    const message = event.message
    if (message.isOwn || Number(message.chatId) === openChatID) return
    const sender = `${message.firstName || ''} ${message.lastName || ''}`.trim() || 'Someone'
    const preview = String(message.content || '').slice(0, 80)
    addNotification(`${sender}: ${preview}`, 'message', {
      title: message.groupTitle ? `New message in ${message.groupTitle}` : 'New group message',
      to: message.groupId ? `/groups/${message.groupId}?section=chat` : '',
      key: `chat-${message.chatId}`,
    })
  }
}

function sendTypingEvent(type, chatId) {
  if (!socket || socket.readyState !== WebSocket.OPEN) return false
  socket.send(JSON.stringify({ type, chat_id: Number(chatId) }))
  return true
}

function clearOutgoingTyping() {
  for (const state of outgoingTyping.values()) {
    window.clearTimeout(state.idleTimer)
  }
  outgoingTyping.clear()
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
      showPopup(event)
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
    clearOutgoingTyping()
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
  clearOutgoingTyping()
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

export function updateChatTyping(chatId, content) {
  const normalizedChatId = Number(chatId)
  if (!Number.isInteger(normalizedChatId) || normalizedChatId <= 0) return

  if (!content.trim()) {
    stopChatTyping(normalizedChatId)
    return
  }

  const now = Date.now()
  const state = outgoingTyping.get(normalizedChatId) || { idleTimer: null, lastStartAt: 0 }
  if (!state.lastStartAt || now - state.lastStartAt >= TYPING_REFRESH_DELAY) {
    if (sendTypingEvent('typing_start', normalizedChatId)) state.lastStartAt = now
  }

  window.clearTimeout(state.idleTimer)
  state.idleTimer = window.setTimeout(() => stopChatTyping(normalizedChatId), TYPING_IDLE_DELAY)
  outgoingTyping.set(normalizedChatId, state)
}

export function stopChatTyping(chatId) {
  const normalizedChatId = Number(chatId)
  const state = outgoingTyping.get(normalizedChatId)
  if (!state) return

  window.clearTimeout(state.idleTimer)
  outgoingTyping.delete(normalizedChatId)
  if (state.lastStartAt) sendTypingEvent('typing_stop', normalizedChatId)
}

export const realtimeStatus = readonly(status)
