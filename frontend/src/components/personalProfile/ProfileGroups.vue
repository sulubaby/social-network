<template>
    <section class="profile-section">
        <div class="section-heading">
            <div>
                <p class="eyebrow">COMMUNITIES</p>
                <h2>Groups</h2>
            </div>
        </div>

        <div class="groups-grid">
            <article class="group-card">
                <div class="group-icon">G</div>
                <div>
                    <h3>Developers Hub</h3>
                    <p>2.4k members</p>
                </div>
                <button>View group</button>
            </article>

            <article class="group-card">
                <div class="group-icon">D</div>
                <div>
                    <h3>Design & Creative</h3>
                    <p>1.8k members</p>
                </div>
                <button>View group</button>
            </article>

            <article class="group-card">
                <div class="group-icon">P</div>
                <div>
                    <h3>Photography</h3>
                    <p>936 members</p>
                </div>
                <button>View group</button>
            </article>
        </div>
    </section>
</template>

<style scoped>
.profile-section {
    scroll-margin-top: 100px;
}

.section-heading {
    margin-bottom: 20px;
}

.eyebrow {
    margin: 0 0 5px;
    color: var(--input-focus);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
    letter-spacing: 2px;
}

h2 {
    margin: 0;
    font-family: "Liter", serif;
    font-size: 29px;
}

.groups-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 18px;
}

.group-card {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 15px;
    padding: 20px;
    border: 2px solid var(--main-color);
    border-radius: 7px;
    background: var(--bg-color);
    box-shadow: 4px 4px var(--main-color);
}

.group-icon {
    width: 55px;
    height: 55px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px solid var(--main-color);
    border-radius: 6px;
    background: var(--input-focus);
    color: white;
    font-family: "Liter", serif;
    font-size: 24px;
}

.group-card h3 {
    margin: 3px 0 5px;
    font-size: 13px;
}

.group-card p {
    margin: 0;
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: 8px;
}

.group-card button {
    grid-column: 1 / -1;
    padding: 9px;
    border: 2px solid var(--main-color);
    border-radius: 4px;
    background: var(--page-background);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
    font-weight: 600;
}

@media (max-width: 800px) {
    .groups-grid {
        grid-template-columns: 1fr;
    }
}
</style>