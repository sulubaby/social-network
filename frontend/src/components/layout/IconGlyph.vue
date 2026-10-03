<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: {
    type: String,
    required: true,
  },
  size: {
    type: [Number, String],
    default: 20,
  },
  strokeWidth: {
    type: [Number, String],
    default: 1.8,
  },
})

const paths = {
  home: ['M3 10.5 12 3l9 7.5', 'M5.5 9.5V21h13V9.5', 'M9.5 21v-5h5v5'],
  profile: ['M20 21a8 8 0 0 0-16 0', 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z'],
  groups: ['M16 21v-1.7a4.3 4.3 0 0 0-4.3-4.3H6.3A4.3 4.3 0 0 0 2 19.3V21', 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z', 'M22 21v-1.7a4.3 4.3 0 0 0-3-4.1', 'M16 3.1a4 4 0 0 1 0 7.8'],
  chat: ['M21 11.5a8.3 8.3 0 0 1-9 8.3 9.4 9.4 0 0 1-4-.9L3 20l1.2-4.1a8 8 0 0 1-1.5-4.4A8.3 8.3 0 0 1 12 3a8.3 8.3 0 0 1 9 8.5Z'],
  bell: ['M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9', 'M10 21h4'],
  mail: ['M4 6h16v12H4z', 'm4 7 8 6 8-6'],
  calendar: ['M6 3v3', 'M18 3v3', 'M4 8h16', 'M5 5h14a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1Z'],
  search: ['m21 21-4.2-4.2', 'M10.8 18a7.2 7.2 0 1 1 0-14.4 7.2 7.2 0 0 1 0 14.4Z'],
  logout: ['M10 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5', 'm14 16 4-4-4-4', 'M18 12H8'],
  lock: ['M7 10V7a5 5 0 0 1 10 0v3', 'M5 10h14v10H5z'],
  image: ['M4 5h16v14H4z', 'm5.5 16 4-4 3 3 2-2 4 4', 'M8.5 9h.01'],
  smile: ['M8.5 10h.01', 'M15.5 10h.01', 'M8 14a5.2 5.2 0 0 0 8 0', 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z'],
  globe: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z', 'M3 12h18', 'M12 3a13.7 13.7 0 0 1 0 18', 'M12 3a13.7 13.7 0 0 0 0 18'],
  upload: ['M12 16V4', 'm7 9 5-5 5 5', 'M5 20h14'],
  close: ['m6 6 12 12', 'm18 6-12 12'],
  check: ['m5 12 4 4L19 6'],
  arrowLeft: ['M19 12H5', 'm12 19-7-7 7-7'],
  arrowRight: ['M5 12h14', 'm12 5 7 7-7 7'],
  more: ['M5 12h.01', 'M12 12h.01', 'M19 12h.01'],
  heart: ['M20.8 8.8c0 5.5-8.8 10.2-8.8 10.2S3.2 14.3 3.2 8.8A4.8 4.8 0 0 1 12 6a4.8 4.8 0 0 1 8.8 2.8Z'],
  comment: ['M20 11.5a7.5 7.5 0 0 1-8 7.5 8.6 8.6 0 0 1-3.6-.8L4 20l1.2-3.7A7.3 7.3 0 0 1 4 11.5 7.5 7.5 0 0 1 12 4a7.5 7.5 0 0 1 8 7.5Z'],
  settings: ['M4 6h8', 'M16 6h4', 'M14 8a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z', 'M4 12h2', 'M10 12h10', 'M8 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z', 'M4 18h10', 'M18 18h2', 'M16 20a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z'],
  trash: ['M4 7h16', 'M10 11v6', 'M14 11v6', 'M6 7l1 13h10l1-13', 'M9 7V4h6v3'],
}

const iconPaths = computed(() => paths[props.name] || paths.more)
</script>

<template>
  <svg
    class="icon-glyph"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="path in iconPaths" :key="path" :d="path" />
  </svg>
</template>

<style scoped>
.icon-glyph {
  display: block;
  flex: 0 0 auto;
}
</style>
