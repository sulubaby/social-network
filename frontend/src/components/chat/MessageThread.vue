<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  messages: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  loadingOlder: { type: Boolean, default: false },
  canLoadOlder: { type: Boolean, default: false },
  autoScroll: { type: Boolean, default: true },
  emptyMessage: { type: String, default: 'No messages yet. Say hello.' },
})
const emit = defineEmits(['reach-top'])
const thread = ref(null)
let previousScrollHeight = 0
let previousScrollTop = 0
let firstMessageIdBeforeOlderLoad = null
let preserveOlderScroll = false
let isNearBottom = true
let initialized = false
let openingChat = props.loading

watch(() => [props.loading, props.messages], async ([loading, currentMessages], [previousLoading, previousMessages] = []) => {
  const messagesChanged = currentMessages !== previousMessages
  const previousMessageCount = previousMessages?.length || 0
  const messageCount = currentMessages.length
  const olderMessagesWerePrepended = preserveOlderScroll
    && messageCount > previousMessageCount
    && currentMessages[0]?.id !== firstMessageIdBeforeOlderLoad

  if (loading && previousLoading === false) openingChat = true

  const shouldOpenAtBottom = !initialized
    || (openingChat && messagesChanged)
    || (openingChat && !loading)
  const shouldFollowNewMessage = messagesChanged && props.autoScroll && isNearBottom

  await nextTick()
  if (!thread.value) return

  if (olderMessagesWerePrepended) {
    // Keep the first visible old message anchored while earlier rows are added.
    thread.value.scrollTop = previousScrollTop + thread.value.scrollHeight - previousScrollHeight
    preserveOlderScroll = false
    return
  }

  if (shouldOpenAtBottom || shouldFollowNewMessage) {
    thread.value.scrollTop = thread.value.scrollHeight
  }

  initialized = true
  if (shouldOpenAtBottom) openingChat = false
}, { immediate: true, flush: 'post' })

watch(() => props.loadingOlder, (loading, wasLoading) => {
  if (wasLoading && !loading && props.messages[0]?.id === firstMessageIdBeforeOlderLoad) {
    preserveOlderScroll = false
  }
}, { flush: 'post' })

function handleScroll() {
  if (!thread.value) return

  const distanceFromBottom = thread.value.scrollHeight - thread.value.scrollTop - thread.value.clientHeight
  isNearBottom = distanceFromBottom <= 80

  if (!props.canLoadOlder || props.loadingOlder) return
  if (thread.value.scrollTop <= 80) {
    previousScrollHeight = thread.value.scrollHeight
    previousScrollTop = thread.value.scrollTop
    firstMessageIdBeforeOlderLoad = props.messages[0]?.id ?? null
    preserveOlderScroll = true
    emit('reach-top')
  }
}

function senderName(message) {
  return `${message.firstName || ''} ${message.lastName || ''}`.trim() || message.username || 'Orbit member'
}

function messageTime(value) {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
}
</script>

<template>
  <div ref="thread" class="message-thread" aria-live="polite" @scroll="handleScroll">
    <p v-if="loadingOlder" class="thread-load-state" role="status">Loading earlier messages...</p>
    <p v-if="loading" class="thread-state">Loading messages...</p>
    <p v-else-if="!messages.length" class="thread-state">{{ emptyMessage }}</p>
    <article v-for="message in messages" v-else :key="message.id" class="message-row" :class="{ 'message-row--own': message.isOwn }">
      <span v-if="!message.isOwn" class="message-avatar" aria-hidden="true">
        <img v-if="message.avatarPath" :src="`/uploads/${message.avatarPath}`" alt="" />
        <span v-else>{{ senderName(message).slice(0, 2).toUpperCase() }}</span>
      </span>
      <div class="message-content">
        <span v-if="!message.isOwn" class="message-sender">{{ senderName(message) }}</span>
        <p>{{ message.content }}</p>
        <time :datetime="message.createdAt">{{ messageTime(message.createdAt) }}</time>
      </div>
    </article>
  </div>
</template>

<style scoped>
.message-thread { display: flex; min-height: 20rem; max-height: 34rem; flex-direction: column; gap: var(--space-3); padding: var(--space-5); overflow-y: auto; background: var(--color-input); }
.thread-load-state { margin: 0; padding: 0 0 var(--space-2); color: var(--color-text-faint); font-size: .75rem; text-align: center; }
.thread-state { margin: auto; padding: var(--space-5); color: var(--color-text-muted); text-align: center; }
.message-row { display: flex; max-width: min(78%, 38rem); align-items: end; gap: var(--space-2); }
.message-row--own { align-self: flex-end; }
.message-avatar { display: grid; width: 2rem; height: 2rem; flex: 0 0 2rem; place-items: center; overflow: hidden; border-radius: 50%; background: var(--gradient-action); color: white; font-size: .6875rem; font-weight: 700; }
.message-avatar img { width: 100%; height: 100%; object-fit: cover; }
.message-content { display: grid; min-width: 0; gap: var(--space-1); }
.message-sender { color: var(--color-text-muted); font-size: .75rem; font-weight: 600; }
.message-content p { max-height: 12rem; margin: 0; padding: var(--space-3) var(--space-4); border: 1px solid var(--color-border); border-radius: var(--radius-medium) var(--radius-medium) var(--radius-medium) var(--radius-small); background: var(--color-surface); color: var(--color-text-soft); line-height: 1.55; overflow-y: auto; overflow-wrap: anywhere; overscroll-behavior: contain; scrollbar-color: var(--color-violet) transparent; scrollbar-width: thin; white-space: pre-wrap; }
.message-row--own .message-content p { border-color: rgb(var(--rgb-mint) / 28%); border-radius: var(--radius-medium) var(--radius-medium) var(--radius-small) var(--radius-medium); background: var(--color-surface-teal); color: var(--color-text); }
.message-content time { color: var(--color-text-faint); font-family: var(--font-meta); font-size: .6875rem; }
.message-row--own time { text-align: right; }
@media (max-width: 520px) { .message-thread { min-height: 18rem; padding: var(--space-4); } .message-row { max-width: 92%; } }
</style>
