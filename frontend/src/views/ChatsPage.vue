<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getPrivateChats,
  getPrivateChatUsers,
  getPrivateMessages,
  openPrivateChat
} from '@/api/chats.js'

import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import MessageComposer from '@/components/chat/MessageComposer.vue'
import MessageThread from '@/components/chat/MessageThread.vue'
import {
  appendUniqueMessage,
  normalizeChatMessage,
  normalizeChatMessages,
} from '@/helpers/chatMessages.js'
import { sendChatMessage, subscribeRealtime } from '@/services/realtime.js'
import { refreshChats } from '@/helpers/useChats.js'
import { refreshNotifications } from '@/helpers/useNotifications.js'
import { markNotificationRead } from '@/api/notifications.js'

const conversations = ref([])
const candidates = ref([])
const activeChat = ref(null)
const messages = ref([])
const search = ref('')
const loadingPage = ref(true)
const loadingMessages = ref(false)
const sending = ref(false)
const loadingOlderMessages = ref(false)
const hasOlderMessages = ref(false)
const messageOffset = ref(0)
const openingUserId = ref(null)
const error = ref('')

const MESSAGE_PAGE_SIZE = 20

const route = useRoute()
const router = useRouter()

let stopMessageListener
let stopErrorListener
let stopConnectionListener
let stopNotificationListener
const pendingRealtimeMessages = []
const pendingHistoryMessages = []

function appendActiveMessage(message) {
  const nextMessages = appendUniqueMessage(messages.value, message)
  if (nextMessages !== messages.value) {
    messages.value = nextMessages
    messageOffset.value += 1
  }
}

function handleRealtimeMessage(event) {
  const message = normalizeChatMessage(event)
  const conversationIndex = conversations.value.findIndex(chat => Number(chat.id) === message.chatId)
  if (conversationIndex === -1) {
    if (loadingPage.value) pendingRealtimeMessages.push(event)
    return
  }

  if (Number(activeChat.value?.id) === message.chatId) {
    if (loadingMessages.value) pendingHistoryMessages.push(message)
    else appendActiveMessage(message)
  }

  const conversation = {
    ...conversations.value[conversationIndex],
    latestMessage: message.content,
    latestMessageTime: message.createdAt,
  }
  conversations.value = [
    conversation,
    ...conversations.value.filter(chat => Number(chat.id) !== message.chatId),
  ]
  if (Number(activeChat.value?.id) === message.chatId) activeChat.value = conversation
  if (message.isOwn) sending.value = false
}

function handleRealtimeError(event) {
  sending.value = false
  error.value = event.message || 'Could not send message.'
}

const filteredCandidates = computed(() => {
  const value = search.value.trim().toLowerCase()

  if (!value) {
    return candidates.value
  }

  return candidates.value.filter(user => {
    const fullName =
      `${user.firstName || ''} ${user.lastName || ''}`
        .trim()
        .toLowerCase()

    return (
      fullName.includes(value) ||
      (user.username || '').toLowerCase().includes(value)
    )
  })
})

onMounted(async () => {
  stopMessageListener = subscribeRealtime('message', handleRealtimeMessage)
  stopErrorListener = subscribeRealtime('error', handleRealtimeError)
  stopConnectionListener = subscribeRealtime('connection', event => {
    if (event.status === 'disconnected') sending.value = false
  })
  stopNotificationListener = subscribeRealtime('notification', async event => {
    const notification = event?.notification
    if (
      notification?.category !== 'messages' ||
      Number(notification.relatedId) !== Number(activeChat.value?.id)
    ) return
    try {
      await markNotificationRead(notification.id)
      await Promise.all([refreshChats(), refreshNotifications()])
    } catch {
      // The persisted notification can still be marked read from the notifications page.
    }
  })

  try {
    const [chatResult, userResult] = await Promise.all([
      getPrivateChats(),
      getPrivateChatUsers(),
    ])

    conversations.value = chatResult?.chats || []
    candidates.value = userResult?.users || []
    pendingRealtimeMessages.splice(0).forEach(handleRealtimeMessage)

    const queryUser = Array.isArray(route.query.user) ? route.query.user[0] : route.query.user
    const targetUserId = Number(queryUser)

    if (Number.isSafeInteger(targetUserId) && targetUserId > 0) {
      await startChat({ id: targetUserId })
      router.replace({ path: '/chats' })
    }

    if (!activeChat.value && conversations.value.length) {
      await selectChat(conversations.value[0])
    }
  } catch (err) {
    error.value = err.message
  } finally {
    loadingPage.value = false
  }
})

