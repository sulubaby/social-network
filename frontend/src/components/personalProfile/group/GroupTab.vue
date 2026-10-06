<script setup>
import { ref, onMounted } from 'vue';
import GroupModal from './GroupModal.vue';
import { getPostGroups } from '@/api/common/friends.js';
import ExistingGroups from './ExistingGroups.vue';

const groups = ref([]);
const loading = ref(true);

const showModal = ref(false);

async function loadGroups() {
    loading.value = true;
    try {
        const result = await getPostGroups();
        groups.value = result || [];
        console.log(result)
    } catch (error) {
        console.error(error);
        groups.value = [];
    } finally {
        loading.value = false;
    }
}

function openCreateModal() {
    showModal.value = true;
}

function closeModal() {
    showModal.value = false;
}

function handleSave() {
    closeModal();
    loadGroups();
}

onMounted(loadGroups);
</script>

<template>
    <div class="groups-tab">
        <div class="tab-header">
            <h2>Groups</h2>
            <button type="button" class="create-button" @click="openCreateModal">
                + New Group
            </button>
        </div>

        <div v-if="loading" class="state-message">
            Loading groups...
        </div>

        <div v-else-if="!groups.length" class="state-message">
            No groups yet. Create one to get started.
        </div>

        <ExistingGroups
            v-else
            :groups="groups"
            @changed="loadGroups"
        />

        <GroupModal
            v-if="showModal"
            :group="null"
            @close="closeModal"
            @save="handleSave"
        />
    </div>
</template>

<style scoped>
* {
    box-sizing: border-box;
}

.groups-tab {
    padding: 20px 0;
}

.tab-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
}

.tab-header h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 800;
    color: #1a1a1a;
}

.create-button {
    padding: 10px 18px;
    border: 3px solid #1a1a1a;
    border-radius: 8px;
    background: #2563eb;
    color: #ffffff;
    font-family: "JetBrains Mono", monospace;
    font-size: 12px;
    font-weight: 800;
    box-shadow: 4px 4px 0 #1a1a1a;
    transition: transform .1s ease, box-shadow .1s ease;
    cursor: pointer;
}

.create-button:hover {
    transform: translate(-1px, -1px);
    box-shadow: 5px 5px 0 #1a1a1a;
}

.create-button:active {
    transform: translate(2px, 2px);
    box-shadow: 2px 2px 0 #1a1a1a;
}

.state-message {
    padding: 30px;
    text-align: center;
    color: #55554f;
    font-family: "JetBrains Mono", monospace;
    font-size: 13px;
}
</style>