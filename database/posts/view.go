package posts

import (
	"database/sql"
	"social/database/users"
	"strconv"
	"strings"
)

/*
CanView says if a user is allowed to see a post (and so like it, comment on it,
open it or share it).

  - my own post: yes
  - the author's profile is private and i do not follow them: no
  - public post (public = 1): yes
  - followers post (private = 1): only accepted followers
  - post for a chosen list of people (group_id): only people on that list
*/
func CanView(db *sql.DB, userID, postID int) (bool, error) {
	var authorID int
	var private int
	var public int
	var groupID sql.NullInt64
	var authorPrivate int

	err := db.QueryRow(`
		SELECT p.user_id, p.public, p.private, p.group_id, COALESCE(pr.is_private, 0)
		FROM posts p
		LEFT JOIN profile pr ON pr.user_id = p.user_id
		WHERE p.id = ?
	`, postID).Scan(
		&authorID,
		&public,
		&private,
		&groupID,
		&authorPrivate,
	)

	if err != nil {
		return false, err
	}

	if authorID == userID {
		return true, nil
	}

	isFollower, err := users.IsFollowing(db, userID, authorID)
	if err != nil {
		return false, err
	}

	if authorPrivate == 1 && !isFollower {
		return false, nil
	}

	if public == 1 {
		return true, nil
	}

	if private == 1 {
		return isFollower, nil
	}

	if groupID.Valid {
		var usersArray string

		err := db.QueryRow(`
			SELECT users
			FROM user_posts_groups
			WHERE id = ?
		`, groupID.Int64).Scan(&usersArray)

		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		userIDString := strconv.Itoa(userID)

		for _, id := range strings.Split(usersArray, ":") {
			if id == userIDString {
				return true, nil
			}
		}
	}

	return false, nil
}
