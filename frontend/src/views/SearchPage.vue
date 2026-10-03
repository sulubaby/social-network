<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import { getSearchResults } from '@/api/search.js'
import { requestFollow } from '@/api/users/profiles.js'

const route = useRoute()
const router = useRouter()
const SEARCH_TABS = [
  { id: 'all', label: 'All' },
  { id: 'users', label: 'People' },
  { id: 'groups', label: 'Groups' },
  { id: 'posts', label: 'Posts' },
]
const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
const results = ref({ users: [], groups: [], posts: [] })
const isLoading = ref(false)
const error = ref('')
const initialTab = typeof route.query.type === 'string' && SEARCH_TABS.some((tab) => tab.id === route.query.type)
  ? route.query.type
  : 'all'
const activeTab = ref(initialTab)
const busyUsers = ref(new Set())
const SEARCH_DEBOUNCE_MS = 350
let searchTimer
let searchRequestID = 0

const visibleSections = computed(() => {
  const sections = [
    { id: 'users', label: 'People', icon: 'profile', items: results.value.users },
    { id: 'groups', label: 'Groups', icon: 'groups', items: results.value.groups },
    { id: 'posts', label: 'Posts', icon: 'image', items: results.value.posts },
  ]

  if (activeTab.value === 'all') return sections.filter((section) => section.items.length)
  return sections.filter((section) => section.id === activeTab.value && section.items.length)
})

const hasVisibleResults = computed(() => visibleSections.value.length > 0)

const counts = computed(() => ({
  all: results.value.users.length + results.value.groups.length + results.value.posts.length,
  users: results.value.users.length,
  groups: results.value.groups.length,
  posts: results.value.posts.length,
}))

function avatarUrl(path) {
  if (!path) return ''
  return path.startsWith('/') ? path : `/uploads/${path}`
}

function initials(user) {
  return `${user.firstName || ''}${user.lastName || ''}`.trim().slice(0, 2).toUpperCase() || '?'
}

function formatTime(value) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'Recently' : date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

async function performSearch() {
  const cleanQuery = query.value.trim()
  const requestID = ++searchRequestID
  error.value = ''
  if (!cleanQuery) {
    results.value = { users: [], groups: [], posts: [] }
    isLoading.value = false
    return
  }

  isLoading.value = true
  try {
    const result = await getSearchResults(cleanQuery)
    if (requestID !== searchRequestID) return
    results.value = {
      users: result?.users || [],
      groups: result?.groups || [],
      posts: result?.posts || [],
    }
  } catch (searchError) {
    if (requestID !== searchRequestID) return
    error.value = searchError.message || 'Could not search Orbit.'
  } finally {
    if (requestID === searchRequestID) isLoading.value = false
  }
}

function updateSearchRoute(cleanQuery) {
  const nextQuery = {}
  if (cleanQuery) nextQuery.q = cleanQuery
  if (cleanQuery && activeTab.value !== 'all') nextQuery.type = activeTab.value
  router.replace({ path: '/search', query: nextQuery })
}

function scheduleSearch({ updateRoute = true } = {}) {
  clearTimeout(searchTimer)
  const cleanQuery = query.value.trim()
  if (updateRoute) updateSearchRoute(cleanQuery)

  searchTimer = setTimeout(() => performSearch(), SEARCH_DEBOUNCE_MS)
}

function submitSearch() {
  clearTimeout(searchTimer)
  updateSearchRoute(query.value.trim())
  performSearch()
}

function selectTab(tabID) {
  activeTab.value = tabID
  updateSearchRoute(query.value.trim())
}

function openUser(userID) {
  router.push(`/user?id=${userID}`)
}

function openGroup(groupID) {
  router.push(`/groups/${groupID}`)
}

async function toggleFollow(user) {
  if (busyUsers.value.has(user.id)) return

  busyUsers.value = new Set(busyUsers.value).add(user.id)
  try {
    const shouldRemove = user.followStatus >= 0
    if (user.followStatus === 1 && !window.confirm(`Unfollow ${user.firstName} ${user.lastName}?`)) return
    const result = await requestFollow(user.id, shouldRemove ? 'DELETE' : 'POST')
    user.followStatus = result?.followStatus ?? (shouldRemove ? -1 : 1)
  } catch (followError) {
    error.value = followError.message || 'Could not update follow status.'
  } finally {
    const next = new Set(busyUsers.value)
    next.delete(user.id)
    busyUsers.value = next
  }
}

watch(query, () => scheduleSearch())

watch(() => [route.query.q, route.query.type], ([value, type]) => {
  const nextQuery = typeof value === 'string' ? value : ''
  const nextTab = SEARCH_TABS.some((tab) => tab.id === type) ? type : 'all'
  if (query.value !== nextQuery) {
    query.value = nextQuery
    scheduleSearch({ updateRoute: false })
  }
  if (activeTab.value !== nextTab) activeTab.value = nextTab
})

