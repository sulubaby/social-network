<script setup>
defineProps({
    groups: {
        type: Array,
        default: () => []
    }
});

defineEmits(['edit', 'remove']);
</script>

<template>
    <section class="groups-section">
        <div class="heading">
            <div>
                <div class="title-row">
                    <div class="section-icon">♥</div>
                    <div>
                        <h3>My Groups</h3>
                        <span>{{ groups.length }} groups</span>
                    </div>
                </div>
            </div>
        </div>

        <div v-if="groups.length" class="groups-list">
            <article
                v-for="group in groups"
                :key="group.id"
                class="group"
            >
                <div class="group-icon">
                    {{ group.name.charAt(0).toUpperCase() }}
                </div>

                <div class="info">
                    <h4>{{ group.name }}</h4>
                    <p>
                        <span class="member-icon">●</span>
                        {{ group.members.length }} members
                    </p>
                </div>

                <div class="actions">
                    <button
                        class="edit"
                        @click="$emit('edit', group)"
                    >
                        Edit
                    </button>

                    <button
                        class="remove"
                        @click="$emit('remove', group.id)"
                    >
                        Remove
                    </button>
                </div>
            </article>
        </div>

        <div v-else class="empty">
            <div class="empty-icon">+</div>
            <strong>No groups yet</strong>
            <span>Create your first friend group.</span>
        </div>
    </section>
</template>

<style scoped>
.groups-section {
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    overflow: hidden;
    box-shadow: 0 3px 12px rgba(15, 23, 42, .05);
}

.heading {
    padding: 18px 20px;
    background: #f8fafc;
    border-bottom: 1px solid #e2e8f0;
}

.title-row {
    display: flex;
    align-items: center;
    gap: 12px;
}

.section-icon {
    width: 38px;
    height: 38px;
    display: grid;
    place-items: center;
    border-radius: 10px;
    background: #dbeafe;
    color: #2563eb;
    font-size: 15px;
}

.heading h3 {
    margin: 0 0 3px;
    color: #172033;
    font-size: 17px;
}

.heading span {
    color: #64748b;
    font-size: 12px;
}

.groups-list {
    display: flex;
    flex-direction: column;
    padding: 0 20px;
}

.group {
    display: flex;
    align-items: center;
    gap: 13px;
    padding: 15px 0;
    border-bottom: 1px solid #eef2f7;
}

.group:last-child {
    border-bottom: 0;
}

.group-icon {
    width: 46px;
    height: 46px;
    display: grid;
    place-items: center;
    flex-shrink: 0;
    border-radius: 11px;
    background: #eff6ff;
    border: 1px solid #dbeafe;
    color: #2563eb;
    font-size: 17px;
    font-weight: 700;
}

.info {
    flex: 1;
    min-width: 0;
}

.info h4 {
    margin: 0 0 5px;
    color: #1e293b;
    font-size: 14px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.info p {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    color: #64748b;
    font-size: 12px;
}

.member-icon {
    color: #94a3b8;
    font-size: 7px;
}

.actions {
    display: flex;
    gap: 6px;
}

.actions button {
    padding: 7px 10px;
    border-radius: 7px;
    font-size: 11px;
    font-weight: 600;
    transition: .2s ease;
}

.edit {
    border: 1px solid #bfdbfe;
    background: #eff6ff;
    color: #2563eb;
}

.edit:hover {
    background: #dbeafe;
    border-color: #93c5fd;
}

.remove {
    border: 1px solid #fecaca;
    background: #fff1f2;
    color: #dc2626;
}

.remove:hover {
    background: #fee2e2;
    border-color: #fca5a5;
}

.empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 45px 20px;
    color: #64748b;
}

.empty-icon {
    width: 44px;
    height: 44px;
    display: grid;
    place-items: center;
    margin-bottom: 10px;
    border-radius: 50%;
    background: #eff6ff;
    color: #2563eb;
    font-size: 22px;
}

.empty strong {
    margin-bottom: 4px;
    color: #334155;
    font-size: 14px;
}

.empty span {
    font-size: 12px;
}

@media (max-width: 500px) {
    .group {
        align-items: flex-start;
    }

    .actions {
        flex-direction: column;
    }

    .actions button {
        min-width: 58px;
    }
}
</style>
