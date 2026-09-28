<script setup>
import { computed } from 'vue'

const props = defineProps({
  type: { type: String, default: 'personal' },
  modelValue: { type: String, default: 'about' },
})
const emit = defineEmits(['update:modelValue', 'changeTab'])
const tabs = computed(() => props.type === 'personal'
  ? ['posts', 'about', 'followers', 'following', 'friends']
  : ['posts', 'about', 'followers', 'following'])

function selectTab(tab) {
  emit('update:modelValue', tab)
  emit('changeTab', tab)
}
</script>

<template>
  <nav class="profile-tabs" aria-label="Profile sections">
    <button v-for="tab in tabs" :key="tab" type="button" :class="{ active: modelValue === tab }" @click="selectTab(tab)">
      {{ tab }}
    </button>
  </nav>
</template>

<style scoped>
.profile-tabs { position: sticky; top: 4rem; z-index: 10; display: flex; overflow-x: auto; border-bottom: 1px solid var(--color-border); background: rgb(var(--rgb-background) / 94%); backdrop-filter: blur(.75rem); }
.profile-tabs button { min-width: max-content; min-height: var(--touch-target); flex: 1; padding: 0 var(--space-4); border: 0; border-bottom: 2px solid transparent; background: transparent; color: var(--color-text-muted); cursor: pointer; font-family: var(--font-meta); font-size: .75rem; font-weight: 600; text-transform: capitalize; }
.profile-tabs button:hover { color: var(--color-text); }
.profile-tabs button.active { border-bottom-color: var(--color-mint); color: var(--color-mint); }
@media (max-width: 800px) { .profile-tabs { top: 0; } }
</style>