onMounted(performSearch)
onBeforeUnmount(() => clearTimeout(searchTimer))
</script>

<template>
  <AuthenticatedLayout active-page="search">
    <main class="search-page">
      <header class="search-page__header">
        <div>
          <p class="orbit-meta">Find your orbit</p>
          <h1>Search</h1>
          <p class="search-page__intro">People, groups, and posts you can actually see.</p>
        </div>

        <form class="search-page__form" role="search" @submit.prevent="submitSearch">
          <IconGlyph name="search" :size="17" />
          <input v-model="query" type="search" placeholder="Try a name, group, or phrase" aria-label="Search Orbit" />
          <button type="submit">Search</button>
        </form>
      </header>

      <nav v-if="query.trim()" class="search-tabs" aria-label="Search result types">
        <button v-for="tab in SEARCH_TABS" :key="tab.id" type="button" :class="{ 'search-tab--active': activeTab === tab.id }" @click="selectTab(tab.id)">
          {{ tab.label }} <span>{{ counts[tab.id] }}</span>
        </button>
      </nav>

      <p v-if="isLoading" class="search-state orbit-surface">Searching your orbit…</p>
      <p v-else-if="error" class="search-state search-state--error orbit-surface">{{ error }}</p>
      <p v-else-if="query.trim() && !hasVisibleResults" class="search-state orbit-surface">No {{ SEARCH_TABS.find((tab) => tab.id === activeTab)?.label.toLowerCase() || 'matching' }} found. Try another filter or phrase.</p>
      <p v-else-if="!query.trim()" class="search-state search-state--empty orbit-surface">Start with a name, group, or post phrase.</p>

      <div v-else class="search-results">
        <section v-for="section in visibleSections" :key="section.id" class="search-section orbit-surface">
          <header class="search-section__header">
            <div>
              <p class="orbit-meta">{{ section.label }}</p>
              <h2><IconGlyph :name="section.icon" :size="19" /> {{ section.items.length }} {{ section.items.length === 1 ? 'result' : 'results' }}</h2>
            </div>
          </header>

          <div v-if="section.id === 'users'" class="search-user-list">
            <article v-for="user in section.items" :key="user.id" class="search-user-card">
              <button class="search-user-card__identity" type="button" @click="openUser(user.id)">
                <span class="search-avatar">
                  <img v-if="avatarUrl(user.avatarPath)" :src="avatarUrl(user.avatarPath)" :alt="`${user.firstName} ${user.lastName}`" />
                  <span v-else>{{ initials(user) }}</span>
                </span>
                <span>
                  <strong>{{ user.firstName }} {{ user.lastName }}</strong>
                  <small>@{{ user.username || 'orbit member' }} <span v-if="user.isPrivate">· private</span></small>
                </span>
              </button>
              <button class="search-follow-button" type="button" :disabled="busyUsers.has(user.id)" @click="toggleFollow(user)">
                {{ user.followStatus === 1 ? 'Following' : user.followStatus === 0 ? 'Requested' : 'Follow' }}
              </button>
            </article>
          </div>

          <div v-else-if="section.id === 'groups'" class="search-card-grid">
            <button v-for="group in section.items" :key="group.id" class="search-result-card" type="button" @click="openGroup(group.id)">
              <span class="search-result-card__icon"><IconGlyph name="groups" :size="20" /></span>
              <span>
                <strong>{{ group.title }}</strong>
                <small>{{ group.memberCount }} members</small>
                <p>{{ group.description }}</p>
              </span>
            </button>
          </div>

          <div v-else class="search-post-list">
            <article v-for="post in section.items" :key="post.id" class="search-post-card">
              <header>
                <span class="search-avatar search-avatar--small">
                  <img v-if="avatarUrl(post.avatarPath)" :src="avatarUrl(post.avatarPath)" alt="" />
                  <span v-else>{{ post.author.slice(0, 2).toUpperCase() }}</span>
                </span>
                <div><strong>{{ post.author }}</strong><small>{{ formatTime(post.createdAt) }} · {{ post.privacy }}</small></div>
              </header>
              <p>{{ post.content }}</p>
              <img v-if="avatarUrl(post.imagePath)" class="search-post-card__image" :src="avatarUrl(post.imagePath)" alt="Image attached to post" />
              <footer><span>{{ post.likeCount }} likes</span><span>{{ post.commentCount }} comments</span></footer>
            </article>
          </div>
        </section>
      </div>
    </main>
  </AuthenticatedLayout>
</template>

