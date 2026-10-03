<script setup>
import { onBeforeUnmount, reactive, ref } from 'vue'

import {
  loggingSession,
  registerUser,
  checkRegistration,
  sendEmailCode,
  verifyEmailCode,
} from '@/api/auth/auth.js'

import {
  validateName,
  validateUsername,
  validateEmail,
  validateDOB,
  validatePassword,
  validateAbout,
  handleEmailInput,
  handleUsernameInput,
  handleNameInput,
  validateAvatar,
} from '@/helpers/validators/registration.js'

import InputHolder from './InputHolder.vue'

import { router } from '@/router/router.js'
import { addNotification } from '@/data/notifications.js'
import IconGlyph from '@/components/layout/IconGlyph.vue'

const signingUp = ref(false)

const avatarInput = ref(null)
const avatarName = ref('')

const loginErrors = reactive({
  identifier: '',
  password: '',
  form: '',
})

const errors = reactive({
  firstName: '',
  lastName: '',
  username: '',
  dob: '',
  email: '',
  about: '',
  password: '',
  avatar: '',
})

const touched = reactive({
  firstName: false,
  lastName: false,
  username: false,
  dob: false,
  email: false,
  about: false,
  password: false,
  avatar: false,
})

const availability = reactive({
  username: null,
  email: null,
})

const CODE_LENGTH = 6
const MAX_ATTEMPTS = 3

const step = ref('form')
const pendingEmail = ref('')
const code = ref('')
const codeError = ref('')
const attemptsLeft = ref(MAX_ATTEMPTS)
const resendIn = ref(0)
const sendingCode = ref(false)
const verifying = ref(false)
const registering = ref(false)

let pendingForm = null
let verifiedEmail = ''
let verifyToken = ''
let resendTimer = null

function stopResendTimer() {
  if (resendTimer) {
    clearInterval(resendTimer)
    resendTimer = null
  }
}

function startResendTimer(seconds) {
  stopResendTimer()

  resendIn.value = seconds

  if (seconds <= 0) return

  resendTimer = setInterval(() => {
    resendIn.value -= 1

    if (resendIn.value <= 0) {
      resendIn.value = 0
      stopResendTimer()
    }
  }, 1000)
}

onBeforeUnmount(stopResendTimer)

function resetVerification() {
  stopResendTimer()

  step.value = 'form'
  pendingEmail.value = ''
  code.value = ''
  codeError.value = ''
  attemptsLeft.value = MAX_ATTEMPTS
  resendIn.value = 0

  pendingForm = null
  verifiedEmail = ''
  verifyToken = ''
}

function validateField(field, value) {
  touched[field] = true

  const validators = {
    avatar: validateAvatar,
    firstName: validateName,
    lastName: validateName,
    username: validateUsername,
    email: validateEmail,
    dob: validateDOB,
    password: validatePassword,
    about: validateAbout,
  }

  errors[field] = validators[field](value)

  if (field === 'username' && errors.username) {
    availability.username = null
  }

  if (field === 'email' && errors.email) {
    availability.email = null
  }
}

function handleName(field, event) {
  const value = handleNameInput(event.target.value)

  event.target.value = value
  validateField(field, value)
}

function handleUsername(event) {
  const value = handleUsernameInput(event.target.value)

  event.target.value = value
  validateField('username', value)

  if (!value) {
    availability.username = null
  }
}

function handleEmail(event) {
  const value = handleEmailInput(event.target.value)

  event.target.value = value
  validateField('email', value)

  if (!value) {
    availability.email = null
  }
}

async function checkAvailability(field, value) {
  if (!value || errors[field]) {
    availability[field] = null
    return
  }

  try {
    const type = field === 'username' ? 'name' : 'email'

    const result = await checkRegistration(type, value)

    availability[field] = result.avilable
  } catch {
    availability[field] = null
  }
}