onUnmounted(() => {
  stopMessageListener?.()
  stopErrorListener?.()
  stopConnectionListener?.()
  stopNotificationListener?.()
})

async function selectChat(chat) {
  if (!chat || loadingMessages.value) {
    return
  }

  // clicking the chat that is already open should not load it again,
  // new messages already come in live through the websocket
  if (Number(activeChat.value?.id) === Number(chat.id) && messages.value.length) {
    return
  }

  activeChat.value = chat
  loadingMessages.value = true
  error.value = ''

  try {
    const result = await getPrivateMessages(chat.id, {
      limit: MESSAGE_PAGE_SIZE,
      offset: 0,
    })

    messages.value = normalizeChatMessages(result?.messages)
    messageOffset.value = messages.value.length
    hasOlderMessages.value = Boolean(result?.hasMore)
    pendingHistoryMessages.splice(0).forEach(message => {
      if (message.chatId === Number(chat.id)) appendActiveMessage(message)
    })
    refreshChats()
  } catch (err) {
    error.value = err.message
    messages.value = []
    pendingHistoryMessages.length = 0
  } finally {
    loadingMessages.value = false
  }
}

async function loadOlderMessages() {
  if (
    !activeChat.value ||
    loadingOlderMessages.value ||
    !hasOlderMessages.value
  ) {
    return
  }

  loadingOlderMessages.value = true
  error.value = ''

  try {
    const result = await getPrivateMessages(
      activeChat.value.id,
      {
        limit: MESSAGE_PAGE_SIZE,
        offset: messageOffset.value,
      }
    )

    const olderMessages = normalizeChatMessages(result?.messages)

    messages.value = [
      ...olderMessages,
      ...messages.value,
    ]

    messageOffset.value += olderMessages.length
    hasOlderMessages.value = Boolean(result?.hasMore)
  } catch (err) {
    error.value =
      err.message || 'Could not load older messages.'
  } finally {
    loadingOlderMessages.value = false
  }
}

async function startChat(user) {
  if (openingUserId.value) {
    return
  }

  openingUserId.value = user.id
  error.value = ''

  try {
    const result = await openPrivateChat(user.id)

    const chat = result.chat

    const existingIndex = conversations.value.findIndex(
      item => item.id === chat.id
    )

    if (existingIndex === -1) {
      conversations.value.unshift(chat)
    } else {
      conversations.value[existingIndex] = chat
    }

    user.chatId = chat.id

    await selectChat(chat)
  } catch (err) {
    error.value = err.message
  } finally {
    openingUserId.value = null
  }
}

function send(content, clear) {
  if (!activeChat.value || sending.value) {
    return
  }

  error.value = ''

  sending.value = true

  try {
    sendChatMessage(activeChat.value.id, content)
    clear()
  } catch (err) {
    error.value = err.message
    sending.value = false
  }
}

function displayName(user) {
  return (
    `${user?.firstName || ''} ${user?.lastName || ''}`.trim() ||
    user?.username ||
    'Orbit member'
  )
}

function initials(user) {
  return displayName(user)
    .slice(0, 2)
    .toUpperCase()
}

function shortTime(value) {
  if (!value) {
    return ''
  }

  const date = new Date(value)

  return Number.isNaN(date.getTime())
    ? ''
    : date.toLocaleTimeString([], {
      hour: 'numeric',
      minute: '2-digit',
    })
}
</script>

