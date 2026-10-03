<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { router } from '@/router/router.js'
import { useRoute } from 'vue-router'
import {
    getGroup,
    deleteGroupApi,
    getGroupPosts,
    groupJoinRequest,
    undoJoinGroup,
    leaveGroupApi,
    answerGroupInvitation,
} from '@/api/groups/Groups.js'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout.vue'
import GroupActivity from '@/components/groups/GroupActivity.vue'
import GroupPostCard from '@/components/groups/GroupPostCard.vue'
import GroupPostComposer from '@/components/groups/GroupPostComposer.vue'
import IconGlyph from '@/components/layout/IconGlyph.vue'
import GroupChat from '@/components/chat/GroupChat.vue'
import { addNotification } from '@/data/notifications.js'

const route = useRoute()
const groupId = route.params.groupId

const group = ref(null)
const groupPosts = ref([])
const loading = ref(true)
const error = ref(null)
const isDeletingGroup = ref(false)
const deleteError = ref('')
const isJoinPending = ref(false)
const joinError = ref('')
const activeSection = ref('overview')
const isLeaving = ref(false)
const isAnsweringInvite = ref(false)

// group posts paging: 20 at a time, the next page loads when the empty div
// at the bottom of the list gets close to the screen
const GROUP_POSTS_PAGE_SIZE = 20
const groupPostsOffset = ref(0)
const hasMorePosts = ref(false)
const loadingMorePosts = ref(false)
const postsSentinel = ref(null)
let postsObserver

const groupSections = [
    { id: 'overview', label: 'Overview', icon: 'groups' },
    { id: 'activity', label: 'Activity & events', icon: 'calendar' },
    { id: 'chat', label: 'Group chat', icon: 'chat' },
    { id: 'posts', label: 'Posts', icon: 'image' },
]

// the sections a link can open directly, like ?section=chat from a message popup
const requestedSection = groupSections.some(section => section.id === route.query.section)
    ? route.query.section
    : ''

async function loadGroup() {
    try {
        const result = await getGroup(groupId)

        group.value = result.group
        if (group.value?.isMember) {
            activeSection.value = requestedSection || 'posts'
            await loadGroupPosts()
        }
    } catch (err) {
        console.error(err)
        error.value = 'Could not load group'
    } finally {
        loading.value = false
    }

    // the sentinel only exists after the page finished loading
    await nextTick()
    postsObserver?.disconnect()
    observePostsEnd()
}

onMounted(loadGroup)

onBeforeUnmount(() => postsObserver?.disconnect())

// loads one page of group posts. append = true adds the next page at the bottom
async function loadGroupPosts(append = false) {
    if (append && (loadingMorePosts.value || !hasMorePosts.value)) return
    if (append) loadingMorePosts.value = true

    try {
        const result = await getGroupPosts(groupId, {
            limit: GROUP_POSTS_PAGE_SIZE,
            offset: append ? groupPostsOffset.value : 0,
        })
        const nextPosts = result?.posts || []
        groupPosts.value = append ? [...groupPosts.value, ...nextPosts] : nextPosts
        hasMorePosts.value = result?.hasMore === true
        groupPostsOffset.value = typeof result?.nextOffset === 'number'
            ? result.nextOffset
            : groupPostsOffset.value + nextPosts.length
    } finally {
        loadingMorePosts.value = false
    }
}

// infinite scroll for group posts
function observePostsEnd() {
    if (!postsSentinel.value || typeof IntersectionObserver === 'undefined') return

    postsObserver = new IntersectionObserver(([entry]) => {
        if (entry.isIntersecting && activeSection.value === 'posts' && hasMorePosts.value) {
            loadGroupPosts(true)
        }
    }, { rootMargin: '0px 0px 320px' })

    postsObserver.observe(postsSentinel.value)
}

async function deleteGroup() {
    if (isDeletingGroup.value || !window.confirm('Delete this group and all of its content?')) return

    isDeletingGroup.value = true
    deleteError.value = ''
    try {
        const result = await deleteGroupApi(groupId)

        if (result?.status) {
            router.replace('/groups')
        }
    } catch (err) {
        console.error(err)
        deleteError.value = err.message || 'Could not delete group.'
    } finally {
        isDeletingGroup.value = false
    }
}

function addGroupPost(post) {
    groupPosts.value.unshift(post)
    // the new post pushed everything down by one, so the next page starts one later
    groupPostsOffset.value += 1
}

