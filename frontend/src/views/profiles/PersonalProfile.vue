<script setup>

import { onMounted, onUnmounted, ref } from 'vue'

import { getUserData } from '@/api/users/personalProfile'
import { getPosts } from '@/api/posts/posts.js'
import { toProfileCardPost } from '@/helpers/profilePosts.js'
import { profileData } from '@/data/usersData'

import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import ProfileHeader from '@/components/personalProfile/ProfileHeader.vue'
import ProfileTabs from '@/components/personalProfile/ProfileTabs.vue'
import AboutTab from '@/components/Profile/AboutTab.vue'
import FollowersTab from '@/components/Profile/FollowersTab.vue'
import PostCard from '@/components/posts/PostCard.vue'

const activeTab = ref('posts')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')

const posts = ref([])

const limit = 20
const offset = ref(0)
const hasMore = ref(true)

let throttleTimeout = null

async function loadPosts(loadMore = false) {
  if (loadMore) {
    if (loadingMore.value || !hasMore.value) {
      return
    }

    loadingMore.value = true
  }

  try {
    // only my own posts, 20 at a time
    const result = await getPosts({
      limit,
      offset: offset.value,
      userId: profileData.userInfo.id,
    })

    const newPosts = (result?.posts || []).map(
      (post, index) => toProfileCardPost(post, posts.value.length + index),
    )

    if (loadMore) {
      posts.value.push(...newPosts)
    } else {
      posts.value = newPosts
    }

    hasMore.value = result?.hasMore === true

    if (typeof result?.nextOffset === 'number') {
      offset.value = result.nextOffset
    } else {
      offset.value += newPosts.length
    }
  } catch (err) {
    console.error(err)

    if (!loadMore) {
      error.value = err.message || 'Could not load your profile.'
    }
  } finally {
    if (loadMore) {
      loadingMore.value = false
    } else {
      loading.value = false
    }
  }

}

function handleScroll() {
  if (throttleTimeout) {
    return
  }

  throttleTimeout = setTimeout(() => {
    throttleTimeout = null

    const scrollPosition = window.innerHeight + window.scrollY
    const pageHeight = document.documentElement.scrollHeight

    if (scrollPosition >= pageHeight - 500) {
      loadPosts(true)
    }
  }, 200)
}

onMounted(async () => {
  try {
    await getUserData()
    await loadPosts()

    window.addEventListener('scroll', handleScroll, {
      passive: true
    })
  } catch (err) {
    console.error(err)

    error.value = err.message || 'Could not load your profile.'
    loading.value = false
  }
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)

  if (throttleTimeout) {
    clearTimeout(throttleTimeout)
    throttleTimeout = null
  }
})

function removePost(postID) {
  posts.value = posts.value.filter(
    (post) => post.id !== postID,
  )
}

</script>

<template>

  <AuthenticatedLayout active-page="profile">

    <main class="profile-page">

      <p v-if="loading" class="profile-state orbit-surface">
        Loading profile...
      </p>

      <div v-else-if="error" class="profile-state profile-state--error orbit-surface" role="alert">
        <p>{{ error }}</p>

        <button type="button" @click="$router.go(0)">
          Try again
        </button>
      </div>

      <template v-else>

        <ProfileHeader :first-name="profileData.userInfo.firstName" :last-name="profileData.userInfo.lastName"
          :username="profileData.userInfo.userName" :bio="profileData.about.bio" :avatar-path="profileData.userInfo.avatar
            ? `/uploads/${profileData.userInfo.avatar}`
            : ''
            " :num-of-posts="profileData.numOfPosts" :num-of-following="profileData.numOfFollowing"
          :num-of-followers="profileData.numOfFollowers" :is-private="profileData.userInfo.isPrivate === 1" add-edit
          @select-tab="activeTab = $event" :dob="profileData.userInfo.dob" />

        <ProfileTabs v-model="activeTab" type="personal" />

        <section v-if="activeTab === 'posts'" class="profile-posts" aria-labelledby="profile-posts-heading">

          <header class="profile-posts__header">

            <p class="orbit-meta">
              Activity
            </p>

            <h2 id="profile-posts-heading">
              Your posts
            </h2>

          </header>

          <div v-if="posts.length" class="profile-posts__grid">

            <PostCard v-for="post in posts" :key="post.id" :post="post" @deleted="removePost" />

          </div>

          <p v-else class="profile-empty orbit-surface">
            No posts yet.
          </p>

          <p v-if="loadingMore" class="profile-loading">
            Loading more posts...
          </p>

          <p v-else-if="!hasMore && posts.length" class="profile-end">
            No more posts.
          </p>

        </section>

        <AboutTab v-else-if="activeTab === 'about'" :about="profileData.about" :profile="profileData.userInfo"
          own-profile />

        <FollowersTab v-else-if="activeTab === 'followers'" type="followers" :target-id="profileData.userInfo.id"
          :follower-list="profileData.followers" own-profile />

        <FollowersTab v-else-if="activeTab === 'following'" type="following" :target-id="profileData.userInfo.id"
          :follower-list="profileData.following" own-profile />

        <FollowersTab v-else-if="activeTab === 'friends'" type="friends" :target-id="profileData.userInfo.id"
          :follower-list="profileData.friends" />

      </template>

    </main>

  </AuthenticatedLayout>

</template>

<style scoped>
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
  width: 100%;
  max-width: 64rem;
  margin-inline: auto;
}

.profile-posts__header {
  margin-bottom: var(--space-4);
}

.profile-posts__header .orbit-meta {
  margin: 0;
  color: var(--color-violet-soft);
}

.profile-posts__header h2 {
  margin: var(--space-1) 0 0;
  font-family: var(--font-display);
  font-size: 1.5rem;
  letter-spacing: 0;
}

.profile-posts__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-4);
  align-items: start;
}

.profile-empty {
  margin: 0;
  padding: var(--space-6);
  color: var(--color-text-muted);
  text-align: center;
}

.profile-loading,
.profile-end {
  margin: var(--space-5) 0;
  color: var(--color-text-muted);
  text-align: center;
}

.profile-loading {
  padding: var(--space-4);
}

.profile-end {
  padding: var(--space-3);
  opacity: 0.7;
}

@media (max-width: 60rem) {

  .profile-posts__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

}

@media (max-width: 40rem) {

  .profile-posts__grid {
    grid-template-columns: 1fr;
  }

}
</style>