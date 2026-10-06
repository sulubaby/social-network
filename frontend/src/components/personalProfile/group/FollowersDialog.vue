```vue
<script setup>
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import FollowersDialog from '../personalProfile/group/FollowersDialog.vue';

const router = useRouter();

const props = defineProps({
    followers: {
        type: Object,
        default: () => ({})
    },
    type: {
        type: String,
        default: 'followers'
    },
    targetId: {
        type: [String, Number],
        default: null
    }
});

const showDialog = ref(false);

const followerList = computed(() => {
    return Object.entries(props.followers).map(([id, follower]) => ({
        id,
        ...follower
    }));
});

const sectionTitle = computed(() => {
    if (props.type === 'following') return 'Following';
    if (props.type === 'friends') return 'Friends';
    return 'Followers';
});

async function takeToProfile(id) {
    await router.push(`/user?id=${id}`);
    window.location.reload();
}

function openDialog() {
    showDialog.value = true;
}

function closeDialog() {
    showDialog.value = false;
}
</script>

<template>
    <section class="followers-section">
        <div class="section-heading">
            <div class="heading-row">
                <div class="heading-title">
                    <span class="heading-accent"></span>

                    <div>
                        <p class="eyebrow">SOCIAL</p>
                        <h2>{{ sectionTitle }}</h2>
                    </div>
                </div>

                <button
                    v-if="followerList.length"
                    type="button"
                    class="show-all-btn"
                    @click="openDialog"
                >
                    <span>Show all</span>
                    <span class="arrow">→</span>
                </button>
            </div>
        </div>

        <div class="followers-card">
            <div v-if="followerList.length" class="followers-grid">
                <article
                    v-for="follower in followerList"
                    :key="follower.id"
                    class="follower-card"
                    @click="takeToProfile(follower.id)"
                >
                    <div class="avatar-container">
                        <img
                            :src="
                                follower.Avatar
                                    ? `/uploads/${follower.Avatar}`
                                    : '/default-avatar.png'
                            "
                            :alt="`${follower.FirstName} ${follower.LastName}`"
                            class="follower-avatar"
                        >
                    </div>

                    <div class="follower-info">
                        <p class="follower-name">
                            {{ follower.FirstName }} {{ follower.LastName }}
                        </p>
                        <span class="profile-label">VIEW PROFILE</span>
                    </div>

                    <span class="card-arrow">↗</span>
                </article>
            </div>

            <div v-else class="empty-state">
                <div class="empty-icon">◎</div>
                <p class="empty-title">No {{ type }} yet</p>
                <p class="empty-text">
                    People will appear here when you have some.
                </p>
            </div>
        </div>

        <FollowersDialog
            v-if="showDialog"
            :type="type"
            :target-id="targetId"
            @close="closeDialog"
            @navigate="takeToProfile"
        />
    </section>
</template>

<style scoped>
.followers-section {
    width: 100%;
    scroll-margin-top: 100px;
}

.section-heading {
    margin-bottom: 20px;
}

.heading-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
}

.heading-title {
    display: flex;
    align-items: center;
    gap: 12px;
}

.heading-accent {
    width: 5px;
    height: 44px;
    flex: 0 0 auto;
    border: 1px solid var(--input-focus);
    border-radius: 2px;
    background: var(--input-focus);
    box-shadow: 3px 3px var(--main-color);
}

.eyebrow {
    margin: 0 0 5px;
    color: var(--input-focus);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 2px;
}

h2 {
    margin: 0;
    color: var(--font-color);
    font-family: "Liter", serif;
    font-size: clamp(22px, 4vw, 30px);
    line-height: 1;
}

.show-all-btn {
    display: flex;
    align-items: center;
    gap: 9px;
    flex: 0 0 auto;
    padding: 9px 13px;
    border: 2px solid var(--main-color);
    border-radius: 6px;
    background: var(--bg-color);
    box-shadow: 4px 4px var(--main-color);
    color: var(--font-color);
    font-family: "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    cursor: pointer;
    transition:
        transform 0.15s ease,
        box-shadow 0.15s ease,
        border-color 0.15s ease;
}

.show-all-btn:hover {
    border-color: var(--input-focus);
    transform: translate(-2px, -2px);
    box-shadow: 6px 6px var(--input-focus);
}

.show-all-btn:active {
    transform: translate(2px, 2px);
    box-shadow: 1px 1px var(--main-color);
}

.arrow {
    color: var(--input-focus);
    font-size: 15px;
    transition: transform 0.15s ease;
}

.show-all-btn:hover .arrow {
    transform: translateX(3px);
}

.followers-card {
    width: 100%;
    padding: clamp(14px, 2.5vw, 22px);
    border: 2px solid var(--main-color);
    border-radius: 8px;
    background: var(--bg-color);
    box-shadow: 7px 7px var(--main-color);
    box-sizing: border-box;
}

.followers-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
    gap: 15px;
}

.follower-card {
    position: relative;
    display: flex;
    align-items: center;
    gap: 13px;
    min-width: 0;
    padding: 13px;
    border: 2px solid var(--main-color);
    border-radius: 7px;
    background: var(--bg-color);
    box-shadow: 4px 4px var(--main-color);
    box-sizing: border-box;
    cursor: pointer;
    overflow: hidden;
    transition:
        transform 0.15s ease,
        box-shadow 0.15s ease,
        border-color 0.15s ease;
}

.follower-card::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    width: 5px;
    height: 100%;
    background: var(--input-focus);
    transform: scaleY(0);
    transform-origin: bottom;
    transition: transform 0.15s ease;
}

.follower-card:hover {
    border-color: var(--input-focus);
    transform: translate(-3px, -3px);
    box-shadow: 7px 7px var(--input-focus);
}

.follower-card:hover::before {
    transform: scaleY(1);
}

.follower-card:active {
    transform: translate(0, 0);
    box-shadow: 1px 1px var(--main-color);
}

.avatar-container {
    flex: 0 0 auto;
    padding: 3px;
    border: 2px solid var(--input-focus);
    border-radius: 50%;
    background: var(--bg-color);
    box-shadow: 2px 2px var(--main-color);
}

.follower-avatar {
    display: block;
    width: clamp(48px, 6vw, 58px);
    height: clamp(48px, 6vw, 58px);
    border: 2px solid var(--main-color);
    border-radius: 50%;
    background: var(--bg-color);
    object-fit: cover;
    box-sizing: border-box;
}

.follower-info {
    min-width: 0;
    flex: 1;
}

.follower-name {
    margin: 0;
    color: var(--font-color);
    font-family: "Hedvig Letters Sans", sans-serif;
    font-size: clamp(12px, 1.8vw, 15px);
    font-weight: 700;
    line-height: 1.35;
    overflow-wrap: anywhere;
}

.profile-label {
    display: block;
    margin-top: 5px;
    color: var(--input-focus);
    font-family: "JetBrains Mono", monospace;
    font-size: 8px;
    font-weight: 700;
    letter-spacing: 0.7px;
    opacity: 0;
    transform: translateY(3px);
    transition:
        opacity 0.15s ease,
        transform 0.15s ease;
}

.follower-card:hover .profile-label {
    opacity: 1;
    transform: translateY(0);
}

.card-arrow {
    flex: 0 0 auto;
    align-self: flex-start;
    color: var(--input-focus);
    font-family: "JetBrains Mono", monospace;
    font-size: 16px;
    font-weight: 700;
    opacity: 0;
    transform: translate(-3px, 3px);
    transition:
        opacity 0.15s ease,
        transform 0.15s ease;
}

.follower-card:hover .card-arrow {
    opacity: 1;
    transform: translate(0, 0);
}

.empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 150px;
    padding: 25px 15px;
    border: 2px dashed var(--main-color);
    border-radius: 7px;
    background: var(--bg-color);
    text-align: center;
}

.empty-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    margin-bottom: 9px;
    border: 2px solid var(--input-focus);
    border-radius: 50%;
    box-shadow: 3px 3px var(--main-color);
    color: var(--input-focus);
    font-family: "Liter", serif;
    font-size: 25px;
}

.empty-title {
    margin: 0;
    color: var(--font-color);
    font-family: "Hedvig Letters Sans", sans-serif;
    font-size: 14px;
    font-weight: 700;
}

.empty-text {
    max-width: 300px;
    margin: 5px 0 0;
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: 9px;
    line-height: 1.5;
}

@media (max-width: 700px) {
    .followers-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .follower-card {
        padding: 11px;
    }

    .profile-label,
    .card-arrow {
        display: none;
    }
}

@media (max-width: 500px) {
    .heading-row {
        align-items: flex-end;
    }

    .heading-accent {
        height: 37px;
    }

    .show-all-btn {
        padding: 8px 10px;
    }

    .show-all-btn .arrow {
        display: none;
    }

    .followers-card {
        padding: 12px;
        box-shadow: 5px 5px var(--main-color);
    }

    .followers-grid {
        grid-template-columns: 1fr;
        gap: 11px;
    }

    .follower-card {
        padding: 12px;
    }

    .follower-avatar {
        width: 50px;
        height: 50px;
    }
}

@media (max-width: 350px) {
    .heading-title {
        gap: 8px;
    }

    .heading-accent {
        width: 4px;
    }

    .followers-card {
        padding: 9px;
    }
}
</style>
