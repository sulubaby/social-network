<script setup>
defineProps({
    groupId: {
        type: [Number, String],
        default: null
    },

    groupName: {
        type: String,
        default: ''
    }
});
</script>

<template>
    <a v-if="groupId && groupName" class="group-badge">
        <span class="group-icon">▦</span>
        {{ groupName }}
    </a>
</template>

<style scoped>
.group-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;

    align-self: flex-start;

    margin: 14px 20px 0;
    padding: 5px 10px;

    border: 1px solid var(--main-color);
    border-radius: 5px;

    background: var(--input-focus);
    box-shadow: 2px 2px var(--main-color);

    color: white;
    text-decoration: none;

    font-family: "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.02em;

    transition: opacity 0.15s;
}

.group-badge:hover {
    opacity: 0.85;
}

.group-icon {
    font-size: 11px;
}

@media (max-width: 650px) {
    .group-badge {
        margin: 10px 14px 0;
    }
}
</style>