// shared helpers so the popups and the notifications page describe
// a notification the same way (title and where clicking it goes)

const TITLES = {
  follow_request: 'Follow request',
  new_follower: 'New follower',
  follow_accepted: 'Request accepted',
  join_request: 'Join request',
  join_accepted: 'Welcome to the group',
  join_rejected: 'Join request declined',
  invitation: 'Group invitation',
  invitation_accepted: 'Invitation accepted',
  event_created: 'New event',
  group_post_comment: 'New comment',
  post_comment: 'New comment',
  post_like: 'New like',
  new_message: 'New message',
}

export function notificationTitle(notification) {
  return TITLES[notification?.type] || 'Orbit update'
}

// the page that a notification is about
export function notificationLink(notification) {
  if (!notification) return ''

  if (notification.category === 'messages' && notification.relatedId) {
    return `/chats?chat=${notification.relatedId}`
  }

  if (notification.category === 'requests' && notification.actorId) {
    return `/user?id=${notification.actorId}`
  }

  if (notification.category === 'events' && notification.groupId) {
    return `/groups/${notification.groupId}?section=activity`
  }

  if (notification.category === 'groups' && notification.groupId) {
    return `/groups/${notification.groupId}`
  }

  // likes and comments are on my own posts, they live on my profile
  if (notification.category === 'posts') return '/me'

  return ''
}
