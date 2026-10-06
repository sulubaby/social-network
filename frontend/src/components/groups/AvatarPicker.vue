<script setup>
import { ref, watch } from 'vue';

const props = defineProps({
    modelValue: {
        type: File,
        default: null
    }
});

const emit = defineEmits(['update:modelValue']);

const fileInput = ref(null);
const preview = ref('');

function openFilePicker() {
    fileInput.value?.click();
}

function handleFileChange(event) {
    const file = event.target.files?.[0];

    if (!file) {
        return;
    }

    const allowedTypes = [
        'image/png',
        'image/jpeg',
        'image/gif'
    ];

    const extension = file.name
        .split('.')
        .pop()
        .toLowerCase();

    if (
        !allowedTypes.includes(file.type) ||
        !['png', 'jpg', 'jpeg', 'gif'].includes(extension)
    ) {
        event.target.value = '';
        emit('update:modelValue', null);
        return;
    }

    if (preview.value) {
        URL.revokeObjectURL(preview.value);
    }

    preview.value = URL.createObjectURL(file);

    emit('update:modelValue', file);
}

function removeImage() {
    if (preview.value) {
        URL.revokeObjectURL(preview.value);
    }

    preview.value = '';
    emit('update:modelValue', null);

    if (fileInput.value) {
        fileInput.value.value = '';
    }
}

watch(
    () => props.modelValue,
    file => {
        if (!file) {
            preview.value = '';
        }
    }
);
</script>

<template>
    <div class="add-picture">
        <input ref="fileInput" type="file" accept=".png,.jpg,.jpeg,.gif" hidden @change="handleFileChange">

        <div v-if="!preview" class="picture-upload" @click="openFilePicker">
            <div class="upload-icon">+</div>
            <strong>Add group photo</strong>
            <span>Click to choose an image</span>
        </div>

        <div v-else class="picture-preview">
            <img :src="preview" alt="Group avatar preview">

            <div class="picture-actions">
                <button type="button" @click="openFilePicker">
                    Change
                </button>

                <button type="button" @click="removeImage">
                    Remove
                </button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.add-picture {
    width: 100%;
}

.picture-upload {
    min-height: 180px;
    border: 2px dashed var(--main-color);
    border-radius: 10px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    cursor: pointer;
    transition: 0.2s;
}

.picture-upload:hover {
    background: var(--input-focus);
}

.upload-icon {
    width: 45px;
    height: 45px;
    border-radius: 50%;
    background: var(--main-color);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 28px;
    font-weight: bold;
}

.picture-upload span {
    font-size: 12px;
    opacity: 0.7;
}

.picture-preview {
    position: relative;
    width: 100%;
}

.picture-preview img {
    width: 100%;
    max-height: 260px;
    object-fit: contain;
    border-radius: 10px;
    display: block;
}

.picture-actions {
    display: flex;
    gap: 8px;
    margin-top: 10px;
}

.picture-actions button {
    border: none;
    padding: 8px 14px;
    border-radius: 6px;
    cursor: pointer;
    font-weight: bold;
}
</style>