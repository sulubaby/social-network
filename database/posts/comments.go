package posts

import (
	"database/sql"
	"errors"
)

var ErrCommentNotFound = errors.New("comment not found")

// DeleteComment removes a comment, but only if i wrote it.
// if nothing got deleted it means the comment is not there or its not mine.
// the comment_count on the post goes down by itself (database trigger)
func DeleteComment(db *sql.DB, userID int, postID, commentID int64) error {
	result, err := db.Exec(
		`DELETE FROM comments WHERE id = ? AND post_id = ? AND user_id = ?`,
		commentID, postID, userID,
	)
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

	return nil
}
