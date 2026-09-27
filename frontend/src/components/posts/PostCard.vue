<script setup>
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'

import CommentInput from '@/components/comments/CommentInput.vue'
import CommentPreview from '@/components/comments/CommentPreview.vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'

import {
  createComment,
  deleteComment,
  getComments,
} from '@/api/posts/comments.js'

import { setPostLike } from '@/api/posts/posts.js'

// the post we get from the parent (home feed or profile page)
const props = defineProps({
  post: {
    type: Object,
    required: true,
  },
})

// we tell the parent when the post got deleted so it can remove it from the list
const emit = defineEmits(['deleted'])

const route = useRoute()

// comments state (they only load when you open the comments)
const comments = ref([])
const commentInput = ref(null)
const commentsError = ref('')
const isLoadingComments = ref(false)
const isLoadingMoreComments = ref(false)
const areCommentsOpen = ref(false)
const commentsLoaded = ref(false)
const hasMoreComments = ref(false)
const commentsList = ref(null)
const commentsSentinel = ref(null)
const commentsOffset = ref(0)
const isSubmittingComment = ref(false)

// comment delete state
const deletingCommentIds = ref([])
const deletedCommentCount = ref(0)
const commentDeleteError = ref('')

// like state, starts with what the server told us
const isLiked = ref(Boolean(props.post.liked))
const likeCount = ref(Number(props.post.likes) || 0)
const isLikePending = ref(false)
const likeError = ref('')

// location map popup
const showLocationDialog = ref(false)

// the ... menu and the delete post popup
const showPostMenu = ref(false)
const showDeleteDialog = ref(false)
const isDeletingPost = ref(false)
const deleteError = ref('')

const COMMENTS_PAGE_SIZE = 20

let commentsObserver

// the delete menu only shows on my own profile page
const canManagePost = computed(() => {
  return route.path === '/me' || route.path === '/profile'
})

// comment number on the button. we take away the ones i deleted
// so the number updates without reloading
const commentCount = computed(() => {
  const postComments = Number(props.post.comments) || 0

  return Math.max(
    postComments,
    comments.value.length + deletedCommentCount.value,
  ) - deletedCommentCount.value
})

// builds the image url. images are served from /uploads/
function imageUrl(imagePath) {
  if (!imagePath) return ''

  return imagePath.startsWith('/')
    ? imagePath
    : `/uploads/${imagePath}`
}

// location is saved as "name:lat:lon", here we split it.
// if its broken we just dont show a location
const postLocation = computed(() => {
  if (!props.post.location) return null

  const parts = String(props.post.location).split(':')

  if (parts.length < 3) return null

  const label = parts[0].trim()
  const lat = Number(parts[1])
  const lon = Number(parts[2])

  if (
    !label ||
    !Number.isFinite(lat) ||
    !Number.isFinite(lon)
  ) {
    return null
  }

  return {
    label,
    lat,
    lon,
  }
})

// link to open the place in google maps
const googleMapsUrl = computed(() => {
  if (!postLocation.value) return ''

  const { lat, lon } = postLocation.value

  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(
    `${lat},${lon}`,
  )}`
})

// small google map shown inside the popup
const googleMapsEmbedUrl = computed(() => {
  if (!postLocation.value) return ''

  const { lat, lon } = postLocation.value

  return `https://www.google.com/maps?q=${encodeURIComponent(
    `${lat},${lon}`,
  )}&output=embed`
})

// open / close the map popup (escape key closes it too)
function openLocationDialog() {
  if (!postLocation.value) return

  showLocationDialog.value = true
}

function closeLocationDialog() {
  showLocationDialog.value = false
}

function handleLocationKeydown(event) {
  if (event.key === 'Escape') {
    closeLocationDialog()
  }
}

// open / close the ... menu
function togglePostMenu() {
  if (!canManagePost.value || isDeletingPost.value) return

  showPostMenu.value = !showPostMenu.value
}

function closePostMenu() {
  showPostMenu.value = false
}

// the "are you sure?" popup before deleting
function openDeleteDialog() {
  if (!canManagePost.value || isDeletingPost.value) return

  closePostMenu()

  deleteError.value = ''
  showDeleteDialog.value = true
}

function closeDeleteDialog() {
  if (isDeletingPost.value) return

  showDeleteDialog.value = false
  deleteError.value = ''
}

function handleDeleteDialogKeydown(event) {
  if (event.key === 'Escape') {
    closeDeleteDialog()
  }
}

// deletes the post (DELETE /api/posts with the post id).
// if it works we tell the parent so the card disappears
async function deletePost() {
  if (!canManagePost.value || isDeletingPost.value) return

  isDeletingPost.value = true
  deleteError.value = ''

  try {
    const response = await fetch('/api/posts', {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        postID: props.post.id,
      }),
    })

    let result = null

    try {
      result = await response.json()
    } catch {
      result = null
    }

    if (!response.ok || result?.status === false) {
      throw new Error(
        result?.message || 'Could not delete the post.',
      )
    }

    showDeleteDialog.value = false

    emit('deleted', props.post.id)
  } catch (error) {
    deleteError.value =
      error.message || 'Could not delete the post.'
  } finally {
    isDeletingPost.value = false
  }
}

// shows time like "5m", "3h", "2d", "1w". older posts show the date
function formatRelativeTime(dateValue) {
  if (!dateValue) return ''

  const date = new Date(dateValue)

  if (Number.isNaN(date.getTime())) {
    return dateValue
  }

  const now = new Date()
  const difference = now.getTime() - date.getTime()

  const minutes = Math.floor(difference / 60000)
  const hours = Math.floor(difference / 3600000)
  const days = Math.floor(difference / 86400000)
  const weeks = Math.floor(difference / 604800000)

  if (difference < 0) return 'just now'

  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m`
  if (hours < 24) return `${hours}h`
  if (days < 7) return `${days}d`
  if (weeks < 5) return `${weeks}w`

  return date.toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    year:
      date.getFullYear() !== now.getFullYear()
        ? 'numeric'
        : undefined,
  })
}

