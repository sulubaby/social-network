<script setup>
import { nextTick, onBeforeUnmount, ref } from 'vue'

// the ... menu on my own comments. it only has a delete option
defineProps({
  disabled: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['remove'])

const isOpen = ref(false)
const trigger = ref(null)
const panel = ref(null)
const position = ref({ top: 0, left: 0 })

// spacing so the menu doesnt touch the screen edge
const VIEWPORT_MARGIN = 8
const TRIGGER_GAP = 4

// works out where to show the menu. under the button normally,
// but above it if there is no space at the bottom of the screen
function placePanel() {
  const triggerRect = trigger.value?.getBoundingClientRect()
  if (!triggerRect || !panel.value) return

  const panelWidth = panel.value.offsetWidth
  const panelHeight = panel.value.offsetHeight

  let top = triggerRect.bottom + TRIGGER_GAP
  if (top + panelHeight > window.innerHeight - VIEWPORT_MARGIN) {
    top = triggerRect.top - panelHeight - TRIGGER_GAP
  }

  const maxLeft = window.innerWidth - panelWidth - VIEWPORT_MARGIN
  const left = Math.max(
    VIEWPORT_MARGIN,
    Math.min(triggerRect.right - panelWidth, maxLeft),
  )

  position.value = { top, left }
}

// clicking anywhere outside the menu closes it
function handlePointerDown(event) {
  if (
    panel.value?.contains(event.target) ||
    trigger.value?.contains(event.target)
  ) {
    return
  }

  close()
}

// escape closes the menu
function handleKeydown(event) {
  if (event.key !== 'Escape') return

  close()
  trigger.value?.focus()
}

// we only listen to clicks/scroll/resize while the menu is open
function addListeners() {
  document.addEventListener('pointerdown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', close)
  window.addEventListener('scroll', close, true)
}

function removeListeners() {
  document.removeEventListener('pointerdown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', close)
  window.removeEventListener('scroll', close, true)
}

// open: show it, wait for it to render so we know its size, then place it
async function open() {
  isOpen.value = true
  await nextTick()
  placePanel()
  addListeners()
}

function close() {
  if (!isOpen.value) return

  isOpen.value = false
  removeListeners()
}

function toggle() {
  if (isOpen.value) {
    close()
  } else {
    open()
  }
}

// the delete option, the parent does the real delete
function remove() {
  close()
  emit('remove')
}

onBeforeUnmount(removeListeners)
</script>

<template>
  <div class="comment-menu">
    <button
      ref="trigger"
      type="button"
      class="comment-menu__trigger"
      aria-label="Comment options"
      aria-haspopup="menu"
      :aria-expanded="isOpen"
      :disabled="disabled"
      @click="toggle"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="5" cy="12" r="1.8" />
        <circle cx="12" cy="12" r="1.8" />
        <circle cx="19" cy="12" r="1.8" />
      </svg>
    </button>

    <!-- the menu is put on the body so it doesnt get cut by the comments box scroll -->
    <Teleport to="body">
      <div
        v-if="isOpen"
        ref="panel"
        class="comment-menu__panel"
        role="menu"
        :style="{ top: `${position.top}px`, left: `${position.left}px` }"
      >
        <button
          type="button"
          class="comment-menu__item"
          role="menuitem"
          @click="remove"
        >
          Remove
        </button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.comment-menu {
  display: flex;
  flex: 0 0 auto;
}

.comment-menu__trigger {
  display: grid;
  width: 2rem;
  height: 2rem;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}

.comment-menu__trigger:hover:not(:disabled),
.comment-menu__trigger:focus-visible,
.comment-menu__trigger[aria-expanded='true'] {
  background: var(--color-surface-raised);
  color: var(--color-text);
}

.comment-menu__trigger:disabled {
  cursor: default;
  opacity: 0.5;
}

.comment-menu__trigger svg {
  width: 1.125rem;
  height: 1.125rem;
  fill: currentColor;
}

.comment-menu__panel {
  position: fixed;
  z-index: 1000;
  min-width: 8rem;
  padding: var(--space-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-surface-raised);
  box-shadow: var(--shadow-raised);
}

.comment-menu__item {
  display: block;
  width: 100%;
  min-height: 2.5rem;
  padding: 0 var(--space-3);
  border: 0;
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-coral);
  cursor: pointer;
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  text-align: left;
}

.comment-menu__item:hover,
.comment-menu__item:focus-visible {
  background: var(--color-surface-coral);
}
</style>
