<script setup>
import OrbitLogo from './OrbitLogo.vue'
import { useNotifications } from '@/helpers/useNotifications.js'
import { useChatCount } from '@/helpers/useChats.js'
import { logout } from '@/api/auth/auth.js'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { router } from '@/router/router.js'
import IconGlyph from './IconGlyph.vue'
import { getSearchResults } from '@/api/search.js'
// search box state and the little dropdown under it
const logoutError = ref('')
const searchText = ref('')
const suggestions = ref({ users: [], groups: [], posts: [] })
const searchOpen = ref(false)
const searchLoading = ref(false)
// suggestionRequestID helps us ignore old search answers that come back late
let suggestionTimer
let suggestionRequestID = 0

// my avatar for the top right corner
const props = defineProps(["avatar"]);

// true if the search found anything at all
const hasSuggestions = computed(() =>
  suggestions.value.users.length > 0 ||
  suggestions.value.groups.length > 0 ||
  suggestions.value.posts.length > 0,
)

// go to the full search page with what i typed
function navigateToSearch(value) {
  const search = value.trim()
  router.push(search ? { path: '/search', query: { q: search } } : '/search')
}

// builds the /uploads/ url for avatars
function avatarUrl(path) {
  if (!path) return ''
  return path.startsWith('/') ? path : `/uploads/${path}`
}

// empty and close the dropdown
function clearSuggestions() {
  suggestions.value = { users: [], groups: [], posts: [] }
  searchOpen.value = false
  searchLoading.value = false
}

// runs every time i type. it waits 350ms after i stop typing
// before asking the server (debounce), so we dont send a request for every letter.
// if an older answer comes back after a newer one we ignore it
function scheduleSuggestions(value) {
  clearTimeout(suggestionTimer)
  const search = value.trim()
  suggestionRequestID += 1

  if (!search) {
    clearSuggestions()
    return
  }

  searchOpen.value = true
  suggestionTimer = setTimeout(async () => {
    const requestID = suggestionRequestID
    searchLoading.value = true
    try {
      const result = await getSearchResults(search)
      if (requestID !== suggestionRequestID) return
      suggestions.value = {
        users: (result?.users || []).slice(0, 5),
        groups: (result?.groups || []).slice(0, 3),
        posts: (result?.posts || []).slice(0, 3),
      }
    } catch {
      if (requestID === suggestionRequestID) suggestions.value = { users: [], groups: [], posts: [] }
    } finally {
      if (requestID === suggestionRequestID) searchLoading.value = false
    }
  }, 350)
}

// pressing enter in the search box
function submitSearch() {
  clearTimeout(suggestionTimer)
  clearSuggestions()
  navigateToSearch(searchText.value)
}

// clicking a result: user -> their profile, group -> group page, post -> search page
function openSuggestion(type, item) {
  clearTimeout(suggestionTimer)
  clearSuggestions()

  if (type === 'users') {
    router.push(`/user?id=${item.id}`)
    return
  }

  if (type === 'groups') {
    router.push(`/groups/${item.id}`)
    return
  }

  router.push({ path: '/search', query: { q: searchText.value.trim(), type: 'posts' } })
}

// watch the search text and stop the timer when leaving the page
watch(searchText, scheduleSuggestions)
onBeforeUnmount(() => clearTimeout(suggestionTimer))

// log out button
async function signOut() {
  try {
    await logout()
    router.push("/login");
    return;
  } catch {
    logoutError.value = 'Could not log out. Please try again.'
  }
}
// the red numbers on the chat and notification icons
const { unreadCount: notificationUnreadCount } = useNotifications()
const { chatCount } = useChatCount()
</script>