<template>
  <AuthenticatedLayout active-page="chats">
    <div class="chats-page">
      <header class="chats-heading">
        <p class="orbit-meta">Messages</p>
        <h1>Chats</h1>
        <p>Conversations with people you follow or who follow you.</p>
      </header>

      <p v-if="error" class="chat-error" role="alert">{{ error }}</p>
      <p v-if="loadingPage" class="chat-state">Loading chats...</p>

      <div v-else class="chat-workspace orbit-surface">
        <aside class="chat-sidebar" aria-label="Private conversations">
          <section class="conversation-section">
            <h2>Conversations</h2>
            <button v-for="chat in conversations" :key="chat.id" class="conversation-row"
              :class="{ 'conversation-row--active': activeChat?.id === chat.id }" type="button"
              @click="selectChat(chat)">
              <span class="user-avatar"><img v-if="chat.otherUser.avatarPath"
                  :src="`/uploads/${chat.otherUser.avatarPath}`" alt="" /><span v-else>{{ initials(chat.otherUser)
                  }}</span></span>
              <span class="conversation-copy"><strong>{{ displayName(chat.otherUser) }}</strong><small>{{
                chat.latestMessage || 'No messages yet' }}</small></span>
              <time>{{ shortTime(chat.latestMessageTime) }}</time>
            </button>
            <p v-if="!conversations.length" class="sidebar-empty">No conversations yet.</p>
          </section>

          <section class="contact-section">
            <h2>Start a chat</h2>
            <label class="contact-search">
              <span class="sr-only">Search chat contacts</span>
              <IconGlyph name="search" :size="16" />
              <input v-model="search" type="search" placeholder="Search people..." />
            </label>
            <button v-for="user in filteredCandidates" :key="user.id" class="contact-row" type="button"
              :disabled="openingUserId === user.id" @click="startChat(user)">
              <span class="user-avatar"><img v-if="user.avatarPath" :src="`/uploads/${user.avatarPath}`" alt="" /><span
                  v-else>{{ initials(user) }}</span></span>
              <span><strong>{{ displayName(user) }}</strong><small>@{{ user.username || 'orbit member' }}</small></span>
              <IconGlyph name="chat" :size="17" />
            </button>
            <p v-if="!filteredCandidates.length" class="sidebar-empty">No matching contacts.</p>
          </section>
        </aside>

        <section class="private-thread" aria-label="Selected conversation">
          <template v-if="activeChat">
            <header class="private-thread__header">
              <span class="user-avatar"><img v-if="activeChat.otherUser.avatarPath"
                  :src="`/uploads/${activeChat.otherUser.avatarPath}`" alt="" /><span v-else>{{
                    initials(activeChat.otherUser) }}</span></span>
              <span><strong>{{ displayName(activeChat.otherUser) }}</strong><small>@{{ activeChat.otherUser.username ||
                'orbit member' }}</small></span>
            </header>
            <MessageThread :messages="messages" :loading="loadingMessages" :loading-older="loadingOlderMessages"
              :can-load-older="hasOlderMessages" @reach-top="loadOlderMessages" />
            <MessageComposer :sending="sending" @send="send" />
          </template>
          <div v-else class="thread-placeholder">
            <IconGlyph name="chat" :size="28" />
            <h2>Select a conversation</h2>
            <p>Choose an existing chat or start one with a contact.</p>
          </div>
        </section>
      </div>
    </div>
  </AuthenticatedLayout>
</template>

<style scoped>
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.chats-page {
  display: grid;
  width: 100%;
  max-width: 72rem;
  margin: 0 auto;
  gap: var(--space-5);
}

.chats-heading .orbit-meta {
  margin: 0;
}

.chats-heading h1 {
  margin: var(--space-2) 0;
  font-family: var(--font-display);
  font-size: 2.25rem;
  letter-spacing: 0;
}

.chats-heading>p:last-child {
  margin: 0;
  color: var(--color-text-muted);
}

.chat-error,
.chat-state {
  margin: 0;
  padding: var(--space-4);
  border-left: 3px solid var(--color-coral);
  background: rgb(var(--rgb-coral) / 8%);
  color: var(--color-coral);
}

.chat-state {
  border-color: var(--color-mint);
  color: var(--color-text-muted);
}

