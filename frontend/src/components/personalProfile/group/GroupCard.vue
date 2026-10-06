<script setup>
defineProps({
    card: {
        type: Object,
        required: true
    }
});

const emit = defineEmits(['edit', 'delete']);
</script>

<template>
    <div :id="card.id" class="group-card">
        <strong>{{ card.name }}</strong>

        <span class="members">
            {{ card.firstName }}
            <template v-if="card.lastName">
                &amp; {{ card.lastName }}
            </template>
            <template v-if="card.extraCount">
                +{{ card.extraCount }} more
            </template>
        </span>

        <div class="card-actions">
            <button type="button" class="action-button edit-button" @click="emit('edit')">
                Edit
            </button>

            <button type="button" class="action-button delete-button" @click="emit('delete')">
                Delete
            </button>
        </div>
    </div>
</template>

<style scoped>
* {
    box-sizing: border-box;
}

.group-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 16px;
    border: 3px solid #1a1a1a;
    border-radius: 12px;
    background: #fff;
    text-align: left;
    box-shadow: 5px 5px 0 #1a1a1a;
}

.group-card strong {
    font-size: 14px;
    color: #1a1a1a;
}

.group-card .members {
    font-family: "JetBrains Mono", monospace;
    font-size: 11px;
    color: #55554f;
}

.card-actions {
    display: flex;
    gap: 8px;
    margin-top: 8px;
}

.action-button {
    flex: 1;
    padding: 6px 10px;
    border: 2px solid #1a1a1a;
    border-radius: 6px;
    font-family: "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 800;
    cursor: pointer;
    box-shadow: 3px 3px 0 #1a1a1a;
    transition: transform .1s ease, box-shadow .1s ease;
}

.action-button:hover {
    transform: translate(-1px, -1px);
    box-shadow: 4px 4px 0 #1a1a1a;
}

.action-button:active {
    transform: translate(2px, 2px);
    box-shadow: 1px 1px 0 #1a1a1a;
}

.edit-button {
    background: #2563eb;
    color: #fff;
}

.delete-button {
    background: #ef4444;
    color: #fff;
}
</style>
