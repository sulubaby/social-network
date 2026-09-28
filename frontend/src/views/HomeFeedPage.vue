<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'

import FeedSidebar from '@/components/posts/FeedSidebar.vue'

import PostCard from '@/components/posts/PostCard.vue'

import PostComposer from '@/components/posts/PostComposer.vue'

import { getPosts } from '@/api/posts/posts.js'

import { profileData } from '@/data/usersData'

// feed state: the posts, loading flags and error text
const posts = ref([])

const isLoading = ref(true)

const isLoadingMore = ref(false)

const hasMorePosts = ref(false)

const feedError = ref('')

const feedSentinel = ref(null)

const FEED_PAGE_SIZE = 20

let feedObserver

// my avatar comes from the shared profile data. the layout already loads it,
// so the feed doesnt need to ask the server for my data a second time
const myAvatar = computed(() => profileData.userInfo.avatar)

// colors for the avatar circle when a user has no picture
const avatarColors = [
  '#3ee6b0',
  '#ff6b8a',
  '#7c5cff',
  '#ffb84d',
  '#4cc3ff'
]

// makes the post date easy to read. sqlite gives "2026-09-27 15:20:00" so we swap the space for a T first
function formatPostTime(value) {
  if (!value) return 'Just now'

  const date = new Date(value.replace(' ', 'T'))

  if (Number.isNaN(date.getTime())) {
    return 'Recently'
  }

  return date.toLocaleString([], {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

// turns public/followers/selected into the text we show on the card
function privacyLabel(value) {
  const labels = {
    public: 'Public',
    followers: 'Followers only',
    selected: 'Selected followers',
  }

  return labels[value] || value
}

// location is saved like "Manama:26.2:50.5", we only show the name part
function formatLocation(value) {
  if (!value) return ''

  const parts = value.split(':')

  return parts[0]?.trim() || ''
}

// changes a post from the server into the shape PostCard wants
function toCardPost(post, index = 0) {
  return {
    id: post.id,
    authorId: post.userId,
    author: post.author || 'Orbit member',
    avatarColor:
      avatarColors[index % avatarColors.length],
    avatarPath: post.avatarPath || '',
    time: formatPostTime(post.createdAt),
    privacy: privacyLabel(post.privacy),
    content: post.content || '',
    likes: post.likeCount || 0,
    liked: Boolean(post.liked),
    comments: post.commentCount || 0,
    imagePath: post.imagePath || '',
    location: post.location || '',
    locationLabel: formatLocation(post.location),
    hasMedia: false,
    mediaDescription: '',
  }
}

// loads the feed. append = true means get the next page for infinite scroll
async function loadPosts({ append = false } = {}) {
  if (append) {
    if (
      isLoadingMore.value ||
      !hasMorePosts.value
    ) {
      return
    }

    isLoadingMore.value = true
  } else {
    isLoading.value = true
  }

  feedError.value = ''

  try {
    const result = await getPosts({
      limit: FEED_PAGE_SIZE,
      offset: append ? posts.value.length : 0,
    })

    const nextPosts = (result?.posts || []).map(
      (post, index) =>
        toCardPost(
          post,
          append
            ? posts.value.length + index
            : index
        )
    )

    posts.value = append
      ? [...posts.value, ...nextPosts]
      : nextPosts

    hasMorePosts.value = Boolean(result?.hasMore)
  } catch (error) {
    feedError.value =
      error?.message ||
      'Could not load your feed.'
  } finally {
    isLoading.value = false
    isLoadingMore.value = false
  }
}

// when i make a new post it goes on top of the feed right away (no reload)
function addPost(post) {
  if (!post) return

  posts.value.unshift(
    toCardPost(post, 0)
  )
  // the post is published, close the popup
  showComposer.value = false
}

// the + button opens the post box in a popup instead of keeping it on the page
const showComposer = ref(false)

function closeComposer() {
  showComposer.value = false
}

// escape closes the popup
function handleComposerKeydown(event) {
  if (event.key === 'Escape') closeComposer()
}

// when the popup opens, put the cursor in the text box and listen for escape
watch(showComposer, async (open) => {
  if (open) {
    window.addEventListener('keydown', handleComposerKeydown)
    await nextTick()
    document.getElementById('post-content')?.focus()
  } else {
    window.removeEventListener('keydown', handleComposerKeydown)
  }
})

// infinite scroll: load more when the bottom of the feed is close to the screen
function observeFeedEnd() {
  if (
    !feedSentinel.value ||
    typeof IntersectionObserver === 'undefined'
  ) {
    return
  }

  feedObserver = new IntersectionObserver(
    ([entry]) => {
      if (
        entry.isIntersecting &&
        hasMorePosts.value &&
        !isLoading.value &&
        !isLoadingMore.value
      ) {
        loadPosts({
          append: true
        })
      }
    },
    {
      rootMargin: '0px 0px 320px',
    }
  )

  feedObserver.observe(feedSentinel.value)
}

// when the page opens: load the first posts, then start watching the scroll
onMounted(async () => {
  await loadPosts()
  observeFeedEnd()
})

onBeforeUnmount(() => {
  feedObserver?.disconnect()
  window.removeEventListener('keydown', handleComposerKeydown)
})
</script>

<template>
  <AuthenticatedLayout active-page="home">
    <div class="feed-layout">
      <div class="home-feed">
        <h1 class="visually-hidden">
          Home feed
        </h1>


        <!-- loading / error / no posts / the list of posts -->
        <p
          v-if="isLoading"
          class="feed-state orbit-surface"
        >
          Loading your feed...
        </p>

        <div
          v-else-if="feedError"
          class="feed-state orbit-surface"
        >
          <p>
            {{ feedError }}
          </p>

          <button
            type="button"
            @click="loadPosts()"
          >
            Try again
          </button>
        </div>

        <p
          v-else-if="posts.length === 0"
          class="feed-state orbit-surface"
        >
          No posts yet. Share something with your orbit.
        </p>

        <template v-else>
          <PostCard
            v-for="post in posts"
            :key="post.id"
            :post="post"
          />
        </template>

        <!-- empty div at the bottom used for infinite scroll -->
        <div
          ref="feedSentinel"
          class="feed-load-sentinel"
          aria-hidden="true"
        ></div>

        <p
          v-if="isLoadingMore"
          class="feed-load-state"
          role="status"
        >
          Loading more posts...
        </p>

        <p
          v-else-if="!hasMorePosts && posts.length"
          class="feed-load-state"
        >
          You’re all caught up.
        </p>
      </div>

      <!-- right side panel -->
      <FeedSidebar />
    </div>

    <!-- floating + button on the right to write a new post -->
    <button
      class="new-post-button"
      type="button"
      aria-label="Create a post"
      title="Create a post"
      @click="showComposer = true"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 5v14M5 12h14" />
      </svg>
    </button>

    <!-- the post box popup. clicking outside or pressing escape closes it -->
    <Teleport to="body">
      <div
        v-if="showComposer"
        class="composer-popup"
        @click.self="closeComposer"
      >
        <div
          class="composer-popup__box"
          role="dialog"
          aria-modal="true"
          aria-label="Create a post"
        >
          <button
            class="composer-popup__close"
            type="button"
            aria-label="Close"
            @click="closeComposer"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="m6 6 12 12M18 6 6 18" />
            </svg>
          </button>

          <PostComposer
            :avatar="myAvatar ? `/uploads/${myAvatar}` : ''"
            @post-created="addPost"
          />
        </div>
      </div>
    </Teleport>
  </AuthenticatedLayout>
</template>

<style scoped>
.new-post-button {
  position: fixed;
  right: 1.25rem;
  /* stay above the bottom menu on phones */
  bottom: calc(4.25rem + 1rem);
  z-index: 30;
  display: grid;
  place-items: center;
  width: 3.5rem;
  height: 3.5rem;
  border: 0;
  border-radius: 50%;
  background: var(--gradient-action);
  box-shadow: 0 0.75rem 1.75rem rgb(0 0 0 / 30%);
  color: white;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.new-post-button:hover {
  transform: scale(1.06);
}

.new-post-button:focus-visible {
  outline: 2px solid white;
  outline-offset: 3px;
}

.new-post-button svg,
.composer-popup__close svg {
  width: 1.6rem;
  height: 1.6rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 2.2;
  stroke-linecap: round;
}

.composer-popup {
  position: fixed;
  inset: 0;
  z-index: 900;
  display: grid;
  place-items: center;
  padding: var(--space-4);
  background: rgb(0 0 0 / 60%);
}

.composer-popup__box {
  position: relative;
  width: min(40rem, 100%);
}

.composer-popup__close {
  position: absolute;
  top: -0.75rem;
  right: -0.75rem;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 2.25rem;
  height: 2.25rem;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
}

.composer-popup__close svg {
  width: 1.1rem;
  height: 1.1rem;
}

@media (min-width: 64rem) {
  .new-post-button {
    right: 2rem;
    bottom: 2rem;
  }
}

.feed-layout {
  display: flex;
  align-items: flex-start;
  gap: var(--space-5);
  min-width: 0;
}

.home-feed {
  display: grid;
  width: 100%;
  max-width: 48rem;
  min-width: 0;
  gap: var(--space-4);
}

.feed-state {
  margin: 0;
  padding: var(--space-5);
  color: var(--color-text-muted);
  text-align: center;
}

.feed-state p {
  margin: 0;
}

.feed-state button {
  margin-top: var(--space-3);
  padding: var(--space-2) var(--space-4);
  border: 0;
  border-radius: 999px;
  background: var(--gradient-action);
  color: white;
  cursor: pointer;
  font-weight: 700;
}

.feed-load-sentinel {
  width: 100%;
  height: 1px;
  pointer-events: none;
}

.feed-load-state {
  display: flex;
  justify-content: center;
  margin: 0;
  padding: var(--space-3) 0 var(--space-4);
  color: var(--color-text-muted);
  font-size: 0.8125rem;
  text-align: center;
}

@media (min-width: 48rem) {
  .home-feed {
    gap: var(--space-5);
  }
}

@media (min-width: 90rem) {
  .feed-layout {
    justify-content: center;
  }
}
</style>