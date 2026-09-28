<script setup>
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { getGroupMessages } from '@/api/chats.js'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import { appendUniqueMessage, normalizeChatMessage, normalizeChatMessages } from '@/helpers/chatMessages.js'
import { sendChatMessage, subscribeRealtime } from '@/services/realtime.js'
import { useChatTyping } from '@/helpers/useChatTyping.js'
import MessageComposer from './MessageComposer.vue'
import MessageThread from './MessageThread.vue'

const props = defineProps({
  groupId: { type: [String, Number], required: true },
  active: { type: Boolean, default: false },
})
const messages = ref([])
const loading = ref(true)
const sending = ref(false)
const loadingOlderMessages = ref(false)
const hasOlderMessages = ref(false)
const messageOffset = ref(0)
const error = ref('')
const chatId = ref(null)
const { typingCount } = useChatTyping(chatId)
const messageThread = ref(null)
const MESSAGE_PAGE_SIZE = 20
let stopMessageListener
let stopErrorListener
let stopConnectionListener
let historyLoaded = false
let initialScrollDone = false
const pendingRealtimeMessages = []

async function scrollInitialHistory() {
  if (!props.active || !historyLoaded || initialScrollDone) return

  // v-show must make the group-chat panel visible before it has scroll geometry.
  await nextTick()
  initialScrollDone = (await messageThread.value?.scrollToBottom()) === true
}

watch(() => props.active, active => {
  if (active) void scrollInitialHistory()
})

function handleRealtimeMessage(event) {
  const message = normalizeChatMessage(event)
  if (!chatId.value) {
    pendingRealtimeMessages.push(event)
    return
  }
  if (message.chatId !== Number(chatId.value)) return
  const nextMessages = appendUniqueMessage(messages.value, message)
  if (nextMessages !== messages.value) {
    messages.value = nextMessages
    messageOffset.value += 1
  }
  if (message.isOwn) sending.value = false
}

onMounted(async () => {
  stopMessageListener = subscribeRealtime('message', handleRealtimeMessage)
  stopErrorListener = subscribeRealtime('error', event => {
    sending.value = false
    error.value = event.message || 'Could not send message.'
  })
  stopConnectionListener = subscribeRealtime('connection', event => {
    if (event.status === 'disconnected') sending.value = false
  })
  try {
    const result = await getGroupMessages(props.groupId, { limit: MESSAGE_PAGE_SIZE, offset: 0 })
    chatId.value = Number(result?.chatId)
    messages.value = normalizeChatMessages(result?.messages)
    pendingRealtimeMessages.splice(0).forEach(handleRealtimeMessage)
    messageOffset.value = messages.value.length
    hasOlderMessages.value = Boolean(result?.hasMore)
    historyLoaded = true
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
    await scrollInitialHistory()
  }
})

onUnmounted(() => {
  stopMessageListener?.()
  stopErrorListener?.()
  stopConnectionListener?.()
})

async function loadOlderMessages() {
  if (loadingOlderMessages.value || !hasOlderMessages.value) return

  loadingOlderMessages.value = true
  error.value = ''
  try {
    const result = await getGroupMessages(props.groupId, {
      limit: MESSAGE_PAGE_SIZE,
      offset: messageOffset.value,
    })
    const olderMessages = normalizeChatMessages(result?.messages)
    messages.value = [...olderMessages, ...messages.value]
    messageOffset.value += olderMessages.length
    hasOlderMessages.value = Boolean(result?.hasMore)
  } catch (err) {
    error.value = err.message || 'Could not load older messages.'
  } finally {
    loadingOlderMessages.value = false
  }
}

function send(content, clear) {
  if (sending.value) return
  sending.value = true
  error.value = ''
  try {
    sendChatMessage(chatId.value, content)
    clear()
  } catch (err) {
    error.value = err.message
    sending.value = false
  }
}
</script>

<template>
  <section class="group-chat orbit-surface" aria-labelledby="group-chat-heading">
    <header>
      <div class="group-chat__icon"><IconGlyph name="chat" :size="19" /></div>
      <div>
        <p class="orbit-meta">Members</p>
        <h2 id="group-chat-heading">Group chat</h2>
      </div>
    </header>
    <p v-if="error" class="group-chat__error" role="alert">{{ error }}</p>
    <MessageThread
      ref="messageThread"
      :messages="messages"
      :loading="loading"
      :loading-older="loadingOlderMessages"
      :can-load-older="hasOlderMessages"
      :typing-count="typingCount"
      empty-message="No group messages yet. Start the conversation."
      @reach-top="loadOlderMessages"
    />
    <MessageComposer :sending="sending" :chat-id="chatId" :active="active" @send="send" />
  </section>
</template>

<style scoped>
.group-chat { overflow: hidden; }
.group-chat > header { display: grid; grid-template-columns: 2.5rem minmax(0, 1fr); align-items: center; gap: var(--space-3); padding: var(--space-5); }
.group-chat__icon { display: grid; width: 2.5rem; height: 2.5rem; place-items: center; border-radius: 50%; background: var(--color-surface-teal); color: var(--color-mint); }
.group-chat .orbit-meta { margin: 0; }
.group-chat h2 { margin: var(--space-1) 0 0; font-family: var(--font-display); font-size: 1.4rem; letter-spacing: 0; }
.group-chat__error { margin: 0; padding: var(--space-3) var(--space-5); border-top: 1px solid var(--color-border); background: rgb(var(--rgb-coral) / 8%); color: var(--color-coral); }
@media (max-width: 520px) { .group-chat > header { padding: var(--space-4); } }
</style>
