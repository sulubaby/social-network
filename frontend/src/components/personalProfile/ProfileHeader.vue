<script setup>
import { ref, watch } from 'vue'

import { requestFollow } from '@/api/users/profiles'
import { useRoute } from 'vue-router'
import { addNotification } from '@/data/notifications'
import IconGlyph from '@/components/layout/IconGlyph.vue'

const route = useRoute()

const props = defineProps({
  addEdit: Boolean,
  firstName: String,
  lastName: String,
  username: String,
  email: String,
  bio: String,
  avatarPath: String,
  numOfPosts: Number,
  numOfFollowing: Number,
  numOfFollowers: Number,
  isFollowing: { type: Number, default: -1 },
  isPrivate: Boolean,
  dob: [Date, String],
  canMessage: Boolean,
})

const emit = defineEmits(['relationship-change', 'select-tab', 'privacy-change'])

const followingStatus = ref(props.isFollowing)
const relationshipBusy = ref(false)
const avatarFailed = ref(false)

watch(() => props.isFollowing, (value) => {
  followingStatus.value = value
})

watch(() => props.avatarPath, () => {
  avatarFailed.value = false
})

function formatDob(value) {
  if (!value) return ''

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return ''

  return date.toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
}

async function changeRelationship(method) {
  if (relationshipBusy.value) return

  // ask before unfollowing, so it does not happen by a wrong click
  if (method === 'DELETE' && followingStatus.value === 1 &&
    !window.confirm(`Unfollow ${props.firstName} ${props.lastName}?`)) return

  relationshipBusy.value = true

  try {
    const result = await requestFollow(route.query.id, method)

    if (!result?.status) {
      throw new Error(result?.message || 'Could not update follow status')
    }

    followingStatus.value = result.followStatus
    emit('relationship-change', result.followStatus)
  } catch (error) {
    addNotification(
      error.message || 'Could not update follow status',
      'error'
    )
  } finally {
    relationshipBusy.value = false
  }
}

function initials() {
  return `${props.firstName || ''}${props.lastName || ''}`
    .slice(0, 2)
    .toUpperCase() || 'O'
}
</script>

<template>
  <section class="profile-header orbit-surface">
    <div class="profile-cover" aria-hidden="true"></div>

    <div class="profile-identity">
      <div class="profile-avatar">
        <img v-if="avatarPath && !avatarFailed" :src="avatarPath" alt="Profile avatar" @error="avatarFailed = true" />
        <span v-else>{{ initials() }}</span>
      </div>

      <div class="profile-copy">
        <div class="profile-title-row">
          <div>
            <p class="orbit-meta">
              {{ addEdit ? 'Your profile' : 'Orbit member' }}
            </p>

            <h1>{{ firstName }} {{ lastName }}</h1>

            <p v-if="username" class="profile-username">
              @{{ username }}
            </p>

            <p v-if="email" class="profile-username">
              Email: {{ email }}
            </p>
            
            <p v-if="formatDob(dob)" class="profile-dob">
              <IconGlyph name="calendar" :size="14" />
              {{ formatDob(dob) }}
            </p>
            
          </div>

          <div class="profile-actions">
            <RouterLink v-if="addEdit" class="profile-action profile-action--primary" to="/me/edit">
              Edit profile
            </RouterLink>

            <button v-else-if="followingStatus === -1" class="profile-action profile-action--primary" type="button"
              :disabled="relationshipBusy" @click="changeRelationship('POST')">
              {{ relationshipBusy ? 'Updating...' : 'Follow' }}
            </button>

            <button v-else-if="followingStatus === 0" class="profile-action" type="button" :disabled="relationshipBusy"
              @click="changeRelationship('DELETE')">
              {{ relationshipBusy ? 'Updating...' : 'Requested' }}
            </button>

            <button v-else class="profile-action" type="button" :disabled="relationshipBusy"
              @click="changeRelationship('DELETE')">
              {{ relationshipBusy ? 'Updating...' : 'Following' }}
            </button>

            <RouterLink v-if="!addEdit && canMessage" class="profile-action"
              :to="{ path: '/chats', query: { user: route.query.id } }">
              <IconGlyph name="chat" :size="16" />
              Message
            </RouterLink>
          </div>
        </div>

        <p class="profile-bio">
          {{ bio || 'No bio added yet.' }}
        </p>

        <div class="privacy-row">
          <p class="privacy-badge" :class="{ 'privacy-badge--private': isPrivate }">
            <IconGlyph :name="isPrivate ? 'lock' : 'globe'" :size="14" />
            {{ isPrivate ? 'Private profile' : 'Public profile' }}
          </p>

          <!-- on my own profile i can switch public / private right here -->
          <button v-if="addEdit" class="privacy-switch" type="button" @click="$emit('privacy-change', !isPrivate)">
            {{ isPrivate ? 'Make public' : 'Make private' }}
          </button>
        </div>
      </div>
    </div>

    <div class="profile-stats" aria-label="Profile statistics">
      <button type="button" @click="$emit('select-tab', 'posts')">
        <strong>{{ numOfPosts || 0 }}</strong>
        <span>Posts</span>
      </button>

      <button type="button" @click="$emit('select-tab', 'followers')">
        <strong>{{ numOfFollowers || 0 }}</strong>
        <span>Followers</span>
      </button>

      <button type="button" @click="$emit('select-tab', 'following')">
        <strong>{{ numOfFollowing || 0 }}</strong>
        <span>Following</span>
      </button>
    </div>
  </section>
