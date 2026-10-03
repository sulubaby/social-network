<script setup>
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getProfileData } from '@/api/users/profiles'
import { getPosts } from '@/api/posts/posts.js'
import { toProfileCardPost } from '@/helpers/profilePosts.js'
import { viewedProfile as profileData } from '@/data/usersData'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import ProfileHeader from '@/components/personalProfile/ProfileHeader.vue'
import ProfileTabs from '@/components/personalProfile/ProfileTabs.vue'
import PrivateProfileIcon from '@/components/ProfileEdit/PrivateProfileIcon.vue'
import AboutTab from '@/components/Profile/AboutTab.vue'
import FollowersTab from '@/components/Profile/FollowersTab.vue'
import PostCard from '@/components/posts/PostCard.vue'
import { subscribeRealtime } from '@/services/realtime.js'

const route = useRoute()

const activeTab = ref('posts')

const loading = ref(true)

const error = ref('')

const posts = ref([])

// paging for this persons posts (20 at a time, more when you scroll down)
const POSTS_PAGE_SIZE = 20
const profileId = ref(0)
const postsOffset = ref(0)
const hasMore = ref(false)
const loadingMore = ref(false)
let throttleTimeout = null

// loads one page of this persons posts. the server only sends the ones i can see
async function loadPosts(loadMore = false) {
  if (loadMore && (loadingMore.value || !hasMore.value)) return
  if (loadMore) loadingMore.value = true

  const id = profileId.value
  try {
    const result = await getPosts({
      limit: POSTS_PAGE_SIZE,
      offset: loadMore ? postsOffset.value : 0,
      userId: id,
    })
    // if i opened another profile while this was loading, drop the old answer
    if (id !== profileId.value) return

    const newPosts = (result?.posts || []).map(
      (post, index) => toProfileCardPost(post, (loadMore ? posts.value.length : 0) + index),
    )
    posts.value = loadMore ? [...posts.value, ...newPosts] : newPosts
    hasMore.value = result?.hasMore === true
    postsOffset.value = typeof result?.nextOffset === 'number'
      ? result.nextOffset
      : postsOffset.value + newPosts.length
  } finally {
    if (loadMore) loadingMore.value = false
  }
}

// throttled scroll: check at most every 200ms if we are near the bottom
function handleScroll() {
  if (throttleTimeout) return

  throttleTimeout = setTimeout(() => {
    throttleTimeout = null
    const nearBottom = window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 500
    if (nearBottom && activeTab.value === 'posts') loadPosts(true)
  }, 200)
}

// when this person accepts my follow request while i look at their profile,
// open it up right away instead of waiting for a reload
let stopNotificationListener
function handleNotification(event) {
  const notification = event?.notification
  if (notification?.type === 'follow_accepted' && Number(notification.actorId) === profileId.value) {
    loadProfile(profileId.value)
  }
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll, { passive: true })
  stopNotificationListener = subscribeRealtime('notification', handleNotification)
})

onUnmounted(() => {
  stopNotificationListener?.()
  window.removeEventListener('scroll', handleScroll)
  clearTimeout(throttleTimeout)
  throttleTimeout = null
})

async function loadProfile(idValue) {
  const id = Number(idValue)

  activeTab.value = 'posts'
  posts.value = []
  hasMore.value = false
  postsOffset.value = 0
  profileId.value = id
  error.value = ''

  if (!Number.isInteger(id) || id <= 0) {
    loading.value = false
    error.value = 'Select a valid member to view their profile.'
    return
  }

  loading.value = true

  try {
    const profileResult = await getProfileData(id, 10)
    if (!profileResult) return

    if (profileResult.showProfile) {
      await loadPosts()
    }
  } catch (err) {
    console.error(err)
    error.value = err.message || 'Could not load this profile.'
  } finally {
    loading.value = false
  }
}

function relationshipChanged(status) {
  const previous = profileData.isFollowing
  profileData.isFollowing = status
  // following them opens the chat, unfollowing may close it (unless they follow me)
  if (status === 1) profileData.canMessage = true

  // keep the follower number in the header right without reloading
  if (previous !== 1 && status === 1) profileData.numOfFollowers += 1
  if (previous === 1 && status !== 1) profileData.numOfFollowers = Math.max(0, profileData.numOfFollowers - 1)

  if (status === 1) {
    // a private profile opens up once i really follow it, so load everything again
    if (!profileData.show) {
      loadProfile(profileData.userInfo.id)
      return
    }
    // i may see more posts now (followers only ones), so reload the first page
    loadPosts()
  }

  if (profileData.userInfo.isPrivate === 1 && status !== 1) {
    profileData.show = false
    posts.value = []
  }
}

