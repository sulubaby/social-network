import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';
import { computed, ref, watch } from 'vue';

const stateSource = readFileSync(new URL('../src/data/chatState.js', import.meta.url), 'utf8');
const socketSource = readFileSync(new URL('../src/api/socket/socket.js', import.meta.url), 'utf8');
const script = source => source.replace(/^import[\s\S]*?;\r?\n/gm, '').replace(/^export /gm, '');

function client(userID = 1, storage = new Map()) {
    const document = new EventTarget();
    document.hidden = false;
    const window = new EventTarget();
    window.location = { protocol: 'http:', host: 'localhost', pathname: '/home' };
    class CustomEvent extends Event {
        constructor(type, options) { super(type); this.detail = options.detail; }
    }
    class WebSocket {
        static OPEN = 1;
        readyState = 1;
    }
    const context = vm.createContext({
        ref, computed, watch, document, window, CustomEvent, WebSocket, console,
        sessionUserId: ref(userID),
        sessionStorage: {
            getItem: key => storage.get(key) ?? null,
            setItem: (key, value) => storage.set(key, value)
        },
        setTyping() {}, setGroupTyping() {}, addNotification() {}, avatarUrl() {},
        postImageUrl() {}, handleIncomingNotification() {},
        incrementUnreadNotificationCount() {}, refreshUnreadNotificationCount() {},
        setUnreadNotificationCount() {}
    });
    vm.runInContext(script(stateSource) + '\nglobalThis.state = { activePage, openGroupPage, hasUnreadChats, markChatSeen };', context);
    vm.runInContext(script(socketSource) + '\nglobalThis.socket = connectToWS();', context);
    return {
        ...context.state, context, storage,
        receive(senderID = 2, groupID = 10, isPrivate = true, silent = false) {
            context.socket.onmessage({ data: JSON.stringify({
                type: 'message', isPrivate, silent,
                data: { Sender: { ID: senderID, firstName: 'Sender' }, GroupID: groupID, content: 'Hello' }
            }) });
        },
        visibility(hidden) {
            document.hidden = hidden;
            document.dispatchEvent(new Event('visibilitychange'));
        }
    };
}

test('unseen private and group chats clear independently after viewing', () => {
    const c = client();
    c.receive(2, 10);
    c.receive(3, 20, false);
    assert.equal(c.hasUnreadChats.value, true);
    c.activePage.value = 'chat:2';
    assert.equal(c.hasUnreadChats.value, true, 'Selecting a chat alone must not clear it');
    c.markChatSeen('chat:2');
    assert.equal(c.hasUnreadChats.value, true, 'The group remains unseen');
    c.activePage.value = 'group:20';
    c.markChatSeen('group:20');
    assert.equal(c.hasUnreadChats.value, false);
});

test('loaded open chats stay seen, but messages after leaving are unseen', () => {
    for (const isPrivate of [true, false]) {
        const c = client();
        const key = isPrivate ? 'chat:2' : 'group:10';
        c.activePage.value = key;
        c.markChatSeen(key);
        c.receive(2, 10, isPrivate);
        assert.equal(c.hasUnreadChats.value, false);
        c.activePage.value = null;
        c.receive(2, 10, isPrivate);
        assert.equal(c.hasUnreadChats.value, true);
        c.markChatSeen(key);
        assert.equal(c.hasUnreadChats.value, true, 'A stale load cannot clear another page');
    }
});

test('group feed and matching private sender IDs do not count as viewing group chat', () => {
    const c = client();
    c.activePage.value = 'chat:2';
    c.markChatSeen('chat:2');
    c.openGroupPage.value = 10;
    c.receive(2, 10, false);
    assert.equal(c.hasUnreadChats.value, true);
});

test('hidden-tab messages clear when the loaded chat becomes visible', () => {
    const c = client();
    c.activePage.value = 'chat:2';
    c.markChatSeen('chat:2');
    c.visibility(true);
    c.receive();
    assert.equal(c.hasUnreadChats.value, true);
    c.visibility(false);
    assert.equal(c.hasUnreadChats.value, false);
});

test('muted messages count, own messages do not', () => {
    const c = client();
    c.receive(1, 10);
    c.receive(1, 20, false);
    assert.equal(c.hasUnreadChats.value, false);
    c.receive(2, 10, true, true);
    assert.equal(c.hasUnreadChats.value, true);
});

test('unseen state survives navigation reloads and is isolated by session user', () => {
    const c = client();
    c.receive();
    const reloaded = client(1, c.storage);
    assert.equal(reloaded.hasUnreadChats.value, true);
    const otherUser = client(3, c.storage);
    assert.equal(otherUser.hasUnreadChats.value, false);
    reloaded.activePage.value = 'chat:2';
    reloaded.markChatSeen('chat:2');
    assert.equal(client(1, c.storage).hasUnreadChats.value, false);
    c.context.sessionUserId.value = 3;
    assert.equal(c.hasUnreadChats.value, false);
});