const formattedTime = computed(() =>
  formatRelativeTime(props.post.time),
)

// fills missing stuff on a comment before we show it
function commentForPreview(comment) {
  return {
    ...comment,
    author: comment.author || 'Orbit member',
    avatarColor: 'var(--gradient-action)',
  }
}

// loads the comments of this post.
// append = true loads the next page when you scroll down inside the comments
async function loadComments({ append = false } = {}) {
  if (append) {
    if (
      isLoadingMoreComments.value ||
      !hasMoreComments.value
    ) {
      return
    }

    isLoadingMoreComments.value = true
  } else {
    if (commentsLoaded.value) return

    isLoadingComments.value = true
  }

  commentsError.value = ''

  const requestOffset = append
    ? commentsOffset.value
    : 0

  try {
    const result = await getComments(props.post.id, {
      limit: COMMENTS_PAGE_SIZE,
      offset: requestOffset,
    })

    const nextComments = (result?.comments || [])
      .map(commentForPreview)

    const combinedComments = append
      ? [...comments.value, ...nextComments]
      : nextComments

    // remove duplicates (by id) and sort oldest first,
    // so a comment never shows twice if pages overlap
    const uniqueComments = new Map(
      combinedComments.map((comment) => [
        comment.id,
        comment,
      ]),
    )

    comments.value = [
      ...uniqueComments.values(),
    ].sort((first, second) => {
      const timeDifference =
        Date.parse(first.createdAt || '') -
        Date.parse(second.createdAt || '')

      return (
        (Number.isFinite(timeDifference)
          ? timeDifference
          : 0) ||
        Number(first.id) - Number(second.id)
      )
    })

    commentsOffset.value = Number.isInteger(
      result?.nextOffset,
    )
      ? result.nextOffset
      : requestOffset + nextComments.length

    hasMoreComments.value = Boolean(result?.hasMore)
    commentsLoaded.value = true
  } catch (error) {
    commentsError.value =
      error.message || 'Could not load comments.'
  } finally {
    isLoadingComments.value = false
    isLoadingMoreComments.value = false
  }
}

// infinite scroll inside the comments box
function observeCommentsEnd() {
  commentsObserver?.disconnect()

  if (
    !areCommentsOpen.value ||
    !commentsList.value ||
    !commentsSentinel.value ||
    typeof IntersectionObserver === 'undefined'
  ) {
    return
  }

  commentsObserver = new IntersectionObserver(
    ([entry]) => {
      if (
        entry.isIntersecting &&
        areCommentsOpen.value &&
        hasMoreComments.value
      ) {
        loadComments({ append: true })
      }
    },
    {
      root: commentsList.value,
      rootMargin: '0px 0px 120px',
    },
  )

  commentsObserver.observe(commentsSentinel.value)
}

// the try again button when comments fail to load
async function retryComments() {
  await loadComments({
    append:
      commentsLoaded.value &&
      comments.value.length > 0,
  })

  await nextTick()

  observeCommentsEnd()
}

function closeComments() {
  areCommentsOpen.value = false
  commentsObserver?.disconnect()
}

