<script setup>
// the small popup messages (toasts) at the top of the screen: errors, saved
// changes and every live notification (follows, groups, events, messages...)
import { useRouter } from 'vue-router'
import { notifications, removeNotification } from '@/data/notifications'
import IconGlyph from './IconGlyph.vue'

const router = useRouter()

const LABELS = {
  error: 'Something went wrong',
  message: 'New message',
  alert: 'Notification',
}

function label(notification) {
  return notification.title || LABELS[notification.type] || 'Orbit update'
}

// clicking a popup that points somewhere opens that page
function open(notification) {
  if (!notification.to) return
  removeNotification(notification.id)
  router.push(notification.to)
}
</script>

<template>
  <div class="notification-container" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="signal">
      <article v-for="notification in notifications" :key="notification.id" class="notification"
        :class="[`notification--${notification.type}`, { 'notification--link': notification.to }]"
        :role="notification.to ? 'link' : 'status'" :tabindex="notification.to ? 0 : -1"
        @click="open(notification)" @keydown.enter="open(notification)">
        <span class="notification__marker" aria-hidden="true"></span>
        <div class="notification__copy">
          <p class="notification__label">{{ label(notification) }}</p>
          <p class="notification__message">{{ notification.message }}</p>
        </div>
        <button type="button" aria-label="Dismiss notification" @click.stop="removeNotification(notification.id)"><IconGlyph name="close" :size="17" /></button>
      </article>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.notification-container { position: fixed; top: 1rem; right: 1rem; left: 1rem; z-index: 100; display: grid; gap: .75rem; pointer-events: none; }
.notification { position: relative; display: grid; grid-template-columns: .25rem 1fr auto; gap: .75rem; align-items: center; max-width: 28rem; margin-left: auto; padding: .9rem 1rem; overflow: hidden; border: 1px solid var(--color-border); border-radius: var(--radius-small); background: rgb(var(--rgb-surface) / 96%); box-shadow: 0 1rem 2.5rem rgb(0 0 0 / 28%); color: var(--color-text); pointer-events: auto; backdrop-filter: blur(.75rem); }
.notification::after { position: absolute; inset: 0; z-index: -1; background: linear-gradient(110deg, rgb(var(--rgb-mint) / 10%), transparent 45%); content: ''; }
.notification__marker { width: .25rem; min-height: 2.5rem; border-radius: 99rem; background: var(--color-mint); }
.notification--error .notification__marker, .notification--alert .notification__marker { background: var(--color-coral); }
.notification--alert::after { background: linear-gradient(110deg, rgb(var(--rgb-coral) / 12%), transparent 45%); }
.notification--link { cursor: pointer; }
.notification--link:hover { border-color: var(--color-violet); }
.notification__copy { min-width: 0; }
.notification__label, .notification__message { margin: 0; overflow-wrap: anywhere; }
.notification__label { color: var(--color-mint); font-family: var(--font-meta); font-size: .6875rem; letter-spacing: .1em; text-transform: uppercase; }
.notification--error .notification__label, .notification--alert .notification__label { color: var(--color-coral); }
.notification__message { margin-top: .2rem; color: var(--color-text-soft); font-size: .9375rem; line-height: 1.35; }
.notification button { display: grid; width: 2.75rem; height: 2.75rem; place-items: center; border: 0; background: transparent; color: var(--color-text-muted); cursor: pointer; font-size: 1.25rem; }
.notification button:hover { color: var(--color-text); }
.signal-enter-active, .signal-leave-active { transition: opacity .2s ease, transform .2s ease; }
.signal-enter-from, .signal-leave-to { opacity: 0; transform: translateY(-.5rem) translateX(1rem); }
@media (min-width: 48rem) { .notification-container { right: 1.5rem; left: auto; width: min(28rem, calc(100vw - 3rem)); } }
@media (prefers-reduced-motion: reduce) { .signal-enter-active, .signal-leave-active { transition: none; } }
</style>
