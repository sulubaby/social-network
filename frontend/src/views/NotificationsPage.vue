<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import {
  applyNotificationAction,
  getNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from '@/api/notifications.js'
import { subscribeRealtime } from '@/services/realtime.js'

// the filter tabs on top of the page
const filters = [
  { id: 'all', label: 'All' },
  { id: 'requests', label: 'Requests' },
  { id: 'groups', label: 'Groups' },
  { id: 'events', label: 'Events' },
  { id: 'messages', label: 'Messages' },
]

// page state: the list, loading flags, errors and the unread number
const notificationItems = ref([])
const isLoading = ref(true)
const isLoadingMore = ref(false)
const hasMoreNotifications = ref(false)
const unreadTotal = ref(0)
const loadError = ref('')
const notificationSentinel = ref(null)
const NOTIFICATION_PAGE_SIZE = 20
let notificationObserver
let stopNotificationListener
let stopConnectionListener
// time of the last full load, so the websocket connect doesnt load the same page again
let lastLoadedAt = 0

const activeFilter = ref('all')

// only show the notifications of the tab that is picked
const visibleNotifications = computed(() => {
  if (activeFilter.value === 'all') return notificationItems.value
  return notificationItems.value.filter((item) => item.type === activeFilter.value)
})

const unreadCount = computed(() => unreadTotal.value)

// turns the date from the server into something like "Sep 27, 2026, 3:20 PM"
function formatNotificationTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Recently'

  return date.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
}

// the text we show after the user already answered (accepted, declined...)
function actionLabel(action) {
  return {
    accept: 'Accepted',
    decline: 'Declined',
    reject: 'Rejected',
    join: 'Joined',
    rsvp: 'Going',
  }[action] || action
}

// takes a notification from the server and adds what the page needs to show it:
// icon, color, time text, and which buttons to show (action)
function notificationForDisplay(notification) {
  const categoryStyles = {
    requests: { icon: 'profile', color: '#7c5cff' },
    groups: { icon: 'groups', color: '#3ee6b0' },
    events: { icon: 'calendar', color: '#ffb84d' },
    messages: { icon: 'chat', color: '#55b7ff' },
  }

  const style =
    categoryStyles[notification.category] || categoryStyles.groups

  let action = ''

  // follow request: 0 = still waiting so show buttons, 1 = already accepted
  if (notification.category === 'requests' && notification.type === 'follow_request') {
    action = notification.followStatus === 0 ? 'follow' : notification.followStatus === 1 ? 'accept' : ''
  }

  // someone asked to join my group
  if (notification.category === 'groups' && notification.type === 'join_request') {
    if (notification.requestStatus === 'accepted') {
      action = 'accept'
    } else if (notification.requestStatus === 'rejected') {
      action = 'reject'
    } else {
      action = 'join_request'
    }
  }

  // someone invited me to a group
  if (notification.category === 'groups' && notification.type === 'invitation') {
    if (notification.invitationStatus === 'accepted') {
      action = 'join'
    } else if (notification.invitationStatus === 'declined') {
      action = 'decline'
    } else {
      action = 'invitation'
    }
  }

  // new event: going / not going
  if (notification.category === 'events' && notification.type === 'event_created') {
    action = notification.eventResponse === 'going' ? 'going' : notification.eventResponse === 'declined' ? 'decline' : 'rsvp'
  }

  return {
    ...notification,
    type: notification.category,
    icon: style.icon,
    color: style.color,
    title: notification.message,
    detail: notification.type.replaceAll('_', ' '),
    time: formatNotificationTime(notification.createdAt),
    unread: !notification.isRead,
    action,
  }
}

// a new notification came from the websocket, put it on top of the list
// (or update it if we already have it)
function receiveRealtimeNotification(event) {
  const notification = event?.notification
  if (!notification?.id) return
  const item = notificationForDisplay(notification)
  const index = notificationItems.value.findIndex(existing => existing.id === item.id)
  if (index >= 0) {
    notificationItems.value[index] = { ...notificationItems.value[index], ...item }
    return
  }
  notificationItems.value.unshift(item)
  if (item.unread) unreadTotal.value += 1
}