<style scoped>
.search-page { width: 100%; max-width: 76rem; margin: 0 auto; }
.search-page__header { display: grid; gap: var(--space-5); align-items: end; margin-bottom: var(--space-5); }
.search-page h1 { margin: var(--space-1) 0 0; font-family: var(--font-display); font-size: clamp(2rem, 5vw, 3.25rem); }
.search-page__intro { margin: var(--space-2) 0 0; color: var(--color-text-muted); }
.search-page__form { display: flex; align-items: center; gap: var(--space-2); min-height: var(--touch-target); padding: .35rem .4rem .35rem var(--space-3); border: 1px solid var(--color-border); border-radius: 999px; background: var(--color-input); color: var(--color-text-faint); }
.search-page__form input { width: 100%; min-width: 0; border: 0; outline: 0; background: transparent; color: var(--color-text); }
.search-page__form button, .search-follow-button { min-height: 2.25rem; padding-inline: var(--space-3); border: 0; border-radius: 999px; background: var(--gradient-action); color: white; font-weight: 700; cursor: pointer; }
.search-tabs { display: flex; flex-wrap: wrap; gap: var(--space-2); margin-bottom: var(--space-5); }
.search-tabs button { min-height: 2.5rem; padding: 0 var(--space-3); border: 1px solid var(--color-border); border-radius: 999px; background: transparent; color: var(--color-text-muted); cursor: pointer; }
.search-tabs button span { margin-left: .3rem; color: var(--color-text-faint); font-family: var(--font-meta); font-size: .7rem; }
.search-tabs .search-tab--active { border-color: var(--color-violet); color: var(--color-text); box-shadow: var(--focus-ring); }
.search-state { padding: var(--space-6); color: var(--color-text-muted); text-align: center; }
.search-state--empty { min-height: 10rem; display: grid; place-items: center; }
.search-state--error { color: var(--color-coral); }
.search-results { display: grid; gap: var(--space-5); }
.search-section { padding: var(--space-5); }
.search-section__header { display: flex; justify-content: space-between; margin-bottom: var(--space-4); }
.search-section h2 { display: flex; align-items: center; gap: .5rem; margin: .2rem 0 0; font-size: 1rem; }
.search-user-list, .search-post-list { display: grid; gap: var(--space-2); }
.search-user-card { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: rgb(var(--rgb-surface-raised) / 48%); }
.search-user-card__identity { display: flex; align-items: center; min-width: 0; gap: var(--space-3); border: 0; background: transparent; color: inherit; text-align: left; cursor: pointer; }
.search-user-card__identity > span:last-child { min-width: 0; display: grid; gap: .2rem; }
.search-user-card strong, .search-post-card strong, .search-result-card strong { color: var(--color-text); }
.search-user-card small, .search-post-card small, .search-result-card small { color: var(--color-text-muted); font-size: .75rem; }
.search-avatar { display: grid; flex: 0 0 2.75rem; width: 2.75rem; height: 2.75rem; aspect-ratio: 1; place-items: center; overflow: hidden; border-radius: 50%; background: var(--gradient-action); color: white; font-weight: 700; }
.search-avatar img { width: 100%; height: 100%; object-fit: cover; object-position: center; }
.search-avatar--small { width: 2.25rem; height: 2.25rem; flex-basis: 2.25rem; font-size: .75rem; }
.search-follow-button:disabled { opacity: .6; cursor: wait; }
.search-card-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr)); gap: var(--space-3); }
.search-result-card { display: grid; grid-template-columns: auto minmax(0, 1fr); gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: rgb(var(--rgb-surface-raised) / 48%); color: inherit; text-align: left; cursor: pointer; }
.search-result-card:hover, .search-user-card:hover, .search-post-card:hover { border-color: var(--color-violet); }
.search-result-card__icon { display: grid; width: 2.5rem; height: 2.5rem; place-items: center; border-radius: 50%; background: var(--color-surface-teal); color: var(--color-mint); }
.search-result-card > span:last-child { display: grid; gap: .2rem; min-width: 0; }
.search-result-card p { margin: .4rem 0 0; color: var(--color-text-muted); font-size: .8125rem; line-height: 1.45; overflow-wrap: anywhere; }
.search-post-card { padding: var(--space-4); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: rgb(var(--rgb-surface-raised) / 48%); }
.search-post-card header { display: flex; align-items: center; gap: var(--space-2); }
.search-post-card header div { display: grid; gap: .15rem; }
.search-post-card > p { margin: var(--space-3) 0 0; color: var(--color-text-soft); line-height: 1.5; }
.search-post-card__image { display: block; width: min(100%, 28rem); max-height: 18rem; margin-top: var(--space-3); border-radius: var(--radius-small); object-fit: cover; }
.search-post-card footer { display: flex; gap: var(--space-4); margin-top: var(--space-3); color: var(--color-text-faint); font-family: var(--font-meta); font-size: .7rem; }
@media (min-width: 52rem) { .search-page__header { grid-template-columns: minmax(0, 1fr) minmax(20rem, 32rem); } }
@media (max-width: 30rem) { .search-user-card { align-items: flex-start; flex-direction: column; } .search-follow-button { align-self: stretch; } }
</style>