.chat-workspace {
  display: grid;
  height: clamp(28rem, calc(100vh - 13rem), 42rem);
  height: clamp(28rem, calc(100dvh - 13rem), 42rem);
  grid-template-columns: minmax(16rem, 21rem) minmax(0, 1fr);
  overflow: hidden;
}

.chat-sidebar {
  min-height: 0;
  overflow-y: auto;
  border-right: 1px solid var(--color-border);
  background: var(--color-sidebar);
}

.conversation-section,
.contact-section {
  padding: var(--space-4);
}

.contact-section {
  border-top: 1px solid var(--color-border);
}

.chat-sidebar h2 {
  margin: 0 0 var(--space-3);
  color: var(--color-text-muted);
  font-family: var(--font-meta);
  font-size: .75rem;
  letter-spacing: .08em;
  text-transform: uppercase;
}

.conversation-row,
.contact-row {
  display: grid;
  width: 100%;
  min-height: 4rem;
  grid-template-columns: 2.5rem minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2);
  border: 0;
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
}

.conversation-row:hover,
.contact-row:hover,
.conversation-row--active {
  background: var(--color-surface-teal);
}

.conversation-row--active {
  box-shadow: inset 3px 0 var(--color-mint);
}

.conversation-copy,
.contact-row>span:nth-child(2),
.private-thread__header>span:last-child {
  display: grid;
  min-width: 0;
  gap: var(--space-1);
}

.conversation-copy strong,
.conversation-copy small,
.contact-row strong,
.contact-row small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversation-copy small,
.contact-row small,
.private-thread__header small {
  color: var(--color-text-faint);
  font-size: .75rem;
}

.conversation-row time {
  color: var(--color-text-faint);
  font-size: .6875rem;
}

.user-avatar {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  background: var(--gradient-action);
  color: white;
  font-size: .75rem;
  font-weight: 700;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.contact-search {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  padding-left: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  color: var(--color-text-faint);
}

.contact-search:focus-within {
  border-color: var(--color-violet);
  box-shadow: var(--focus-ring);
}

.contact-search input {
  width: 100%;
  min-width: 0;
  padding: var(--space-3) var(--space-3) var(--space-3) 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text);
}

.sidebar-empty {
  margin: 0;
  padding: var(--space-4) var(--space-2);
  color: var(--color-text-faint);
  font-size: .8125rem;
  text-align: center;
}

.private-thread {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
}

.private-thread__header {
  position: sticky;
  top: 0;
  z-index: 2;
  flex: 0 0 auto;
  display: flex;
  min-height: 4.75rem;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.private-thread :deep(.message-thread) {
  flex: 1;
  min-height: 0;
  max-height: none;
}

.private-thread :deep(.message-composer) {
  position: sticky;
  bottom: 0;
  z-index: 2;
  flex: 0 0 auto;
}

.thread-placeholder {
  display: grid;
  margin: auto;
  justify-items: center;
  padding: var(--space-6);
  color: var(--color-text-faint);
  text-align: center;
}

.thread-placeholder h2 {
  margin: var(--space-3) 0 var(--space-2);
  color: var(--color-text);
  font-size: 1.25rem;
}

.thread-placeholder p {
  margin: 0;
}

@media (max-width: 760px) {
  .chat-workspace {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(8rem, 0.8fr) minmax(16rem, 1.2fr);
    height: clamp(24rem, calc(100dvh - 19rem), 44rem);
  }

  .chat-sidebar {
    display: grid;
    grid-template-rows: minmax(0, 0.8fr) minmax(0, 1.2fr);
    border-right: 0;
    border-bottom: 1px solid var(--color-border);
  }

  .private-thread {
    min-height: 0;
  }

  .conversation-section,
  .contact-section {
    min-height: 0;
    overflow-y: auto;
  }
}

@media (max-width: 520px) {
  .chats-heading h1 {
    font-size: 1.875rem;
  }

  .chat-workspace {
    margin-inline: calc(var(--space-3) * -1);
    border-radius: 0;
  }
}
</style>