// loads the notifications from the server.
// append = true is for infinite scroll (add the next page at the bottom)
async function loadNotifications({ append = false } = {}) {
  if (append) {
    if (isLoadingMore.value || !hasMoreNotifications.value) return
    isLoadingMore.value = true
  } else {
    isLoading.value = true
    lastLoadedAt = Date.now()
  }
  loadError.value = ''

  try {
    const result = await getNotifications('all', {
      limit: NOTIFICATION_PAGE_SIZE,
      offset: append ? notificationItems.value.length : 0,
    })
    const nextItems = (result?.notifications || []).map(notificationForDisplay)
    notificationItems.value = append ? [...notificationItems.value, ...nextItems] : nextItems
    unreadTotal.value = result?.unreadCount || 0
    hasMoreNotifications.value = Boolean(result?.hasMore)
  } catch (error) {
    loadError.value = error.message || 'Could not load notifications.'
  } finally {
    isLoading.value = false
    isLoadingMore.value = false
  }
}

// clicking a notification marks it as read
async function markAsRead(item) {
  if (!item.unread) return

  try {
    await markNotificationRead(item.id)
    item.unread = false
    unreadTotal.value = Math.max(0, unreadTotal.value - 1)
  } catch (error) {
    loadError.value = error.message || 'Could not mark notification as read.'
  }
}

// the "mark all as read" button
async function markAllAsRead() {
  try {
    await markAllNotificationsRead()
    unreadTotal.value = 0
    notificationItems.value.forEach((item) => {
      item.unread = false
    })
  } catch (error) {
    loadError.value = error.message || 'Could not mark notifications as read.'
  }
}

// runs when a button in a notification is clicked (accept, decline, join...)
// busy stops double clicks, and if it fails we put the old state back
async function chooseAction(item, action) {
  if (item.busy) return
  item.busy = true
  const previousAction = item.action

  try {
    await applyNotificationAction(item.id, action)
    item.action = action
    if (item.unread) unreadTotal.value = Math.max(0, unreadTotal.value - 1)
    item.unread = false
  } catch (error) {
    item.action = previousAction
    loadError.value = error.message || 'Could not complete notification action.'
  } finally {
    item.busy = false
  }
}

// infinite scroll: when the empty div at the bottom comes on screen we load the next page
function observeNotificationEnd() {
  if (!notificationSentinel.value || typeof IntersectionObserver === 'undefined') return

  notificationObserver = new IntersectionObserver(([entry]) => {
    if (entry.isIntersecting && hasMoreNotifications.value && !isLoading.value && !isLoadingMore.value) {
      loadNotifications({ append: true })
    }
  }, {
    // Prefetch before the user reaches the end of a long notification list.
    rootMargin: '0px 0px 320px',
  })

  notificationObserver.observe(notificationSentinel.value)
}

// when the page opens: listen for live notifications, start the scroll watcher, load the first page
onMounted(async () => {
  stopNotificationListener = subscribeRealtime('notification', receiveRealtimeNotification)
  stopConnectionListener = subscribeRealtime('connection', event => {
    if (event.status === 'connected' && !isLoading.value && Date.now() - lastLoadedAt > 3000) loadNotifications()
  })
  observeNotificationEnd()
  await loadNotifications()
})

// clean up the listeners when we leave the page
onBeforeUnmount(() => {
  notificationObserver?.disconnect()
  stopNotificationListener?.()
  stopConnectionListener?.()
})
</script>