// sends a new comment and adds it at the bottom of the list
async function addComment(comment) {
  isSubmittingComment.value = true
  commentsError.value = ''

  try {
    const result = await createComment(
      props.post.id,
      comment,
    )

    if (!result?.comment) {
      throw new Error('Could not create comment')
    }

    comments.value.push(
      commentForPreview(result.comment),
    )

    commentInput.value?.reset()
  } catch (error) {
    commentsError.value =
      error.message || 'Could not create comment.'
  } finally {
    isSubmittingComment.value = false
  }
}

// deletes one of my comments. deletingCommentIds stops double clicks on the same one
async function removeComment(commentId) {
  if (
    deletingCommentIds.value.includes(commentId)
  ) {
    return
  }

  deletingCommentIds.value.push(commentId)
  commentDeleteError.value = ''

  try {
    const result = await deleteComment(
      props.post.id,
      commentId,
    )

    if (!result) return

    if (!result.status) {
      throw new Error(
        result.message || 'Could not delete comment',
      )
    }

    comments.value = comments.value.filter(
      (comment) => comment.id !== commentId,
    )

    deletedCommentCount.value += 1

    commentsOffset.value = Math.max(
      0,
      commentsOffset.value - 1,
    )
  } catch (error) {
    commentDeleteError.value =
      error.message || 'Could not delete comment.'
  } finally {
    deletingCommentIds.value =
      deletingCommentIds.value.filter(
        (id) => id !== commentId,
      )
  }
}

// like / unlike. we wait for the server and then use the real count it sends back
async function toggleLike() {
  if (isLikePending.value) return

  const nextLiked = !isLiked.value

  isLikePending.value = true
  likeError.value = ''

  try {
    const result = await setPostLike(
      props.post.id,
      nextLiked,
    )

    if (!result?.status) {
      throw new Error('Could not update the like')
    }

    isLiked.value = Boolean(result.liked)
    likeCount.value =
      Number(result.likeCount) || 0
  } catch (error) {
    likeError.value =
      error.message || 'Could not update the like.'
  } finally {
    isLikePending.value = false
  }
}

// open or close the comments. first time we open them we load them
async function toggleComments() {
  areCommentsOpen.value =
    !areCommentsOpen.value

  if (areCommentsOpen.value) {
    await loadComments()

    await nextTick()

    observeCommentsEnd()
  } else {
    closeComments()
  }
}

// first 2 letters of the name, used when there is no avatar picture
function initials(author) {
  if (!author) return '?'

  return author
    .slice(0, 2)
    .toUpperCase()
}

// stop the scroll watcher when the card is removed
onBeforeUnmount(() => {
  commentsObserver?.disconnect()
})
</script>