watch(() => route.query.id, loadProfile, { immediate: true })
</script>

<template>
  <AuthenticatedLayout active-page="profile">
    <main class="profile-page">
      <p
        v-if="loading"
        class="profile-state orbit-surface"
      >
        Loading profile...
      </p>

      <div
        v-else-if="error"
        class="profile-state profile-state--error orbit-surface"
        role="alert"
      >
        <p>{{ error }}</p>

        <button
          type="button"
          @click="loadProfile(route.query.id)"
        >
          Try again
        </button>
      </div>

      <template v-else>
        <ProfileHeader
          :email="profileData.userInfo.email"
          :first-name="profileData.userInfo.firstName"
          :last-name="profileData.userInfo.lastName"
          :username="profileData.userInfo.userName"
          :bio="profileData.about.bio"
          :avatar-path="profileData.userInfo.avatar ? `/uploads/${profileData.userInfo.avatar}` : ''"
          :num-of-posts="profileData.numOfPosts"
          :num-of-following="profileData.numOfFollowing"
          :num-of-followers="profileData.numOfFollowers"
          :is-following="profileData.isFollowing"
          :is-private="profileData.userInfo.isPrivate === 1"
          :dob="profileData.userInfo.dob"
          :can-message="profileData.canMessage"
          @relationship-change="relationshipChanged"
          @select-tab="activeTab = $event"
        />

        <template v-if="profileData.show">
          <ProfileTabs
            v-model="activeTab"
            type="user"
          />

          <section
            v-if="activeTab === 'posts'"
            class="profile-posts"
            aria-labelledby="member-posts-heading"
          >
            <header class="profile-posts-header">
              <h2 id="member-posts-heading">Posts</h2>
              <span>{{ profileData.numOfPosts }} {{ profileData.numOfPosts === 1 ? 'post' : 'posts' }}</span>
            </header>

            <div
              v-if="posts.length"
              class="posts-grid"
            >
              <PostCard
                v-for="post in posts"
                :key="post.id"
                :post="post"
              />
            </div>

            <p
              v-else
              class="profile-empty orbit-surface"
            >
              No posts available.
            </p>

            <p v-if="loadingMore" class="profile-loading">
              Loading more posts...
            </p>
          </section>

          <AboutTab
            v-else-if="activeTab === 'about'"
            :about="profileData.about"
            :profile="profileData.userInfo"
          />

          <FollowersTab
            v-else-if="activeTab === 'followers'"
            type="followers"
            :target-id="profileData.userInfo.id"
            :follower-list="profileData.followers"
          />

          <FollowersTab
            v-else-if="activeTab === 'following'"
            type="following"
            :target-id="profileData.userInfo.id"
            :follower-list="profileData.following"
          />
        </template>

        <PrivateProfileIcon v-else />
      </template>
    </main>
  </AuthenticatedLayout>
</template>

<style scoped>
.profile-loading {
  margin: var(--space-5) 0;
  padding: var(--space-4);
  color: var(--color-text-muted);
  text-align: center;
}

.profile-page {
  display: grid;
  width: 100%;
  max-width: 64rem;
  margin: 0 auto;
  gap: var(--space-5);
}

.profile-state {
  margin: 0;
  padding: var(--space-6);
  color: var(--color-text-muted);
  text-align: center;
}

.profile-state p {
  margin: 0;
}

.profile-state--error {
  color: var(--color-coral);
}

.profile-state button {
  min-height: var(--touch-target);
  margin-top: var(--space-3);
  padding: 0 var(--space-4);
  border: 0;
  border-radius: var(--radius-small);
  background: var(--gradient-action);
  color: white;
  cursor: pointer;
  font-weight: 700;
}

/* same width as the home feed column */
.profile-posts {
  display: grid;
  width: 100%;
  max-width: 48rem;
  margin-inline: auto;
  gap: var(--space-4);
}

/* the posts title is a slim bar that sticks right under the tabs bar,
   so it always has room and never slides under the tabs */
.profile-posts-header {
  position: sticky;
  top: calc(4rem + var(--touch-target) + 1px);
  z-index: 9;
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  margin: 0 0 var(--space-4);
  padding: var(--space-6) 0 var(--space-3);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-background);
}

.profile-posts-header h2 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.35rem;
  letter-spacing: 0;
}

.profile-posts-header span {
  color: var(--color-text-muted);
  font-size: 0.85rem;
}

/* one post per row, same width as the home feed */
.posts-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--space-4);
}

.posts-grid :deep(.post-card) {
  min-width: 0;
  width: 100%;
}

.profile-empty {
  margin: 0;
  padding: var(--space-6);
  color: var(--color-text-muted);
  text-align: center;
}


</style>