<template>
  <header class="top-navigation">
    <a class="brand" href="/home-feed" aria-label="Orbit home">
      <OrbitLogo />
      <span>orbit</span>
    </a>

    <!-- search box with the live suggestions dropdown -->
    <div class="search-shell">
      <form class="search" role="search" @submit.prevent="submitSearch">
        <IconGlyph name="search" :size="16" />
        <input v-model="searchText" autocomplete="off" type="search" placeholder="Search people, groups, posts…"
          aria-label="Search" @focus="searchOpen = Boolean(searchText.trim())" />
      </form>

      <div v-if="searchOpen && searchText.trim()" class="search-suggestions" role="dialog"
        aria-label="Search suggestions">
        <p v-if="searchLoading" class="search-suggestions__state">Looking around your orbit…</p>

        <template v-else-if="hasSuggestions">
          <section v-if="suggestions.users.length" class="search-suggestions__section">
            <p class="search-suggestions__label">People</p>
            <button v-for="user in suggestions.users" :key="`user-${user.id}`" type="button" class="search-suggestion"
              @click="openSuggestion('users', user)">
              <span class="search-suggestion__avatar">
                <img v-if="avatarUrl(user.avatarPath)" :src="avatarUrl(user.avatarPath)"
                  :alt="`${user.firstName} ${user.lastName}`" />
                <span v-else>{{ `${user.firstName || ''}${user.lastName || ''}`.trim().slice(0, 2).toUpperCase() || '?'
                }}</span>
              </span>
              <span class="search-suggestion__copy"><strong>{{ user.firstName }} {{ user.lastName }}</strong><small>@{{
                user.username || 'orbit member' }}</small></span>
            </button>
          </section>

          <section v-if="suggestions.groups.length" class="search-suggestions__section">
            <p class="search-suggestions__label">Groups</p>
            <button v-for="group in suggestions.groups" :key="`group-${group.id}`" type="button"
              class="search-suggestion" @click="openSuggestion('groups', group)">
              <span class="search-suggestion__icon">
                <IconGlyph name="groups" :size="17" />
              </span>
              <span class="search-suggestion__copy"><strong>{{ group.title }}</strong><small>{{ group.memberCount }}
                  members</small></span>
            </button>
          </section>

          <section v-if="suggestions.posts.length" class="search-suggestions__section">
            <p class="search-suggestions__label">Posts</p>
            <button v-for="post in suggestions.posts" :key="`post-${post.id}`" type="button" class="search-suggestion"
              @click="openSuggestion('posts', post)">
              <span class="search-suggestion__icon">
                <IconGlyph name="image" :size="17" />
              </span>
              <span class="search-suggestion__copy"><strong>{{ post.author }}</strong><small>{{ post.content
                  }}</small></span>
            </button>
          </section>
        </template>

        <p v-else class="search-suggestions__state">No matches yet.</p>
        <button v-if="!searchLoading" type="button" class="search-suggestions__all" @click="submitSearch">See all
          results for
          “{{ searchText.trim() }}”</button>
      </div>
    </div>

    <!-- messages, notifications, my avatar and log out -->
    <nav class="top-actions" aria-label="Account shortcuts">
      <a class="icon-link orbit-touch-target" href="/chats" aria-label="Messages"
        :title="`${chatCount} unread message${chatCount === 1 ? '' : 's'}`">
        <IconGlyph name="chat" :size="19" />
        <span v-if="chatCount" class="badge badge--message">{{ chatCount }}</span>
      </a>
      <a class="icon-link orbit-touch-target" href="/notifications" aria-label="Notifications">
        <IconGlyph name="bell" :size="19" />
        <span v-if="notificationUnreadCount" class="badge badge--notification">{{ notificationUnreadCount }}</span>
      </a>
      <a class="avatar orbit-touch-target" href="/profile" aria-label="My profile">
        <img v-if="avatarUrl(avatar)" :src="avatarUrl(avatar)" alt="My profile" />
        <IconGlyph v-else name="profile" :size="18" stroke-width="2" />
      </a>
      <button class="sign-out" type="button" @click="signOut">Log out</button>
    </nav>
    <p v-if="logoutError" role="alert">{{ logoutError }}</p>
  </header>
</template>

<style scoped>
@import '../../styles/global.css';
@import '../../styles/variables.css';

.top-navigation {
  position: sticky;
  top: 0;
  z-index: 20;
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  min-height: 4rem;
  padding: var(--space-2) var(--space-3);
  background: rgb(var(--rgb-background) / 92%);
  border-bottom: 1px solid var(--color-border);
  backdrop-filter: blur(1rem);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-text);
  font-family: var(--font-display);
  font-size: 1.25rem;
  font-weight: 700;
  text-decoration: none;
}

.search {
  display: none;
  align-items: center;
  gap: var(--space-2);
  justify-self: center;
  width: min(100%, 28.5rem);
  min-height: var(--touch-target);
  padding-inline: var(--space-4);
  background: var(--color-input);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  color: var(--color-text-faint);
}