</template>

<style scoped>
.privacy-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
}

.privacy-switch {
  min-height: 2rem;
  padding: 0 var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: transparent;
  color: var(--color-text-soft);
  cursor: pointer;
  font-size: .8125rem;
  font-weight: 600;
}

.privacy-switch:hover {
  border-color: var(--color-violet);
  color: var(--color-text);
}

.profile-header {
  overflow: hidden;
}

.profile-cover {
  height: 7rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface-raised);
  box-shadow: inset 0 -3px var(--color-violet);
}

.profile-identity {
  display: grid;
  grid-template-columns: 8rem minmax(0, 1fr);
  gap: var(--space-5);
  padding: 0 var(--space-6) var(--space-5);
}

.profile-avatar {
  display: grid;
  width: 8rem;
  height: 8rem;
  margin-top: -3.5rem;
  place-items: center;
  overflow: hidden;
  border: 4px solid var(--color-surface);
  border-radius: 50%;
  background: var(--gradient-action);
  color: white;
  font-family: var(--font-display);
  font-size: 2rem;
  font-weight: 700;
  box-shadow: 0 .75rem 2rem rgb(0 0 0 / 28%);
}

.profile-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.profile-copy {
  min-width: 0;
  padding-top: var(--space-5);
}

.profile-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.profile-title-row .orbit-meta {
  margin: 0 0 var(--space-1);
  color: var(--color-violet-soft);
}

h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 2rem;
  line-height: 1.15;
  letter-spacing: 0;
}

.profile-username {
  margin: var(--space-1) 0 0;
  color: var(--color-text-muted);
  font-family: var(--font-meta);
  font-size: .8125rem;
}

.profile-dob {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin: var(--space-2) 0 0;
  color: var(--color-text-muted);
  font-family: var(--font-meta);
  font-size: .8rem;
}

.profile-bio {
  max-width: 42rem;
  margin: var(--space-3) 0;
  color: var(--color-text-soft);
  line-height: 1.6;
}

.privacy-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  color: var(--color-mint);
  font-size: .8125rem;
  font-weight: 600;
}

.privacy-badge--private {
  color: var(--color-amber-soft);
}

.profile-actions {
  display: flex;
  flex: 0 0 auto;
  gap: var(--space-2);
}

.profile-action {
  display: inline-flex;
  min-height: var(--touch-target);
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: 0 var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--color-text-soft);
  cursor: pointer;
  font-weight: 700;
  text-decoration: none;
}

.profile-action:hover:not(:disabled) {
  border-color: var(--color-violet);
  background: var(--color-input);
}

.profile-action--primary {
  border: 0;
  background: var(--gradient-action);
  color: white;
}

.profile-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-top: 1px solid var(--color-border);
}

.profile-stats button {
  display: grid;
  min-height: 4.5rem;
  place-content: center;
  gap: var(--space-1);
  border: 0;
  border-right: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  text-align: center;
}

.profile-stats button:last-child {
  border-right: 0;
}

.profile-stats button:hover {
  background: var(--color-input);
  color: var(--color-text);
}

.profile-stats strong {
  color: var(--color-text);
  font-family: var(--font-display);
  font-size: 1.15rem;
}

.profile-stats span {
  font-size: .75rem;
}

@media (max-width: 650px) {
  .profile-cover {
    height: 5.5rem;
  }

  .profile-identity {
    grid-template-columns: 1fr;
    gap: var(--space-3);
    padding: 0 var(--space-4) var(--space-4);
  }

  .profile-avatar {
    width: 6.5rem;
    height: 6.5rem;
    margin-top: -3.25rem;
  }

  .profile-copy {
    padding-top: 0;
  }

  .profile-title-row {
    align-items: stretch;
    flex-direction: column;
  }

  h1 {
    font-size: 1.65rem;
  }

  .profile-actions,
  .profile-action {
    width: 100%;
  }

  .profile-stats button {
    min-height: 4rem;
  }
}
</style>
