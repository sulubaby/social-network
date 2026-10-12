import { createApp } from 'vue'
import App from './App.vue';
import { router } from './router/router.js';
import { initTheme } from './helpers/common/theme.js';
import "./styles/global.css"

// set dark or light before the app shows so the page does not flash
initTheme();

// start vue with the router and put it in the #app div
createApp(App)
    .use(router)
    .mount('#app')