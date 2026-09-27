// database functions for comments on normal (non group) posts
package comments

import (
	"database/sql"
	"errors"
	"social/internal/models"
	"strings"
)

// returned when the user is not allowed to see the post
var ErrPostNotVisible = errors.New("post is not available")

// CreateComment saves a new comment and returns it with the author info.
// a comment can be text, an image, or both. text is max 200 characters
func CreateComment(db *sql.DB, userID int, postID int64, content, imagePath string) (models.Comment, error) {
	content = strings.TrimSpace(content)
	if (content == "" && imagePath == "") || len([]rune(content)) > 200 {
		return models.Comment{}, errors.New("comment must contain 1 to 200 characters of text, an image, or both")
	}

	// you cant comment on a post you are not allowed to see
	canView, err := CanViewPost(db, userID, postID)
	if err != nil {
		return models.Comment{}, err
	}
	if !canView {
		return models.Comment{}, ErrPostNotVisible
	}

	result, err := db.Exec(`
		INSERT INTO comments (user_id, post_id, content, image_path)
		VALUES (?, ?, ?, ?)
	`, userID, postID, content, imagePath)
	if err != nil {
		return models.Comment{}, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return models.Comment{}, err
	}

	return GetCommentByID(db, commentID)
}

// GetCommentByID gets one comment with the author name and avatar
func GetCommentByID(db *sql.DB, commentID int64) (models.Comment, error) {
	var comment models.Comment

	err := db.QueryRow(commentQuery+" WHERE comments.id = ?", commentID).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.UserID,
		&comment.Author,
		&comment.AvatarPath,
		&comment.Content,
		&comment.ImagePath,
		&comment.CreatedAt,
	)

	return comment, err
}

// ListComments returns one page of comments. Keeping pagination here prevents
// a popular post from forcing every comment into one database response.
func ListComments(db *sql.DB, userID int, postID int64, pagination ...int) ([]models.Comment, error) {
	canView, err := CanViewPost(db, userID, postID)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrPostNotVisible
	}

	query := commentQuery + " WHERE comments.post_id = ? ORDER BY comments.created_at ASC, comments.id ASC"
	args := []any{postID}
	if len(pagination) >= 2 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, pagination[0], pagination[1])
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.Comment{}
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Author,
			&comment.AvatarPath,
			&comment.Content,
			&comment.ImagePath,
			&comment.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// CanViewPost checks if a user can see a post using the privacy rules:
// my own post, public post, followers post (i must follow them),
// or selected post (i must be in post_viewers)
func CanViewPost(db *sql.DB, viewerID int, postID int64) (bool, error) {
	var canView bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM posts
			WHERE posts.id = ?
			AND posts.group_id IS NULL
			AND (
				posts.user_id = ?
				OR posts.privacy = 'public'
				OR (
					posts.privacy = 'followers'
					AND EXISTS (
						SELECT 1
						FROM user_followers
						WHERE follower_id = ?
						AND target_id = posts.user_id
						AND status = 1
					)
				)
				OR (
					posts.privacy = 'selected'
					AND EXISTS (
						SELECT 1
						FROM post_viewers
						WHERE post_id = posts.id
						AND viewer_id = ?
					)
				)
			)
		)
	`, postID, viewerID, viewerID, viewerID).Scan(&canView)

	return canView, err
}

// base SELECT for comments. if the user has no username we show first + last name
const commentQuery = `
	SELECT
		comments.id,
		comments.post_id,
		comments.user_id,
		COALESCE(users.username, users.first_name || ' ' || users.last_name),
		COALESCE(profile.avatar_path, ''),
		comments.content,
		comments.image_path,
		comments.created_at
	FROM comments
	JOIN user AS users ON users.id = comments.user_id
	LEFT JOIN profile ON profile.user_id = comments.user_id
`
