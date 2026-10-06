<script setup>
defineProps({
    modelValue: {
        type: Boolean,
        default: false
    }
});

defineEmits(['update:modelValue']);
</script>

<template>
    <div class="hide-comments-row">
        <div class="text-group">
            <p class="option-title">Hide comments</p>
            <p class="option-subtitle">Other people won't be able to comment on this post</p>
        </div>

        <label class="switch">
            <input
                type="checkbox"
                :checked="modelValue"
                @change="$emit('update:modelValue', $event.target.checked)"
            >

            <span class="slider"></span>
        </label>
    </div>
</template>

<style scoped>
.hide-comments-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    padding: 16px 18px;
    border: 2px solid var(--main-color);
    border-radius: 6px;
    background: var(--bg-color);
}

.text-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.option-title {
    margin: 0;
    color: var(--font-color);
    font-weight: 600;
    font-size: 14px;
}

.option-subtitle {
    margin: 0;
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
}

.switch {
    position: relative;
    flex-shrink: 0;
    width: 50px;
    height: 24px;
}

.switch input {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
}

.slider {
    position: absolute;
    inset: 0;
    border: 2px solid var(--main-color);
    border-radius: 20px;
    background: var(--page-background);
    cursor: pointer;
    transition: 0.2s ease;
}

.slider::before {
    position: absolute;
    content: "";
    width: 16px;
    height: 16px;
    left: 2px;
    top: 2px;
    border-radius: 50%;
    background: var(--main-color);
    transition: 0.2s ease;
}

.switch input:checked + .slider {
    background: var(--input-focus);
}

.switch input:checked + .slider::before {
    transform: translateX(26px);
    background: white;
}
</style>