function removeGroupPost(postId) {
    groupPosts.value = groupPosts.value.filter((post) => post.id !== postId)
    groupPostsOffset.value = Math.max(0, groupPostsOffset.value - 1)
}

// leaving: the creator cannot leave (they delete the group instead)
async function leaveGroup() {
    if (isLeaving.value || !window.confirm('Leave this group? You will need to be accepted again to come back.')) return

    isLeaving.value = true
    joinError.value = ''
    try {
        await leaveGroupApi(groupId)
        addNotification(`You left ${group.value.title}`)
        router.replace('/groups')
    } catch (err) {
        joinError.value = err.message || 'Could not leave the group.'
    } finally {
        isLeaving.value = false
    }
}

// someone invited me: join or decline right here
async function answerInvitation(action) {
    if (isAnsweringInvite.value) return

    isAnsweringInvite.value = true
    joinError.value = ''
    try {
        await answerGroupInvitation(groupId, action)
        if (action === 'join') {
            addNotification(`Welcome to ${group.value.title}`)
            loading.value = true
            await loadGroup()
        } else {
            group.value.invitationId = null
        }
    } catch (err) {
        joinError.value = err.message || 'Could not answer the invitation.'
    } finally {
        isAnsweringInvite.value = false
    }
}

async function toggleJoinRequest() {
    if (!group.value || group.value.isMember || isJoinPending.value) return

    isJoinPending.value = true
    joinError.value = ''

    try {
        const result = group.value.isRequested
            ? await undoJoinGroup(groupId)
            : await groupJoinRequest(groupId)

        if (!result?.status) {
            throw new Error('Could not update your join request.')
        }

        group.value.isRequested = !group.value.isRequested
    } catch (err) {
        joinError.value = err.message || 'Could not update your join request.'
    } finally {
        isJoinPending.value = false
    }
}
</script>

