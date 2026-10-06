<script setup>
import { onMounted } from 'vue';
import NotificationContainer from './components/layout/NotificationContainer.vue';
import { connectToWS } from './api/socket/socket.js';

onMounted(() => {
    connectToWS();
});

</script>

<template>    
    <RouterView/>

    <NotificationContainer/>
</template>

<style>
</style>