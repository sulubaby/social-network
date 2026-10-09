package posts

import (
	"database/sql"
	"social/internal/models"
)

/*
FillRelationships tells each post how the viewer is connected to its author,
so the post can show a follow button like on instagram:

  - "self"      my own post (no button)
  - "friend"    we follow each other
  - "following" i follow them
  - "requested" i asked to follow a private account and wait for an answer
  - "none"      i do not follow them yet (show "Follow")
*/
func FillRelationships(db *sql.DB, viewerID int, posts []models.Post) {
	cache := map[int]string{}

	for i := range posts {
		authorID := posts[i].UserId

		if relationship, ok := cache[authorID]; ok {
			posts[i].Relationship = relationship
			continue
		}

		relationship := relationshipWith(db, viewerID, authorID)
		cache[authorID] = relationship
		posts[i].Relationship = relationship
	}
}

func relationshipWith(db *sql.DB, viewerID, authorID int) string {
	if viewerID == authorID {
		return "self"
	}

	var mine, theirs sql.NullInt64

	// my follow of them, and their follow of me (status 1 = accepted, 0 = request)
	err := db.QueryRow(`
		SELECT
			(SELECT status FROM user_followers WHERE follower_id = ? AND target_id = ?),
			(SELECT status FROM user_followers WHERE follower_id = ? AND target_id = ?)
	`, viewerID, authorID, authorID, viewerID).Scan(&mine, &theirs)
	if err != nil {
		return "none"
	}

	switch {
	case !mine.Valid:
		return "none"
	case mine.Int64 == 0:
		return "requested"
	case theirs.Valid && theirs.Int64 == 1:
		return "friend"
	default:
		return "following"
	}
}
