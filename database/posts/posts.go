package posts

import (
	"database/sql"
	"errors"
	"fmt"
	"social/internal/models"
	"strconv"
	"strings"
)

// the location text cant be longer than this
const maxLocationLength = 200

// errors for the "selected followers" privacy option
var (
	ErrSelectedFollowersRequired = errors.New("select at least one follower")
	ErrInvalidPostViewer         = errors.New("selected viewers must follow the post author")
)

// CreatePost saves a new post. for "selected" posts it also saves who can see it.
// its all in one transaction, so if one step fails nothing gets saved
func CreatePost(db *sql.DB, userID int, request models.CreatePostRequest) (models.Post, error) {
	tx, err := db.Begin()
	if err != nil {
		return models.Post{}, err
	}
	defer tx.Rollback()

	// if the privacy is selected, every picked person must really follow me
	selectedIDs := uniqueIDs(request.SelectedFollowerIDs)
	if request.Privacy == models.PostPrivacySelected {
		if len(selectedIDs) == 0 {
			return models.Post{}, ErrSelectedFollowersRequired
		}

		for _, viewerID := range selectedIDs {
			var followsAuthor int
			err = tx.QueryRow(`
				SELECT COUNT(*)
				FROM user_followers AS follows
				WHERE follower_id = ? AND target_id = ? AND status = 1
			`, viewerID, userID).Scan(&followsAuthor)
			if err != nil {
				return models.Post{}, err
			}
			if followsAuthor == 0 {
				return models.Post{}, ErrInvalidPostViewer
			}
		}
	}

	// save the post itself (group_id is NULL because this is not a group post)
	result, err := tx.Exec(`
	INSERT INTO posts (type, title, content, image_path, user_id, group_id, privacy, location)
	VALUES ('post', '', ?, ?, ?, NULL, ?, ?)
	`, request.Content, request.ImagePath, userID, request.Privacy, nullableText(request.Location))
	if err != nil {
		return models.Post{}, err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		return models.Post{}, err
	}

	// save the list of people allowed to see this post
	if request.Privacy == models.PostPrivacySelected {
		for _, viewerID := range selectedIDs {
			_, err = tx.Exec(`
				INSERT INTO post_viewers (post_id, viewer_id)
				VALUES (?, ?)
			`, postID, viewerID)
			if err != nil {
				return models.Post{}, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return models.Post{}, err
	}

	return GetPostByID(db, postID)
}

// GetPostByID gets one post with the author name and avatar
func GetPostByID(db *sql.DB, postID int64) (models.Post, error) {
	var post models.Post

	err := db.QueryRow(`
		SELECT
			posts.id,
			posts.user_id,
			COALESCE(users.username, users.first_name || ' ' || users.last_name),
			COALESCE(profile.avatar_path, ''),
			posts.content,
			posts.image_path,
			posts.privacy,
			COALESCE(posts.location, ''),
			posts.created_at,
			posts.like_count,
			posts.comment_count
		FROM posts
		JOIN user AS users ON users.id = posts.user_id
		LEFT JOIN profile ON profile.user_id = users.id
		WHERE posts.id = ?
	`, postID).Scan(
		&post.ID,
		&post.UserID,
		&post.Author,
		&post.AvatarPath,
		&post.Content,
		&post.ImagePath,
		&post.Privacy,
		&post.Location,
		&post.CreatedAt,
		&post.LikeCount,
		&post.CommentCount,
	)

	return post, err
}

// ListFeedPosts builds the home feed for the logged in user.
// a post shows up if: its mine, or its public, or its for followers and i follow
// the author, or its selected and i am in the list.
// liked tells the frontend if i already liked the post
func ListFeedPosts(db *sql.DB, viewerID int, pagination ...int) ([]models.Post, error) {
	query := `
		SELECT
			posts.id,
			posts.user_id,
			COALESCE(users.username, users.first_name || ' ' || users.last_name),
			COALESCE(profile.avatar_path, ''),
			posts.content,
			posts.image_path,
			posts.privacy,
			COALESCE(posts.location, ''),
			posts.created_at,
			posts.like_count,
			EXISTS (
				SELECT 1
				FROM post_reactions AS viewer_reactions
				WHERE viewer_reactions.post_id = posts.id
				AND viewer_reactions.user_id = ?
				AND viewer_reactions.value = 1
			) AS liked,
			posts.comment_count
		FROM posts
		JOIN user AS users ON users.id = posts.user_id
		LEFT JOIN profile ON profile.user_id = users.id
		WHERE posts.group_id IS NULL
		AND (
			posts.user_id = ?
			OR posts.privacy = 'public'
			OR (
				posts.privacy = 'followers'
				AND EXISTS (
					SELECT 1
					FROM user_followers AS follows
					WHERE follows.follower_id = ?
					AND follows.target_id = posts.user_id
					AND follows.status = 1
				)
			)
			OR (
				posts.privacy = 'selected'
				AND EXISTS (
					SELECT 1
					FROM post_viewers
					WHERE post_viewers.post_id = posts.id
					AND post_viewers.viewer_id = ?
				)
			)
		)
		ORDER BY posts.created_at DESC, posts.id DESC
	`
	args := []any{viewerID, viewerID, viewerID, viewerID}
	if len(pagination) >= 2 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pagination[0], pagination[1])
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []models.Post{}
	for rows.Next() {
		var post models.Post
		err = rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Author,
			&post.AvatarPath,
			&post.Content,
			&post.ImagePath,
			&post.Privacy,
			&post.Location,
			&post.CreatedAt,
			&post.LikeCount,
			&post.Liked,
			&post.CommentCount,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// uniqueIDs removes duplicates and bad ids (0 or less) from a list
func uniqueIDs(ids []int) []int {
	seen := make(map[int]bool)
	unique := make([]int, 0, len(ids))

	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}

	return unique
}

// IsPostPrivacy checks the privacy is public, followers or selected
func IsPostPrivacy(value string) bool {
	switch value {
	case models.PostPrivacyPublic, models.PostPrivacyFollowers, models.PostPrivacySelected:
		return true
	default:
		return false
	}
}

// ValidateSelectedIDs: selected privacy needs at least one person,
// and the other privacy types shouldnt send a list at all
func ValidateSelectedIDs(privacy string, ids []int) error {
	if privacy == models.PostPrivacySelected && len(uniqueIDs(ids)) == 0 {
		return ErrSelectedFollowersRequired
	}
	if privacy != models.PostPrivacySelected && len(ids) > 0 {
		return fmt.Errorf("selected followers are only allowed for selected privacy")
	}
	return nil
}

// nullableText saves NULL in the database instead of an empty string
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// IsValidLocation checks the location looks like "name:lat:lon".
// empty is fine because location is optional
func IsValidLocation(value string) bool {
	if value == "" {
		return true
	}
	if len([]rune(value)) > maxLocationLength {
		return false
	}

	parts := strings.Split(value, ":")
	if len(parts) != 3 || strings.TrimSpace(parts[0]) == "" {
		return false
	}

	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || !(lat >= -90 && lat <= 90) {
		return false
	}

	lon, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || !(lon >= -180 && lon <= 180) {
		return false
	}

	return true
}