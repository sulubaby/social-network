// handlers for comments on normal posts: list them, add one (text and/or image), delete one
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"social/database/comments"
	"social/database/notifications"
	"social/database/posts"
	"social/internal/helpers"
	"social/internal/models"
	"strconv"
	"strings"
)

var (
	ErrPostNotVisible  = errors.New("post not visible")
	ErrCommentNotFound = errors.New("comment not found")
)

// size limits: json body 4KB, whole upload 6MB, the image itself 5MB
const maxCommentBodySize = 4 << 10
const maxCommentMultipartSize = 6 << 20
const maxCommentImageSize = 5 * 1024 * 1024

// Comments handles /api/posts/{postID}/comments
// GET = list the comments, POST = add a comment
func (app App) Comments(w http.ResponseWriter, r *http.Request) {
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

	// same url, different job depending on the method
	switch r.Method {
	case http.MethodGet:
		app.listComments(w, r, userID, postID)
	case http.MethodPost:
		app.createComment(w, r, userID, postID)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status":  false,
			"message": "method not allowed",
		})
	}
}

// listComments sends back one page of comments for a post
func (app App) listComments(w http.ResponseWriter, r *http.Request, userID int, postID int64) {
	page, err := parsePage(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "limit must be between 1 and 50 and offset cannot be negative",
		})
		return
	}

	result, err := comments.ListComments(app.DB, userID, postID, page.Limit+1, page.Offset)
	if errors.Is(err, comments.ErrPostNotVisible) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "post is not available",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not load comments",
		})
		return
	}
	result, hasMore := trimPage(result, page, true)
	// mark which comments are mine so the frontend shows the delete menu
	for i := range result {
		result[i].Own = result[i].UserID == userID
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"comments":   result,
		"hasMore":    hasMore,
		"nextOffset": page.Offset + len(result),
	})
}

// createComment adds a new comment.
// if the request is multipart it can have an image, if not its just json text
func (app App) createComment(w http.ResponseWriter, r *http.Request, userID int, postID int64) {
	var request models.CreateCommentRequest
	var imagePath string

	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, maxCommentMultipartSize)
		if err := r.ParseMultipartForm(maxCommentMultipartSize); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "request body too large",
			})
			return
		}

		// the text part of the form
		request.Content = r.FormValue("content")

		// the image is optional, ErrMissingFile just means no image was sent
		imageFile, imageHeader, fileErr := r.FormFile("image")
		if fileErr != nil && fileErr != http.ErrMissingFile {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid image upload",
			})
			return
		}

		if fileErr == nil {
			defer imageFile.Close()

			if imageHeader.Size > maxCommentImageSize {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  false,
					"message": "image must be smaller than 5 MB",
				})
				return
			}

			// check the real file type from the first bytes, not from the file name.
			// someone can rename a .exe to .png so we dont trust the name
			fileBytes := make([]byte, 512)
			bytesRead, readErr := imageFile.Read(fileBytes)

			if readErr != nil && readErr != io.EOF {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  false,
					"message": "could not read image",
				})
				return
			}

			contentType := http.DetectContentType(fileBytes[:bytesRead])

			if contentType != "image/jpeg" &&
				contentType != "image/png" &&
				contentType != "image/gif" {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  false,
					"message": "only JPEG, PNG, and GIF images are allowed",
				})
				return
			}

			// go back to the start of the file before saving it
			if _, seekErr := imageFile.Seek(0, io.SeekStart); seekErr != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  false,
					"message": "could not reset image",
				})
				return
			}

			// save the image in the uploads folder and keep the path
			savedPath, saveErr := helpers.SaveUploads(imageFile, imageHeader, "comment")
			if saveErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"status":  false,
					"message": "could not save image",
				})
				return
			}

			imagePath = savedPath
		}
	} else {
		// no image, so the body is normal json like {"content": "hi"}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCommentBodySize))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid comment body",
			})
			return
		}
	}

	// a comment needs text or an image (or both), and text max 200 chars
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" && imagePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "comment must contain text, an image, or both",
		})
		return
	}
	if len([]rune(request.Content)) > 200 {
		// the picture was already saved above, we dont need it anymore
		helpers.DeleteUpload(imagePath)
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "comment text must contain 1 to 200 characters",
		})
		return
	}

	// save it. this also checks i am allowed to see the post
	comment, err := comments.CreateComment(app.DB, userID, postID, request.Content, imagePath)
	if err != nil {
		// the comment was not saved (post hidden or db error), so remove its picture
		helpers.DeleteUpload(imagePath)
	}
	if errors.Is(err, comments.ErrPostNotVisible) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "post is not available",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not create comment",
		})
		return
	}

	comment.Own = true
	app.notifyPostOwner(postID, userID, "post_comment", "commented on your post")

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  true,
		"comment": comment,
	})
}

// DeleteComment handles DELETE /api/posts/{postID}/comments/{commentID}
// only the person who wrote the comment can delete it
func (app App) DeleteComment(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := strconv.ParseInt(r.PathValue("commentID"), 10, 64)
	if err != nil || commentID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid comment id",
		})
		return
	}

	err = posts.DeleteComment(app.DB, userID, postID, commentID)
	if errors.Is(err, posts.ErrCommentNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "comment not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not delete comment",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "comment deleted",
	})
}

// notifyPostOwner tells the author of a post that someone liked or commented on it
func (app App) notifyPostOwner(postID int64, actorID int, notificationType, action string) {
	var ownerID int
	if err := app.DB.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&ownerID); err != nil {
		log.Printf("find post owner: %v", err)
		return
	}
	if ownerID == actorID {
		return
	}
	// one alert per person and post, so liking twice does not stack up
	if err := notifications.DeleteFromActor(app.DB, ownerID, "posts", notificationType, actorID, &postID); err != nil {
		log.Printf("clear old post notification: %v", err)
	}
	app.notify(ownerID, actorID, "posts", notificationType, app.userFullName(actorID)+" "+action, &postID)
}
