```vue
<script setup>
import { computed, ref, watch } from 'vue'
import { createPost } from '@/api/posts/posts.js'
import { getFollowers } from '@/api/users/profiles.js'
import { normalizePostAudience } from '@/helpers/postAudience.js'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import LocationDialog from './LocationDialog.vue'

// tells the home feed when a post is made so it can show it on top
const emit = defineEmits(['post-created'])

// my avatar, shown next to the text box
const props = defineProps({
  avatar: {
    type: String,
    default: ''
  }
})

// same 5MB limit as the backend
const MAX_FILE_SIZE = 5 * 1024 * 1024

// everything the user types or picks in the post box
const content = ref('')
const postVisibility = ref('public')
const feeling = ref('')
const location = ref('')
const showLocationDialog = ref(false)
const selectedFile = ref(null)
const selectedFollowerIds = ref([])
const followers = ref([])
const followersOffset = ref(0)
const hasMoreFollowers = ref(false)
const followersError = ref('')
const isLoadingFollowers = ref(false)
const previewUrl = ref('')
const fileInput = ref(null)
const message = ref('')
const messageType = ref('success')
const isPosting = ref(false)

// loads my followers (20 at a time) for the "selected followers" option
async function loadFollowers({ append = false } = {}) {
  if (isLoadingFollowers.value) {
    return
  }

  isLoadingFollowers.value = true
  followersError.value = ''

  try {
    const offset = append ? followersOffset.value : 0
    const result = await getFollowers('', 20, offset)

    if (!result?.status) {
      throw new Error(
        result?.message || 'Could not load your followers.'
      )
    }

    const nextFollowers = normalizePostAudience(result?.data)

    followers.value = append
      ? [...followers.value, ...nextFollowers]
      : nextFollowers

    followersOffset.value = offset + nextFollowers.length
    hasMoreFollowers.value = nextFollowers.length === 20
  } catch (error) {
    followersError.value =
      error.message || 'Could not load your followers.'
  } finally {
    isLoadingFollowers.value = false
  }
}

// the load more button in the followers list
function loadMoreFollowers() {
  loadFollowers({ append: true })
}

// when a picture is picked: check the size and make a preview url for it
function selectFile(event) {
  const file = event.target.files[0] || null

  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }

  if (file && file.size > MAX_FILE_SIZE) {
    selectedFile.value = null
    message.value = 'File must be smaller than 5 MB.'
    messageType.value = 'error'
    event.target.value = ''
    return
  }

  selectedFile.value = file
  message.value = ''
  messageType.value = 'success'
  previewUrl.value = file
    ? URL.createObjectURL(file)
    : ''
}

// the Photo / GIF button clicks the hidden file input
function openFilePicker() {
  fileInput.value?.click()
}

// remove the picked picture and free the preview url
function removeFile() {
  selectedFile.value = null

  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }

  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

// the location popup gives back "name:lat:lon"
function handleLocationSelected(value) {
  location.value = value
  showLocationDialog.value = false
  message.value = ''
}

function removeLocation() {
  location.value = ''
}

// only the name part of the location, for showing it
const locationLabel = computed(() => {
  if (!location.value) {
    return ''
  }

  return location.value.split(':')[0].trim()
})

// the post button only works when there is text or a picture,
// and if privacy is selected at least one person must be picked
const canPost = computed(() => {
  const hasContent =
    content.value.trim() !== '' ||
    selectedFile.value !== null

  const hasSelectedFollowers =
    postVisibility.value !== 'selected' ||
    selectedFollowerIds.value.length > 0

  return (
    hasContent &&
    hasSelectedFollowers &&
    !isLoadingFollowers.value &&
    !isPosting.value
  )
})

// builds the form and sends the post.
// we use FormData because it can carry the image file too
async function preparePost() {
  if (!canPost.value) {
    return
  }

  const formData = new FormData()

  formData.append('content', content.value)
  formData.append('privacy', postVisibility.value)

  const selectedAudience =
    postVisibility.value === 'selected'
      ? selectedFollowerIds.value
      : []

  formData.append(
    'selectedFollowerIds',
    JSON.stringify(selectedAudience)
  )

  formData.append('location', location.value)

  if (selectedFile.value) {
    formData.append('image', selectedFile.value)
  }

  isPosting.value = true
  message.value = ''

  try {
    // send it. if it works we clear the form and tell the feed
    const result = await createPost(formData)

    if (!result?.status) {
      throw new Error(
        result?.message || 'Could not create post'
      )
    }

    content.value = ''
    feeling.value = ''
    location.value = ''
    postVisibility.value = 'public'
    selectedFollowerIds.value = []

    removeFile()

    emit('post-created', result.post)

    message.value = 'Post published successfully.'
    messageType.value = 'success'
  } catch (error) {
    message.value =
      error.message || 'Could not create post'

    messageType.value = 'error'
  } finally {
    isPosting.value = false
  }
}

// when privacy changes: if its not selected, clear the picked people.
// if it is selected, load the followers (only the first time)
watch(postVisibility, (value) => {
  if (value !== 'selected') {
    selectedFollowerIds.value = []
    return
  }

  if (
    followers.value.length === 0 &&
    !followersError.value
  ) {
    loadFollowers()
  }
})
</script>

<template>
  <!-- the whole post box. submit runs preparePost -->
  <form class="post-composer orbit-surface" @submit.prevent="preparePost">
    <div class="post-composer__input-row">
      <div class="post-composer__avatar">
        <img v-if="props.avatar" :src="props.avatar" alt="Profile avatar" />

        <IconGlyph v-else name="profile" :size="18" />
      </div>

      <label class="visually-hidden" for="post-content">
        Post content
      </label>

      <textarea id="post-content" v-model="content" maxlength="500" placeholder="What's happening in your orbit?"
        rows="2" @input="message = ''" />
    </div>

    <!-- picked picture name + preview -->
    <div v-if="selectedFile" class="selected-file">
      <span>{{ selectedFile.name }}</span>

      <button type="button" aria-label="Remove selected file" @click="removeFile">
        <IconGlyph name="close" :size="16" />
      </button>
    </div>

    <div v-if="previewUrl" class="selected-preview">
      <img :src="previewUrl" alt="Preview of the selected media" />
    </div>

    <!-- picked location with an x to remove it -->
    <div v-if="locationLabel" class="selected-location">
      <div class="selected-location__icon">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 21s7-6.1 7-12a7 7 0 1 0-14 0c0 5.9 7 12 7 12Z" />
          <circle cx="12" cy="9" r="2.25" />
        </svg>
      </div>

      <div class="selected-location__content">
        <span>Location</span>
        <strong>{{ locationLabel }}</strong>
      </div>

      <button type="button" aria-label="Remove location" @click="removeLocation">
        <IconGlyph name="close" :size="16" />
      </button>
    </div>

    <!-- bottom bar: photo, location, privacy and the post button -->
    <div class="post-composer__toolbar">
      <div class="post-composer__tools">
        <button class="composer-action composer-action--media" type="button" @click="openFilePicker">
          <IconGlyph name="image" :size="17" />

          <span>Photo / GIF</span>
        </button>

        <input ref="fileInput" class="file-input" type="file" accept="image/jpeg,image/png,image/gif"
          @change="selectFile" />

        <button class="composer-action composer-action--location" type="button" @click="showLocationDialog = true">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 21s7-6.1 7-12a7 7 0 1 0-14 0c0 5.9 7 12 7 12Z" />
            <circle cx="12" cy="9" r="2.25" />
          </svg>

          <span>
            {{ location ? 'Change location' : 'Location' }}
          </span>
        </button>
      </div>

      <div class="post-composer__actions">
        <!-- privacy dropdown -->
        <label class="privacy-control">
          <IconGlyph name="globe" :size="16" />

          <span class="visually-hidden">
            Post visibility
          </span>

          <select v-model="postVisibility" aria-label="Post visibility">
            <option value="public">
              Public
            </option>

            <option value="followers">
              Followers only
            </option>

            <option value="selected">
              Selected followers
            </option>
          </select>
        </label>

        <p class="privacy-description" aria-live="polite">
          <template v-if="postVisibility === 'public'">
            Anyone on Orbit can see this post.
          </template>

          <template v-else-if="postVisibility === 'followers'">
            People who follow you can see this post.
          </template>

          <template v-else>
            Only the followers you choose can see this post.
          </template>
        </p>

        <!-- list of my followers with checkboxes, only for selected privacy -->
        <div v-if="postVisibility === 'selected'" class="selected-followers">
          <p class="selected-followers__label">
            Choose approved followers
          </p>

          <p v-if="isLoadingFollowers && followers.length === 0" class="selected-followers__state">
            Loading your followers...
          </p>

          <div v-else-if="
            followersError &&
            followers.length === 0
          " class="selected-followers__state selected-followers__state--error">
            {{ followersError }}

            <button class="load-followers-button" type="button" :disabled="isLoadingFollowers" @click="loadFollowers()">
              Try again
            </button>
          </div>

          <p v-else-if="followers.length === 0" class="selected-followers__state">
            You have no approved followers yet.
          </p>

          <div v-else class="selected-followers__list">
            <p v-if="followersError" class="selected-followers__state selected-followers__state--error">
              {{ followersError }}

              <button class="load-followers-button" type="button" :disabled="isLoadingFollowers" @click="loadFollowers">
                Try again
              </button>
            </p>

            <label v-for="person in followers" :key="person.id" class="selected-follower">
              <input v-model="selectedFollowerIds" type="checkbox" :value="person.id" />

              <span>
                {{ person.name }}
              </span>
            </label>

            <button v-if="hasMoreFollowers" class="load-followers-button" type="button" :disabled="isLoadingFollowers"
              @click="loadMoreFollowers">
              {{
                isLoadingFollowers
                  ? 'Loading...'
                  : 'Show more followers'
              }}
            </button>
          </div>
        </div>

        <button class="post-button" type="submit" :disabled="!canPost">
          {{ isPosting ? 'Posting...' : 'Post' }}
        </button>
      </div>
    </div>

    <!-- success or error message -->
    <p v-if="message" class="post-composer__message" :class="{
      'post-composer__message--error':
        messageType === 'error'
    }" role="status">
      {{ message }}
    </p>

    <!-- popup to search and pick a location -->
    <LocationDialog v-if="showLocationDialog" :model-value="location" @update:model-value="handleLocationSelected"
      @close="showLocationDialog = false" />
  </form>
</template>

<style scoped>
.post-composer {
  top: 4rem;
  z-index: 15;
  align-self: start;
  width: 100%;
  max-height: calc(100vh - 8.5rem);
  max-height: calc(100dvh - 8.5rem);
  overflow-y: auto;
  padding: var(--space-4);
  box-shadow: 0 0.75rem 1.75rem rgb(0 0 0 / 24%);
  overscroll-behavior: contain;
}

.post-composer__input-row {
  display: grid;
  grid-template-columns: var(--touch-target) minmax(0, 1fr);
  align-items: start;
  gap: var(--space-3);
}

.post-composer__avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.75rem;
  height: 2.75rem;
  flex-shrink: 0;
  overflow: hidden;
  border: 2px solid var(--color-border);
  border-radius: 50%;
  background: var(--gradient-action);
  color: white;
  box-shadow: var(--shadow-soft);
}

.post-composer__avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
}

textarea {
  width: 100%;
  min-height: 4.5rem;
  padding: var(--space-2) 0;
  overflow: hidden;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text);
  line-height: 1.5;
  resize: vertical;
}

textarea::placeholder {
  color: var(--color-text-faint);
}

.selected-file {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin: var(--space-3) 0 0 calc(var(--touch-target) + var(--space-3));
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-small);
  background: var(--color-input);
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.selected-file span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-file button {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: var(--touch-target);
  min-height: var(--touch-target);
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}

.selected-preview {
  margin: var(--space-3) 0 0 calc(var(--touch-target) + var(--space-3));
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
}

.selected-preview img {
  display: block;
  width: 100%;
  max-height: 18rem;
  object-fit: contain;
}

.selected-location {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin: var(--space-3) 0 0 calc(var(--touch-target) + var(--space-3));
  padding: 0.7rem 0.8rem;
  border: 1px solid color-mix(in srgb,
      var(--color-violet) 30%,
      var(--color-border));
  border-radius: 0.8rem;
  background: color-mix(in srgb,
      var(--color-violet) 7%,
      var(--color-input));
}

.selected-location__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.2rem;
  height: 2.2rem;
  flex-shrink: 0;
  border-radius: 0.65rem;
  background: color-mix(in srgb,
      var(--color-violet) 14%,
      var(--color-input));
  color: var(--color-violet-soft);
}

.selected-location__icon svg {
  width: 1rem;
  height: 1rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.selected-location__content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 0.15rem;
}

.selected-location__content span {
  color: var(--color-text-faint);
  font-size: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.selected-location__content strong {
  overflow: hidden;
  color: var(--color-text);
  font-size: 0.82rem;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-location>button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  flex-shrink: 0;
  padding: 0;
  border: 0;
  border-radius: 0.55rem;
  background: transparent;
  color: var(--color-text-faint);
  cursor: pointer;
}

.selected-location>button:hover {
  background: color-mix(in srgb,
      var(--color-coral) 12%,
      transparent);
  color: var(--color-coral);
}

.post-composer__toolbar {
  display: flex;
  align-items: stretch;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

.post-composer__tools,
.post-composer__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.post-composer__actions {
  justify-content: space-between;
  flex-wrap: wrap;
}

.composer-action,
.privacy-control,
.post-button {
  min-height: var(--touch-target);
}

.composer-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding-inline: var(--space-2);
  border: 0;
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 0.875rem;
}

.composer-action:hover {
  background: var(--color-input);
  color: var(--color-text);
}

.composer-action svg {
  width: 1.25rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}

.composer-action--media {
  color: var(--color-mint);
}

.composer-action--media span {
  color: var(--color-text-muted);
}

.composer-action--location {
  color: var(--color-violet-soft);
}

.composer-action--location:hover {
  background: color-mix(in srgb,
      var(--color-violet) 9%,
      var(--color-input));
  color: var(--color-violet-soft);
}

.composer-action--location svg {
  width: 1.15rem;
  height: 1.15rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

.privacy-control {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding-inline: var(--space-3);
  border: 1px solid var(--color-blue);
  border-radius: 999px;
  color: var(--color-blue);
}

.privacy-control select {
  max-width: 9.5rem;
  border: 0;
  outline: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 0.875rem;
}

.privacy-control option {
  background: var(--color-surface-raised);
  color: var(--color-text);
}

.privacy-description {
  flex: 1 1 100%;
  margin: 0;
  color: var(--color-text-faint);
  font-size: 0.75rem;
  line-height: 1.4;
}

.selected-followers {
  display: grid;
  flex: 1 1 100%;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: var(--color-input);
}

.selected-followers__list {
  display: grid;
  max-height: 12rem;
  gap: var(--space-1);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.selected-followers__label,
.selected-followers__state {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.selected-followers__state--error {
  color: var(--color-coral);
}

.selected-follower {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-height: var(--touch-target);
  color: var(--color-text-soft);
  cursor: pointer;
}

.selected-follower input {
  width: 1.1rem;
  height: 1.1rem;
  accent-color: var(--color-violet);
}

.load-followers-button {
  min-height: var(--touch-target);
  border: 0;
  background: transparent;
  color: var(--color-violet-soft);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.load-followers-button:disabled {
  opacity: 0.6;
  cursor: wait;
}

.post-button {
  min-width: 5rem;
  padding-inline: var(--space-4);
  border: 0;
  border-radius: 999px;
  background: var(--gradient-action);
  color: white;
  cursor: pointer;
  font-weight: 700;
}

.post-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.post-composer__message {
  margin: var(--space-3) 0 0;
  color: var(--color-mint);
  font-size: 0.875rem;
}

.post-composer__message--error {
  color: var(--color-coral);
}

@media (min-width: 48rem) {
  .post-composer {
    padding: var(--space-5);
  }

  .post-composer__toolbar {
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
  }

  .post-composer__actions {
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .selected-followers {
    order: -1;
  }
}

@media (max-width: 36rem) {
  .selected-location {
    margin-left: 0;
  }
}
</style>
```
