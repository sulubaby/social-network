<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getFriends } from '@/api/common/friends'
import { removeFollower, requestFollow, searchFollowing, searchFollows } from '@/api/users/profiles'
import { addNotification } from '@/data/notifications'
import { profileData } from '@/data/usersData'
import IconGlyph from '@/components/layout/IconGlyph.vue'

const props = defineProps({
  type: { type: String, default: 'followers' },
  targetId: { type: [String, Number], default: null },
  followerList: { type: Object, default: () => ({}) },
  ownProfile: { type: Boolean, default: false },
})

const router = useRouter()
const route = useRoute()
const PAGE_SIZE = 20
const showDialog = ref(false)
const list = ref([])
const offset = ref(0)
const loading = ref(false)
const searching = ref(false)
const hasMore = ref(true)
const error = ref('')
const searchResults = ref([])
const searchQuery = ref('')
const scrollBox = ref(null)
const busyIds = ref([])
let searchDebounce

const targetId = computed(() => props.targetId || route.query.id || '')
const title = computed(() => props.type === 'following' ? 'Following' : props.type === 'friends' ? 'Friends' : 'Followers')
const emptyText = computed(() => `No ${title.value.toLowerCase()} yet.`)
const previewList = computed(() => normalizeUsers(props.followerList).slice(0, 6))
const displayedList = computed(() => searchQuery.value.trim() ? searchResults.value : list.value)
const canManage = computed(() => props.ownProfile && (props.type === 'followers' || props.type === 'following'))
const actionLabel = computed(() => props.type === 'followers' ? 'Remove follower' : 'Unfollow')
const busyLabel = computed(() => props.type === 'followers' ? 'Removing...' : 'Unfollowing...')

function normalizeUsers(users = {}) {
  return Object.entries(users || {}).map(([key, user]) => ({
    id: Number(user.ID || user.id || key),
    firstName: user.FirstName || user.firstName || '',
    lastName: user.LastName || user.lastName || '',
    username: user.UserName || user.username || '',
    avatar: user.Avatar || user.avatar || '',
  }))
}

function initials(user) {
  return `${user.firstName}${user.lastName}`.slice(0, 2).toUpperCase() || 'O'
}

function buildListUrl() {
  const endpoint = props.type === 'following'
    ? '/api/profile/following'
    : props.type === 'friends' ? '/api/friends/' : '/api/profile/follow'
  const params = new URLSearchParams({ offset: offset.value.toString() })
  if (props.type !== 'friends' && targetId.value) params.set('targetid', targetId.value)
  return `${endpoint}?${params.toString()}`
}

async function fetchPage() {
  if (loading.value || !hasMore.value || searchQuery.value.trim()) return
  loading.value = true
  error.value = ''
  try {
    const response = await fetch(buildListUrl(), { credentials: 'include' })
    const result = await response.json()
    if (!response.ok || !result.status) throw new Error(result.message || 'Could not load connections')
    const page = normalizeUsers(result.data)
    list.value.push(...page.filter(user => !list.value.some(existing => existing.id === user.id)))
    offset.value += PAGE_SIZE
    hasMore.value = page.length === PAGE_SIZE
  } catch (err) {
    error.value = err.message || 'Could not load connections.'
  } finally {
    loading.value = false
  }
}

function openDialog() {
  showDialog.value = true
  list.value = []
  offset.value = 0
  hasMore.value = true
  error.value = ''
  searchQuery.value = ''
  fetchPage()
}

function closeDialog() {
  showDialog.value = false
  searchQuery.value = ''
  searchResults.value = []
}

async function runSearch(query) {
  searching.value = true
  error.value = ''
  try {
    const result = props.type === 'following'
      ? await searchFollows(query, targetId.value)
      : props.type === 'friends'
        ? await getFriends(query, targetId.value)
        : await searchFollowing(query, targetId.value)
    searchResults.value = normalizeUsers(result.data)
  } catch (err) {
    searchResults.value = []
    error.value = err.message || 'Could not search connections.'
  } finally {
    searching.value = false
  }
}

watch(searchQuery, value => {
  clearTimeout(searchDebounce)
  const query = value.trim()
  if (!query) {
    searchResults.value = []
    searching.value = false
    error.value = ''
    return
  }
  searchDebounce = setTimeout(() => runSearch(query), 300)
})