async function checkUsernameAvailability(event) {
  const value = event.target.value.trim()

  validateField('username', value)

  if (errors.username) {
    availability.username = null
    return
  }

  await checkAvailability('username', value)
}

async function checkEmailAvailability(event) {
  const value = event.target.value.trim()

  validateField('email', value)

  if (errors.email) {
    availability.email = null
    return
  }

  await checkAvailability('email', value)
}

function inputClass(field) {
  if (!touched[field]) return ''

  if (errors[field]) return 'input-error'

  if (availability[field] === false) return 'input-error'

  if (availability[field] === true) return 'input-valid'

  return ''
}

function updateAvatar(file) {
  avatarName.value = file?.name || ''
  validateField('avatar', file)
}

function handleAvatar(event) {
  updateAvatar(event.target.files?.[0] || null)
}

function handleAvatarDrop(event) {
  const files = event.dataTransfer?.files

  if (!files?.length) return

  try {
    avatarInput.value.files = files
  } catch {
    // Browsers can block assigning a dropped FileList.
  }

  updateAvatar(files[0])
}

function validateForm(form) {
  const data = new FormData(form)

  validateField('firstName', data.get('FirstName') || '')
  validateField('lastName', data.get('LastName') || '')
  validateField('username', data.get('UserName') || '')
  validateField('dob', data.get('dob') || '')
  validateField('email', data.get('Email') || '')
  validateField('about', data.get('About') || '')
  validateField('password', data.get('Password') || '')
  validateField('avatar', data.get('Avatar'))

  return (
    Object.values(errors).every(error => !error) &&
    availability.email !== false &&
    (
      !data.get('UserName') ||
      availability.username !== false
    )
  )
}

async function requestCode(email) {
  sendingCode.value = true
  codeError.value = ''

  try {
    const result = await sendEmailCode(email)

    code.value = ''
    attemptsLeft.value = result.attempts ?? MAX_ATTEMPTS
    startResendTimer(result.cooldown ?? 60)
    step.value = 'code'
  } catch (error) {
    if (error.status === 409) {
      touched.email = true
      availability.email = false
      step.value = 'form'
      addNotification(error.message, 'error')
      return
    }

    if (error.status === 429) {
      startResendTimer(error.retryAfter ?? 60)
      step.value = 'code'
      codeError.value = error.message
      return
    }

    addNotification(`Could not send code: ${error.message}`, 'error')
  } finally {
    sendingCode.value = false
  }
}

async function finishRegistration() {
  if (!pendingForm) return

  registering.value = true
  pendingForm.set('VerifyToken', verifyToken)

  try {
    const result = await registerUser(pendingForm)

    if (!result.status) {
      addNotification(`Failed to register: ${result.message}`, 'error')
      step.value = 'form'
      return
    }

    addNotification('Account created successfully.', 'success')

    // sign the new account in right away with what was just typed
    const email = String(pendingForm.get('Email') || '').trim().toLowerCase()
    const password = String(pendingForm.get('Password') || '')
    const remember = pendingForm.get('Remember') !== null

    resetVerification()
    signingUp.value = false

    try {
      const login = await loggingSession({ Identifier: email, Pass: password, Remember: remember })
      if (login?.status) router.replace('/home')
    } catch {
      // the account exists, the person can still sign in with the form
    }
  } catch (error) {
    if (error.status === 403) {
      verifiedEmail = ''
      verifyToken = ''
      code.value = ''
      attemptsLeft.value = 0
      codeError.value = 'Your email verification expired. Request a new code.'
      step.value = 'code'
    } else {
      step.value = 'form'
    }

    addNotification(`Failed to register: ${error.message}`, 'error')
  } finally {
    registering.value = false
  }
}

async function sendData(event) {
  event.preventDefault()

  if (sendingCode.value || registering.value) return

  const form = event.target

  if (!validateForm(form)) return

  pendingForm = new FormData(form)

  const email = String(pendingForm.get('Email') || '').trim().toLowerCase()

  pendingEmail.value = email

  if (verifyToken && verifiedEmail === email) {
    await finishRegistration()
    return
  }

  await requestCode(email)
}

