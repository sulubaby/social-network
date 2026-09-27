package posts

import "database/sql"

// DeletePost deletes a post only if it belongs to this user.
// comments, reactions and viewers get deleted with it (ON DELETE CASCADE)
func DeletePost(db *sql.DB, postID, userID int) error {
	_, err := db.Exec(`
		DELETE FROM posts
			WHERE id = ? AND user_id = ?
	`, postID, userID)
	return err
}