function handleScroll() {
  const box = scrollBox.value
  if (box && box.scrollHeight - box.scrollTop - box.clientHeight < 120) fetchPage()
}

function handleKeydown(event) {
  if (event.key === 'Escape' && showDialog.value) closeDialog()
}

function applyRemoval(id) {
  const key = String(id)
  list.value = list.value.filter(user => user.id !== id)
  searchResults.value = searchResults.value.filter(user => user.id !== id)
  if (showDialog.value) offset.value = Math.max(0, offset.value - 1)

  if (props.type === 'followers') {
    if (profileData.followers) delete profileData.followers[key]
    profileData.numOfFollowers = Math.max(0, (profileData.numOfFollowers || 0) - 1)
  } else {
    if (profileData.following) delete profileData.following[key]
    profileData.numOfFollowing = Math.max(0, (profileData.numOfFollowing || 0) - 1)
  }

  if (profileData.friends) delete profileData.friends[key]
}

async function handleAction(user) {
  if (busyIds.value.includes(user.id)) return
  const name = `${user.firstName || ''} ${user.lastName || ''}`.trim() || 'this user'
  const question = props.type === 'followers' ? `Remove ${name} from your followers?` : `Unfollow ${name}?`
  if (!window.confirm(question)) return
  busyIds.value.push(user.id)
  try {
    const result = props.type === 'followers'
      ? await removeFollower(user.id)
      : await requestFollow(user.id, 'DELETE')
    if (!result) return
    if (!result.status) throw new Error(result.message || 'Could not update connection')
    applyRemoval(user.id)
    addNotification(props.type === 'followers' ? 'Follower removed' : 'Unfollowed', 'success')
  } catch (err) {
    addNotification(err.message || 'Could not update connection', 'error')
  } finally {
    busyIds.value = busyIds.value.filter(id => id !== user.id)
  }
}

async function goToProfile(id) {
  closeDialog()
  await router.push({ path: '/user', query: { id } })
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  clearTimeout(searchDebounce)
})
</script>

<template>
  <section class="connections orbit-surface">
    <header class="connections-header">
      <div>
        <p class="orbit-meta">Connections</p>
        <h2>{{ title }}</h2>
      </div>
      <button v-if="previewList.length" type="button" class="text-button" @click="openDialog">Show all</button>
    </header>

    <div v-if="previewList.length" class="preview-grid">
      <div v-for="user in previewList" :key="user.id" class="person">
        <button type="button" class="person-link" @click="goToProfile(user.id)">
          <span class="avatar">
            <img v-if="user.avatar" :src="`/uploads/${user.avatar}`" alt="" />
            <span v-else>{{ initials(user) }}</span>
          </span>
          <span class="person-copy">
            <strong>{{ user.firstName }} {{ user.lastName }}</strong>
            <small v-if="user.username">@{{ user.username }}</small>
          </span>
        </button>
        <button v-if="canManage" type="button" class="action-button" :disabled="busyIds.includes(user.id)"
          @click="handleAction(user)">
          {{ busyIds.includes(user.id) ? busyLabel : actionLabel }}
        </button>
      </div>
    </div>
    <p v-else class="empty-state">{{ emptyText }}</p>
  </section>

  <Teleport to="body">
    <div v-if="showDialog" class="dialog-backdrop" @click.self="closeDialog">
      <section class="dialog" role="dialog" aria-modal="true" :aria-label="title">
        <header class="dialog-header">
          <div>
            <p class="orbit-meta">Connections</p>
            <h2>{{ title }}</h2>
          </div>
          <button type="button" class="icon-button" aria-label="Close" title="Close" @click="closeDialog">
            <IconGlyph name="close" :size="18" />
          </button>
        </header>

        <label class="search-field">
          <IconGlyph name="search" :size="17" />
          <input v-model="searchQuery" type="search" placeholder="Search by name..." />
        </label>

        <div ref="scrollBox" class="dialog-list" @scroll="handleScroll">
          <div v-for="user in displayedList" :key="user.id" class="person person--row">
            <button type="button" class="person-link" @click="goToProfile(user.id)">
              <span class="avatar">
                <img v-if="user.avatar" :src="`/uploads/${user.avatar}`" alt="" />
                <span v-else>{{ initials(user) }}</span>
              </span>
              <span class="person-copy">
                <strong>{{ user.firstName }} {{ user.lastName }}</strong>
                <small v-if="user.username">@{{ user.username }}</small>
              </span>
              <IconGlyph v-if="!canManage" name="arrowRight" :size="17" />
            </button>
            <button v-if="canManage" type="button" class="action-button" :disabled="busyIds.includes(user.id)"
              @click="handleAction(user)">
              {{ busyIds.includes(user.id) ? busyLabel : actionLabel }}
            </button>
          </div>

          <p v-if="loading || searching" class="dialog-status">{{ searching ? 'Searching...' : 'Loading...' }}</p>
          <p v-else-if="error" class="dialog-status dialog-status--error" role="alert">{{ error }}</p>
          <p v-else-if="!displayedList.length" class="dialog-status">{{ searchQuery.trim() ? 'No users found.' :
            emptyText }}</p>
          <p v-else-if="!searchQuery.trim() && !hasMore" class="dialog-status">You have reached the end.</p>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.connections {
  padding: var(--space-5);
}

