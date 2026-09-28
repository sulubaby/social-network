import { computed, onBeforeUnmount, ref, watch } from 'vue'

import { subscribeRealtime } from '@/services/realtime.js'

const STALE_TYPING_DELAY = 5000

export function useChatTyping(chatId) {
  const typingUsers = ref(new Map())
  const staleTimers = new Map()

  function currentChatId() {
    return Number(chatId.value)
  }

  function removeUser(userId) {
    const normalizedUserId = Number(userId)
    window.clearTimeout(staleTimers.get(normalizedUserId))
    staleTimers.delete(normalizedUserId)
    if (!typingUsers.value.has(normalizedUserId)) return

    const nextUsers = new Map(typingUsers.value)
    nextUsers.delete(normalizedUserId)
    typingUsers.value = nextUsers
  }

  function clearTypingUsers() {
    for (const timer of staleTimers.values()) window.clearTimeout(timer)
    staleTimers.clear()
    typingUsers.value = new Map()
  }

  function handleTypingStart(event) {
    if (Number(event?.chatId) !== currentChatId()) return

    const userId = Number(event?.userId)
    if (!Number.isInteger(userId) || userId <= 0) return

    const nextUsers = new Map(typingUsers.value)
    nextUsers.set(userId, true)
    typingUsers.value = nextUsers

    window.clearTimeout(staleTimers.get(userId))
    staleTimers.set(userId, window.setTimeout(() => removeUser(userId), STALE_TYPING_DELAY))
  }

  function handleTypingStop(event) {
    if (Number(event?.chatId) === currentChatId()) removeUser(event?.userId)
  }

  function handleMessage(event) {
    const message = event?.message
    if (Number(message?.chatId) === currentChatId()) removeUser(message?.senderId)
  }

  const unsubscribeStart = subscribeRealtime('typing_start', handleTypingStart)
  const unsubscribeStop = subscribeRealtime('typing_stop', handleTypingStop)
  const unsubscribeMessage = subscribeRealtime('message', handleMessage)
  const unsubscribeConnection = subscribeRealtime('connection', event => {
    if (event.status === 'disconnected') clearTypingUsers()
  })

  watch(chatId, clearTypingUsers)

  onBeforeUnmount(() => {
    unsubscribeStart()
    unsubscribeStop()
    unsubscribeMessage()
    unsubscribeConnection()
    clearTypingUsers()
  })

  return {
    typingCount: computed(() => typingUsers.value.size),
  }
}
