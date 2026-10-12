<script setup>
import { onMounted } from 'vue';
import NotificationContainer from './components/layout/NotificationContainer.vue';
import { connectToWS } from './api/socket/socket.js';

// open the websocket once for the whole app, chats and live notifications use it
onMounted(() => {
    connectToWS();
});

</script>

<template>    
    <!-- the page of the current url goes here -->
    <RouterView/>

    <!-- the popups for new notifications, outside the router so they show on every page -->
    <NotificationContainer/>
</template>

<style>
</style>