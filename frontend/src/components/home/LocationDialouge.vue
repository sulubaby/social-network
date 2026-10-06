<script setup>
defineProps({
    show: {
        type: Boolean,
        default: false
    },

    display: {
        type: String,
        default: ''
    },

    embedUrl: {
        type: String,
        default: ''
    },

    externalUrl: {
        type: String,
        default: ''
    }
});

const emit = defineEmits(['close']);

function close() {
    emit('close');
}
</script>

<template>
    <Teleport to="body">
        <div v-if="show" class="location-dialog-overlay" @click.self="close">
            <div class="location-dialog">
                <header class="location-dialog-header">
                    <span>{{ display }}</span>

                    <button type="button" class="dialog-close" @click="close">
                        ✕
                    </button>
                </header>

                <iframe v-if="embedUrl" class="location-dialog-map" :src="embedUrl" loading="lazy"
                    referrerpolicy="no-referrer-when-downgrade"></iframe>

                <a :href="externalUrl" target="_blank" rel="noopener noreferrer"
                    class="location-dialog-external">
                    Open in Google Maps
                </a>
            </div>
        </div>
    </Teleport>
</template>

<style scoped>
.location-dialog-overlay {
    position: fixed;
    inset: 0;
    z-index: 1000;

    display: flex;
    align-items: center;
    justify-content: center;

    padding: 20px;

    background: rgba(0, 0, 0, 0.5);
}

.location-dialog {
    width: 100%;
    max-width: 600px;

    display: flex;
    flex-direction: column;

    border: 2px solid var(--main-color);
    border-radius: 8px;
    overflow: hidden;

    background: var(--bg-color);
    box-shadow: 7px 7px var(--main-color);
}

.location-dialog-header {
    display: flex;
    align-items: center;
    justify-content: space-between;

    padding: 14px 16px;

    border-bottom: 2px solid var(--main-color);

    color: var(--main-color);

    font-family: "Liter", serif;
    font-size: 14px;
    font-weight: 600;
}

.dialog-close {
    padding: 2px 6px;

    border: none;
    background: transparent;

    color: var(--font-color-sub);

    font-size: 14px;
    font-weight: 600;
}

.dialog-close:hover {
    color: var(--main-color);
}

.location-dialog-map {
    width: 100%;
    height: 350px;
    border: none;
}

.location-dialog-external {
    display: block;

    padding: 12px 16px;

    border-top: 2px solid var(--main-color);

    background: var(--input-focus);

    color: white;
    text-align: center;
    text-decoration: none;

    font-family: "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 600;
}

.location-dialog-external:hover {
    opacity: 0.9;
}

@media (max-width: 650px) {
    .location-dialog-map {
        height: 250px;
    }
}
</style>