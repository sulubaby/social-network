import { ref } from 'vue';

import { getUnreadNotificationCount } from '@/api/common/notifications';
import { addNotification } from './notifications';
import { messageSent } from './chatState';

// the bell number, shared by every component that shows it
export const unreadNotificationCount = ref(0);

// countVersion goes up on every change so an old fetch that comes back late cannot overwrite a newer count
let countVersion = 0;

// refreshUnreadNotificationCount asks the server for the real count
export async function refreshUnreadNotificationCount() {
    const version = ++countVersion;

    try {
        const result = await getUnreadNotificationCount();

        if (version !== countVersion) {
            return;
        }

        unreadNotificationCount.value = Math.max(0, Number(result.count) || 0);
    } catch (err) {
        console.error(err);
    }
}

// the functions below change the count locally, for example when the websocket sends a new one
export function setUnreadNotificationCount(count) {
    countVersion++;
    unreadNotificationCount.value = Math.max(0, Number(count) || 0);
}

export function incrementUnreadNotificationCount() {
    countVersion++;
    unreadNotificationCount.value += 1;
}

export function clearUnreadNotificationCount() {
    countVersion++;
    unreadNotificationCount.value = 0;
}

// handleIncomingNotification shows the error when the websocket could not send something
export function handleIncomingNotification(data) {
    if (data && data.error) {
        messageSent.value = false;
        addNotification(data.message, 'error');
    }
}
