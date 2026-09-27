<script setup>
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getProfileData } from '@/api/users/profiles'
import { getPosts } from '@/api/posts/posts.js'
import { toProfileCardPost } from '@/helpers/profilePosts.js'
import { profileData } from '@/data/usersData'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import ProfileHeader from '@/components/personalProfile/ProfileHeader.vue'
import ProfileTabs from '@/components/personalProfile/ProfileTabs.vue'
import PrivateProfileIcon from '@/components/ProfileEdit/PrivateProfileIcon.vue'
import AboutTab from '@/components/Profile/AboutTab.vue'
import FollowersTab from '@/components/Profile/FollowersTab.vue'
import PostCard from '@/components/posts/PostCard.vue'

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

onMounted(() => window.addEventListener('scroll', handleScroll, { passive: true }))

onUnmounted(() => {
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
  profileData.isFollowing = status

  // once i follow them i may see more posts (followers only ones), so reload the first page
  if (status === 1) loadPosts()

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
              <p class="orbit-meta">Activity</p>
              <h2 id="member-posts-heading">Posts</h2>
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

            <p v-if="loadingMore" class="profile-loading">
              Loading more posts...
            </p>

            <p
              v-else
              class="profile-empty orbit-surface"
            >
              No posts available.
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

.profile-posts {
  display: grid;
  width: 100%;
  max-width: 64rem;
  margin-inline: auto;
  gap: var(--space-4);
}

.profile-posts-header {
  margin-bottom: var(--space-1);
}

.profile-posts .orbit-meta {
  margin: 0;
  color: var(--color-violet-soft);
}

.profile-posts h2 {
  margin: var(--space-1) 0 0;
  font-family: var(--font-display);
  font-size: 1.5rem;
  letter-spacing: 0;
}

.posts-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-4);
  width: 100%;
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

@media (max-width: 900px) {
  .posts-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 600px) {
  .posts-grid {
    grid-template-columns: 1fr;
  }
}
</style>
