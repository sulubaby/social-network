<template>
    <div class="private-container">
        <p>This account is private</p>
    </div>
</template>

<style scoped>
.private-container {
    display: flex;
    margin-top: 100px;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: clamp(24px, 5vw, 40px) clamp(16px, 4vw, 24px);
    border: 2px solid var(--main-color);
    border-radius: 7px;
    background: var(--bg-color);
    box-shadow: 5px 5px var(--main-color);
    text-align: center;
}

.icon {
    font-size: clamp(24px, 4vw, 32px);
}

.private-container p {
    margin: 0;
    color: var(--font-color);
    font-family: "JetBrains Mono", monospace;
    font-size: clamp(11px, 1.6vw, 13px);
    font-weight: 600;
}
</style>