<script setup>
defineProps({
    activeTab: String
})

defineEmits(['update:activeTab'])

const tabs = [
    { id: 'personal', label: 'Personal Info' },
    { id: 'additional', label: 'Additional Info' }
]
</script>

<template>
    <nav class="edit-tabs">
        <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            class="tab-button"
            :class="{ active: activeTab === tab.id }"
            @click="$emit('update:activeTab', tab.id)"
        >
            {{ tab.label }}
        </button>
    </nav>
</template>

<style scoped>
.edit-tabs {
    position: sticky;
    top: 64px;
    z-index: 50;
    display: flex;
    margin: clamp(16px, 3vw, 25px) 0;
    overflow-x: auto;
    border: 2px solid var(--main-color);
    border-radius: 6px;
    background: var(--bg-color);
    box-shadow: 5px 5px var(--main-color);
    -webkit-overflow-scrolling: touch;
}

.tab-button {
    flex: 1 1 0;
    min-width: max-content;
    padding: clamp(11px, 2vw, 15px) clamp(12px, 2.5vw, 18px);
    border: none;
    border-right: 2px solid var(--main-color);
    background: transparent;
    color: var(--font-color-sub);
    text-align: center;
    font-family: "JetBrains Mono", monospace;
    font-size: clamp(9px, 1.4vw, 10px);
    font-weight: 600;
    white-space: nowrap;
}

.tab-button:last-child {
    border-right: 0;
}

.tab-button:hover {
    background: var(--page-background);
    color: var(--main-color);
}

.tab-button.active {
    background: var(--input-focus);
    color: white;
}

@media (max-width: 800px) {
    .edit-tabs {
        top: 0;
    }
}
</style>