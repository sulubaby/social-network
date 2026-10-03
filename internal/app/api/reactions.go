package api

import (
	"errors"
	"log"
	"net/http"
	"social/database/notifications"
	"social/database/posts"
	"strconv"
)

func (app App) LikePost(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}
	
	postID, err := strconv.ParseInt(r.PathValue("postID"), 10, 64)
	if err != nil || postID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid post id",
		})
		return
	}

	var result posts.ReactionResult
	if r.Method == http.MethodPut {
		result, err = posts.LikePost(app.DB, userID, postID)
	} else if r.Method == http.MethodDelete {
		result, err = posts.UnlikePost(app.DB, userID, postID)
	} else {
		w.Header().Set("Allow", "PUT, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status":  false,
			"message": "method not allowed",
		})
		return
	}

	if errors.Is(err, posts.ErrPostNotVisible) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "post is not available",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not update post reaction",
		})
		return
	}

	if r.Method == http.MethodPut {
		app.notifyPostOwner(postID, userID, "post_like", "liked your post")
	} else {
		// unliking takes the alert back
		var ownerID int
		if err := app.DB.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&ownerID); err == nil {
			if err := notifications.DeleteFromActor(app.DB, ownerID, "posts", "post_like", userID, &postID); err != nil {
				log.Printf("remove like notification: %v", err)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    true,
		"liked":     result.Liked,
		"likeCount": result.LikeCount,
	})
}