function handleCode(event) {
  const value = event.target.value.replace(/\D/g, '').slice(0, CODE_LENGTH)

  event.target.value = value
  code.value = value
  codeError.value = ''
}

async function verifyCode() {
  if (verifying.value || registering.value || attemptsLeft.value <= 0) return

  if (code.value.length !== CODE_LENGTH) {
    codeError.value = `Enter the ${CODE_LENGTH}-digit code`
    return
  }

  verifying.value = true
  codeError.value = ''

  try {
    const result = await verifyEmailCode(pendingEmail.value, code.value)

    verifiedEmail = pendingEmail.value
    verifyToken = result.token

    await finishRegistration()
  } catch (error) {
    if (typeof error.attemptsLeft === 'number') {
      attemptsLeft.value = error.attemptsLeft
    }

    code.value = ''
    codeError.value = error.message
  } finally {
    verifying.value = false
  }
}

async function resendCode() {
  if (sendingCode.value || resendIn.value > 0) return

  await requestCode(pendingEmail.value)
}

function backToForm() {
  step.value = 'form'
  code.value = ''
  codeError.value = ''
}

async function loggUser(event) {
  event.preventDefault()

  loginErrors.identifier = ''
  loginErrors.password = ''
  loginErrors.form = ''

  const formData = new FormData(event.target)

  const identifier = String(formData.get('Identifier') || '').trim()
  const password = formData.get('Pass')
  const remember = formData.get('Remember') !== null

  if (!identifier) {
    loginErrors.identifier = 'Email or username is required'
  }

  if (!password) {
    loginErrors.password = 'Password is required'
  }

  if (loginErrors.identifier || loginErrors.password) return

  try {
    const result = await loggingSession({
      Identifier: identifier,
      Pass: password,
      Remember: remember,
    })

    if (result.status) router.replace('/home')
  } catch (error) {
    loginErrors.form =
      error.message || 'Invalid email, username, or password'
  }
}
</script>

