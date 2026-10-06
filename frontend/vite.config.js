import { fileURLToPath, URL } from 'node:url'
import { SERVERPORT } from './src/data/routes.js'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:' + SERVERPORT,
        changeOrigin: true,
        ws: true
      },
      '/uploads/': {
        target: 'http://localhost:' + SERVERPORT,
        changeOrigin: true
      }
    }
  }
})
