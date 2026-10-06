<template>
    <section class="profile-section">
        <div class="section-heading">
            <div>
                <p class="eyebrow">ACTIVITY</p>
                <h2>Posts</h2>
            </div>

            <button class="primary-button">Create post</button>
        </div>

        <article class="post-card">
            <div class="post-header">
                <div class="small-avatar">A</div>

                <div>
                    <strong>Alex Anderson</strong>
                    <span>@alexanderson · 2h</span>
                </div>
            </div>

            <p class="post-text">
                Just finished working on something I've been thinking
                about for a while. There is something really satisfying
                about finally seeing an idea become real.
            </p>

            <div class="post-actions">
                <button>♡ 24</button>
                <button>○ 8 comments</button>
                <button>↗ Share</button>
            </div>
        </article>

        <article class="post-card">
            <div class="post-header">
                <div class="small-avatar">A</div>

                <div>
                    <strong>Alex Anderson</strong>
                    <span>@alexanderson · 1d</span>
                </div>
            </div>

            <p class="post-text">
                Sometimes keeping things simple is the hardest part.
            </p>

            <div class="post-actions">
                <button>♡ 41</button>
                <button>○ 12 comments</button>
                <button>↗ Share</button>
            </div>
        </article>
    </section>
</template>

<style scoped>
.profile-section {
    scroll-margin-top: 100px;
    margin-bottom: 35px;
}

.section-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 18px;
}

.eyebrow {
    margin: 0 0 5px;
    color: var(--input-focus);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
    font-weight: 600;
    letter-spacing: 2px;
}

h2 {
    margin: 0;
    font-family: "Liter", serif;
    font-size: 29px;
}

.primary-button {
    padding: 10px 15px;
    border: 2px solid var(--main-color);
    border-radius: 5px;
    background: var(--input-focus);
    box-shadow: 4px 4px var(--main-color);
    color: white;
    font-family: "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 600;
}

.post-card {
    margin-bottom: 20px;
    padding: 25px;
    border: 2px solid var(--main-color);
    border-radius: 7px;
    background: var(--bg-color);
    box-shadow: 5px 5px var(--main-color);
}

.post-header {
    display: flex;
    align-items: center;
    gap: 12px;
}

.small-avatar {
    width: 42px;
    height: 42px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px solid var(--main-color);
    border-radius: 50%;
    background: var(--input-focus);
    color: white;
    font-family: "Liter", serif;
    font-size: 18px;
}

.post-header strong,
.post-header span {
    display: block;
}

.post-header strong {
    font-size: 12px;
}

.post-header span {
    margin-top: 3px;
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
}

.post-text {
    margin: 22px 0;
    color: var(--font-color);
    font-size: 14px;
    line-height: 1.7;
}

.post-actions {
    display: flex;
    gap: 10px;
    padding-top: 15px;
    border-top: 2px solid var(--page-background);
}

.post-actions button {
    border: 0;
    background: transparent;
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
}

.post-actions button:hover {
    color: var(--input-focus);
}
</style>