<template>
  <AuthenticatedLayout active-page="notifications">
    <div class="notifications-layout">
      <section class="notifications-page orbit-surface" aria-labelledby="notifications-title">
      <div class="notifications-sticky-controls">
        <!-- title, unread count and the mark all as read button -->
        <header class="notifications-page__header">
          <div>
            <p class="orbit-meta">Stay in the loop</p>

            <h1 id="notifications-title">
              Notifications
            </h1>

            <p class="notifications-page__summary">
              {{ unreadCount ? `${unreadCount} unread updates` : 'You are all caught up' }}
            </p>
          </div>

          <button class="mark-all-button" type="button" :disabled="unreadCount === 0" @click="markAllAsRead">
            Mark all as read
          </button>
        </header>

        <!-- filter tabs -->
        <nav class="notification-filters" aria-label="Notification filters">
          <button v-for="filter in filters" :key="filter.id" type="button"
            :class="{ 'notification-filter--active': activeFilter === filter.id }" @click="activeFilter = filter.id">
            {{ filter.label }}
          </button>
        </nav>
      </div>

      <!-- loading / error / list / empty states -->
      <p v-if="isLoading" class="notifications-state">
        Loading notifications...
      </p>

      <div v-else-if="loadError" class="notifications-state notifications-state--error">
        <span>{{ loadError }}</span>

        <button type="button" @click="loadNotifications">
          Try again
        </button>
      </div>

      <div v-else-if="visibleNotifications.length" class="notification-list">
        <!-- one notification -->
        <article v-for="item in visibleNotifications" :key="item.id" class="notification-item"
          :class="{ 'notification-item--unread': item.unread }" @click="markAsRead(item)">
          <div class="notification-item__icon" :style="{ background: item.color }" aria-hidden="true">
            <IconGlyph :name="item.icon" :size="19" :stroke-width="2" />
          </div>

          <div class="notification-item__body">
            <h2>{{ item.title }}</h2>
            <p>{{ item.detail }}</p>
            <time>{{ item.time }}</time>
          </div>

          <!-- Follow request -->
          <!-- the buttons change depending on the notification type.
               if it was already answered we just show the result text -->
          <div v-if="item.action === 'follow'" class="notification-item__actions">
            <button type="button" class="action-button action-button--primary"
              @click.stop="chooseAction(item, 'accept')">
              Accept
            </button>

            <button type="button" class="action-button" @click.stop="chooseAction(item, 'decline')">
              Decline
            </button>
          </div>

          <!-- Someone requested to join my group -->
          <div v-else-if="item.action === 'join_request'" class="notification-item__actions">
            <button type="button" class="action-button action-button--primary"
              @click.stop="chooseAction(item, 'accept')">
              Accept
            </button>

            <button type="button" class="action-button" @click.stop="chooseAction(item, 'reject')">
              Reject
            </button>
          </div>

          <!-- I received a group invitation -->
          <div v-else-if="item.action === 'invitation'" class="notification-item__actions">
            <button type="button" class="action-button action-button--primary" @click.stop="chooseAction(item, 'join')">
              Join
            </button>

            <button type="button" class="action-button" @click.stop="chooseAction(item, 'decline')">
              Decline
            </button>
          </div>

          <!-- Event -->
          <div v-else-if="item.action === 'rsvp'" class="notification-item__actions">
            <button type="button" class="action-button action-button--primary" @click.stop="chooseAction(item, 'rsvp')">
              RSVP
            </button>

            <button type="button" class="action-button" @click.stop="chooseAction(item, 'decline')">
              Decline
            </button>
          </div>

          <span v-else-if="item.action" class="notification-item__result" :class="{
            'notification-item__result--negative':
              item.action === 'reject' || item.action === 'decline'
          }">
            {{ actionLabel(item.action) }}
          </span>
        </article>

      </div>

      <div v-else class="notifications-empty">
        <p>Nothing here yet.</p>
      </div>

      <!-- empty div for infinite scroll -->
      <div ref="notificationSentinel" class="notification-load-sentinel" aria-hidden="true"></div>
      <p v-if="isLoadingMore" class="notification-load-state" role="status">Loading more updates...</p>
      <p v-else-if="!hasMoreNotifications && notificationItems.length" class="notification-load-state">You’re all caught up.</p>
      </section>

      <!-- side box that explains the notification types -->
      <aside class="notification-legend orbit-surface" aria-labelledby="notification-legend-title">
        <p class="orbit-meta">How Orbit speaks</p>
        <h2 id="notification-legend-title">Two kinds of signals</h2>

        <div class="notification-legend__item">
          <span class="notification-legend__icon notification-legend__icon--coral"><IconGlyph name="bell" :size="19" /></span>
          <div>
            <strong>Coral bell</strong>
            <p>Requests, group updates, and event reminders.</p>
          </div>
        </div>

        <div class="notification-legend__item">
          <span class="notification-legend__icon notification-legend__icon--mint"><IconGlyph name="chat" :size="19" /></span>
          <div>
            <strong>Mint bubble</strong>
            <p>New messages waiting in your chats.</p>
          </div>
        </div>
      </aside>
    </div>
  </AuthenticatedLayout>
</template>

<style scoped>
.notifications-page {
  width: 100%;
  min-width: 0;
  margin: 0 auto;
  padding: var(--space-4);
}

.notifications-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--space-5);
  width: min(100%, 74rem);
  margin: 0 auto;
}

.notifications-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.notifications-sticky-controls {
  position: sticky;
  top: 4rem;
  z-index: 10;
  padding: 0 0 var(--space-2);
  background: var(--color-surface);
  box-shadow: 0 0.75rem 1rem rgb(0 0 0 / 12%);
}

.notifications-page h1,
.notifications-page__summary,
.notification-item h2,
.notification-item p,
.notification-item time {
  margin: 0;
}

.notifications-page h1 {
  margin-top: var(--space-1);
  font-family: var(--font-display);
  font-size: clamp(1.75rem, 5vw, 2.5rem);
}

.notifications-page__summary {
  margin-top: var(--space-2);
  color: var(--color-text-muted);
}

