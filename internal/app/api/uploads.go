package api

import (
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"social/database/comments"
	"strings"
)

// A private post's image must have the same audience as the post itself.
func (app App) ServeUpload(w http.ResponseWriter, r *http.Request) {
	// the url looks like /uploads/posts/abc.png
	// we only allow "folder/file" with no extra slashes, so nobody can do ../ tricks
	userID, _ := r.Context().Value("userID").(int)
	name := strings.TrimPrefix(r.URL.Path, "/uploads/")
	parts := strings.Split(name, "/")
	if len(parts) != 2 || filepath.Base(parts[1]) != parts[1] || strings.Contains(parts[1], "\\") {
		http.NotFound(w, r)
		return
	}
	// avatars are for everyone. post and comment images are only for
	// people who can see that post. if not allowed we just say 404
	if parts[0] != "avatars" {
		switch parts[0] {
		case "posts":
			allowed, err := app.canViewPostUpload(userID, name)
			if err != nil || !allowed {
				http.NotFound(w, r)
				return
			}
		case "comments":
			allowed, err := app.canViewCommentUpload(userID, name)
			if err != nil || !allowed {
				http.NotFound(w, r)
				return
			}
		default:
			http.NotFound(w, r)
			return
		}
	}
	// dont cache private images and dont let the browser guess the file type
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.Join("uploads", parts[0], parts[1]))
}

// canViewPostUpload finds which post has this image and checks i can see it.
// if its not a normal post it can be a group post, then i must be a member
func (app App) canViewPostUpload(userID int, imagePath string) (bool, error) {
	var postID int64
	err := app.DB.QueryRow(`SELECT id FROM posts WHERE image_path = ?`, imagePath).Scan(&postID)
	if err == nil {
		return comments.CanViewPost(app.DB, userID, postID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	var allowed bool
	err = app.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM group_posts gp
			JOIN group_members gm ON gm.group_id = gp.group_id
			WHERE gp.image_path = ?
			  AND gm.user_id = ?
		)
	`, imagePath, userID).Scan(&allowed)
	return allowed, err
}

// canViewCommentUpload: a comment image follows the same rules as the post it is on
func (app App) canViewCommentUpload(userID int, imagePath string) (bool, error) {
	var postID int64
	err := app.DB.QueryRow(`SELECT post_id FROM comments WHERE image_path = ?`, imagePath).Scan(&postID)
	if err == nil {
		return comments.CanViewPost(app.DB, userID, postID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	// not a normal comment, maybe a comment in a group. then i must be a member
	var allowed bool
	err = app.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM group_post_comments gc
			JOIN group_posts gp ON gp.id = gc.post_id
			JOIN group_members gm ON gm.group_id = gp.group_id
			WHERE gc.image_path = ?
			  AND gm.user_id = ?
		)
	`, imagePath, userID).Scan(&allowed)
	return allowed, err
}
