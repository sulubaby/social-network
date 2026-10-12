import { buildCommentRequest } from '@/helpers/commentMedia';

// getComments loads the comments of a post, or the replies of one comment when replyTo is given
export async function getComments(postId, replyTo = null) {
    const params = new URLSearchParams();

    params.append('postId', postId.toString());

    if (replyTo !== null) {
        params.append('replyTo', replyTo.toString());
    }

    const response = await fetch(
        `/api/post/comment?${params.toString()}`,
        {
            method: 'GET',
            credentials: 'include'
        }
    );

    const data = await response.json();

    if (!response.ok || !data.status) {
        throw new Error(
            data.message || 'Failed to load comments'
        );
    }

    console.log(data.comments)
    return data.comments || [];
}

// addComment sends text and maybe an image, buildCommentRequest picks json or form data
export async function addComment(
    postId,
    content,
    replyTo = null,
    image = null
) {
    const response = await fetch(
        '/api/post/comment',
        buildCommentRequest(postId, content, replyTo, image)
    );

    const data = await response.json();

    if (!response.ok || !data.status) {
        throw new Error(
            data.message || 'Failed to add comment'
        );
    }

    return data.comment;
}

// deleteComment removes the comment and its replies
export async function deleteComment(commentId) {
    const response = await fetch(
        `/api/post/comment?commentId=${commentId}`,
        {
            method: 'DELETE',
            credentials: 'include'
        }
    );

    const data = await response.json();

    if (!response.ok || !data.status) {
        throw new Error(
            data.message || 'Failed to delete comment'
        );
    }

    return data;
}

// voteComment is like 1 or dislike -1, the same vote again removes it
export async function voteComment(commentId, vote) {
    const response = await fetch(
        `/api/post/comment/vote?commentId=${commentId}&vote=${vote}`,
        {
            method: 'POST',
            credentials: 'include'
        }
    );

    const data = await response.json();

    if (!response.ok || !data.status) {
        throw new Error(
            data.message || 'Failed to vote on comment'
        );
    }

    return data;
}