<template>
  <article class="post-card orbit-surface">
    <!-- top of the post: avatar, name, time, privacy, location and the ... menu -->
    <header class="post-card__header">
      <div class="post-card__author-area">
        <!-- clicking the author opens their profile -->
        <RouterLink
          v-if="post.authorId"
          class="post-card__author-link"
          :to="{
            path: '/user',
            query: { id: post.authorId },
          }"
          :aria-label="`View ${post.author}'s profile`"
        >
          <div
            class="post-card__avatar"
            :style="{ background: post.avatarColor }"
            aria-hidden="true"
          >
            <img
              v-if="post.avatarPath"
              :src="imageUrl(post.avatarPath)"
              alt=""
            />

            <span v-else>
              {{ initials(post.author) }}
            </span>
          </div>

          <div class="post-card__author">
            <div class="post-card__author-name-row">
              <h2>
                {{ post.author }}
              </h2>
            </div>

            <p>
              {{ formattedTime }}
              <span aria-hidden="true">•</span>
              {{ post.privacy }}
            </p>
          </div>
        </RouterLink>

        <div
          v-else
          class="post-card__author-link"
        >
          <div
            class="post-card__avatar"
            :style="{ background: post.avatarColor }"
            aria-hidden="true"
          >
            <img
              v-if="post.avatarPath"
              :src="imageUrl(post.avatarPath)"
              alt=""
            />

            <span v-else>
              {{ initials(post.author) }}
            </span>
          </div>

          <div class="post-card__author">
            <div class="post-card__author-name-row">
              <h2>
                {{ post.author }}
              </h2>
            </div>

            <p>
              {{ formattedTime }}
              <span aria-hidden="true">•</span>
              {{ post.privacy }}
            </p>
          </div>
        </div>

        <!-- location button, opens the map popup -->
        <button
          v-if="postLocation"
          type="button"
          class="post-card__location-small"
          :title="`View location: ${postLocation.label}`"
          @click="openLocationDialog"
        >
          <svg
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              d="M12 21s7-6.1 7-12a7 7 0 1 0-14 0c0 5.9 7 12 7 12Z"
            />

            <circle
              cx="12"
              cy="9"
              r="2.25"
            />
          </svg>

          <span>
            {{ postLocation.label }}
          </span>
        </button>

        <!-- ... menu with delete (only on my profile) -->
        <div
          v-if="canManagePost"
          class="post-card__menu"
        >
          <button
            type="button"
            class="post-card__menu-button"
            :disabled="isDeletingPost"
            aria-label="Post options"
            :aria-expanded="showPostMenu"
            @click="togglePostMenu"
          >
            <span aria-hidden="true">•••</span>
          </button>

          <div
            v-if="showPostMenu"
            class="post-card__menu-dropdown"
          >
            <button
              type="button"
              class="post-card__menu-item post-card__menu-item--danger"
              :disabled="isDeletingPost"
              @click="openDeleteDialog"
            >
              <IconGlyph
                name="trash"
                :size="16"
              />

              <span>
                Delete post
              </span>
            </button>
          </div>
        </div>
      </div>
    </header>

    <p class="post-card__content">
      {{ post.content }}
    </p>

    <!-- post image if there is one -->
    <div
      v-if="post.imagePath"
      class="post-card__media post-card__media--uploaded"
    >
      <img
        :src="imageUrl(post.imagePath)"
        alt="Image attached to this post"
      />
    </div>

    <div
      v-else-if="post.hasMedia"
      class="post-card__media"
      role="img"
      :aria-label="post.mediaDescription"
    >
      <span
        class="post-card__sun"
        aria-hidden="true"
      />

      <span
        class="post-card__mountain post-card__mountain--back"
        aria-hidden="true"
      />

      <span
        class="post-card__mountain post-card__mountain--front"
        aria-hidden="true"
      />
    </div>

    <!-- like and comment buttons -->
    <footer class="post-card__actions">
      <button
        class="post-action"
        :class="{
          'post-action--liked': isLiked,
        }"
        type="button"
        :aria-pressed="isLiked"
        :aria-label="
          isLiked
            ? 'Unlike this post'
            : 'Like this post'
        "
        :disabled="isLikePending"
        @click="toggleLike"
      >
        <IconGlyph
          name="heart"
          :size="21"
        />

        <span>
          {{ likeCount }} likes
        </span>
      </button>

      <button
        class="post-action post-action--comments"
        type="button"
        :aria-expanded="areCommentsOpen"
        :aria-controls="`comments-${post.id}`"
        @click="toggleComments"
      >
        <IconGlyph
          name="comment"
          :size="21"
        />

        <span>
          {{ commentCount }} comments
        </span>

        <span class="post-action__hint">
          {{ areCommentsOpen ? 'Hide' : 'View' }}
        </span>
      </button>
    </footer>

    <p
      v-if="likeError"
      class="post-action-error"
      role="alert"
    >
      {{ likeError }}
    </p>

    <p
      v-if="deleteError && !showDeleteDialog"
      class="post-action-error"
      role="alert"
    >
      {{ deleteError }}
    </p>

    <!-- the comments box, only shows when opened -->
    <section
      v-if="areCommentsOpen"
      :id="`comments-${post.id}`"
      class="comments-panel"
      aria-label="Comments"
    >
      <header class="comments-panel__header">
        <div>
          <p class="comments-panel__eyebrow">
            The conversation
          </p>

          <h3>
            Comments
            <span>{{ commentCount }}</span>
          </h3>
        </div>

        <button
          type="button"
          class="comments-panel__close"
          aria-label="Close comments"
          @click="closeComments"
        >
          <IconGlyph
            name="close"
            :size="17"
          />
        </button>
      </header>

      <!-- loading / error / no comments / the comment list -->
      <p
        v-if="isLoadingComments"
        class="comments-state"
      >
        Loading comments...
      </p>

      <div
        v-else-if="
          commentsError &&
          !comments.length
        "
        class="comments-state comments-state--error"
      >
        <span>
          {{ commentsError }}
        </span>

        <button
          type="button"
          @click="retryComments"
        >
          Retry
        </button>
      </div>

      <p
        v-else-if="!comments.length"
        class="comments-state comments-state--empty"
      >
        No comments yet. Start the conversation.
      </p>

      <div
        v-else
        ref="commentsList"
        class="comments-list"
      >
        <CommentPreview
          v-for="comment in comments"
          :key="comment.id"
          :comment="comment"
          :deleting="deletingCommentIds.includes(comment.id)"
          @delete="removeComment(comment.id)"
        />

        <!-- empty div for loading more comments on scroll -->
        <div
          ref="commentsSentinel"
          class="comments-sentinel"
          aria-hidden="true"
        />
      </div>

      <p
        v-if="isLoadingMoreComments"
        class="comments-load-state"
        role="status"
      >
        Loading more comments...
      </p>

      <p
        v-if="commentDeleteError"
        class="comments-state comments-state--error"
        role="alert"
      >
        {{ commentDeleteError }}
      </p>

      <div
        v-if="commentsError && comments.length"
        class="comments-state comments-state--error"
      >
        <span>
          {{ commentsError }}
        </span>

        <button
          type="button"
          @click="retryComments"
        >
          Retry
        </button>
      </div>

      <!-- box to write a comment -->
      <CommentInput
        ref="commentInput"
        :input-id="`comment-${post.id}`"
        :disabled="isSubmittingComment"
        @submit="addComment"
      />
    </section>

    <!-- map popup. Teleport puts it on the body so it covers the whole page -->
    <Teleport to="body">
      <div
        v-if="
          showLocationDialog &&
          postLocation
        "
        class="location-map-dialog"
        tabindex="-1"
        @keydown="handleLocationKeydown"
      >
        <button
          type="button"
          class="location-map-dialog__backdrop"
          aria-label="Close map"
          @click="closeLocationDialog"
        />

        <section
          class="location-map-dialog__panel"
          role="dialog"
          aria-modal="true"
          aria-labelledby="location-map-title"
        >
          <header class="location-map-dialog__header">
            <div class="location-map-dialog__title">
              <p class="location-map-dialog__eyebrow">
                Post location
              </p>

              <h2 id="location-map-title">
                {{ postLocation.label }}
              </h2>
            </div>

            <button
              type="button"
              class="location-map-dialog__close"
              aria-label="Close map"
              @click="closeLocationDialog"
            >
              <IconGlyph
                name="close"
                :size="18"
              />
            </button>
          </header>

          <div class="location-map-dialog__map">
            <iframe
              :src="googleMapsEmbedUrl"
              title="Google Maps location"
              loading="lazy"
              referrerpolicy="no-referrer-when-downgrade"
            />
          </div>

          <footer class="location-map-dialog__footer">
            <div class="location-map-dialog__coordinates">
              {{ postLocation.lat.toFixed(6) }},
              {{ postLocation.lon.toFixed(6) }}
            </div>

            <a
              class="location-map-dialog__open"
              :href="googleMapsUrl"
              target="_blank"
              rel="noopener noreferrer"
            >
              <svg
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path d="M14 5h5v5" />
                <path d="m19 5-8 8" />
                <path
                  d="M19 13v5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5"
                />
              </svg>

              Open in Google Maps
            </a>
          </footer>
        </section>
      </div>
    </Teleport>

    <!-- delete confirm popup -->
    <Teleport to="body">
      <div
        v-if="showDeleteDialog"
        class="delete-post-dialog"
        tabindex="-1"
        @keydown="handleDeleteDialogKeydown"
      >
        <button
          type="button"
          class="delete-post-dialog__backdrop"
          aria-label="Close delete confirmation"
          :disabled="isDeletingPost"
          @click="closeDeleteDialog"
        />

        <section
          class="delete-post-dialog__panel"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-post-title"
          aria-describedby="delete-post-description"
        >
          <div class="delete-post-dialog__icon">
            <IconGlyph
              name="trash"
              :size="21"
            />
          </div>

          <h2 id="delete-post-title">
            Delete post?
          </h2>

          <p id="delete-post-description">
            Are you sure you want to delete this post?
            This action cannot be undone.
          </p>

          <p
            v-if="deleteError"
            class="delete-post-dialog__error"
            role="alert"
          >
            {{ deleteError }}
          </p>

          <div class="delete-post-dialog__actions">
            <button
              type="button"
              class="delete-post-dialog__cancel"
              :disabled="isDeletingPost"
              @click="closeDeleteDialog"
            >
              Cancel
            </button>

            <button
              type="button"
              class="delete-post-dialog__confirm"
              :disabled="isDeletingPost"
              @click="deletePost"
            >
              {{
                isDeletingPost
                  ? 'Deleting...'
                  : 'Delete post'
              }}
            </button>
          </div>
        </section>
      </div>
    </Teleport>
  </article>