<template>
  <section class="auth-section" aria-label="Account access">
    <div class="auth-card">
      <nav class="auth-tabs" aria-label="Account access">
        <button
          class="auth-tab"
          :class="{ 'auth-tab--active': !signingUp }"
          type="button"
          @click="signingUp = false"
        >
          Sign in
        </button>

        <button
          class="auth-tab"
          :class="{ 'auth-tab--active': signingUp }"
          type="button"
          @click="signingUp = true"
        >
          Create account
        </button>
      </nav>

      <div v-if="!signingUp" class="auth-panel">
        <form class="auth-form" @submit.prevent="loggUser">
          <div class="input-group">
            <label for="login-email">EMAIL OR USERNAME *</label>

            <InputHolder
              id="login-email"
              type="text"
              name="Identifier"
              :min-length="3"
              :max-length="75"
              place-holder="noa@orbit.app"
              required
              autocomplete="username"
            />

            <span
              v-if="loginErrors.identifier"
              class="input-error-message"
            >
              {{ loginErrors.identifier }}
            </span>
          </div>

          <div class="input-group">
            <label for="login-password">PASSWORD *</label>

            <InputHolder
              id="login-password"
              type="password"
              name="Pass"
              :min-length="8"
              :max-length="100"
              place-holder="••••••••••"
              required
              autocomplete="current-password"
            />

            <span
              v-if="loginErrors.password"
              class="input-error-message"
            >
              {{ loginErrors.password }}
            </span>
          </div>

          <span
            v-if="loginErrors.form"
            class="input-error-message"
            role="alert"
          >
            {{ loginErrors.form }}
          </span>

          <label class="remember-row">
            <input type="checkbox" name="Remember" checked />
            <span>
              Keep me signed in on this device
              <small>session cookie</small>
            </span>
          </label>

          <button class="auth-submit" type="submit">
            Let's go
            <IconGlyph name="arrowRight" :size="16" />
          </button>
        </form>

        <p class="auth-footer">
          Don't have an account?
          <button type="button" @click="signingUp = true">
            Create account
          </button>
        </p>
      </div>

      <div v-else class="auth-panel">
        <form
          v-show="step === 'form'"
          class="auth-form"
          @submit.prevent="sendData"
        >
          <div class="input-group">
            <label for="signup-email">EMAIL *</label>

            <InputHolder
              id="signup-email"
              type="email"
              name="Email"
              :min-length="5"
              :max-length="75"
              place-holder="noa@orbit.app"
              :class="inputClass('email')"
              autocomplete="email"
              required
              @input="handleEmail"
              @blur="checkEmailAvailability"
            />

            <span
              v-if="touched.email && errors.email"
              class="input-error-message"
            >
              {{ errors.email }}
            </span>

            <span
              v-else-if="availability.email === false"
              class="input-error-message"
            >
              Email is already in use
            </span>
          </div>

          <div class="input-group">
            <label for="signup-password">PASSWORD *</label>

            <InputHolder
              id="signup-password"
              type="password"
              name="Password"
              :min-length="8"
              :max-length="75"
              place-holder="••••••••••"
              :class="inputClass('password')"
              autocomplete="new-password"
              required
              @input="validateField('password', $event.target.value)"
            />

            <span
              v-if="touched.password && errors.password"
              class="input-error-message"
            >
              {{ errors.password }}
            </span>
          </div>

          <div class="input-row">
            <div class="input-group">
              <label for="first-name">FIRST NAME *</label>

              <InputHolder
                id="first-name"
                type="text"
                name="FirstName"
                :min-length="2"
                :max-length="15"
                place-holder="Noa"
                :class="inputClass('firstName')"
                autocomplete="given-name"
                required
                @input="handleName('firstName', $event)"
              />

              <span
                v-if="touched.firstName && errors.firstName"
                class="input-error-message"
              >
                {{ errors.firstName }}
              </span>
            </div>

            <div class="input-group">
              <label for="last-name">LAST NAME *</label>

              <InputHolder
                id="last-name"
                type="text"
                name="LastName"
                :min-length="2"
                :max-length="15"
                place-holder="Ferreira"
                :class="inputClass('lastName')"
                autocomplete="family-name"
                required
                @input="handleName('lastName', $event)"
              />

              <span
                v-if="touched.lastName && errors.lastName"
                class="input-error-message"
              >
                {{ errors.lastName }}
              </span>
            </div>
          </div>

          <div class="input-row">
            <div class="input-group">
              <label for="dob">DATE OF BIRTH *</label>

              <InputHolder
                id="dob"
                type="date"
                name="dob"
                :class="inputClass('dob')"
                required
                @input="validateField('dob', $event.target.value)"
              />

              <span
                v-if="touched.dob && errors.dob"
                class="input-error-message"
              >
                {{ errors.dob }}
              </span>
            </div>

            <div class="input-group">
              <label for="username">
                NICKNAME
                <span>— OPTIONAL</span>
              </label>

              <InputHolder
                id="username"
                type="text"
                name="UserName"
                :min-length="3"
                :max-length="12"
                place-holder="@noa.png"
                :class="inputClass('username')"
                autocomplete="nickname"
                @input="handleUsername"
                @blur="checkUsernameAvailability"
              />

              <span
                v-if="touched.username && errors.username"
                class="input-error-message"
              >
                {{ errors.username }}
              </span>

              <span
                v-else-if="availability.username === false"
                class="input-error-message"
              >
                Username is already taken
              </span>
            </div>
          </div>

          <div class="input-group">
            <label for="avatar">
              AVATAR
              <span>— OPTIONAL · JPG, PNG, GIF</span>
            </label>

            <label
              class="upload-zone"
              for="avatar"
              @dragover.prevent
              @drop.prevent="handleAvatarDrop"
            >
              <span class="upload-zone__icon">
                <IconGlyph name="upload" :size="18" />
              </span>

              <span>
                {{ avatarName || 'Drop an image or click to browse' }}
              </span>

              <input
                id="avatar"
                ref="avatarInput"
                type="file"
                name="Avatar"
                accept="image/jpeg,image/png,image/gif"
                @change="handleAvatar"
              />
            </label>

            <span
              v-if="touched.avatar && errors.avatar"
              class="input-error-message"
            >
              {{ errors.avatar }}
            </span>
          </div>

          <div class="input-group">
            <label for="about">
              ABOUT ME
              <span>— OPTIONAL</span>
            </label>

            <textarea
              id="about"
              name="About"
              maxlength="1000"
              placeholder="Trail runner, pixel-art hobbyist, espresso before noon…"
              :class="inputClass('about')"
              @input="validateField('about', $event.target.value)"
            ></textarea>

            <span
              v-if="touched.about && errors.about"
              class="input-error-message"
            >
              {{ errors.about }}
            </span>
          </div>

          <label class="remember-row">
            <input type="checkbox" name="Remember" checked />

            <span>
              Keep me signed in on this device
              <small>session cookie</small>
            </span>
          </label>

          <button
            class="auth-submit"
            type="submit"
            :disabled="sendingCode || registering"
          >
            {{ sendingCode ? 'Sending code…' : 'Create my orbit' }}
            <IconGlyph name="arrowRight" :size="16" />
          </button>
        </form>

        <form
          v-show="step === 'code'"
          class="auth-form"
          @submit.prevent="verifyCode"
        >
          <div class="verify-intro">
            <h2 class="verify-title">Verify your email</h2>

            <p class="verify-text">
              We sent a 6-digit code to
              <strong>{{ pendingEmail }}</strong>.
              It expires in 10 minutes.
            </p>
          </div>

          <div class="input-group">
            <label for="verify-code">VERIFICATION CODE *</label>

            <InputHolder
              id="verify-code"
              type="text"
              name="Code"
              :max-length="6"
              place-holder="123456"
              class="code-input"
              inputmode="numeric"
              autocomplete="one-time-code"
              :value="code"
              :disabled="attemptsLeft <= 0 || verifying || registering"
              @input="handleCode"
            />

            <span
              v-if="codeError"
              class="input-error-message"
              role="alert"
            >
              {{ codeError }}
            </span>

            <span v-else-if="attemptsLeft > 0" class="verify-hint">
              {{ attemptsLeft }} {{ attemptsLeft === 1 ? 'try' : 'tries' }} left
            </span>
          </div>

          <button
            class="auth-submit"
            type="submit"
            :disabled="
              attemptsLeft <= 0 ||
              verifying ||
              registering ||
              code.length !== 6
            "
          >
            {{
              verifying || registering
                ? 'Verifying…'
                : 'Verify and create account'
            }}
            <IconGlyph name="arrowRight" :size="16" />
          </button>

          <div class="verify-actions">
            <button
              class="link-button"
              type="button"
              :disabled="sendingCode || resendIn > 0"
              @click="resendCode"
            >
              {{
                sendingCode
                  ? 'Sending…'
                  : resendIn > 0
                    ? `Resend code in ${resendIn}s`
                    : 'Resend code'
              }}
            </button>

            <button class="link-button" type="button" @click="backToForm">
              Change email
            </button>
          </div>
        </form>

        <p class="auth-footer">
          Already orbiting?
          <button type="button" @click="signingUp = false">
            Sign in
          </button>
        </p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.auth-section {
  width: 100%;
  max-width: 44rem;
  margin-inline: auto;
}

