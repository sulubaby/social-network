<script setup>
import { computed, reactive, ref } from 'vue'
import FormField from '@/components/ProfileEdit/FormField.vue'
import AvatarUploader from '@/components/ProfileEdit/AvatarUploader.vue'
import { updateUserInfo } from '@/api/users/editProfile'
import { addNotification } from '@/data/notifications'
import { profileData } from '@/data/usersData'

const props = defineProps({
  firstName: String,
  lastName: String,
  username: String,
  email: String,
  bio: String,
  avatarPath: String,
  isPrivate: Boolean,
})
const emit = defineEmits(['avatar-change'])
const form = reactive({
  FirstName: props.firstName || '',
  LastName: props.lastName || '',
  Username: props.username || '',
  Email: props.email || '',
  About: props.bio || '',
  Password: '',
  IsPrivate: Boolean(props.isPrivate),
})
const savedSnapshot = ref(snapshot())
const saving = ref(false)
const feedback = ref('')
const saveError = ref('')
const isDirty = computed(() => snapshot() !== savedSnapshot.value)

function snapshot() {
  return JSON.stringify({
    FirstName: form.FirstName,
    LastName: form.LastName,
    Username: form.Username,
    Email: form.Email,
    About: form.About,
    Password: form.Password,
    IsPrivate: form.IsPrivate,
  })
}

async function updateInfo() {
  if (saving.value || !isDirty.value) return
  // changing public / private is a big change, so ask first
  const privacyChanged = Boolean(form.IsPrivate) !== (profileData.userInfo.isPrivate === 1)
  if (privacyChanged && !window.confirm(form.IsPrivate
    ? 'Make your profile private? Only your followers will see your posts and info.'
    : 'Make your profile public? Everyone will see your posts and info.')) return
  saving.value = true
  feedback.value = ''
  saveError.value = ''
  try {
    const result = await updateUserInfo({ ...form, IsPrivate: form.IsPrivate ? 1 : 0 })
    profileData.userInfo.firstName = form.FirstName
    profileData.userInfo.lastName = form.LastName
    profileData.userInfo.userName = form.Username
    profileData.userInfo.email = form.Email
    profileData.userInfo.about = form.About
    profileData.userInfo.isPrivate = form.IsPrivate ? 1 : 0
    profileData.about.bio = form.About
    form.Password = ''
    savedSnapshot.value = snapshot()
    feedback.value = result.message || 'Profile saved.'
    addNotification(feedback.value, 'success')
  } catch (err) {
    saveError.value = err.message || 'Could not save profile changes.'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="edit-section orbit-surface" aria-labelledby="personal-info-heading">
    <header>
      <p class="orbit-meta">Account</p>
      <h2 id="personal-info-heading">Personal information</h2>
    </header>

    <div class="edit-group">
      <div class="edit-group__heading"><h3>Profile picture</h3><p>Shown across posts, comments, and conversations.</p></div>
      <AvatarUploader :src="avatarPath" @change="$emit('avatar-change', $event)" />
    </div>

    <form @submit.prevent="updateInfo">
      <div class="edit-group">
        <div class="edit-group__heading"><h3>Basic information</h3><p>Use the name and username people know you by.</p></div>
        <div class="field-grid">
          <FormField id="firstName" v-model="form.FirstName" label="First name" placeholder="First name" />
          <FormField id="lastName" v-model="form.LastName" label="Last name" placeholder="Last name" />
          <FormField id="username" v-model="form.Username" label="Username" placeholder="Username" />
          <FormField id="email" v-model="form.Email" label="Email" type="email" placeholder="Email address" />
        </div>
        <FormField id="bio" v-model="form.About" label="About me" type="textarea" placeholder="Tell people about yourself" />
        <FormField id="password" v-model="form.Password" label="New password (optional)" type="password" placeholder="Leave blank to keep your current password" />
      </div>

      <fieldset class="visibility-group">
        <legend>Profile visibility</legend>
        <label :class="{ selected: !form.IsPrivate }">
          <input v-model="form.IsPrivate" type="radio" :value="false" />
          <span><strong>Public</strong><small>Anyone can view your profile and public activity.</small></span>
        </label>
        <label :class="{ selected: form.IsPrivate }">
          <input v-model="form.IsPrivate" type="radio" :value="true" />
          <span><strong>Private</strong><small>Only approved followers can view your full profile.</small></span>
        </label>
      </fieldset>

      <p v-if="feedback" class="form-feedback" role="status">{{ feedback }}</p>
      <p v-if="saveError" class="form-feedback form-feedback--error" role="alert">{{ saveError }}</p>
      <div class="form-actions">
        <RouterLink to="/me">Cancel</RouterLink>
        <button type="submit" :disabled="saving || !isDirty">{{ saving ? 'Saving...' : 'Save changes' }}</button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.edit-section { padding: var(--space-5); }
.edit-section > header { margin-bottom: var(--space-5); }
.edit-section .orbit-meta { margin: 0; color: var(--color-violet-soft); }
.edit-section h2 { margin: var(--space-1) 0 0; font-family: var(--font-display); font-size: 1.5rem; letter-spacing: 0; }
form, .edit-group { display: grid; gap: var(--space-4); }
.edit-group { padding: var(--space-5) 0; border-top: 1px solid var(--color-border); }
.edit-group__heading h3 { margin: 0; font-size: 1rem; }
.edit-group__heading p { margin: var(--space-1) 0 0; color: var(--color-text-muted); font-size: .875rem; }
.field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-4); }
.visibility-group { display: grid; gap: var(--space-3); margin: 0; padding: var(--space-5) 0; border: 0; border-top: 1px solid var(--color-border); }
.visibility-group legend { margin-bottom: var(--space-3); padding: 0; font-weight: 700; }
.visibility-group label { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; gap: var(--space-3); padding: var(--space-4); border: 1px solid var(--color-border); border-radius: var(--radius-small); cursor: pointer; }
.visibility-group label.selected { border-color: var(--color-mint); background: var(--color-surface-teal); }
.visibility-group input { margin-top: .25rem; accent-color: var(--color-mint); }
.visibility-group span { display: grid; gap: var(--space-1); }
.visibility-group small { color: var(--color-text-muted); line-height: 1.5; }
.form-feedback { margin: 0; padding: var(--space-3) var(--space-4); border-left: 3px solid var(--color-mint); background: var(--color-surface-teal); color: var(--color-text-soft); }
.form-feedback--error { border-color: var(--color-coral); background: var(--color-surface-coral); color: var(--color-coral-soft); }
.form-actions { display: flex; justify-content: flex-end; gap: var(--space-3); padding-top: var(--space-4); border-top: 1px solid var(--color-border); }
.form-actions a, .form-actions button { display: inline-flex; min-height: var(--touch-target); align-items: center; justify-content: center; padding: 0 var(--space-4); border-radius: var(--radius-small); font-weight: 700; text-decoration: none; }
.form-actions a { border: 1px solid var(--color-border); color: var(--color-text-soft); }
.form-actions button { border: 0; background: var(--gradient-action); color: white; cursor: pointer; }
@media (max-width: 600px) {
  .edit-section { padding: var(--space-4); }
  .field-grid { grid-template-columns: 1fr; }
  .form-actions { display: grid; }
}
</style>