<template>
    <AuthenticatedLayout active-page="groups">
    <div class="group-page">
        <div class="group-container">

            <p v-if="loading" class="group-page__state">
                Loading group...
            </p>

            <p v-else-if="error" class="group-page__state group-page__state--error" role="alert">
                {{ error }}
            </p>

            <template v-else-if="group">
                <header class="group-header orbit-surface">
                    <div class="group-header__art" aria-hidden="true">
                        <span class="group-header__art-orbit group-header__art-orbit--outer"></span>
                        <span class="group-header__art-orbit group-header__art-orbit--inner"></span>
                        <span class="group-header__art-star"></span>
                        <span class="group-header__art-label">ORBIT / COMMUNITY</span>
                    </div>

                    <div class="group-header__copy">
                        <p class="orbit-meta">A place to gather</p>
                        <div class="group-header__title-row">
                            <h1>{{ group.title }}</h1>
                            <span class="group-header__badge">Community</span>
                        </div>
                        <p>{{ group.description }}</p>
                        <div class="group-header__signals">
                            <span>
                                <IconGlyph name="groups" :size="17" />
                                {{ group.memberCount }} {{ group.memberCount === 1 ? 'member' : 'members' }}
                            </span>
                            <span>
                                <span class="group-header__signal-dot"></span>
                                Shared space for thoughtful conversations
                            </span>
                        </div>
                    </div>

                    <div class="group-header__aside">
                        <template v-if="!group.isMember && group.invitationId">
                            <p class="group-header__member-state">
                                <IconGlyph name="groups" :size="17" />
                                You were invited to this group
                            </p>
                            <button class="group-header__join" type="button" :disabled="isAnsweringInvite" @click="answerInvitation('join')">
                                <span>{{ isAnsweringInvite ? 'Joining...' : 'Accept invitation' }}</span>
                                <IconGlyph name="check" :size="17" />
                            </button>
                            <button class="group-header__delete" type="button" :disabled="isAnsweringInvite" @click="answerInvitation('decline')">
                                Decline
                            </button>
                        </template>

                        <button
                            v-else-if="!group.isMember"
                            class="group-header__join"
                            type="button"
                            :disabled="isJoinPending"
                            @click="toggleJoinRequest"
                        >
                            <span>{{ isJoinPending ? 'Updating...' : group.isRequested ? 'Request sent' : 'Request to join' }}</span>
                            <IconGlyph :name="group.isRequested ? 'check' : 'arrowRight'" :size="17" />
                        </button>

                        <p v-if="group.isMember" class="group-header__member-state">
                            <IconGlyph name="check" :size="17" />
                            You are part of this orbit
                        </p>

                        <button v-if="group.isCreator" class="group-header__delete" :disabled="isDeletingGroup" @click="deleteGroup">
                            {{ isDeletingGroup ? 'Deleting...' : 'Delete group' }}
                        </button>
                        <button v-else-if="group.isMember" class="group-header__delete" type="button" :disabled="isLeaving" @click="leaveGroup">
                            {{ isLeaving ? 'Leaving...' : 'Leave group' }}
                        </button>
                    </div>
                    <p v-if="joinError || deleteError" class="group-header__error" role="alert">{{ joinError || deleteError }}</p>
                </header>

                <section class="group-membership" aria-label="Group membership">
                    <div>
                        <IconGlyph name="groups" :size="19" />
                        <span>People who make this space what it is</span>
                    </div>
                    <span class="group-membership__count">{{ group.memberCount }} total</span>
                </section>

                <nav v-if="group.isMember" class="group-sections" aria-label="Group sections">
                    <button
                        v-for="section in groupSections"
                        :key="section.id"
                        class="group-section-tab"
                        :class="{ 'group-section-tab--active': activeSection === section.id }"
                        type="button"
                        :aria-current="activeSection === section.id ? 'page' : undefined"
                        @click="activeSection = section.id"
                    >
                        <IconGlyph :name="section.icon" :size="16" />
                        <span>{{ section.label }}</span>
                    </button>
                </nav>

                <section v-if="!group.isMember" class="group-welcome orbit-surface" aria-labelledby="group-welcome-title">
                    <div>
                        <p class="orbit-meta">Before you join</p>
                        <h2 id="group-welcome-title">Bring your perspective into the circle.</h2>
                        <p>
                            This is a member-led space. Read the description, get a feel for the conversation, and request access when it feels like your kind of orbit.
                        </p>
                    </div>
                    <div class="group-welcome__details">
                        <span><IconGlyph name="lock" :size="17" /> Membership is approved</span>
                        <span><IconGlyph name="chat" :size="17" /> Posts and chats unlock after joining</span>
                    </div>
                </section>

                <div v-else class="group-section-panels">
                    <section v-show="activeSection === 'overview'" class="group-section-panel group-overview orbit-surface" aria-labelledby="group-overview-heading">
                        <div>
                            <p class="orbit-meta">Your shared orbit</p>
                            <h2 id="group-overview-heading">A quick read before you explore</h2>
                            <p>{{ group.description }}</p>
                        </div>
                        <div class="group-overview__signals">
                            <div>
                                <span class="group-overview__icon group-overview__icon--mint"><IconGlyph name="groups" :size="18" /></span>
                                <span><strong>{{ group.memberCount }} members</strong><small>A familiar circle to share with</small></span>
                            </div>
                            <div>
                                <span class="group-overview__icon group-overview__icon--violet"><IconGlyph name="calendar" :size="18" /></span>
                                <span><strong>Activity & events</strong><small>Plan meetups and keep everyone in sync</small></span>
                            </div>
                            <div>
                                <span class="group-overview__icon group-overview__icon--coral"><IconGlyph name="chat" :size="18" /></span>
                                <span><strong>Member conversations</strong><small>Posts and chat open after you join</small></span>
                            </div>
                        </div>
                    </section>

                    <section v-show="activeSection === 'activity'" class="group-section-panel" aria-labelledby="group-activity-heading">
                        <h2 id="group-activity-heading" class="visually-hidden">Group activity and events</h2>
                        <GroupActivity :group-id="groupId" />
                    </section>

                    <section v-show="activeSection === 'chat'" class="group-section-panel" aria-labelledby="group-chat-section-heading">
                        <h2 id="group-chat-section-heading" class="visually-hidden">Group chat</h2>
                        <GroupChat :group-id="groupId" :active="activeSection === 'chat'" />
                    </section>

                    <section v-show="activeSection === 'posts'" class="group-section-panel group-feed orbit-surface" aria-labelledby="group-posts-heading">
                        <div class="group-feed__heading">
                            <div>
                                <p class="orbit-meta">Conversation</p>
                                <h2 id="group-posts-heading">Group posts</h2>
                            </div>
                            <span>{{ groupPosts.length }}{{ hasMorePosts ? '+' : '' }} {{ groupPosts.length === 1 ? 'post' : 'posts' }}</span>
                        </div>
                        <GroupPostComposer :group-id="groupId" @post-created="addGroupPost" />
                        <div v-if="groupPosts.length" class="group-posts">
                            <GroupPostCard
                                v-for="post in groupPosts"
                                :key="post.id"
                                :group-id="groupId"
                                :post="post"
                                @post-deleted="removeGroupPost"
                            />
                        </div>
                        <p v-else class="group-feed__state">No posts yet. Start the group conversation.</p>
                        <div ref="postsSentinel" aria-hidden="true"></div>
                        <p v-if="loadingMorePosts" class="group-feed__state" role="status">Loading more posts...</p>
                    </section>
                </div>
            </template>

        </div>
    </div>
    </AuthenticatedLayout>
