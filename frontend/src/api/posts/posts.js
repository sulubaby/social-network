import { checkSessionResponse } from '@/helpers/auth/auth'
import { router } from '@/router/router'

// create a post. formData has content, privacy, location, selected followers and the image
export async function createPost(formData) {
    const response = await fetch('/api/posts', {
        method: 'POST',
        credentials: 'include',
        body: formData,
    })

    if (!checkSessionResponse(response)) {
        router.replace('/login')
        return
    }

    const result = await response.json()

    if (!response.ok) {
        throw new Error(result.message || 'Failed to create post')
    }

    return result
}

// get one page of the home feed
export async function getPosts({ limit = 20, offset = 0 } = {}) {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    const response = await fetch(`/api/posts?${params}`, {
        method: 'GET',
        credentials: 'include',
    })

    if (!checkSessionResponse(response)) {
        router.replace('/login')
        return
    }

    const result = await response.json()

    if (!response.ok) {
        throw new Error(result.message || 'Could not load posts')
    }

    return result
}

// like = PUT, unlike = DELETE on the same url
export async function setPostLike(postId, liked) {
    const response = await fetch(`/api/posts/${postId}/like`, {
        method: liked ? 'PUT' : 'DELETE',
        credentials: 'include',
    })

    if (!checkSessionResponse(response)) {
        router.replace('/login')
        return
    }

    const result = await response.json()
    if (!response.ok) {
        throw new Error(result.message || 'Could not update the like')
    }

    return result
}