.connections-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.connections h2,
.dialog h2 {
  margin: var(--space-1) 0 0;
  font-family: var(--font-display);
  font-size: 1.2rem;
  letter-spacing: 0;
}

.text-button {
  min-height: var(--touch-target);
  padding: 0 var(--space-3);
  border: 0;
  background: transparent;
  color: var(--color-violet-soft);
  cursor: pointer;
  font-weight: 700;
}

.preview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-2);
}

.person {
  display: flex;
  min-width: 0;
  min-height: 4.25rem;
  align-items: center;
  gap: var(--space-2);
  padding-right: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
  color: var(--color-text);
}

.person:hover {
  border-color: var(--color-violet);
  background: var(--color-surface-raised);
}

.person-link {
  display: flex;
  min-width: 0;
  min-height: 4.25rem;
  flex: 1;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
  font: inherit;
}

.action-button {
  min-height: 2.25rem;
  flex: 0 0 auto;
  padding: 0 var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-coral-soft);
  cursor: pointer;
  font-size: .8125rem;
  font-weight: 700;
  white-space: nowrap;
}

.action-button:hover:not(:disabled) {
  border-color: var(--color-coral);
  background: var(--color-input);
}

.action-button:disabled {
  cursor: default;
  opacity: .6;
}

.avatar {
  display: grid;
  width: 2.75rem;
  height: 2.75rem;
  flex: 0 0 2.75rem;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  background: var(--gradient-action);
  color: white;
  font-size: .75rem;
  font-weight: 800;
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.person-copy {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: .2rem;
}

.person-copy strong,
.person-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.person-copy strong {
  font-size: .875rem;
}

.person-copy small {
  color: var(--color-text-muted);
}

.empty-state,
.dialog-status {
  margin: 0;
  padding: var(--space-6);
  color: var(--color-text-muted);
  text-align: center;
}

.dialog-backdrop {
  position: fixed;
  inset: 0;
  display: flex;
  justify-content: flex-end;
  background: rgb(0 0 0 / 62%);
  z-index: 1000;
}

.dialog {
  display: flex;
  width: min(27rem, 100%);
  height: 100%;
  flex-direction: column;
  border-left: 1px solid var(--color-border);
  background: var(--color-surface);
  box-shadow: -1rem 0 3rem rgb(0 0 0 / 25%);
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.icon-button {
  display: grid;
  width: var(--touch-target);
  height: var(--touch-target);
  place-items: center;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
  color: var(--color-text);
  cursor: pointer;
}

.search-field {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: var(--space-4);
  padding: 0 var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
  color: var(--color-text-muted);
}

.search-field:focus-within {
  border-color: var(--color-violet);
}

.search-field input {
  width: 100%;
  min-height: var(--touch-target);
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text);
  font: inherit;
}

.dialog-list {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: var(--space-2);
  overflow-y: auto;
  padding: 0 var(--space-4) var(--space-5);
}

.person--row {
  flex: 0 0 auto;
  width: 100%;
}

.person--row .person-link>svg {
  color: var(--color-text-faint);
}

.dialog-status--error {
  color: var(--color-coral-soft);
}

@media (max-width: 600px) {
  .connections {
    padding: var(--space-4);
  }

  .preview-grid {
    grid-template-columns: 1fr;
  }

  .dialog {
    border-left: 0;
  }
}
</style>