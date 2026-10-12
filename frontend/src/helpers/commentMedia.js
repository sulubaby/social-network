import { ref, onBeforeUnmount } from 'vue';
import { CHAT_MEDIA_ACCEPT, validateChatMedia } from '@/helpers/chatMedia';
import { cleanText, validateCommentText } from '@/helpers/limits';

// comments take the same images and gifs as chats
export const COMMENT_MEDIA_ACCEPT = CHAT_MEDIA_ACCEPT;

// a blob url is the local preview before upload, anything else is a saved file in uploads
export function commentImageSrc(path) {
    if (!path) {
        return '';
    }

    return path.startsWith('blob:') ? path : `/uploads/${path}`;
}

// buildCommentRequest makes the fetch options for a new comment
// no image means a small json body, with an image we need form data
export function buildCommentRequest(postId, content, replyTo = null, image = null) {
    content = cleanText(content);

    const contentError = validateCommentText(content, Boolean(image));

    if (contentError) {
        throw new Error(contentError);
    }

    if (!image) {
        return {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                postId,
                content,
                replyTo
            })
        };
    }

    const form = new FormData();

    form.append('postId', String(postId));
    form.append('content', content);

    if (replyTo !== null) {
        form.append('replyTo', String(replyTo));
    }

    form.append('image', image);

    return {
        method: 'POST',
        credentials: 'include',
        body: form
    };
}

// useCommentMedia keeps the picked image of a comment box and its preview
export function useCommentMedia() {
    const file = ref(null);
    const previewUrl = ref('');
    const error = ref('');
    const fileInput = ref(null);

    // free the preview url so the browser does not keep the file in memory
    function release() {
        if (previewUrl.value) {
            URL.revokeObjectURL(previewUrl.value);
        }

        file.value = null;
        previewUrl.value = '';
    }

    function clearMedia() {
        release();
        error.value = '';
    }

    function openPicker() {
        fileInput.value?.click();
    }

    // check the picked file first, if it is fine show the preview
    function onSelect(event) {
        const selected = event.target.files?.[0];

        // reset the input so picking the same file again still works
        event.target.value = '';

        if (!selected) {
            return;
        }

        const message = validateChatMedia(selected);

        if (message) {
            error.value = message;
            return;
        }

        release();
        error.value = '';
        file.value = selected;
        previewUrl.value = URL.createObjectURL(selected);
    }

    // takeMedia hands the file to the send code and empties the box right away
    function takeMedia() {
        if (!file.value) {
            return null;
        }

        const media = {
            file: file.value,
            previewUrl: previewUrl.value
        };

        file.value = null;
        previewUrl.value = '';
        error.value = '';

        return media;
    }

    // restoreMedia puts the image back if sending failed
    function restoreMedia(media) {
        if (!media) {
            return;
        }

        release();
        file.value = media.file;
        previewUrl.value = media.previewUrl;
    }

    function revokeMedia(media) {
        if (media?.previewUrl) {
            URL.revokeObjectURL(media.previewUrl);
        }
    }

    onBeforeUnmount(release);

    return {
        file,
        previewUrl,
        error,
        fileInput,
        clearMedia,
        openPicker,
        onSelect,
        takeMedia,
        restoreMedia,
        revokeMedia
    };
}