</template>

<style scoped>
.group-page {
    width: 100%;
    padding: 0;
    background: var(--color-background);
    color: var(--color-text);
}

.group-container {
    display: grid;
    width: 100%;
    max-width: 64rem;
    margin: 0 auto;
    gap: var(--space-5);
}

.group-header {
    position: relative;
    display: grid;
    gap: var(--space-4);
    overflow: hidden;
    padding: var(--space-4);
    background:
        linear-gradient(135deg, rgb(var(--rgb-surface) / 98%), rgb(var(--rgb-background) / 98%)),
        var(--color-surface);
}

.group-header::after {
    position: absolute;
    right: -6rem;
    bottom: -7rem;
    width: 16rem;
    height: 16rem;
    border: 1px solid rgb(var(--rgb-violet) / 22%);
    border-radius: 50%;
    content: '';
    pointer-events: none;
}

.group-header__art {
    position: relative;
    min-height: 8rem;
    overflow: hidden;
    border: 1px solid rgb(var(--rgb-violet) / 28%);
    border-radius: var(--radius-small);
    background:
        radial-gradient(circle at 18% 26%, rgb(var(--rgb-coral) / 82%) 0 0.25rem, transparent 0.3rem),
        radial-gradient(circle at 78% 68%, rgb(var(--rgb-mint) / 75%) 0 0.2rem, transparent 0.25rem),
        linear-gradient(145deg, rgb(var(--rgb-violet) / 22%), rgb(var(--rgb-coral) / 10%));
}

.group-header__art-orbit {
    position: absolute;
    top: 50%;
    left: 50%;
    border: 1px solid rgb(var(--rgb-violet-soft) / 45%);
    border-radius: 50%;
    transform: translate(-50%, -50%) rotate(-18deg);
}

.group-header__art-orbit--outer {
    width: 13rem;
    height: 5.5rem;
}

.group-header__art-orbit--inner {
    width: 7rem;
    height: 3rem;
    border-color: rgb(var(--rgb-mint) / 60%);
    transform: translate(-50%, -50%) rotate(35deg);
}

.group-header__art-star {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 1.15rem;
    height: 1.15rem;
    border: 3px solid var(--color-text);
    border-radius: 50%;
    background: var(--color-coral);
    box-shadow: 0 0 0 0.45rem rgb(var(--rgb-coral) / 15%), 0 0 2rem rgb(var(--rgb-coral) / 45%);
    transform: translate(-50%, -50%);
}

.group-header__art-label {
    position: absolute;
    right: var(--space-3);
    bottom: var(--space-3);
    color: var(--color-text-soft);
    font-family: var(--font-meta);
    font-size: 0.625rem;
    letter-spacing: 0.12em;
}

.group-header__copy {
    min-width: 0;
}

.group-header .orbit-meta {
    margin: 0 0 var(--space-2);
    color: var(--color-violet-soft);
}

.group-header__title-row {
    display: flex;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-3);
}

.group-header h1 {
    margin: 0;
    color: var(--color-text);
    font-family: var(--font-display);
    font-size: clamp(1.75rem, 7vw, 2.5rem);
    font-weight: 700;
    line-height: 1.15;
    letter-spacing: 0;
}

.group-header__badge {
    margin-top: 0.25rem;
    padding: 0.35rem 0.55rem;
    border: 1px solid rgb(var(--rgb-mint) / 30%);
    border-radius: 999px;
    background: rgb(var(--rgb-mint) / 9%);
    color: var(--color-mint);
    font-family: var(--font-meta);
    font-size: 0.625rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}

