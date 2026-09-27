<script setup>
import CommentMenu from '@/components/comments/CommentMenu.vue'

// shows one comment. deleting is true while this comment is being deleted
defineProps({
  comment: {
    type: Object,
    required: true,
  },
  deleting: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['delete'])

// builds the image url for avatar or comment picture
function imageUrl(imagePath) {
  if (!imagePath) return ''
  return imagePath.startsWith('/') ? imagePath : `/uploads/${imagePath}`
}

// first 2 letters of the name when there is no avatar
function initials(author) {
  return author.slice(0, 2).toUpperCase()
}
</script>

<template>
  <div class="comment-preview" :class="{ 'comment-preview--deleting': deleting }">
    <div class="comment-preview__avatar" :style="{ background: comment.avatarColor }" aria-hidden="true">
      <img v-if="comment.avatarPath" :src="imageUrl(comment.avatarPath)" alt="" />
      <span v-else>{{ initials(comment.author) }}</span>
    </div>
    <div class="comment-preview__body">
      <div class="comment-preview__header">
        <strong>{{ comment.author }}</strong>

        <!-- delete menu, only on my own comments -->
        <CommentMenu v-if="comment.own" :disabled="deleting" @remove="emit('delete')" />
      </div>
      <p v-if="comment.content">{{ comment.content }}</p>
      <img
        v-if="comment.imagePath"
        class="comment-preview__image"
        :src="imageUrl(comment.imagePath)"
        alt=""
        loading="lazy"
      />
    </div>
  </div>
</template>

<style scoped>
.comment-preview {
  display: grid;
  grid-template-columns: 2.25rem minmax(0, 1fr);
  align-items: start;
  gap: var(--space-3);
  margin-top: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-small);
  background: var(--color-input);
}

.comment-preview__avatar {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  aspect-ratio: 1;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  background: var(--color-input);
  color: var(--color-text);
  font-size: 0.8125rem;
  font-weight: 700;
}

.comment-preview__avatar img {
  width: 100%;
  height: 100%;
  border-radius: inherit;
  object-fit: cover;
  object-position: center;
}

.comment-preview__body {
  min-width: 0;
}

.comment-preview strong,
.comment-preview p {
  margin: 0;
}

.comment-preview__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.comment-preview--deleting {
  opacity: 0.6;
}

.comment-preview strong {
  color: var(--color-text);
  font-size: 0.875rem;
  font-weight: 500;
}

.comment-preview p {
  color: var(--color-text-soft);
  font-size: 0.875rem;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.comment-preview__image {
  display: block;
  max-width: 12rem;
  max-height: 12rem;
  margin-top: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  object-fit: cover;
}

</style>