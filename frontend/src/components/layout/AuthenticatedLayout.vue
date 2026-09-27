<script setup>
import { onMounted, ref } from 'vue'

import SideNavigation from './SideNavigation.vue'
import TopNavigation from './TopNavigation.vue'

import { getUserData } from '@/api/users/personalProfile.js'
import { addNotification } from '@/data/notifications'

defineProps({
  activePage: {
    type: String,
    required: true,
  },
})

const user = ref({
  UserInfo: {
    Avatar: '',
  },
})

async function getData() {
  try {
    const result = await getUserData()

    if (!result.status) {
      addNotification(result.message || 'could not get data')
      return
    }

    user.value = result.data
  } catch (err) {
    addNotification(err || 'could not get data')
  }
}

onMounted(getData)
</script>

<template>
  <div class="authenticated-layout">
    <TopNavigation :avatar="user.UserInfo.Avatar" />

    <div class="authenticated-layout__body">
      <SideNavigation :active-page="activePage" />

      <main class="authenticated-layout__content">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.authenticated-layout {
  min-height: 100vh;
  background: var(--color-background);
}

.authenticated-layout__body {
  display: flex;
  min-height: calc(100vh - 4rem);
}

.authenticated-layout__content {
  width: 100%;
  min-width: 0;
  padding: var(--space-4) var(--space-3) 5.5rem;
}

@media (min-width: 48rem) {
  .authenticated-layout__content {
    padding: var(--space-5);
  }
}

@media (min-width: 64rem) {
  .authenticated-layout__content {
    width: calc(100% - 14.5rem);
    max-width: calc(var(--content-max-width) - 14.5rem);
    margin-inline: auto;
    padding: var(--space-5) var(--space-6);
  }
}
</style>