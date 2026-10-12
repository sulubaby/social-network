package posts

import (
	"database/sql"
	"log"
	"social/database/profiles"
	"social/database/users"
	"social/internal/models"
	"strconv"
	"strings"
)

// FilterPosts keeps only the posts of targetID that userID is allowed to see on their profile
func FilterPosts(db *sql.DB, posts *[]models.Post, userID, targetID int) ([]models.Post, error) {
	var returnPosts []models.Post

	// a private profile shows nothing to people who do not follow it (even the public posts)
	if userID != targetID {
		isPrivate, err := profiles.IsPrivate(db, targetID)
		if err != nil {
			return nil, err
		}
		if isPrivate {
			status, err := profiles.CheckFollower(db, userID, targetID)
			if err != nil || status != 1 {
				return []models.Post{}, nil
			}
		}
	}

	// old rows have no group_id and use the public and private columns
	for _, p := range *posts {
		if p.GroupId == nil {
			if p.Public == 1 {
				returnPosts = append(returnPosts, p)
				continue
			}

			if p.Private == 1 {
				isFollower, err := users.IsFollowing(db, userID, targetID)
				if err != nil {
					continue
				}

				if isFollower {
					returnPosts = append(returnPosts, p)
				}
			}

			continue
		}

		groupID := *p.GroupId

		// 0 is public
		if groupID == 0 {
			returnPosts = append(returnPosts, p)
			continue
		}

		if groupID == -1 {
			isFollower, err := profiles.CheckFollower(db, userID, targetID)
			// no row just means i do not follow them, the post is simply hidden
			if err == sql.ErrNoRows {
				continue
			}
			if err != nil {
				return nil, err
			}

			if isFollower == 1 {
				returnPosts = append(returnPosts, p)
			}

			continue
		}

		// anything else is a chosen list, users are saved like "1:5:9"
		log.Println("group post")
		var users string

		// the list has to belong to the author of the post
		err := db.QueryRow(
			`SELECT users FROM user_posts_groups WHERE id = ? AND user_id = ?`,
			groupID, p.UserId,
		).Scan(&users)

		log.Printf("users %s", users)
		if err != nil {
			log.Println("hereerr")
			if err == sql.ErrNoRows {
				continue
			}

			return nil, err
		}

		isMember := false

		log.Println(strings.Split(users, ":"))
		log.Println(userID)
		for _, idStr := range strings.Split(users, ":") {
			if idStr == "" {
				continue
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				return nil, err
			}

			if id == userID {
				isMember = true
				break
			}
		}

		// and the chosen person must still follow the author
		if isMember {
			following, err := profiles.CheckFollower(db, userID, p.UserId)
			if err == nil && following == 1 {
				returnPosts = append(returnPosts, p)
			}
		}
	}

	return returnPosts, nil
}
