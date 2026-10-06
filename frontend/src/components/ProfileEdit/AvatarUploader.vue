<script setup>
import { ref } from 'vue';
import { checkSessionResponse } from '@/helpers/auth/auth';
import { addNotification } from '@/data/notifications';
import { router } from '@/router/router';

const props = defineProps({
    src: {
        type: String,
        default: ''
    }
});

const emit = defineEmits(['change']);

const preview = ref(props.src);
const fileInput = ref(null);
const uploading = ref(false);
const error = ref('');

const allowedTypes = ['image/png', 'image/jpeg', 'image/gif'];
const maxFileSize = 5 * 1024 * 1024;

function triggerUpload() {
    if (!uploading.value) {
        fileInput.value?.click();
    }
}

function validateFile(file) {
    if (!allowedTypes.includes(file.type)) {
        return 'Only JPG, GIF and PNG images are allowed';
    }

    if (file.size > maxFileSize) {
        return 'Image must be smaller than 5MB';
    }

    return '';
}

async function uploadAvatar(file) {
    const formData = new FormData();
    formData.append('avatar', file);

    uploading.value = true;

    try {
        const response = await fetch('/api/profile/avatar', {
            method: 'PATCH',
            credentials: 'include',
            body: formData
        });

        const result = await response.json();

        if (!checkSessionResponse(response)) {
            router.replace('/login');
            return;
        }

        if (!response.ok) {
            throw new Error(result.message || 'Failed to update avatar');
        }

        emit('change', file);

        addNotification(
            result.message || 'Avatar updated successfully',
            'success'
        );
    } catch (err) {
        console.error(err);

        preview.value = props.src;

        addNotification(
            err.message || 'Failed to update avatar',
            'error'
        );
    } finally {
        uploading.value = false;
    }
}

async function onFileChange(event) {
    const file = event.target.files?.[0];

    if (!file) {
        return;
    }

    error.value = '';

    const validationError = validateFile(file);

    if (validationError) {
        error.value = validationError;
        event.target.value = '';
        return;
    }

    const previousPreview = preview.value;

    preview.value = URL.createObjectURL(file);

    try {
        await uploadAvatar(file);
    } catch (err) {
        preview.value = previousPreview;
    }

    event.target.value = '';
}
</script>

<template>
    <div class="avatar-uploader">
        <div class="avatar-preview">
            <img
                v-if="preview"
                :src="preview"
                alt="Profile avatar"
            >
        </div>

        <div class="avatar-actions">
            <button
                type="button"
                class="upload-button"
                :disabled="uploading"
                @click="triggerUpload"
            >
                {{ uploading ? 'Uploading...' : 'Change avatar' }}
            </button>

            <span class="hint">
                JPG, GIF or PNG, up to 5MB
            </span>

            <span
                v-if="error"
                class="error"
            >
                {{ error }}
            </span>
        </div>

        <input
            ref="fileInput"
            type="file"
            accept="image/png, image/jpeg, image/gif"
            class="hidden-input"
            @change="onFileChange"
        >
    </div>
</template>

<style scoped>
.avatar-uploader {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: clamp(12px, 3vw, 20px);
}

.avatar-preview {
    flex-shrink: 0;
    width: clamp(72px, 12vw, 100px);
    height: clamp(72px, 12vw, 100px);
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    border: 3px solid var(--bg-color);
    outline: 2px solid var(--main-color);
    border-radius: 50%;
    background: var(--main-color);
    box-shadow: 4px 4px var(--main-color);
}

.avatar-preview img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.avatar-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
}

.upload-button {
    padding: clamp(8px, 1.5vw, 10px) clamp(12px, 2vw, 16px);
    border: 2px solid var(--main-color);
    border-radius: 5px;
    background: var(--input-focus);
    box-shadow: 3px 3px var(--main-color);
    color: white;
    font-family: "JetBrains Mono", monospace;
    font-size: clamp(8px, 1.4vw, 9px);
    font-weight: 600;
    white-space: nowrap;
    cursor: pointer;
}

.upload-button:hover:not(:disabled) {
    transform: translate(-1px, -1px);
}

.upload-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.hint {
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: clamp(7px, 1.2vw, 8px);
}

.error {
    color: #d9534f;
    font-family: "JetBrains Mono", monospace;
    font-size: clamp(7px, 1.2vw, 8px);
}

.hidden-input {
    display: none;
}

@media (max-width: 400px) {
    .avatar-uploader {
        flex-direction: column;
        align-items: flex-start;
    }
}
</style>
