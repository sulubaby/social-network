import { checkSessionResponse } from '@/helpers/auth/auth'
import { router } from '@/router/router'

// small helper for all the notification requests.
// sends the cookie, goes to /login if the session is gone,
// and throws the server message if something failed
async function request(url, options = {}) {
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
    throw new Error(result.message || 'Could not update notifications')
  }

  return result
}

// get one page of notifications. category can be all, requests, groups, events or messages
export function getNotifications(category = 'all', { limit = 20, offset = 0 } = {}) {
  const params = new URLSearchParams({
    category,
    limit: String(limit),
    offset: String(offset),
  })
  return request(`/api/notifications?${params}`)
}

// mark one notification as read
export function markNotificationRead(notificationId) {
  return request(`/api/notifications/${notificationId}/read`, { method: 'PATCH' })
}

// the "mark all as read" button
export function markAllNotificationsRead() {
  return request('/api/notifications/read-all', { method: 'PATCH' })
}

// when the user clicks a button inside a notification (accept, decline, join, rsvp...)
export function applyNotificationAction(notificationId, action) {
  return request(`/api/notifications/${notificationId}/action`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ action }),
  })
}