.auth-card {
  padding: 2.25rem 3rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-medium);
  background: var(--color-surface);
  box-shadow: var(--shadow-raised);
}

.auth-tabs {
  display: flex;
  gap: 1.75rem;
  margin-bottom: 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.auth-tab {
  min-height: 2.75rem;
  padding: 0 0 .75rem;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: .875rem;
  font-weight: 600;
}

.auth-tab:hover,
.auth-tab--active {
  color: var(--color-text);
}

.auth-tab--active {
  border-color: var(--color-coral);
}

.auth-form,
.input-group {
  display: grid;
  gap: .5rem;
}

.auth-form {
  gap: 1rem;
}

.input-group {
  gap: .25rem;
}

.input-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: .75rem;
}

.input-group {
  min-width: 0;
}

.input-group > label {
  color: var(--color-text-muted);
  font-family: var(--font-meta);
  font-size: .6875rem;
  letter-spacing: .08em;
}

.input-group > label span {
  color: var(--color-text-faint);
}

.input-group textarea {
  width: 100%;
  min-height: 4.5rem;
  padding: .75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  outline: 0;
  background: var(--color-input);
  color: var(--color-text);
  resize: vertical;
}

.input-group textarea:focus {
  border-color: var(--color-violet);
  box-shadow: var(--focus-ring);
}

