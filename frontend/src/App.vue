<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import NotificationContainer from './components/layout/NotificationContainer.vue'

const route = useRoute()

// pages like /user?id=2 -> /user?id=3 or /groups/1 -> /groups/2 reuse the same
// component, so give each person / group its own key and the page loads fresh.
// other query changes (like typing in /search) must not reload the page
const viewKey = computed(() => (route.path === '/user' ? `/user/${route.query.id}` : route.path))
</script>

<template>
    <RouterView :key="viewKey" />
    <NotificationContainer />
</template>