.group-header__copy > p:last-child {
    max-width: 52ch;
    margin: var(--space-3) 0 0;
    color: var(--color-text-muted);
    font-size: 0.9375rem;
    line-height: 1.65;
}

.group-header__signals {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-4);
    margin-top: var(--space-4);
    color: var(--color-text-faint);
    font-size: 0.8125rem;
}

.group-header__signals span {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
}

.group-header__signals :deep(.icon-glyph) { color: var(--color-violet-soft); }

.group-header__signal-dot {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
    background: var(--color-mint);
    box-shadow: 0 0 0 0.25rem rgb(var(--rgb-mint) / 12%);
}

.group-header__aside {
    display: grid;
    align-items: start;
    gap: var(--space-3);
}

.group-header__join {
    display: inline-flex;
    min-width: 100%;
    min-height: var(--touch-target);
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: 0 var(--space-4);
    border: 0;
    border-radius: var(--radius-small);
    background: var(--gradient-action);
    color: #fff;
    font-family: var(--font-body);
    font-size: 0.875rem;
    font-weight: 700;
    cursor: pointer;
    transition: transform 160ms ease, opacity 160ms ease;
}

.group-header__join:hover:not(:disabled) {
    transform: translateY(-1px);
}

.group-header__join:disabled {
    cursor: wait;
    opacity: 0.7;
}

.group-header__member-state {
    display: inline-flex;
    min-height: var(--touch-target);
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    margin: 0;
    padding: 0 var(--space-3);
    border: 1px solid rgb(var(--rgb-mint) / 25%);
    border-radius: var(--radius-small);
    background: rgb(var(--rgb-mint) / 8%);
    color: var(--color-mint);
    font-size: 0.8125rem;
    font-weight: 600;
}

.group-header__delete {
    min-height: var(--touch-target);
    padding: 0 var(--space-4);
    border: 1px solid var(--color-coral);
    border-radius: var(--radius-small);
    background: transparent;
    color: var(--color-coral);
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
}

.group-header__delete:hover:not(:disabled) {
    background: var(--color-coral);
    color: var(--color-background);
}

.group-header__error {
    grid-column: 1 / -1;
    margin: 0;
    color: var(--color-coral);
    font-size: .875rem;
}

.group-membership {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: 0 var(--space-1) var(--space-4);
    border-bottom: 1px solid var(--color-border);
    color: var(--color-text-muted);
}

.group-membership > div {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.875rem;
}

.group-membership > div :deep(.icon-glyph) {
    color: var(--color-violet-soft);
}

.group-membership__count {
    color: var(--color-text);
    font-family: var(--font-meta);
    font-size: 0.75rem;
}

.group-sections {
    display: flex;
    gap: var(--space-2);
    overflow-x: auto;
    padding: var(--space-2);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-medium);
    background: rgb(var(--rgb-surface) / 72%);
    scrollbar-width: none;
}

.group-sections::-webkit-scrollbar { display: none; }

.group-section-tab {
    display: inline-flex;
    min-width: max-content;
    min-height: var(--touch-target);
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: 0 var(--space-4);
    border: 1px solid transparent;
    border-radius: var(--radius-small);
    background: transparent;
    color: var(--color-text-muted);
    cursor: pointer;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    white-space: nowrap;
    transition: border-color 160ms ease, background 160ms ease, color 160ms ease;
}

.group-section-tab:hover,
.group-section-tab:focus-visible {
    border-color: var(--color-border-strong);
    color: var(--color-text);
}

.group-section-tab--active {
    border-color: rgb(var(--rgb-violet) / 55%);
    background: rgb(var(--rgb-violet) / 16%);
    color: var(--color-violet-soft);
}

.group-section-panels { min-width: 0; }
.group-section-panel { min-width: 0; }

.group-overview {
    display: grid;
    gap: var(--space-6);
    padding: var(--space-5);
    border-color: rgb(var(--rgb-violet) / 30%);
    background:
        linear-gradient(135deg, rgb(var(--rgb-violet) / 11%), transparent 58%),
        var(--color-surface);
}

.group-overview .orbit-meta { margin: 0 0 var(--space-2); color: var(--color-violet-soft); }
.group-overview h2 { max-width: 28ch; margin: 0; font-family: var(--font-display); font-size: clamp(1.25rem, 4vw, 1.75rem); line-height: 1.2; }
.group-overview p:not(.orbit-meta) { max-width: 58ch; margin: var(--space-3) 0 0; color: var(--color-text-muted); line-height: 1.65; }

