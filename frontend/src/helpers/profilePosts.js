const avatarColors = ['#3ee6b0', '#ff6b8a', '#7c5cff', '#ffb84d', '#4cc3ff']

function formatPostTime(value) {
  if (!value) return 'Just now'
  const date = new Date(String(value).replace(' ', 'T'))
  return Number.isNaN(date.getTime())
    ? 'Recently'
    : date.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
}

function privacyLabel(value) {
  return { public: 'Public', followers: 'Followers only', selected: 'Selected followers' }[value] || value
}

function locationLabel(value) {
  return value ? String(value).split(':')[0].trim() : ''
}

// changes a post from the server into the shape PostCard wants (same as the home feed)
export function toProfileCardPost(post, index = 0) {
  return {
    id: post.id,
    authorId: post.userId,
    author: post.author || 'Orbit member',
    avatarColor: avatarColors[index % avatarColors.length],
    avatarPath: post.avatarPath || '',
    time: formatPostTime(post.createdAt),
    privacy: privacyLabel(post.privacy),
    content: post.content || '',
    likes: post.likeCount || 0,
    liked: Boolean(post.liked),
    comments: post.commentCount || 0,
    imagePath: post.imagePath || '',
    location: post.location || '',
    locationLabel: locationLabel(post.location),
    hasMedia: false,
    mediaDescription: '',
  }
}
