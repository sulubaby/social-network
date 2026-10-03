<script setup>
import { computed, reactive, ref } from 'vue'
import FormField from '@/components/ProfileEdit/FormField.vue'
import AvatarUploader from '@/components/ProfileEdit/AvatarUploader.vue'
import { updateUserInfo } from '@/api/users/editProfile'
import { sendEmailCode, verifyEmailCode } from '@/api/auth/auth'
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

// a new email needs a code from that inbox before we can save it
const savedEmail = ref(props.email || '')
const emailChanged = computed(() => form.Email.trim().toLowerCase() !== savedEmail.value.toLowerCase())
const codeSentTo = ref('')
const emailCode = ref('')
const verifyToken = ref('')
const verifiedEmail = ref('')
const emailBusy = ref(false)
const emailError = ref('')
const needsCode = computed(() => emailChanged.value && verifiedEmail.value !== form.Email.trim().toLowerCase())

async function sendCode() {
  emailBusy.value = true
  emailError.value = ''
  try {
    await sendEmailCode(form.Email)
    codeSentTo.value = form.Email.trim().toLowerCase()
    addNotification('We sent a code to ' + form.Email, 'success')
  } catch (err) {
    emailError.value = err.message
  } finally {
    emailBusy.value = false
  }
}

async function checkCode() {
  emailBusy.value = true
  emailError.value = ''
  try {
    const result = await verifyEmailCode(codeSentTo.value, emailCode.value.trim())
    verifyToken.value = result.token
    verifiedEmail.value = codeSentTo.value
    emailCode.value = ''
  } catch (err) {
    emailError.value = err.message
  } finally {
    emailBusy.value = false
  }
}

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
  if (needsCode.value) {
    saveError.value = 'Please verify your new email first.'
    return
  }
  // changing public / private is a big change, so ask first
  const privacyChanged = Boolean(form.IsPrivate) !== (profileData.userInfo.isPrivate === 1)
  if (privacyChanged && !window.confirm(form.IsPrivate
    ? 'Make your profile private? Only your followers will see your posts and info.'
    : 'Make your profile public? Everyone will see your posts and info.')) return
  saving.value = true
  feedback.value = ''
  saveError.value = ''
  try {
    const result = await updateUserInfo({ ...form, IsPrivate: form.IsPrivate ? 1 : 0, VerifyToken: verifyToken.value })
    savedEmail.value = form.Email.trim().toLowerCase()
    verifyToken.value = ''
    codeSentTo.value = ''
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
        <div v-if="needsCode" class="email-check">
          <p>To use a new email, enter the code we send to it.</p>
          <div class="email-check__row">
            <button type="button" :disabled="emailBusy" @click="sendCode">{{ codeSentTo === form.Email.trim().toLowerCase() ? 'Send again' : 'Send code' }}</button>
            <template v-if="codeSentTo === form.Email.trim().toLowerCase()">
              <input v-model="emailCode" inputmode="numeric" maxlength="6" placeholder="6 digit code" aria-label="Email code" />
              <button type="button" :disabled="emailBusy || emailCode.trim().length !== 6" @click="checkCode">Verify</button>
            </template>
          </div>
          <p v-if="emailError" class="email-check__error" role="alert">{{ emailError }}</p>
        </div>
        <p v-else-if="emailChanged" class="email-check__ok">New email verified.</p>
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
.email-check { display: grid; gap: var(--space-2); padding: var(--space-3) var(--space-4); border: 1px solid var(--color-border); border-radius: var(--radius-small); }
.email-check p { margin: 0; color: var(--color-text-soft); font-size: .875rem; }
.email-check__row { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.email-check__row input { min-height: var(--touch-target); width: 9rem; padding: 0 var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: transparent; color: inherit; }
.email-check__row button { min-height: var(--touch-target); padding: 0 var(--space-4); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: transparent; color: var(--color-text-soft); font-weight: 700; cursor: pointer; }
.email-check__error { color: var(--color-coral-soft) !important; }
.email-check__ok { margin: 0; color: var(--color-mint); font-size: .875rem; }
@media (max-width: 600px) {
  .edit-section { padding: var(--space-4); }
  .field-grid { grid-template-columns: 1fr; }
  .form-actions { display: grid; }
}
</style>