.upload-zone {
  display: flex;
  align-items: center;
  gap: .75rem;
  min-height: 3.5rem;
  padding: .75rem;
  border: 1px dashed var(--color-violet);
  border-radius: var(--radius-small);
  background: rgb(var(--rgb-violet) / 5%);
  color: var(--color-text-soft);
  cursor: pointer;
}

.upload-zone:hover {
  background: rgb(var(--rgb-violet) / 11%);
}

.upload-zone__icon {
  color: var(--color-violet);
  font-size: 1.25rem;
  line-height: 1;
}

.upload-zone input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
}

.remember-row {
  display: flex;
  align-items: flex-start;
  gap: .5rem;
  color: var(--color-text-muted);
  font-size: .75rem;
}

.remember-row input {
  width: 1rem;
  height: 1rem;
  margin: 0;
  accent-color: var(--color-violet);
}

.remember-row small {
  color: var(--color-text-faint);
  font-family: var(--font-meta);
  font-size: .6875rem;
}

.auth-submit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: .45rem;
  width: 100%;
  min-height: 3.25rem;
  border: 0;
  border-radius: 999px;
  background: var(--gradient-action);
  color: var(--color-text);
  cursor: pointer;
  font-weight: 700;
}

.auth-submit:hover {
  filter: brightness(1.08);
}

.auth-submit:disabled {
  cursor: not-allowed;
  filter: none;
  opacity: .55;
}

.verify-intro {
  display: grid;
  gap: .375rem;
}

.verify-title {
  margin: 0;
  color: var(--color-text);
  font-size: 1.125rem;
}

.verify-text {
  margin: 0;
  color: var(--color-text-muted);
  font-size: .875rem;
  line-height: 1.5;
}

.verify-text strong {
  color: var(--color-text);
  overflow-wrap: anywhere;
}

.verify-hint {
  color: var(--color-text-faint);
  font-size: .75rem;
}

.code-input {
  font-family: var(--font-meta);
  font-size: 1.25rem;
  letter-spacing: .5em;
  text-align: center;
}

.verify-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: .75rem;
}

.link-button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-violet-soft);
  cursor: pointer;
  font: inherit;
  font-size: .8125rem;
}

.link-button:hover:not(:disabled) {
  color: var(--color-text);
}

.link-button:disabled {
  color: var(--color-text-faint);
  cursor: not-allowed;
}

.auth-footer {
  margin: .5rem 0 0;
  color: var(--color-text-muted);
  font-size: .8125rem;
  text-align: center;
}

.auth-footer button {
  border: 0;
  background: transparent;
  color: var(--color-violet-soft);
  cursor: pointer;
  font: inherit;
}

.auth-footer button:hover {
  color: var(--color-text);
}

.input-error-message {
  color: var(--color-coral);
  font-size: .75rem;
}

@media (max-width: 30rem) {
  .input-row {
    grid-template-columns: 1fr;
  }
}
</style>
