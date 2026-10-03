<script setup>
import { ref } from 'vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'

// inputId makes the input id unique for each post, disabled is true while sending
defineProps({
  inputId: {
    type: String,
    required: true,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

// we dont send the comment from here, we give it to the PostCard with a submit event
const emit = defineEmits(['submit'])

// only small jpeg/png/gif pictures, same rules as the backend
const MAX_FILE_SIZE = 5 * 1024 * 1024
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/gif']

const content = ref('')
const selectedImage = ref(null)
const previewUrl = ref('')
const imageError = ref('')
const fileInput = ref(null)

// PostCard calls this after the comment is sent to clear the box
function reset() {
  content.value = ''
  removeImage()
}
defineExpose({ reset })

// the picture button clicks the hidden file input
function openFilePicker() {
  fileInput.value?.click()
}

// check the picked picture (type and size) and make a preview of it
function selectImage(event) {
  const file = event.target.files[0] || null

  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }

  if (!file) {
    selectedImage.value = null
    return
  }

  if (!ALLOWED_TYPES.includes(file.type)) {
    selectedImage.value = null
    imageError.value = 'Only GIF, PNG, JPEG or JPG images are allowed.'
    event.target.value = ''
    return
  }

  if (file.size > MAX_FILE_SIZE) {
    selectedImage.value = null
    imageError.value = 'Image must be smaller than 5 MB.'
    event.target.value = ''
    return
  }

  selectedImage.value = file
  imageError.value = ''
  previewUrl.value = URL.createObjectURL(file)
}

// remove the picture and free the preview url
function removeImage() {
  selectedImage.value = null
  imageError.value = ''

  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }

  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

// send the text and/or picture up to the PostCard. empty comments are ignored
function submitComment() {
  const cleanContent = content.value.trim()
  if (!cleanContent && !selectedImage.value) return

  emit('submit', { content: cleanContent, image: selectedImage.value })
}
</script>

<template>
  <form class="comment-input" @submit.prevent="submitComment">
    <!-- small preview of the picked picture -->
    <div v-if="previewUrl" class="comment-input__preview">
      <img :src="previewUrl" alt="Selected image preview" />
      <button type="button" aria-label="Remove image" @click="removeImage">
        <IconGlyph name="close" :size="14" />
      </button>
    </div>

    <p v-if="imageError" class="comment-input__error" role="alert">{{ imageError }}</p>

    <!-- text box, picture button, hidden file input and send button -->
    <div class="comment-input__row">
      <label class="visually-hidden" :for="inputId">Write a comment</label>
      <input
        :id="inputId"
        v-model="content"
        :disabled="disabled"
        maxlength="200"
        placeholder="Write a comment..."
        type="text"
      />

      <button
        type="button"
        class="comment-input__image-button"
        :disabled="disabled"
        aria-label="Add image"
        @click="openFilePicker"
      >
        <IconGlyph name="image" :size="18" />
      </button>

      <input
        ref="fileInput"
        class="file-input"
        type="file"
        accept="image/jpeg,image/png,image/gif"
        @change="selectImage"
      />

      <button type="submit" :disabled="disabled || (!content.trim() && !selectedImage)" aria-label="Send comment">
        <svg class="comment-input__send-icon" viewBox="0 0 24 24" aria-hidden="true">
          <path d="m4 12 16-8-6 16-2-6-8-2Zm8 2 3-3" />
        </svg>
      </button>
    </div>
  </form>
</template>

<style scoped>
.comment-input {
  margin-top: var(--space-3);
}

.comment-input__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--touch-target) var(--touch-target);
  align-items: center;
  padding-left: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-input);
}

.comment-input__row input {
  width: 100%;
  min-width: 0;
  min-height: var(--touch-target);
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text);
}

.comment-input__row input::placeholder {
  color: var(--color-text-faint);
}

.comment-input__row input:disabled {
  cursor: wait;
  opacity: 0.7;
}

.comment-input__image-button,
.comment-input__row button[type='submit'] {
  display: grid;
  min-width: var(--touch-target);
  min-height: var(--touch-target);
  place-items: center;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--color-violet);
  cursor: pointer;
}

.comment-input__image-button:disabled,
.comment-input__row button[type='submit']:disabled {
  color: var(--color-text-faint);
  cursor: not-allowed;
}

.comment-input__image-button svg {
  width: 1.1rem;
}

.comment-input__send-icon {
  width: 1.25rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}

/* stronger than ".comment-input__row input" so the hidden picker stays 1px
   (it used to take the full width and make the whole page scroll sideways) */
.comment-input__row .file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

.comment-input__preview {
  position: relative;
  display: inline-block;
  margin: 0 0 var(--space-2) var(--space-4);
}

.comment-input__preview img {
  display: block;
  max-width: 8rem;
  max-height: 8rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-small);
  object-fit: cover;
}

.comment-input__preview button {
  position: absolute;
  top: -0.4rem;
  right: -0.4rem;
  display: grid;
  width: 1.5rem;
  height: 1.5rem;
  place-items: center;
  border: 0;
  border-radius: 50%;
  background: var(--color-surface-raised);
  color: var(--color-text);
  cursor: pointer;
}

.comment-input__error {
  margin: 0 0 var(--space-2) var(--space-4);
  color: var(--color-coral);
  font-size: 0.75rem;
}
</style>
