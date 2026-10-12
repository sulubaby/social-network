// postReaction likes or dislikes a post, react is { postID, value }
export async function postReaction(react) {
    const resp = await fetch("/api/post/reaction", {
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify(react)
    });

    const result = await resp.json();
    if (!resp.ok) {
        throw new Error(result.message || "could not insert reaction")
    }

    return result;
}