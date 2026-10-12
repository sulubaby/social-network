package posts

import (
	"database/sql"
	"social/internal/models"
)

// InsertReaction toggles a like or dislike, returns true when it was removed
func InsertReaction(db *sql.DB, reaction models.Reaction) (error, bool) {
	tx, err := db.Begin()

	if err != nil {
		return err, false
	}

	defer tx.Rollback()

	// same reaction again means the user wants to undo it
	result, err := tx.Exec(`
		DELETE FROM post_reactions
		WHERE post_id = ? AND user_id = ? AND value = ?
	`, reaction.PostID, reaction.UserID, reaction.Value)

	if err != nil {
		return err, false
	}

	removed, err := result.RowsAffected()

	if err != nil {
		return err, false
	}

	if removed > 0 {
		if err := tx.Commit(); err != nil {
			return err, false
		}

		return nil, true
	}

	// otherwise add it, or switch like to dislike with the upsert
	_, err = tx.Exec(`
		INSERT INTO post_reactions (post_id, user_id, value)
		VALUES (?, ?, ?)
		ON CONFLICT(post_id, user_id) DO UPDATE SET value = excluded.value
	`, reaction.PostID, reaction.UserID, reaction.Value)

	if err != nil {
		return err, false
	}

	if err := tx.Commit(); err != nil {
		return err, false
	}

	return nil, false
}
