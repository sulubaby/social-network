<script setup>
import { logout } from '@/api/auth/auth'
import { addNotification } from '@/data/notifications'
import { useNotifications } from '@/helpers/useNotifications.js'
import { useChatCount } from '@/helpers/useChats.js'
import { getGroups } from '@/api/groups/Groups'
import { onMounted, ref } from 'vue'
import IconGlyph from './IconGlyph.vue'

// activePage is the page we are on, so its link gets highlighted
defineProps({
  activePage: {
    type: String,
    required: true,
  },
})

// up to 4 of my groups shown under the menu
const myGroups = ref([])

onMounted(async () => {
  try {
    const result = await getGroups()
    myGroups.value = (result?.groups || []).filter((group) => group.isMember).slice(0, 4)
  } catch {
    // The main navigation should remain usable if the groups request fails.
  }
})

// the menu links. badgeType says which red number to show next to it
const links = [
  { name: 'home', label: 'Home', href: '/home-feed', icon: 'home' },
  { name: 'profile', label: 'Profile', href: '/profile', icon: 'profile' },
  { name: 'groups', label: 'Groups', href: '/groups', icon: 'groups' },
  { name: 'chats', label: 'Chats', href: '/chats', icon: 'chat', badgeType: 'message' },
  { name: 'notifications', label: 'Notifications', href: '/notifications', icon: 'bell', badgeType: 'notification' },
  { name: 'settings', label: 'Settings', href: '/settings', icon: 'settings' },
]

// unread numbers for the badges
const { unreadCount: notificationUnreadCount } = useNotifications()
const { chatCount } = useChatCount()

// log out, if it fails show a small error popup
async function logoutHandler() {
    try {
        await logout();
    } catch (err) {
        addNotification(err, 'error');
    }
}


</script>

<template>
  <aside class="side-navigation">
    <!-- main menu for big screens -->
    <nav aria-label="Main navigation">
      <a v-for="link in links" :key="link.name" class="navigation-link"
        :class="{ 'navigation-link--active': activePage === link.name }" :href="link.href"
        :aria-current="activePage === link.name ? 'page' : undefined">
        <span class="navigation-link__icon"><IconGlyph :name="link.icon" :size="18" /></span>
        <span class="navigation-link__label">{{ link.label }}</span>
        <span
          v-if="link.name === 'notifications' ? notificationUnreadCount : link.name === 'chats' ? chatCount : link.badge"
          class="navigation-link__badge"
          :class="`navigation-link__badge--${link.badgeType}`"
        >
          {{ link.name === 'notifications' ? notificationUnreadCount : link.name === 'chats' ? chatCount : link.badge }}
        </span>
      </a>
    </nav>

    <!-- my groups shortcuts -->
    <section v-if="myGroups.length" class="my-groups" aria-labelledby="my-groups-title">
      <h2 id="my-groups-title">My groups</h2>
      <a v-for="group in myGroups" :key="group.id" :href="`/groups/${group.id}`" class="my-group-link">
        <span class="my-group-link__dot" aria-hidden="true"></span>
        <span>{{ group.title }}</span>
      </a>
    </section>

    <button class="logout-link" type="button" @click="logoutHandler"><IconGlyph name="logout" :size="18" /> <span>Log out</span></button>
  </aside>

  <!-- same links as a bottom bar on phones -->
  <nav class="mobile-navigation" aria-label="Mobile navigation">
    <a v-for="link in links" :key="link.name" class="mobile-link"
      :class="{ 'mobile-link--active': activePage === link.name }" :href="link.href" :aria-label="link.label"
      :aria-current="activePage === link.name ? 'page' : undefined">
      <IconGlyph :name="link.icon" :size="19" />
      <span>{{ link.label }}</span>
    </a>
  </nav>
</template>

<style scoped>
@import '../../styles/global.css';
@import '../../styles/variables.css';

.side-navigation {
  display: none;
}

.mobile-navigation {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  min-height: 4.25rem;
  background: rgb(var(--rgb-background) / 96%);
  border-top: 1px solid var(--color-border);
  backdrop-filter: blur(1rem);
}

.mobile-link {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
  color: var(--color-text-faint);
  font-size: 0.65rem;
  text-decoration: none;
}

.mobile-link > span {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-link > .icon-glyph {
  width: 1.25rem;
  height: 1.25rem;
}

.mobile-link--active {
  color: var(--color-mint);
}

@media (min-width: 64rem) {
  .side-navigation {
    position: sticky;
    top: 4rem;
    display: flex;
    align-self: start;
    flex-direction: column;
    justify-content: space-between;
    width: 14.5rem;
    height: calc(100vh - 4rem);
    padding: var(--space-4) var(--space-3) var(--space-5);
    background: var(--color-sidebar);
    border-right: 1px solid var(--color-border);
  }

  .navigation-link {
    display: grid;
    grid-template-columns: 1.75rem 1fr auto;
    align-items: center;
    min-height: var(--touch-target);
    padding-inline: var(--space-3);
    border-radius: var(--radius-small);
    color: var(--color-text-muted);
    text-decoration: none;
  }

  .navigation-link:hover,
  .navigation-link--active {
    background: rgb(var(--rgb-mint) / 9%);
    color: var(--color-mint);
  }

  .navigation-link__icon {
    font-size: 1.25rem;
  }

  .navigation-link__badge {
    display: grid;
    min-width: 1.35rem;
    height: 1.35rem;
    place-items: center;
    border-radius: 999px;
    color: var(--color-on-accent);
    font-size: 0.75rem;
    font-weight: 700;
  }

  .navigation-link__badge--message {
    background: var(--color-mint);
  }

  .navigation-link__badge--notification {
    background: var(--color-coral);
  }

  .my-groups {
    display: grid;
    gap: var(--space-2);
    margin-top: var(--space-5);
    padding-inline: var(--space-3);
  }

  .my-groups h2 {
    margin: 0 0 var(--space-1);
    color: var(--color-text-faint);
    font-family: var(--font-meta);
    font-size: 0.6875rem;
    font-weight: 500;
    letter-spacing: 0.09em;
    text-transform: uppercase;
  }

  .my-group-link {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    color: var(--color-text-muted);
    font-size: 0.8125rem;
    text-decoration: none;
  }

  .my-group-link span:last-child {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .my-group-link:hover {
    color: var(--color-mint);
  }

  .my-group-link__dot {
    width: 0.5rem;
    height: 0.5rem;
    flex: 0 0 auto;
    border-radius: 50%;
    background: var(--color-violet);
  }

  .logout-link {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: var(--touch-target);
    padding-inline: var(--space-3);
    border: 0;
    background: transparent;
    color: var(--color-text-faint);
    cursor: pointer;
    font: inherit;
    text-decoration: none;
  }

  .mobile-navigation {
    display: none;
  }
}
</style>
