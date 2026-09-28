package posts

import (
	"database/sql"
	"social/internal/models"
)

/*
function to get the posts of one user for their profile page, one page at a time.
it uses the same privacy rules as the home feed (visiblePostsQuery), and on top of
that if the profile is private you only see the posts when you follow them
(or when its your own profile).

Parameters:

	db *sql.DB,
	viewerID int
		-> the logged in user
	authorID int
		-> the user whose profile we are looking at
	limit, offset int

Returns:

	[]models.Post
		-> empty list if there is nothing to show

	error
		-> nil if success
*/
func ListProfilePosts(db *sql.DB, viewerID, authorID, limit, offset int) ([]models.Post, error) {
	query := visiblePostsQuery + `
		AND posts.user_id = ?
		AND (
			posts.user_id = ?
			OR COALESCE(profile.is_private, 0) = 0
			OR EXISTS (
				SELECT 1
				FROM user_followers AS follows
				WHERE follows.follower_id = ?
				AND follows.target_id = posts.user_id
				AND follows.status = 1
			)
		)
		ORDER BY posts.created_at DESC, posts.id DESC
		LIMIT ? OFFSET ?
	`

	return queryPosts(db, query,
		viewerID, viewerID, viewerID, viewerID, viewerID,
		authorID,
		viewerID, viewerID,
		limit, offset,
	)
}
