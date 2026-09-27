import { checkSessionResponse } from '@/helpers/auth/auth'
import { router } from '@/router/router'

// helper for the comment requests: sends the cookie, handles logout and errors
async function requestComments(url, options = {}) {
  const response = await fetch(url, {
    credentials: 'include',
    ...options,
  })

  if (!checkSessionResponse(response)) {
    router.replace('/login')
    return
  }

  const result = await response.json()
  if (!response.ok) {
    throw new Error(result.message || 'Could not load comments')
  }

  return result
}

// get one page of comments of a post
export async function getComments(postId, { limit = 20, offset = 0 } = {}) {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  })

  return requestComments(`/api/posts/${postId}/comments?${params}`)
}

// add a comment. if there is an image we have to send FormData,
// if its just text we send normal json
export async function createComment(postId, comment) {
  if (comment.image) {
    const formData = new FormData()
    formData.append('content', comment.content)
    formData.append('image', comment.image)

    return requestComments(`/api/posts/${postId}/comments`, {
      method: 'POST',
      body: formData,
    })
  }

  return requestComments(`/api/posts/${postId}/comments`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ content: comment.content }),
  })
}

// delete my comment
export async function deleteComment(postId, commentId) {
  return requestComments(`/api/posts/${postId}/comments/${commentId}`, {
    method: 'DELETE',
  })
}