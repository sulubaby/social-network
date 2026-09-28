package posts

import (
	"database/sql"
	"errors"
)

var ErrPostNotVisible = errors.New("post is not available")

type ReactionResult struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"likeCount"`
}

// LikePost adds a like for a visible post. Repeating the request is safe.
func LikePost(db *sql.DB, userID int, postID int64) (ReactionResult, error) {
	tx, err := db.Begin()
	if err != nil {
		return ReactionResult{}, err
	}
	defer tx.Rollback()

	visible, err := canViewPost(tx, userID, postID)
	if err != nil {
		return ReactionResult{}, err
	}
	if !visible {
		return ReactionResult{}, ErrPostNotVisible
	}

	var currentValue int
	err = tx.QueryRow(`
		SELECT value
		FROM post_reactions
		WHERE user_id = ? AND post_id = ?
	`, userID, postID).Scan(&currentValue)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.Exec(`
			INSERT INTO post_reactions (user_id, post_id, value)
			VALUES (?, ?, 1)
		`, userID, postID)
	case err != nil:
		return ReactionResult{}, err
	case currentValue != 1:
		// Delete then insert so the existing triggers adjust both counts.
		if _, err = tx.Exec(`DELETE FROM post_reactions WHERE user_id = ? AND post_id = ?`, userID, postID); err == nil {
			_, err = tx.Exec(`
				INSERT INTO post_reactions (user_id, post_id, value)
				VALUES (?, ?, 1)
			`, userID, postID)
		}
	}
	if err != nil {
		return ReactionResult{}, err
	}

	result, err := reactionResult(tx, postID, true)
	if err != nil {
		return ReactionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReactionResult{}, err
	}
	return result, nil
}

// UnlikePost removes the current user's like. Repeating the request is safe.
func UnlikePost(db *sql.DB, userID int, postID int64) (ReactionResult, error) {
	tx, err := db.Begin()
	if err != nil {
		return ReactionResult{}, err
	}
	defer tx.Rollback()

	visible, err := canViewPost(tx, userID, postID)
	if err != nil {
		return ReactionResult{}, err
	}
	if !visible {
		return ReactionResult{}, ErrPostNotVisible
	}

	if _, err := tx.Exec(`
		DELETE FROM post_reactions
		WHERE user_id = ? AND post_id = ? AND value = 1
	`, userID, postID); err != nil {
		return ReactionResult{}, err
	}

	result, err := reactionResult(tx, postID, false)
	if err != nil {
		return ReactionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReactionResult{}, err
	}
	return result, nil
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

func canViewPost(db queryRower, viewerID int, postID int64) (bool, error) {
	var visible bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM posts
			WHERE posts.id = ?
			AND posts.group_id IS NULL
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
		)
	`, postID, viewerID, viewerID, viewerID, viewerID).Scan(&visible)
	return visible, err
}

func reactionResult(db queryRower, postID int64, liked bool) (ReactionResult, error) {
	var likeCount int
	err := db.QueryRow(`SELECT like_count FROM posts WHERE id = ?`, postID).Scan(&likeCount)
	if err != nil {
		return ReactionResult{}, err
	}
	return ReactionResult{Liked: liked, LikeCount: likeCount}, nil
}
