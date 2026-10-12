import { reactive } from 'vue';

// notifications here are the small popups on the screen, not the notifications page
export const notifications = reactive([]);

// postDialog is the post that opens on top of the page when a notification is clicked
export const postDialog = reactive({
    show: false,
    postId: null
});

export function openPostDialog(postId) {
    if (!postId) {
        return;
    }

    postDialog.postId = Number(postId);
    postDialog.show = true;
}

export function closePostDialog() {
    postDialog.show = false;
    postDialog.postId = null;
}

// postImageUrl gives the picture for the notification, videos dont get a thumbnail
export function postImageUrl(imagePath) {
    if (!imagePath) {
        return '';
    }

    const path = imagePath.toLowerCase();

    if (
        path.endsWith('.mp4') ||
        path.endsWith('.webm') ||
        path.endsWith('.mov') ||
        path.endsWith('.avi')
    ) {
        return '';
    }

    return `/uploads/${imagePath}`;
}

export function avatarUrl(avatarPath) {
    return avatarPath ? `/uploads/${avatarPath}` : '';
}

// each popup gets its own id so we can remove the right one
let nextNotificationId = 0;

// addNotification shows a popup, it goes away alone after a few seconds
// popups you can click stay a bit longer
export function addNotification(message, type = 'success', options = {}) {
    const id = ++nextNotificationId;

    const clickable = !!(options.postId || options.route);

    notifications.push({
        id,
        message,
        type,
        avatar: options.avatar || '',
        initial: options.initial || '',
        image: options.image || '',
        postId: options.postId || null,
        route: options.route || null,
        notificationId: options.notificationId || null
    });

    setTimeout(() => {
        removeNotification(id);
    }, clickable ? 6000 : 3000);
}

// removeNotification closes one popup
export function removeNotification(id) {
    const index = notifications.findIndex(
        notification => notification.id === id
    );

    if (index !== -1) {
        notifications.splice(index, 1);
    }
}