.search-shell {
  display: none;
  position: relative;
  width: min(100%, 28.5rem);
}

.search-shell .search {
  width: 100%;
}

.search-suggestions {
  position: absolute;
  top: calc(100% + var(--space-2));
  right: 0;
  left: 0;
  z-index: 30;
  display: grid;
  gap: var(--space-2);
  max-height: min(32rem, calc(100vh - 6rem));
  overflow-y: auto;
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-medium);
  background: rgb(var(--rgb-surface) / 98%);
  box-shadow: 0 1rem 2.5rem rgb(0 0 0 / 28%);
}

.search-suggestions__section {
  display: grid;
  gap: .2rem;
}

.search-suggestions__label {
  margin: var(--space-2) var(--space-2) .2rem;
  color: var(--color-text-faint);
  font-family: var(--font-meta);
  font-size: .68rem;
  letter-spacing: .12em;
  text-transform: uppercase;
}

.search-suggestions__state {
  margin: 0;
  padding: var(--space-4);
  color: var(--color-text-muted);
  text-align: center;
}

.search-suggestion {
  display: grid;
  width: 100%;
  grid-template-columns: 2.25rem minmax(0, 1fr);
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2);
  border: 0;
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
}

.search-suggestion:hover,
.search-suggestion:focus-visible {
  background: var(--color-surface-teal);
}

.search-suggestion__avatar,
.search-suggestion__icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  aspect-ratio: 1;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  background: var(--gradient-action);
  color: white;
  font-size: .72rem;
  font-weight: 700;
}

.search-suggestion__avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
}

.search-suggestion__icon {
  background: rgb(var(--rgb-mint) / 14%);
  color: var(--color-mint);
}

.search-suggestion__copy {
  display: grid;
  min-width: 0;
  gap: .1rem;
}

.search-suggestion__copy strong,
.search-suggestion__copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-suggestion__copy small {
  color: var(--color-text-muted);
  font-size: .75rem;
}

.search-suggestions__all {
  min-height: var(--touch-target);
  margin-top: var(--space-1);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: transparent;
  color: var(--color-violet-soft);
  cursor: pointer;
  font: inherit;
  font-size: .8125rem;
  font-weight: 700;
}

.search-suggestions__all:hover,
.search-suggestions__all:focus-visible {
  border-color: var(--color-violet);
  color: var(--color-text);
}

.top-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-1);
}

.sign-out {
  min-height: 44px;
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: .8125rem;
}

.sign-out:hover {
  color: var(--color-coral);
}

.icon-link,
.avatar {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  text-decoration: none;
}

.icon-link {
  width: var(--touch-target);
}



.avatar {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--touch-target);
  height: var(--touch-target);
  overflow: hidden;
  border: 2px solid var(--color-border);
  border-radius: 50%;
  background: var(--gradient-action);
  color: white;
  text-decoration: none;
  box-shadow: 0 0.25rem 0.75rem rgb(0 0 0 / 20%);
}

.avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
  border-radius: 50%;
}

.avatar::after {
  position: absolute;
  right: 0.05rem;
  bottom: 0.05rem;
  width: 0.55rem;
  height: 0.55rem;
  border: 2px solid var(--color-sidebar);
  border-radius: 50%;
  background: var(--color-mint);
  content: "";
}

.badge {
  position: absolute;
  top: 0.125rem;
  right: 0.125rem;
  display: grid;
  min-width: 1.1rem;
  height: 1.1rem;
  padding-inline: 0.25rem;
  place-items: center;
  border-radius: 999px;
  color: var(--color-on-accent);
  font-size: 0.7rem;
  font-weight: 700;
}

.badge--message {
  background: var(--color-mint);
}

.badge--notification {
  background: var(--color-coral);
}

@media (min-width: 48rem) {
  .top-navigation {
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 28.5rem) minmax(0, 1fr);
    padding-inline: var(--space-5);
  }

  .search-shell {
    display: flex;
    justify-self: center;
  }

  .search-shell .search {
    display: flex;
  }

  .search input {
    width: 100%;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--color-text);
  }

  .search input::placeholder {
    color: var(--color-text-faint);
  }

  .search:focus-within {
    border-color: var(--color-blue);
    box-shadow: var(--focus-ring);
  }

  .top-actions {
    justify-self: end;
    gap: var(--space-2);
  }
}
</style>
