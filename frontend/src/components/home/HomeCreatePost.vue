<script setup>
</script>

<template>
    <div class="story-container">
        <div class="story-icon">
            <span>+</span>
        </div>

        <div class="story-content">
            <h3>Create story</h3>
            <p>Share a photo, video or write something</p>
        </div>
    </div>
</template>

<style scoped>
.story-container {
    width: 100%;
    min-height: 74px;

    display: flex;
    align-items: center;
    gap: 12px;

    padding: 12px 18px;

    border: 2px solid #d8d8d8;
    border-radius: 12px;

    background: var(--bg-color);

    box-shadow: 0 2px 2px rgba(0, 0, 0, 0.12);

    cursor: pointer;

    transition:
        transform 0.15s,
        box-shadow 0.15s,
        border-color 0.15s;
}

.story-container:hover {
    border-color: var(--main-color);
    transform: translateY(-1px);
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.12);
}

.story-icon {
    flex-shrink: 0;

    width: 42px;
    height: 42px;

    display: flex;
    align-items: center;
    justify-content: center;

    border-radius: 50%;

    background: #e1efff;
    color: var(--input-focus);
}

.story-icon span {
    margin-top: -2px;

    font-family: "Hedvig Letters Sans", sans-serif;
    font-size: 28px;
    font-weight: 400;
    line-height: 1;
}

.story-content {
    min-width: 0;
}

.story-content h3 {
    margin: 0 0 3px;

    color: var(--main-color);

    font-family: "Hedvig Letters Sans", sans-serif;
    font-size: 16px;
    font-weight: 600;
    line-height: 1.2;
}

.story-content p {
    margin: 0;

    color: var(--font-color-sub);

    font-family: "Hedvig Letters Sans", sans-serif;
    font-size: 14px;
    line-height: 1.3;
}

@media (max-width: 650px) {
    .story-container {
        min-height: 68px;
        padding: 10px 14px;
    }

    .story-icon {
        width: 40px;
        height: 40px;
    }

    .story-content h3 {
        font-size: 15px;
    }

    .story-content p {
        font-size: 13px;
    }
}
</style>