<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import { THEMES, currentTheme, setTheme } from '@/helpers/theme.js'
import { addNotification } from '@/data/notifications.js'
import { forgetSession, router } from '@/router/router'
import { disconnectRealtime } from '@/services/realtime.js'

const showDeleteDialog = ref(false)
const confirmText = ref('')
const confirmInput = ref(null)

const canDelete = computed(() => confirmText.value.trim() === 'DELETE')

function chooseTheme(theme) {
  if (currentTheme.value === theme.id) return
  setTheme(theme.id)
  addNotification(`${theme.label} theme applied`)
}

function openDeleteDialog() {
  confirmText.value = ''
  showDeleteDialog.value = true
}

function closeDeleteDialog() {
  showDeleteDialog.value = false
  confirmText.value = ''
}

async function confirmDelete() {
  if (!canDelete.value) return
  closeDeleteDialog()

  try {
    const resp = await fetch("/api/user", {
      method: "DELETE",
      credentials: 'include'
    })

    const result = await resp.json().catch(() => null)
    if (!resp.ok || !result?.status) {
      addNotification(result?.message || 'Failed to delete account', 'error')
      return
    }

    // the account and its cookie are gone, close the live connection too
    disconnectRealtime()
    forgetSession()
    addNotification('Your account was deleted.')
    router.replace("/login")
  } catch (err) {
    addNotification(err.message || 'Failed to delete account', 'error')
  }
}

function onKeydown(event) {
  if (event.key === 'Escape') closeDeleteDialog()
}

watch(showDeleteDialog, async (open) => {
  if (open) {
    window.addEventListener('keydown', onKeydown)
    await nextTick()
    confirmInput.value?.focus()
    return
  }

  window.removeEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <AuthenticatedLayout active-page="settings">
    <div class="settings-page">
      <header class="settings-heading">
        <p class="orbit-meta">Preferences</p>
        <h1>Settings</h1>
        <p>Personalize how Orbit looks and manage your account.</p>
      </header>

      <section class="settings-section orbit-surface" aria-labelledby="appearance-title">
        <div class="settings-section__head">
          <h2 id="appearance-title">Appearance</h2>
          <p>Pick a theme. Your choice is saved in a cookie and applied every time you open Orbit.</p>
        </div>

        <div class="theme-grid" role="radiogroup" aria-label="Theme">
          <button v-for="theme in THEMES" :key="theme.id" type="button" role="radio" class="theme-card"
            :class="{ 'theme-card--active': currentTheme === theme.id }" :aria-checked="currentTheme === theme.id"
            @click="chooseTheme(theme)">
            <span class="theme-preview"
              :style="{ background: theme.preview.background, borderColor: theme.preview.surface }" aria-hidden="true">
              <span class="theme-preview__bar" :style="{ background: theme.preview.surface }"></span>
              <span class="theme-preview__body">
                <span class="theme-preview__card" :style="{ background: theme.preview.surface }">
                  <span class="theme-preview__line" :style="{ background: theme.preview.text }"></span>
                  <span class="theme-preview__line theme-preview__line--short"
                    :style="{ background: theme.preview.text }"></span>
                </span>
                <span class="theme-preview__dots">
                  <span :style="{ background: theme.preview.primary }"></span>
                  <span :style="{ background: theme.preview.secondary }"></span>
                </span>
              </span>
            </span>

            <span class="theme-card__meta">
              <strong>{{ theme.label }}</strong>
              <small>{{ theme.description }}</small>
            </span>

            <span v-if="currentTheme === theme.id" class="theme-card__check" aria-hidden="true">
              <IconGlyph name="check" :size="14" :stroke-width="3" />
            </span>
          </button>
        </div>
      </section>

      <section class="settings-section settings-section--danger orbit-surface" aria-labelledby="danger-title">
        <div class="settings-section__head">
          <h2 id="danger-title">Danger zone</h2>
          <p>Deleting your account permanently removes your profile, posts, comments, messages and group memberships.
          </p>
        </div>

        <button class="danger-button" type="button" @click="openDeleteDialog">
          <IconGlyph name="trash" :size="17" />
          <span>Delete account</span>
        </button>
      </section>
    </div>

    <Teleport to="body">
      <div v-if="showDeleteDialog" class="dialog-backdrop" @click.self="closeDeleteDialog">
        <div class="dialog orbit-surface" role="dialog" aria-modal="true" aria-labelledby="delete-title"
          aria-describedby="delete-description">
          <span class="dialog__icon" aria-hidden="true">
            <IconGlyph name="trash" :size="22" />
          </span>
          <h2 id="delete-title">Delete your account?</h2>
          <p id="delete-description">This will permanently remove your profile, posts, comments, messages and group
            memberships. This action cannot be undone.</p>

          <label class="dialog__field">
            <span>Type <strong>DELETE</strong> to confirm</span>
            <input ref="confirmInput" v-model="confirmText" type="text" autocomplete="off" spellcheck="false"
              @keydown.enter.prevent="confirmDelete" />
          </label>

          <div class="dialog__actions">
            <button type="button" class="ghost-button" @click="closeDeleteDialog">Cancel</button>
            <button type="button" class="danger-button danger-button--solid" :disabled="!canDelete"
              @click="confirmDelete">
              Delete my account
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </AuthenticatedLayout>
</template>

<style scoped>
.settings-page {
  display: grid;
  width: 100%;
  max-width: 58rem;
  margin: 0 auto;
  gap: var(--space-5);
}

.settings-heading .orbit-meta {
  margin: 0;
  color: var(--color-violet-soft);
}

.settings-heading h1 {
  margin: var(--space-2) 0;
  font-family: var(--font-display);
  font-size: 2.25rem;
  line-height: 1.15;
  letter-spacing: 0;
}

.settings-heading p:last-child {
  margin: 0;
  color: var(--color-text-muted);
}

.settings-section {
  display: grid;
  gap: var(--space-5);
  padding: var(--space-5);
}

.settings-section__head h2 {
  margin: 0 0 var(--space-1);
  font-family: var(--font-display);
  font-size: 1.25rem;
}

.settings-section__head p {
  margin: 0;
  color: var(--color-text-muted);
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(11rem, 1fr));
  gap: var(--space-4);
}

.theme-card {
  position: relative;
  display: grid;
  align-content: start;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 2px solid var(--color-border);
  border-radius: var(--radius-medium);
  background: var(--color-input);
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s ease, transform 0.15s ease;
}

.theme-card:hover {
  border-color: var(--color-border-hover);
  transform: translateY(-2px);
}

.theme-card--active,
.theme-card--active:hover {
  border-color: var(--color-mint);
  box-shadow: var(--focus-ring);
}

.theme-card__meta {
  display: grid;
  gap: 0.15rem;
}

.theme-card__meta strong {
  font-family: var(--font-display);
  font-size: 1rem;
}

.theme-card__meta small {
  color: var(--color-text-muted);
  font-size: 0.75rem;
  line-height: 1.4;
}

.theme-card__check {
  position: absolute;
  top: 1.1rem;
  right: 1.1rem;
  display: grid;
  width: 1.5rem;
  height: 1.5rem;
  place-items: center;
  border-radius: 50%;
  background: var(--color-mint);
  color: var(--color-on-accent);
}

.theme-preview {
  display: grid;
  grid-template-rows: 1.1rem 1fr;
  height: 6rem;
  overflow: hidden;
  border: 1px solid;
  border-radius: var(--radius-small);
}

.theme-preview__bar {
  display: block;
  opacity: 0.9;
}

.theme-preview__body {
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem;
}

.theme-preview__card {
  display: grid;
  flex: 1;
  align-content: center;
  gap: 0.3rem;
  padding: 0.4rem;
  border-radius: 0.3rem;
}

.theme-preview__line {
  display: block;
  height: 0.25rem;
  border-radius: 999px;
  opacity: 0.7;
}

.theme-preview__line--short {
  width: 60%;
  opacity: 0.4;
}

.theme-preview__dots {
  display: grid;
  gap: 0.35rem;
  align-content: center;
}

.theme-preview__dots span {
  display: block;
  width: 1rem;
  height: 1rem;
  border-radius: 50%;
}

.settings-section--danger {
  border-color: rgb(var(--rgb-coral) / 45%);
  background: linear-gradient(90deg, rgb(var(--rgb-coral) / 7%), var(--color-surface));
}

.settings-section--danger .settings-section__head h2 {
  color: var(--color-coral);
}

.danger-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  justify-self: start;
  gap: var(--space-2);
  min-height: var(--touch-target);
  padding: 0 var(--space-4);
  border: 1px solid var(--color-coral);
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-coral);
  cursor: pointer;
  font-weight: 700;
}

