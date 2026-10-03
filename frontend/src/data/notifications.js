import { reactive } from 'vue';

// the popups (toasts) at the top of the screen
export const notifications = reactive([]);

let nextID = 1
const timers = new Map()

// type: 'success' | 'error' | 'alert' (coral, notifications) | 'message' (mint, chats)
// options.title: bold first line, options.to: page to open on click,
// options.key: a popup with the same key replaces the old one instead of stacking
// (so ten messages from the same chat show as one popup that keeps updating)
export function addNotification(message, type = 'success', options = {}) {
    const displayMessage = message instanceof Error ? message.message : String(message);
    const duration = options.duration ?? (type === 'error' ? 5000 : 4500)

    if (options.key) {
        const existing = notifications.find(item => item.key === options.key)
        if (existing) removeNotification(existing.id)
    }

    const id = nextID++
    notifications.push({
        id,
        key: options.key || '',
        title: options.title || '',
        to: options.to || '',
        message: displayMessage,
        type
    });

    // keep at most 4 popups on screen
    while (notifications.length > 4) removeNotification(notifications[0].id)

    timers.set(id, setTimeout(() => removeNotification(id), duration))
}

export function removeNotification(id) {
    clearTimeout(timers.get(id))
    timers.delete(id)

    const index = notifications.findIndex(
        notification => notification.id === id
    );

    if (index !== -1) {
        notifications.splice(index, 1);
    }
}
