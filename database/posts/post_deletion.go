package posts

import (
	"database/sql"
	"errors"
)

var ErrPostNotFound = errors.New("post not found")

// DeletePost deletes a post only if it belongs to this user.
// comments, reactions and viewers get deleted with it (ON DELETE CASCADE).
// it gives back the image paths of the post and its comments so the handler
// can delete the files too
func DeletePost(db *sql.DB, postID, userID int) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var postImage string
	err = tx.QueryRow(`
		SELECT COALESCE(image_path, '')
		FROM posts
		WHERE id = ? AND user_id = ? AND group_id IS NULL
	`, postID, userID).Scan(&postImage)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}

	images := []string{}
	if postImage != "" {
		images = append(images, postImage)
	}

	rows, err := tx.Query(`
		SELECT image_path
		FROM comments
		WHERE post_id = ? AND COALESCE(image_path, '') <> ''
	`, postID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			rows.Close()
			return nil, err
		}
		images = append(images, path)
	}
	rows.Close()

	if _, err = tx.Exec(`DELETE FROM posts WHERE id = ? AND user_id = ?`, postID, userID); err != nil {
		return nil, err
	}

	return images, tx.Commit()
}
