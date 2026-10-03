<script setup>
import { computed } from 'vue'

const props = defineProps({
  about: { type: Object, required: true },
  profile: { type: Object, default: () => ({}) },
  ownProfile: Boolean,
})

const details = computed(() => [
  { label: 'Username', value: props.profile.userName ? `@${props.profile.userName}` : '' },
  { label: 'Email', value: props.profile.email },
  { label: 'Date of birth', value: formatDate(props.profile.dob) },
  { label: 'Work', value: props.about.work },
  { label: 'Education', value: props.about.education },
  { label: 'Hobbies', value: props.about.hobbies },
  { label: 'Interests', value: props.about.intrests },
  { label: 'Travel', value: props.about.travel },
].filter((item) => item.value))

const links = computed(() => [
  { label: 'Website', value: props.about.website },
  { label: 'LinkedIn', value: props.about.linkedin },
  { label: 'Twitter / X', value: props.about.twitter },
  { label: 'Instagram', value: props.about.instgram },
].filter((item) => item.value))

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString([], { dateStyle: 'long' })
}
</script>

<template>
  <section class="about-section orbit-surface" aria-labelledby="about-heading">
    <header>
      <p class="orbit-meta">Profile details</p>
      <h2 id="about-heading">About</h2>
    </header>

    <div class="about-summary">
      <span>About me</span>
      <p>{{ about.bio || 'No bio added yet.' }}</p>
    </div>

    <dl v-if="details.length" class="detail-list">
      <div v-for="item in details" :key="item.label">
        <dt>{{ item.label }}</dt>
        <dd>{{ item.value }}</dd>
      </div>
    </dl>
    <p v-else class="about-empty">No additional profile information yet.</p>

    <div v-if="links.length" class="profile-links">
      <h3>Links</h3>
      <a v-for="link in links" :key="link.label" :href="link.value" target="_blank" rel="noopener noreferrer">
        <span>{{ link.label }}</span>
        <strong>{{ link.value }}</strong>
      </a>
    </div>
  </section>
</template>

<style scoped>
.about-section { padding: var(--space-5); }
.about-section header { margin-bottom: var(--space-5); }
.about-section .orbit-meta { margin: 0 0 var(--space-1); color: var(--color-violet-soft); }
.about-section h2 { margin: 0; font-family: var(--font-display); font-size: 1.5rem; letter-spacing: 0; }
.about-summary { display: grid; grid-template-columns: minmax(7rem, 10rem) minmax(0, 1fr); gap: var(--space-4); padding-bottom: var(--space-5); border-bottom: 1px solid var(--color-border); }
.about-summary > span, dt { color: var(--color-text-faint); font-family: var(--font-meta); font-size: .75rem; text-transform: uppercase; }
.about-summary p { margin: 0; color: var(--color-text-soft); line-height: 1.65; white-space: pre-wrap; }
.detail-list { display: grid; margin: 0; }
.detail-list > div { display: grid; grid-template-columns: minmax(7rem, 10rem) minmax(0, 1fr); gap: var(--space-4); padding: var(--space-4) 0; border-bottom: 1px solid var(--color-border); }
dd { margin: 0; color: var(--color-text-soft); overflow-wrap: anywhere; }
.about-empty { margin: 0; padding: var(--space-5) 0; color: var(--color-text-muted); }
.profile-links { display: grid; gap: var(--space-2); padding-top: var(--space-5); }
.profile-links h3 { margin: 0 0 var(--space-2); font-size: 1rem; }
.profile-links a { display: grid; grid-template-columns: minmax(7rem, 10rem) minmax(0, 1fr); gap: var(--space-4); padding: var(--space-3) 0; color: var(--color-text-soft); text-decoration: none; }
.profile-links a:hover strong { color: var(--color-mint); }
.profile-links span { color: var(--color-text-faint); font-size: .8125rem; }
.profile-links strong { overflow: hidden; color: var(--color-violet-soft); font-size: .875rem; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
@media (max-width: 560px) {
  .about-section { padding: var(--space-4); }
  .about-summary, .detail-list > div, .profile-links a { grid-template-columns: 1fr; gap: var(--space-2); }
}
</style>
