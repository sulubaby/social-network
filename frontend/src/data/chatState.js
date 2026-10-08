import { computed, ref, watch } from 'vue';
import { sessionUserId } from './currentUser';

export let activePage = ref(null);
export let messageSent = ref(true);
export const openGroupPage = ref(null);
export const chatsSidebarOpen = ref(false);

// Navigation uses full-page links, so keep unseen chats across reloads per user.
const unseenChats = ref({});
let viewedChat = null;
export const hasUnreadChats = computed(() => Object.keys(unseenChats.value).length > 0);

function saveUnseenChats() {
    if (!sessionUserId.value) return;
    try {
        sessionStorage.setItem(`unseen-chats:${sessionUserId.value}`, JSON.stringify(unseenChats.value));
    } catch {
        // The live indicator still works when browser storage is unavailable.
    }
}

watch(sessionUserId, () => {
    unseenChats.value = {};
    viewedChat = null;
    if (!sessionUserId.value) return;
    try {
        const stored = JSON.parse(sessionStorage.getItem(`unseen-chats:${sessionUserId.value}`) || '{}');
        if (stored && typeof stored === 'object' && !Array.isArray(stored)) {
            unseenChats.value = Object.fromEntries(
                Object.keys(stored).filter(key => /^(chat|group):[1-9]\d*$/.test(key)).map(key => [key, true])
            );
        }
    } catch {
        // Ignore unavailable storage or an invalid saved value.
    }
}, { flush: 'sync', immediate: true });

watch(activePage, () => { viewedChat = null; }, { flush: 'sync' });

export function markChatSeen(key) {
    if (activePage.value !== key) return;
    viewedChat = key;
    if (!document.hidden && unseenChats.value[key]) {
        delete unseenChats.value[key];
        saveUnseenChats();
    }
}

export function recordIncomingChat(payload) {
    const message = payload.data;
    const senderID = Number(message?.Sender?.ID);
    const groupID = Number(message?.GroupID);
    if (!sessionUserId.value || !Number.isInteger(senderID) || senderID <= 0 ||
        senderID === sessionUserId.value || !Number.isInteger(groupID) || groupID <= 0) return;

    // The authenticated socket delivers private messages only to their recipient.
    const key = payload.isPrivate ? `chat:${senderID}` : `group:${groupID}`;
    if (viewedChat === key && activePage.value === key && !document.hidden) return;
    unseenChats.value[key] = true;
    saveUnseenChats();
}

if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
        if (viewedChat) markChatSeen(viewedChat);
    });
}

export function openChatsSidebar() {
    chatsSidebarOpen.value = true;
}

export function closeChatsSidebar() {
    chatsSidebarOpen.value = false;
}

export function toggleChatsSidebar() {
    chatsSidebarOpen.value = !chatsSidebarOpen.value;
}