</template>

<style scoped>
.post-card {
  width: 100%;
  padding: var(--space-4);
}

.post-card__header {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: center;
  gap: var(--space-3);
}

.post-card__author-area {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.5rem;
}

.post-card__author-link {
  display: grid;
  flex: 1;
  grid-template-columns: var(--touch-target) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  color: inherit;
  text-decoration: none;
}

.post-card__author-link:hover .post-card__author h2,
.post-card__author-link:focus-visible .post-card__author h2 {
  color: var(--color-violet-soft);
}

.post-card__avatar {
  display: grid;
  flex: 0 0 var(--touch-target);
  width: var(--touch-target);
  height: var(--touch-target);
  place-items: center;
  aspect-ratio: 1;
  overflow: hidden;
  border-radius: 50%;
  color: #0b0d17;
  font-weight: 700;
  background: var(--color-input);
}

.post-card__avatar img {
  width: 100%;
  height: 100%;
  border-radius: inherit;
  object-fit: cover;
  object-position: center;
}

.post-card__author {
  min-width: 0;
}

.post-card__author-name-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.4rem;
}

.post-card__author h2,
.post-card__author p,
.post-card__content {
  margin: 0;
}

.post-card__author h2 {
  overflow: hidden;
  color: var(--color-text);
  font-size: 1rem;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-card__author p {
  color: var(--color-text-faint);
  font-size: 0.8125rem;
}

.post-card__location-small {
  display: inline-flex;
  max-width: 9rem;
  flex-shrink: 0;
  align-items: center;
  gap: 0.25rem;
  padding: 0.2rem 0.45rem;
  border: 1px solid color-mix(
    in srgb,
    var(--color-violet) 25%,
    var(--color-border)
  );
  border-radius: 999px;
  background: color-mix(
    in srgb,
    var(--color-violet) 7%,
    var(--color-input)
  );
  color: var(--color-violet-soft);
  cursor: pointer;
  font-size: 0.62rem;
  font-weight: 650;
  line-height: 1;
  transition:
    background 160ms ease,
    border-color 160ms ease,
    transform 160ms ease;
}

.post-card__location-small:hover {
  border-color: var(--color-violet);
  background: color-mix(
    in srgb,
    var(--color-violet) 14%,
    var(--color-input)
  );
  transform: translateY(-1px);
}

.post-card__location-small svg {
  width: 0.75rem;
  height: 0.75rem;
  flex-shrink: 0;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.post-card__location-small span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-card__menu {
  position: relative;
  flex-shrink: 0;
}

.post-card__menu-button {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  place-items: center;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 0.9rem;
  letter-spacing: 0.08em;
  line-height: 1;
  transition:
    background 160ms ease,
    border-color 160ms ease,
    color 160ms ease;
}

.post-card__menu-button:hover,
.post-card__menu-button:focus-visible {
  border-color: var(--color-border);
  background: var(--color-input);
  color: var(--color-text);
}

.post-card__menu-button:disabled {
  cursor: wait;
  opacity: 0.5;
}

.post-card__menu-dropdown {
  position: absolute;
  z-index: 20;
  top: calc(100% + 0.35rem);
  right: 0;
  min-width: 10rem;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-surface-raised);
  box-shadow: var(--shadow-soft);
}

.post-card__menu-item {
  display: flex;
  width: 100%;
  min-height: 2.6rem;
  align-items: center;
  gap: 0.55rem;
  padding: 0.55rem 0.75rem;
  border: 0;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  font-size: 0.78rem;
  text-align: left;
}

.post-card__menu-item:hover,
.post-card__menu-item:focus-visible {
  background: var(--color-input);
}

.post-card__menu-item--danger {
  color: var(--color-coral);
}

.post-card__menu-item:disabled {
  cursor: wait;
  opacity: 0.6;
}

.post-card__content {
  margin-top: var(--space-4);
  color: var(--color-text-soft);
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.post-card__media {
  position: relative;
  min-height: 12rem;
  margin-top: var(--space-4);
  overflow: hidden;
  border-radius: var(--radius-medium);
  background: linear-gradient(
    110deg,
    #44538e 0%,
    #aa5e9d 54%,
    #ff8e8b 100%
  );
}

.post-card__media--uploaded {
  min-height: 0;
  background: var(--color-input);
}

.post-card__media--uploaded img {
  display: block;
  width: 100%;
  max-height: 30rem;
  object-fit: contain;
}

.post-card__sun {
  position: absolute;
  top: 18%;
  right: 18%;
  width: clamp(2.75rem, 9vw, 4.5rem);
  aspect-ratio: 1;
  border-radius: 50%;
  background: #ffe1a3;
}

.post-card__mountain {
  position: absolute;
  right: -4%;
  bottom: -1px;
  left: -4%;
  height: 55%;
  background: #1c2447;
  clip-path: polygon(
    0 85%,
    18% 30%,
    35% 76%,
    51% 20%,
    66% 72%,
    84% 12%,
    100% 80%,
    100% 100%,
    0 100%
  );
}

.post-card__mountain--front {
  height: 42%;
  background: #11182f;
  clip-path: polygon(
    0 100%,
    31% 30%,
    49% 78%,
    63% 22%,
    79% 83%,
    100% 34%,
    100% 100%
  );
}

.post-card__actions {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-top: var(--space-2);
}

.post-action {
  display: inline-flex;
  min-width: var(--touch-target);
  min-height: var(--touch-target);
  align-items: center;
  gap: var(--space-2);
  padding-inline: var(--space-2);
  border: 0;
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 0.875rem;
}

.post-action--comments {
  align-items: center;
  font: inherit;
}

.post-action__hint {
  color: var(--color-violet-soft);
  font-size: 0.75rem;
  opacity: 0;
  transform: translateX(-0.25rem);
  transition:
    opacity 160ms ease,
    transform 160ms ease;
}

.post-action--comments:hover .post-action__hint,
.post-action--comments:focus-visible .post-action__hint {
  opacity: 1;
  transform: translateX(0);
}

.post-action:hover {
  background: var(--color-input);
  color: var(--color-text);
}

.post-action--liked {
  color: var(--color-coral);
}

.post-action--liked :deep(.icon-glyph) {
  fill: currentColor;
}

.post-action:disabled {
  cursor: wait;
  opacity: 0.65;
}

.post-action :deep(.icon-glyph) {
  width: 1.35rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.7;
}

.post-action-error {
  margin: var(--space-2) 0 0;
  color: var(--color-coral);
  font-size: 0.8125rem;
}

.comments-panel {
  margin-top: var(--space-4);
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-medium);
  background: rgb(11 13 23 / 46%);
  animation: comments-panel-in 180ms ease-out;
}

.comments-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.comments-panel__eyebrow {
  margin: 0 0 var(--space-1);
  color: var(--color-mint);
  font-family: var(--font-meta);
  font-size: 0.6875rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.comments-panel h3 {
  margin: 0;
  color: var(--color-text);
  font-size: 1rem;
}

.comments-panel h3 span {
  color: var(--color-text-faint);
  font-family: var(--font-meta);
  font-size: 0.75rem;
  font-weight: 400;
}

.comments-panel__close {
  display: grid;
  min-width: var(--touch-target);
  min-height: var(--touch-target);
  place-items: center;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}

.comments-panel__close:hover,
.comments-panel__close:focus-visible {
  border-color: var(--color-violet);
  color: var(--color-text);
}

.comments-list {
  display: grid;
  gap: var(--space-2);
  max-height: min(28rem, 55vh);
  overflow-y: auto;
  padding-right: var(--space-2);
  overscroll-behavior: contain;
}

.comments-sentinel {
  width: 100%;
  height: 1px;
  pointer-events: none;
}

.comments-load-state {
  margin: var(--space-2) 0 0;
  color: var(--color-text-faint);
  font-size: 0.8125rem;
  text-align: center;
}

.comments-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin: var(--space-3) 0 0;
  color: var(--color-text-faint);
  font-size: 0.875rem;
}

.comments-state--empty {
  display: block;
  padding: var(--space-3);
  border-radius: var(--radius-small);
  background: var(--color-surface);
}

.comments-state--error {
  color: var(--color-coral);
}

.comments-state button {
  min-height: var(--touch-target);
  padding-inline: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

@keyframes comments-panel-in {
  from {
    opacity: 0;
    transform: translateY(-0.35rem);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.location-map-dialog {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.location-map-dialog__backdrop {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  background: rgb(0 0 0 / 65%);
  backdrop-filter: blur(4px);
  cursor: default;
}

.location-map-dialog__panel {
  position: relative;
  z-index: 1;
  display: flex;
  width: min(100%, 48rem);
  max-height: 90vh;
  overflow: hidden;
  flex-direction: column;
  border: 1px solid var(--color-border);
  border-radius: 1rem;
  background: var(--color-surface-raised);
  box-shadow: 0 1.5rem 4rem rgb(0 0 0 / 40%);
}

.location-map-dialog__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.location-map-dialog__title {
  min-width: 0;
}

.location-map-dialog__eyebrow {
  margin: 0 0 0.2rem;
  color: var(--color-violet-soft);
  font-family: var(--font-meta);
  font-size: 0.65rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.location-map-dialog__header h2 {
  margin: 0;
  overflow: hidden;
  color: var(--color-text);
  font-size: 1rem;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.location-map-dialog__close {
  display: flex;
  width: 2.4rem;
  height: 2.4rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}

.location-map-dialog__close:hover {
  background: var(--color-input);
  color: var(--color-text);
}

.location-map-dialog__map {
  width: 100%;
  height: min(55vh, 28rem);
  background: var(--color-input);
}

.location-map-dialog__map iframe {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
}

.location-map-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);
}

.location-map-dialog__coordinates {
  min-width: 0;
  color: var(--color-text-faint);
  font-family: var(--font-meta);
  font-size: 0.7rem;
}

.location-map-dialog__open {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0 1rem;
  border-radius: 999px;
  background: var(--gradient-action);
  color: white;
  font-size: 0.8rem;
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
  transition:
    transform 160ms ease,
    box-shadow 160ms ease;
}

.location-map-dialog__open:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-soft);
}

.location-map-dialog__open svg {
  width: 1rem;
  height: 1rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.delete-post-dialog {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.delete-post-dialog__backdrop {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  background: rgb(0 0 0 / 68%);
  backdrop-filter: blur(5px);
}

.delete-post-dialog__panel {
  position: relative;
  z-index: 1;
  width: min(100%, 25rem);
  padding: 1.5rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-medium);
  background: var(--color-surface-raised);
  box-shadow: 0 1.5rem 4rem rgb(0 0 0 / 45%);
  text-align: center;
  animation: delete-dialog-in 160ms ease-out;
}

.delete-post-dialog__icon {
  display: grid;
  width: 3rem;
  height: 3rem;
  margin: 0 auto 1rem;
  place-items: center;
  border-radius: 50%;
  background: color-mix(
    in srgb,
    var(--color-coral) 12%,
    var(--color-input)
  );
  color: var(--color-coral);
}

.delete-post-dialog__panel h2 {
  margin: 0;
  color: var(--color-text);
  font-size: 1.1rem;
  font-weight: 650;
}

.delete-post-dialog__panel p {
  margin: 0.65rem 0 0;
  color: var(--color-text-muted);
  font-size: 0.82rem;
  line-height: 1.5;
}

.delete-post-dialog__error {
  color: var(--color-coral) !important;
}

.delete-post-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.6rem;
  margin-top: 1.35rem;
}

.delete-post-dialog__cancel,
.delete-post-dialog__confirm {
  min-height: 2.6rem;
  padding: 0 1rem;
  border-radius: var(--radius-small);
  cursor: pointer;
  font-size: 0.78rem;
  font-weight: 650;
  transition:
    background 160ms ease,
    border-color 160ms ease,
    opacity 160ms ease;
}

.delete-post-dialog__cancel {
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text);
}

.delete-post-dialog__cancel:hover {
  background: var(--color-input);
}

.delete-post-dialog__confirm {
  border: 1px solid var(--color-coral);
  background: var(--color-coral);
  color: white;
}

.delete-post-dialog__confirm:hover {
  opacity: 0.9;
}

.delete-post-dialog__cancel:disabled,
.delete-post-dialog__confirm:disabled {
  cursor: wait;
  opacity: 0.6;
}

@keyframes delete-dialog-in {
  from {
    opacity: 0;
    transform: translateY(0.5rem) scale(0.98);
  }

  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@media (max-width: 48rem) {
  .post-card__author-area {
    gap: 0.35rem;
  }

  .post-card__location-small {
    max-width: 7rem;
  }
}

@media (max-width: 36rem) {
  .post-card {
    padding: var(--space-3);
  }

  .post-card__author-area {
    align-items: flex-start;
  }

  .post-card__location-small {
    margin-top: 0.2rem;
    max-width: 6rem;
    padding: 0.18rem 0.4rem;
    font-size: 0.58rem;
  }

  .post-card__location-small svg {
    width: 0.7rem;
    height: 0.7rem;
  }

  .post-card__menu-button {
    width: 2rem;
    height: 2rem;
  }

  .post-card__menu-dropdown {
    right: 0;
  }

  .location-map-dialog {
    align-items: flex-end;
    padding: 0;
  }

  .location-map-dialog__panel {
    width: 100%;
    max-height: 92vh;
    border-radius: 1rem 1rem 0 0;
  }

  .location-map-dialog__map {
    height: 55vh;
  }

  .location-map-dialog__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .location-map-dialog__open {
    width: 100%;
  }

  .delete-post-dialog {
    align-items: flex-end;
    padding: 0;
  }

  .delete-post-dialog__panel {
    width: 100%;
    border-radius: 1rem 1rem 0 0;
    padding: 1.35rem;
  }

  .delete-post-dialog__actions {
    flex-direction: column-reverse;
  }

  .delete-post-dialog__cancel,
  .delete-post-dialog__confirm {
    width: 100%;
  }
}

@media (min-width: 48rem) {
  .post-card {
    padding: var(--space-5);
  }

  .post-card__media {
    min-height: 15rem;
  }
}
</style>