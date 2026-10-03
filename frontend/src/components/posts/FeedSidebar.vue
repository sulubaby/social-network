<script setup>
import { computed } from 'vue'
import { useNotifications } from '@/helpers/useNotifications.js'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import { notificationLink } from '@/helpers/notificationDisplay.js'
// the right side box on the home feed. it shows my 4 newest notifications
// using the same shared list as the top bar
const { items, unreadCount, error } = useNotifications()
const recent = computed(() => items.value.slice(0, 4))
</script>

<template>
  <aside class="feed-sidebar orbit-surface" aria-label="Recent activity">
    <header><h2>Your orbit</h2><span>{{ unreadCount }} unread</span></header>
    <p v-if="error" role="status">{{ error }}</p>
    <p v-else-if="!recent.length">No notifications yet. Updates from your community will appear here.</p>
    <ul v-else>
      <li v-for="item in recent" :key="item.id"><RouterLink :to="notificationLink(item) || '/notifications'">{{ item.message }}</RouterLink></li>
    </ul>
    <a class="activity-link" href="/notifications">View notifications <IconGlyph name="arrowRight" :size="15" /></a>
    <a class="activity-link" href="/groups">Explore groups <IconGlyph name="arrowRight" :size="15" /></a>
  </aside>
</template>

<style scoped>
.feed-sidebar { display: none; }
@media (min-width: 64rem) {
  .feed-sidebar { display: grid; gap: var(--space-4); width: 20rem; flex-shrink: 0; padding: var(--space-5); }
}
header { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
h2, p { margin: 0; }
h2 { font-size: 1.125rem; }
span, p { color: var(--color-text-muted); font-size: .875rem; }
ul { margin: 0; padding: 0; list-style: none; }
li { padding-block: .75rem; border-bottom: 1px solid var(--color-border); overflow-wrap: anywhere; }
a { color: var(--color-text-soft); text-decoration: none; }
a:hover { color: var(--color-mint); }
.activity-link { display: inline-flex; align-items: center; gap: .4rem; padding-block: .5rem; color: var(--color-mint); }
</style>