.group-overview__signals { display: grid; gap: var(--space-3); }
.group-overview__signals > div { display: grid; grid-template-columns: 2.75rem minmax(0, 1fr); align-items: center; gap: var(--space-3); }
.group-overview__icon { display: grid; width: 2.75rem; height: 2.75rem; place-items: center; border: 1px solid; border-radius: 50%; }
.group-overview__icon--mint { border-color: rgb(var(--rgb-mint) / 35%); background: rgb(var(--rgb-mint) / 10%); color: var(--color-mint); }
.group-overview__icon--violet { border-color: rgb(var(--rgb-violet-soft) / 35%); background: rgb(var(--rgb-violet) / 12%); color: var(--color-violet-soft); }
.group-overview__icon--coral { border-color: rgb(var(--rgb-coral) / 35%); background: rgb(var(--rgb-coral) / 10%); color: var(--color-coral); }
.group-overview__signals span:last-child { display: grid; gap: 0.2rem; min-width: 0; }
.group-overview__signals strong { color: var(--color-text); font-size: 0.875rem; }
.group-overview__signals small { color: var(--color-text-faint); font-size: 0.75rem; line-height: 1.4; }

.group-welcome {
    display: grid;
    gap: var(--space-5);
    padding: var(--space-5);
    border-color: rgb(var(--rgb-violet) / 30%);
    background:
        linear-gradient(135deg, rgb(var(--rgb-violet) / 11%), transparent 58%),
        var(--color-surface);
}

.group-welcome .orbit-meta {
    margin: 0 0 var(--space-2);
    color: var(--color-violet-soft);
}

.group-welcome h2 {
    max-width: 25ch;
    margin: 0;
    color: var(--color-text);
    font-family: var(--font-display);
    font-size: clamp(1.25rem, 4vw, 1.75rem);
    line-height: 1.2;
}

.group-welcome > div:first-child > p:last-child {
    max-width: 58ch;
    margin: var(--space-3) 0 0;
    color: var(--color-text-muted);
    font-size: 0.9375rem;
    line-height: 1.65;
}

.group-welcome__details {
    display: grid;
    align-content: center;
    gap: var(--space-3);
    color: var(--color-text-soft);
    font-size: 0.8125rem;
}

.group-welcome__details span {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
}

.group-welcome__details :deep(.icon-glyph) {
    color: var(--color-mint);
}

.group-feed {
    display: grid;
    gap: var(--space-5);
    padding: var(--space-5);
}

.group-feed__heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: var(--space-3);
}

.group-feed__heading h2 {
    margin: var(--space-1) 0 0;
    color: var(--color-text);
    font-family: var(--font-display);
    font-size: 1.75rem;
    letter-spacing: 0;
}

.group-feed__heading > span {
    color: var(--color-text-muted);
    font-size: .875rem;
}

.group-posts {
    display: grid;
    gap: var(--space-5);
}

.group-feed__state {
    margin: 0;
    padding: var(--space-5);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-small);
    color: var(--color-text-muted);
    text-align: center;
}

.group-page__state {
    margin: var(--space-7) 0;
    padding: var(--space-6);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-medium);
    text-align: center;
    color: var(--color-text-muted);
}
.group-page__state--error { color: var(--color-coral); }

@media (min-width: 48rem) {
    .group-container { gap: var(--space-7); }
    .group-header {
        grid-template-columns: minmax(10rem, 0.35fr) minmax(0, 1fr) auto;
        align-items: center;
        gap: var(--space-6);
        padding: var(--space-5);
    }

    .group-header__art { min-height: 10rem; }

    .group-header__aside {
        min-width: 12rem;
        justify-items: stretch;
    }

    .group-header__join { min-width: 0; }

    .group-header__delete {
        width: 100%;
    }

    .group-welcome {
        grid-template-columns: minmax(0, 1fr) minmax(15rem, 0.55fr);
        align-items: center;
        padding: var(--space-6);
    }

    .group-feed { padding: var(--space-6); }
    .group-feed__heading { align-items: start; }
    .group-feed__heading h2 { font-size: 1.5rem; }

    .group-overview {
        grid-template-columns: minmax(0, 1fr) minmax(16rem, 0.75fr);
        align-items: start;
        padding: var(--space-6);
    }

    .group-overview__signals {
        padding-left: var(--space-5);
        border-left: 1px solid var(--color-border);
    }
}
</style>
