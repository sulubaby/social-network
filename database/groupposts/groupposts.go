// database functions for posts and comments inside groups
package groupposts

import (
	"database/sql"
	"errors"
	"social/internal/models"
	"strings"
)

// errors we send back so the handler knows which status code to use
var (
	ErrPostNotFound    = errors.New("group post not found")
	ErrCommentNotFound = errors.New("group post comment not found")
	ErrNotOwner        = errors.New("group content belongs to another user")
)

// IsMember checks if the user is in the group. only members can see or post
func IsMember(db *sql.DB, groupID int64, userID int) (bool, error) {
	var isMember bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = ? AND user_id = ?
		)
	`, groupID, userID).Scan(&isMember)
	return isMember, err
}

// PostBelongsToGroup makes sure the post is really in this group
// (so you cant use a group you are in to reach a post from another group)
func PostBelongsToGroup(db *sql.DB, groupID, postID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM group_posts
			WHERE id = ? AND group_id = ?
		)
	`, postID, groupID).Scan(&exists)
	return exists, err
}

// CreatePost saves a new group post and returns it with the author info
func CreatePost(db *sql.DB, groupID int64, userID int, content, imagePath string) (models.GroupPost, error) {
	content = strings.TrimSpace(content)
	result, err := db.Exec(`
		INSERT INTO group_posts (group_id, user_id, content, image_path)
		VALUES (?, ?, ?, ?)
	`, groupID, userID, content, imagePath)
	if err != nil {
		return models.GroupPost{}, err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		return models.GroupPost{}, err
	}
	return getPost(db, groupID, postID)
}

// ListPosts gets the posts of a group, newest first.
// pagination is optional: [limit, offset]
func ListPosts(db *sql.DB, groupID int64, pagination ...int) ([]models.GroupPost, error) {
	query := groupPostSelect + `
		WHERE gp.group_id = ?
		ORDER BY gp.created_at DESC, gp.id DESC
	`
	args := []any{groupID}
	if len(pagination) >= 2 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pagination[0], pagination[1])
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []models.GroupPost{}
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

// CreateComment adds a comment on a group post. group comments are text only (1 to 200 chars)
func CreateComment(db *sql.DB, postID int64, userID int, content, imagePath string) (models.GroupPostComment, error) {
	content = strings.TrimSpace(content)
	if imagePath != "" {
		return models.GroupPostComment{}, errors.New("comment images are not supported")
	}
	if content == "" || len([]rune(content)) > 200 {
		return models.GroupPostComment{}, errors.New("comment text must contain 1 to 200 characters")
	}
	result, err := db.Exec(`
		INSERT INTO group_post_comments (post_id, user_id, content, image_path)
		VALUES (?, ?, ?, ?)
	`, postID, userID, content, imagePath)
	if err != nil {
		return models.GroupPostComment{}, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return models.GroupPostComment{}, err
	}
	return getComment(db, postID, commentID)
}

