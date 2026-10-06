<script setup>
import { onMounted, ref } from 'vue';

const emit = defineEmits(['changeTab']);

const props = defineProps({
    type: {
        type: String,
        default: 'personal'
    }
});

const activeTab = ref('about');

let tabs = [];

if (props.type === 'personal') {
    tabs = [
        'posts',
        'friends',
        'groups',
        'following',
        'followers',
        'about'
    ];
} else {
    tabs = [
        'posts',
        'following',
        'followers',
        'about'
    ];
}

function selectTab(tab) {
    activeTab.value = tab;
    emit('changeTab', tab);
}

onMounted(() => {
    selectTab('about');
});
</script>

<template>
    <nav class="profile-tabs">
        <button
            v-for="tab in tabs"
            :key="tab"
            type="button"
            :class="{ active: activeTab === tab }"
            @click="selectTab(tab)"
        >
            {{ tab }}
        </button>
    </nav>
</template>

<style scoped>
.profile-tabs {
    position: sticky;
    top: 64px;
    z-index: 50;
    display: flex;
    margin: 25px 0;
    overflow-x: auto;
    border: 2px solid var(--main-color);
    border-radius: 6px;
    background: var(--bg-color);
    box-shadow: 5px 5px var(--main-color);
}

.profile-tabs button {
    flex: 1;
    min-width: 105px;
    padding: 15px 18px;
    border: 0;
    border-right: 2px solid var(--main-color);
    background: transparent;
    color: var(--font-color-sub);
    text-align: center;
    font-family: "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 600;
    cursor: pointer;
}

.profile-tabs button:last-child {
    border-right: 0;
}

.profile-tabs button:hover {
    background: var(--page-background);
    color: var(--main-color);
}

.profile-tabs button.active {
    background: var(--input-focus);
    color: white;
}

@media (max-width: 800px) {
    .profile-tabs {
        top: 0;
    }
}
</style>

