<script setup>
import { ref } from 'vue'

defineProps({
    show: {
        type: Boolean,
        default: false
    }
})

const emit = defineEmits(['close', 'create'])

const title = ref('')
const description = ref('')
const formError = ref('')

function submitGroup() {
    formError.value = ''
    if (!title.value.trim() || !description.value.trim()) {
        formError.value = 'The title and the description are both required.'
        return
    }

    emit('create', {
        title: title.value,
        description: description.value
    })

    title.value = ''
    description.value = ''
}
</script>

<template>
    <div v-if="show" class="modal-overlay" @click.self="emit('close')">
        <div class="modal">
            <p class="modal-eyebrow">New Community</p>

            <h2>Create Group</h2>

            <form @submit.prevent="submitGroup">
                <div class="form-group">
                    <label for="group-title">
                        Group Title
                    </label>

                    <input id="group-title" v-model="title" type="text" maxlength="45" placeholder="Enter group title" />
                </div>

                <div class="form-group">
                    <label for="group-description">
                        Description
                    </label>

                    <textarea id="group-description" v-model="description" maxlength="500"
                        placeholder="Tell people what this group is about"></textarea>
                </div>

                <p v-if="formError" class="form-error" role="alert">{{ formError }}</p>

                <div class="actions">
                    <button class="cancel-button" type="button" @click="emit('close')">
                        Cancel
                    </button>

                    <button class="create-button" type="submit">
                        Create Group
                    </button>
                </div>
            </form>
        </div>
    </div>
</template>

<style scoped>
.form-error {
    margin: 0 0 1rem;
    color: var(--color-coral);
    font-size: .875rem;
}

.modal-overlay {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    background: rgb(5 6 12 / 75%);
    backdrop-filter: blur(6px);
}

.modal {
    width: min(100%, 460px);
    padding: var(--space-5);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-medium);
    background: var(--color-surface);
    box-shadow: var(--shadow-raised);
}

.modal-eyebrow {
    margin: 0 0 var(--space-1);
    color: var(--color-violet);
    font-family: var(--font-meta);
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 2px;
    text-transform: uppercase;
}

.modal h2 {
    margin: 0 0 var(--space-5);
    color: var(--color-text);
    font-family: var(--font-display);
    font-size: 1.4rem;
}

form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
}

.form-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
}

label {
    color: var(--color-text-muted);
    font-family: var(--font-meta);
    font-size: 0.6875rem;
    font-weight: 500;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}

input,
textarea {
    width: 100%;
    padding: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-small);
    background: var(--color-input);
    color: var(--color-text);
    font-size: 0.875rem;
}

input::placeholder,
textarea::placeholder {
    color: var(--color-text-faint);
}

input:focus,
textarea:focus {
    border-color: var(--color-violet);
    outline: none;
    box-shadow: var(--focus-ring);
}

textarea {
    min-height: 120px;
    resize: vertical;
}

.actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-top: var(--space-2);
}

.cancel-button,
.create-button {
    min-height: var(--touch-target);
    padding: 0 var(--space-4);
    border-radius: var(--radius-small);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
}

.cancel-button {
    border: 1px solid var(--color-border);
    background: var(--color-surface-raised);
    color: var(--color-text-soft);
}

.cancel-button:hover {
    border-color: var(--color-violet);
}

.create-button {
    border: 0;
    background: var(--gradient-action);
    color: #fff;
}

.create-button:hover {
    opacity: 0.9;
}

@media (max-width: 30rem) {
    .modal {
        padding: var(--space-4);
    }

    .actions {
        flex-direction: column-reverse;
    }

    .cancel-button,
    .create-button {
        width: 100%;
    }
}
</style>