.danger-button:hover:not(:disabled) {
  background: rgb(var(--rgb-coral) / 12%);
}

.danger-button--solid {
  background: var(--color-coral);
  color: var(--color-on-accent);
}

.danger-button--solid:hover:not(:disabled) {
  background: var(--color-coral);
  filter: brightness(1.08);
}

.ghost-button {
  min-height: var(--touch-target);
  padding: 0 var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-text-soft);
  cursor: pointer;
  font-weight: 600;
}

.ghost-button:hover {
  border-color: var(--color-border-hover);
  background: var(--color-input);
}

.dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: grid;
  place-items: center;
  padding: var(--space-4);
  background: rgb(0 0 0 / 62%);
  backdrop-filter: blur(0.25rem);
}

.dialog {
  display: grid;
  width: min(28rem, 100%);
  gap: var(--space-4);
  padding: var(--space-5);
  box-shadow: var(--shadow-raised);
}

.dialog__icon {
  display: grid;
  width: 3rem;
  height: 3rem;
  place-items: center;
  border-radius: 50%;
  background: rgb(var(--rgb-coral) / 14%);
  color: var(--color-coral);
}

.dialog h2 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.375rem;
}

.dialog p {
  margin: 0;
  color: var(--color-text-muted);
}

.dialog__field {
  display: grid;
  gap: var(--space-2);
  color: var(--color-text-soft);
  font-size: 0.875rem;
}

.dialog__field strong {
  color: var(--color-coral);
  font-family: var(--font-meta);
}

.dialog__field input {
  min-height: var(--touch-target);
  padding: 0 var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
  color: var(--color-text);
  font-family: var(--font-meta);
}

.dialog__field input:focus {
  border-color: var(--color-coral);
  outline: none;
}

.dialog__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: var(--space-3);
}

@media (max-width: 600px) {
  .settings-heading h1 {
    font-size: 1.875rem;
  }

  .settings-section {
    padding: var(--space-4);
  }

  .dialog__actions>button {
    flex: 1;
  }
}
</style>