.mark-all-button,
.notification-filter,
.action-button {
  min-height: var(--touch-target);
  border-radius: 999px;
  cursor: pointer;
}

.mark-all-button {
  padding-inline: var(--space-3);
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.mark-all-button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.notification-filters {
  display: flex;
  gap: var(--space-2);
  margin-top: var(--space-3);
  overflow-x: auto;
  padding-bottom: var(--space-1);
}

.notification-filters button {
  min-height: var(--touch-target);
  padding-inline: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  white-space: nowrap;
}

.notification-filters button:hover,
.notification-filter--active {
  border-color: var(--color-violet) !important;
  background: rgb(var(--rgb-violet) / 16%) !important;
  color: var(--color-text) !important;
}

.notification-list {
  display: grid;
  gap: var(--space-3);
  margin-top: var(--space-5);
}

.notification-load-sentinel {
  width: 100%;
  height: 1px;
  pointer-events: none;
}

.notification-load-state {
  margin: var(--space-2) 0 0;
  padding: var(--space-3) 0 var(--space-1);
  color: var(--color-text-muted);
  font-size: 0.8125rem;
  text-align: center;
}

.notification-item {
  display: grid;
  grid-template-columns: var(--touch-target) minmax(0, 1fr);
  align-items: start;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid transparent;
  border-radius: var(--radius-medium);
  background: var(--color-input);
  cursor: pointer;
}

.notification-item--unread {
  border-color: rgb(var(--rgb-coral) / 48%);
  background: linear-gradient(90deg, rgb(var(--rgb-coral) / 9%), var(--color-input));
}

.notification-item__icon {
  display: grid;
  width: var(--touch-target);
  height: var(--touch-target);
  place-items: center;
  border-radius: 50%;
  color: var(--color-on-accent);
  font-weight: 700;
}

.notification-item__body {
  min-width: 0;
}

.notification-item__body h2 {
  color: var(--color-text);
  font-size: 1rem;
  font-weight: 600;
}

.notification-item__body p {
  margin-top: var(--space-1);
  color: var(--color-text-muted);
  overflow-wrap: anywhere;
}

.notification-item__body time {
  display: block;
  margin-top: var(--space-2);
  color: var(--color-text-faint);
  font-family: var(--font-meta);
  font-size: 0.75rem;
}

.notification-item__actions,
.notification-item__result {
  grid-column: 2;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.action-button {
  padding-inline: var(--space-3);
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-muted);
}

.action-button--primary {
  border-color: var(--color-mint);
  color: var(--color-mint);
}

.notification-item__result {
  color: var(--color-mint);
  font-size: 0.875rem;
  text-transform: capitalize;
}

.notifications-empty {
  margin: var(--space-6) 0 0;
  color: var(--color-text-muted);
  text-align: center;
}

.notifications-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-top: var(--space-6);
  color: var(--color-text-muted);
}

.notifications-state--error {
  color: var(--color-coral);
}

.notifications-state button {
  min-height: var(--touch-target);
  padding-inline: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.notification-legend {
  align-self: start;
  padding: var(--space-5);
}

.notification-legend h2 {
  margin: var(--space-1) 0 0;
  font-family: var(--font-display);
  font-size: 1.25rem;
}

.notification-legend__item {
  display: grid;
  grid-template-columns: 2.5rem minmax(0, 1fr);
  gap: var(--space-3);
  align-items: start;
  margin-top: var(--space-5);
}

.notification-legend__icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  place-items: center;
  border-radius: 50%;
  font-size: 1.25rem;
  font-weight: 700;
}

.notification-legend__icon--coral {
  background: rgb(var(--rgb-coral) / 16%);
  color: var(--color-coral);
}

.notification-legend__icon--mint {
  background: rgb(var(--rgb-mint) / 16%);
  color: var(--color-mint);
}

.notification-legend strong {
  color: var(--color-text);
  font-size: 0.9375rem;
}

.notification-legend p:not(.orbit-meta) {
  margin: var(--space-1) 0 0;
  color: var(--color-text-muted);
  font-size: 0.875rem;
  line-height: 1.5;
}

@media (min-width: 48rem) {
  .notifications-layout {
    grid-template-columns: minmax(0, 1fr) 18rem;
    align-items: start;
  }

  .notifications-page {
    padding: var(--space-6);
  }

  .notification-item {
    grid-template-columns: var(--touch-target) minmax(0, 1fr) auto;
    align-items: center;
    padding: var(--space-4);
  }

  .notification-item__actions,
  .notification-item__result {
    grid-column: auto;
  }
}

.notification-item__result--negative {
  color: var(--color-coral);
}
</style>
