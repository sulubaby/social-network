<script setup>

import { ref, nextTick, watch } from 'vue'

import IconGlyph from '@/components/layout/IconGlyph.vue'

const props = defineProps({
  sending: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['send'])

const content = ref('')
const showEmojiPicker = ref(false)
const textarea = ref(null)
let focusAfterSend = false

const emojis = [
  '😀', '😂', '🤣', '😊', '😍', '🥰', '😘', '😎',
  '😭', '😢', '😡', '🤔', '😴', '🙄', '😮', '😱',
  '👍', '👎', '👏', '🙌', '❤️', '🔥', '🎉', '💯',
  '😂', '🤣', '😅', '😆', '😉', '😋', '🤩', '🥳',
  '💀', '🤡', '👀', '💔', '✨', '⭐', '🚀', '💪'
]

function submit() {
  if (!content.value.trim()) return

  emit('send', content.value, () => {
    content.value = ''
    focusAfterSend = true
  })
}

watch(() => props.sending, async sending => {
  if (sending || !focusAfterSend) return

  await nextTick()
  textarea.value?.focus()
  focusAfterSend = false
})

function handleKeydown(event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    submit()
  }
}

async function addEmoji(emoji) {
  const element = textarea.value

  if (!element) {
    content.value += emoji
    return
  }

  const start = element.selectionStart
  const end = element.selectionEnd

  content.value =
    content.value.substring(0, start) +
    emoji +
    content.value.substring(end)

  showEmojiPicker.value = false

  await nextTick()

  const cursorPosition = start + emoji.length

  element.focus()
  element.setSelectionRange(cursorPosition, cursorPosition)
}

</script>

<template>

  <form class="message-composer" @submit.prevent="submit">

    <label class="sr-only" for="chat-message">
      Message
    </label>

    <div class="message-input">

      <textarea id="chat-message" ref="textarea" v-model="content" rows="2" maxlength="2000"
        placeholder="Write a message..." :disabled="sending" @keydown="handleKeydown" />

      <div class="composer-actions">

        <div class="emoji-container">

          <button type="button" class="emoji-button" :disabled="sending" aria-label="Add emoji"
            @click="showEmojiPicker = !showEmojiPicker">
            😊
          </button>

          <div v-if="showEmojiPicker" class="emoji-picker">

            <button v-for="emoji in emojis" :key="emoji" type="button" class="emoji" @click="addEmoji(emoji)">
              {{ emoji }}
            </button>

          </div>

        </div>

        <button type="submit" :disabled="sending || !content.trim()" class="send-button">
          <span>
            {{ sending ? 'Sending...' : 'Send' }}
          </span>

          <IconGlyph name="arrowRight" :size="17" />
        </button>

      </div>

    </div>

  </form>

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

.message-composer {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: end;
  gap: var(--space-3);
  padding: var(--space-4);
  border-top: 1px solid var(--color-border);
  background: var(--color-surface);
}

.message-input {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: var(--space-3);
}

textarea {
  width: 100%;
  min-height: var(--touch-target);
  max-height: 10rem;
  resize: vertical;
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  outline: none;
  background: var(--color-input);
  color: var(--color-text);
  font: inherit;
  line-height: 1.45;
}

textarea:focus {
  border-color: var(--color-violet);
  box-shadow: var(--focus-ring);
}

.composer-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.emoji-container {
  position: relative;
}

.emoji-button {
  display: inline-flex;
  width: var(--touch-target);
  min-height: var(--touch-target);
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
  color: var(--color-text);
  cursor: pointer;
  font-size: 1.25rem;
}

.emoji-button:hover {
  background: var(--color-surface-hover);
}

.emoji-picker {
  position: absolute;
  right: 0;
  bottom: calc(100% + var(--space-2));
  z-index: 20;
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  width: 280px;
  max-height: 220px;
  overflow-y: auto;
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-surface);
  box-shadow: var(--shadow-medium);
}

.emoji {
  display: flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  border-radius: var(--radius-small);
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 1.25rem;
}

.emoji:hover {
  background: var(--color-input);
}

.send-button {
  display: inline-flex;
  min-height: var(--touch-target);
  align-items: center;
  gap: var(--space-2);
  padding: 0 var(--space-4);
  border: 0;
  border-radius: var(--radius-small);
  background: var(--gradient-action);
  color: white;
  cursor: pointer;
  font-weight: 700;
}

.send-button:disabled,
.emoji-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

@media (max-width: 520px) {

  .message-input {
    grid-template-columns: 1fr;
  }

  .composer-actions {
    justify-content: flex-end;
  }

  .emoji-picker {
    right: 0;
    width: min(280px, calc(100vw - 2rem));
  }

}
</style>
