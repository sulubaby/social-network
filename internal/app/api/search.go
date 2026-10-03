package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Search returns the small set of results needed by the global search screen.
// Posts use the same privacy rules as the home feed, so searching never reveals
// a post that the signed-in user could not already see in their feed.
func (app *App) Search(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": true,
			"query":  "",
			"users":  []any{},
			"groups": []any{},
			"posts":  []any{},
		})
		return
	}

	pattern := "%" + query + "%"
	excludeGroupID, _ := strconv.Atoi(r.URL.Query().Get("excludeGroupId"))
	users, err := searchUsers(app.DB, userID, query, excludeGroupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not search people"})
		return
	}

	groups, err := searchGroups(app.DB, pattern)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not search groups"})
		return
	}

	posts, err := searchPosts(app.DB, userID, pattern)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not search posts"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": true,
		"query":  query,
		"users":  users,
		"groups": groups,
		"posts":  posts,
	})
}

// searchUsers finds people by username or name.
// excludeGroupID is used by the group invite search to hide people already in the group
func searchUsers(db *sql.DB, currentUserID int, query string, excludeGroupID int) ([]map[string]any, error) {
	groupFilter := ""
	args := []any{currentUserID, currentUserID}
	if excludeGroupID > 0 {
		groupFilter = `
		  AND NOT EXISTS (
			SELECT 1 FROM group_members
			WHERE group_id = ? AND user_id = u.id
		  )`
		args = append(args, excludeGroupID)
	}
	// People search is prefix-based, so typing "o" starts with accounts whose
	// username or display name starts with "o" instead of unrelated contains matches.
	prefixPattern := query + "%"
	args = append(args, prefixPattern, prefixPattern, prefixPattern, prefixPattern)

	rows, err := db.Query(`
		SELECT
			u.id,
			COALESCE(u.username, ''),
			u.first_name,
			u.last_name,
			COALESCE(p.avatar_path, ''),
			COALESCE(p.is_private, 0),
			COALESCE(relationship.status, -1)
		FROM user AS u
		LEFT JOIN profile AS p ON p.user_id = u.id
		LEFT JOIN user_followers AS relationship
			ON relationship.follower_id = ?
			AND relationship.target_id = u.id
		WHERE u.id <> ?
		`+groupFilter+`
		  AND (
			COALESCE(u.username, '') LIKE ?
			OR u.first_name LIKE ?
			OR u.last_name LIKE ?
			OR (u.first_name || ' ' || u.last_name) LIKE ?
		  )
		ORDER BY u.first_name, u.last_name
		LIMIT 12
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]any, 0)
	for rows.Next() {
		var id, private, followStatus int
		var username, firstName, lastName, avatar string
		if err := rows.Scan(&id, &username, &firstName, &lastName, &avatar, &private, &followStatus); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{
			"id":           id,
			"username":     username,
			"firstName":    firstName,
			"lastName":     lastName,
			"avatarPath":   avatar,
			"isPrivate":    private == 1,
			"followStatus": followStatus,
		})
	}

	return results, rows.Err()
}

// searchGroups finds groups by title or description and counts their members
func searchGroups(db *sql.DB, pattern string) ([]map[string]any, error) {
	rows, err := db.Query(`
		SELECT
			g.id,
			g.title,
			g.description,
			COUNT(gm.user_id)
		FROM groups AS g
		LEFT JOIN group_members AS gm ON gm.group_id = g.id
		WHERE g.title LIKE ? OR g.description LIKE ?
		GROUP BY g.id, g.title, g.description
		ORDER BY g.id DESC
		LIMIT 12
	`, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]any, 0)
	for rows.Next() {
		var id, memberCount int
		var title, description string
		if err := rows.Scan(&id, &title, &description, &memberCount); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{
			"id":          id,
			"title":       title,
			"description": description,
			"memberCount": memberCount,
		})
	}

	return results, rows.Err()
}

// searchPosts finds posts by text, only ones i am allowed to see
func searchPosts(db *sql.DB, currentUserID int, pattern string) ([]map[string]any, error) {
	rows, err := db.Query(`
		SELECT
			posts.id,
			posts.user_id,
			users.first_name || ' ' || users.last_name,
			COALESCE(profile.avatar_path, ''),
			posts.content,
			posts.image_path,
			posts.privacy,
			posts.created_at,
			posts.comment_count,
			posts.like_count
		FROM posts
		JOIN user AS users ON users.id = posts.user_id
		LEFT JOIN profile ON profile.user_id = users.id
		WHERE (posts.content LIKE ? OR users.username LIKE ? OR users.first_name LIKE ? OR users.last_name LIKE ?)
		  AND (
			posts.user_id = ?
			OR (
				posts.privacy = 'public'
				-- a private account's posts are only for its followers, even the public ones
				AND (
					NOT EXISTS (
						SELECT 1 FROM profile AS author_profile
						WHERE author_profile.user_id = posts.user_id AND author_profile.is_private = 1
					)
					OR EXISTS (
						SELECT 1 FROM user_followers AS public_follows
						WHERE public_follows.follower_id = ? AND public_follows.target_id = posts.user_id AND public_follows.status = 1
					)
				)
			)
			OR (
				posts.privacy = 'followers'
				AND EXISTS (
					SELECT 1 FROM user_followers
					WHERE follower_id = ? AND target_id = posts.user_id AND status = 1
				)
			)
			OR (
				posts.privacy = 'selected'
				AND EXISTS (
					SELECT 1 FROM post_viewers
					WHERE post_id = posts.id AND viewer_id = ?
				)
			)
		  )
		ORDER BY posts.created_at DESC, posts.id DESC
		LIMIT 12
	`, pattern, pattern, pattern, pattern, currentUserID, currentUserID, currentUserID, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]any, 0)
	for rows.Next() {
		var id, authorID, comments, likes int
		var author, avatar, content, imagePath, privacy string
		var createdAt time.Time
		if err := rows.Scan(&id, &authorID, &author, &avatar, &content, &imagePath, &privacy, &createdAt, &comments, &likes); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{
			"id":           id,
			"authorId":     authorID,
			"author":       author,
			"avatarPath":   avatar,
			"content":      content,
			"imagePath":    imagePath,
			"privacy":      privacy,
			"createdAt":    createdAt,
			"commentCount": comments,
			"likeCount":    likes,
		})
	}

	return results, rows.Err()
}