// ListComments gets the comments of a group post, oldest first.
// pagination is optional: [limit, offset]
func ListComments(db *sql.DB, postID int64, pagination ...int) ([]models.GroupPostComment, error) {
	query := groupCommentSelect + `
		WHERE gc.post_id = ?
		ORDER BY gc.created_at ASC, gc.id ASC
	`
	args := []any{postID}
	if len(pagination) >= 2 {
		query += `LIMIT ? OFFSET ?`
		args = append(args, pagination[0], pagination[1])
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []models.GroupPostComment{}
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

// DeletePost deletes a group post if i am the owner.
// we use a transaction so the check and the delete happen together
func DeletePost(db *sql.DB, groupID, postID int64, userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var ownerID int
	// first find who wrote the post
	err = tx.QueryRow(`
		SELECT user_id
		FROM group_posts
		WHERE id = ? AND group_id = ?
	`, postID, groupID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPostNotFound
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrNotOwner
	}

	result, err := tx.Exec(`
		DELETE FROM group_posts
		WHERE id = ? AND group_id = ? AND user_id = ?
	`, postID, groupID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrPostNotFound
	}
	return tx.Commit()
}

// DeleteComment deletes a comment on a group post if i wrote it.
// same idea as DeletePost: check the owner first, then delete
func DeleteComment(db *sql.DB, groupID, postID, commentID int64, userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var ownerID int
	err = tx.QueryRow(`
		SELECT gc.user_id
		FROM group_post_comments gc
		JOIN group_posts gp ON gp.id = gc.post_id
		WHERE gc.id = ?
		  AND gc.post_id = ?
		  AND gp.group_id = ?
	`, commentID, postID, groupID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCommentNotFound
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrNotOwner
	}

	result, err := tx.Exec(`
		DELETE FROM group_post_comments
		WHERE id = ?
		  AND post_id = ?
		  AND user_id = ?
		  AND EXISTS (
			SELECT 1
			FROM group_posts
			WHERE id = ? AND group_id = ?
		  )
	`, commentID, postID, userID, postID, groupID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCommentNotFound
	}
	return tx.Commit()
}

// base SELECT for group posts, with the author info and how many comments it has
const groupPostSelect = `
	SELECT
		gp.id,
		gp.group_id,
		gp.user_id,
		COALESCE(u.username, ''),
		u.first_name,
		u.last_name,
		COALESCE(p.avatar_path, ''),
		gp.content,
		gp.image_path,
		gp.created_at,
		(SELECT COUNT(*) FROM group_post_comments gc WHERE gc.post_id = gp.id)
	FROM group_posts gp
	JOIN user u ON u.id = gp.user_id
	LEFT JOIN profile p ON p.user_id = u.id
`

// getPost gets one post from a group
func getPost(db *sql.DB, groupID, postID int64) (models.GroupPost, error) {
	post, err := scanPost(db.QueryRow(groupPostSelect+`
		WHERE gp.id = ? AND gp.group_id = ?
	`, postID, groupID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.GroupPost{}, ErrPostNotFound
	}
	return post, err
}

// lets scanPost and scanComment work with QueryRow and Query rows
type rowScanner interface {
	Scan(dest ...any) error
}

// scanPost reads one row into a GroupPost
func scanPost(row rowScanner) (models.GroupPost, error) {
	var post models.GroupPost
	err := row.Scan(
		&post.ID,
		&post.GroupID,
		&post.UserID,
		&post.Username,
		&post.FirstName,
		&post.LastName,
		&post.AvatarPath,
		&post.Content,
		&post.ImagePath,
		&post.CreatedAt,
		&post.CommentCount,
	)
	return post, err
}

// base SELECT for group comments with the author info
const groupCommentSelect = `
	SELECT
		gc.id,
		gc.post_id,
		gc.user_id,
		COALESCE(u.username, ''),
		u.first_name,
		u.last_name,
		COALESCE(p.avatar_path, ''),
		gc.content,
		gc.image_path,
		gc.created_at
	FROM group_post_comments gc
	JOIN user u ON u.id = gc.user_id
	LEFT JOIN profile p ON p.user_id = u.id
`

// getComment gets one comment of a post
func getComment(db *sql.DB, postID, commentID int64) (models.GroupPostComment, error) {
	return scanComment(db.QueryRow(groupCommentSelect+`
		WHERE gc.id = ? AND gc.post_id = ?
	`, commentID, postID))
}

// scanComment reads one row into a GroupPostComment
func scanComment(row rowScanner) (models.GroupPostComment, error) {
	var comment models.GroupPostComment
	err := row.Scan(
		&comment.ID,
		&comment.PostID,
		&comment.UserID,
		&comment.Username,
		&comment.FirstName,
		&comment.LastName,
		&comment.AvatarPath,
		&comment.Content,
		&comment.ImagePath,
		&comment.CreatedAt,
	)
	return comment, err
}
