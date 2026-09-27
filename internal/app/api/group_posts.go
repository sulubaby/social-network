// handlers for group posts and their comments. every handler first checks
// that you are a member of the group
package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"social/database/groupposts"
	"social/internal/helpers"
	"strconv"
	"strings"
)

// GetGroupPosts handles GET /api/groups/{id}/posts
func (app App) GetGroupPosts(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}

	posts, err := groupposts.ListPosts(app.DB, groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load group posts"})
		return
	}
	for i := range posts {
		posts[i].IsOwner = posts[i].UserID == userID
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "posts": posts})
}

// CreateGroupPost handles POST /api/groups/{id}/posts
// the post can have text, an image, or both
func (app App) CreateGroupPost(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}

	// read and check the form (text length + image type and size)
	form, err := parseGroupContentForm(w, r, maxPostBodySize, maxPostImageSize, 500, "post")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": err.Error()})
		return
	}
	if form.file != nil {
		defer form.file.Close()
	}

	// save the image if there is one
	imagePath := ""
	if form.file != nil {
		imagePath, err = helpers.SaveUploads(form.file, form.header, "post")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not save post image"})
			return
		}
	}

	post, err := groupposts.CreatePost(app.DB, groupID, userID, form.content, imagePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not create group post"})
		return
	}
	post.IsOwner = true
	writeJSON(w, http.StatusCreated, map[string]any{"status": true, "post": post})
}

// GetGroupPostComments handles GET /api/groups/{id}/posts/{postID}/comments (one page)
func (app App) GetGroupPostComments(w http.ResponseWriter, r *http.Request) {
	userID, groupID, postID, ok := groupPostRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	if !app.requireGroupPost(w, groupID, postID) {
		return
	}

	page, err := parsePage(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "limit must be between 1 and 50 and offset cannot be negative"})
		return
	}

	comments, err := groupposts.ListComments(app.DB, postID, page.Limit+1, page.Offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load group post comments"})
		return
	}
	comments, hasMore := trimPage(comments, page, true)
	for i := range comments {
		comments[i].IsOwner = comments[i].UserID == userID
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"comments":   comments,
		"hasMore":    hasMore,
		"nextOffset": page.Offset + len(comments),
	})
}

// CreateGroupPostComment handles POST on the same url. group comments are text only
func (app App) CreateGroupPostComment(w http.ResponseWriter, r *http.Request) {
	userID, groupID, postID, ok := groupPostRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	if !app.requireGroupPost(w, groupID, postID) {
		return
	}

	content, err := parseTextComment(w, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": err.Error()})
		return
	}

	comment, err := groupposts.CreateComment(app.DB, postID, userID, content, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not create group post comment"})
		return
	}
	comment.IsOwner = true
	writeJSON(w, http.StatusCreated, map[string]any{"status": true, "comment": comment})
}

// DeleteGroupPost handles DELETE /api/groups/{id}/posts/{postID}
// 404 if the post is not there, 403 if its not mine
func (app App) DeleteGroupPost(w http.ResponseWriter, r *http.Request) {
	userID, groupID, postID, ok := groupPostRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}

	err := groupposts.DeletePost(app.DB, groupID, postID, userID)
	if errors.Is(err, groupposts.ErrPostNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "group post not found"})
		return
	}
	if errors.Is(err, groupposts.ErrNotOwner) {
		writeJSON(w, http.StatusForbidden, map[string]any{"status": false, "message": "you can only delete your own group posts"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not delete group post"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "post deleted"})
}

// DeleteGroupPostComment handles DELETE .../comments/{commentID}, same rules as above
func (app App) DeleteGroupPostComment(w http.ResponseWriter, r *http.Request) {
	userID, groupID, postID, ok := groupPostRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	commentID, err := strconv.ParseInt(r.PathValue("commentID"), 10, 64)
	if err != nil || commentID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid comment id"})
		return
	}

	err = groupposts.DeleteComment(app.DB, groupID, postID, commentID, userID)
	if errors.Is(err, groupposts.ErrCommentNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "group post comment not found"})
		return
	}
	if errors.Is(err, groupposts.ErrNotOwner) {
		writeJSON(w, http.StatusForbidden, map[string]any{"status": false, "message": "you can only delete your own group post comments"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not delete group post comment"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "comment deleted"})
}

// groupRequestIdentity gets the logged in user id and the group id from the url.
// if something is wrong it already wrote the error, so the handler just returns
func groupRequestIdentity(w http.ResponseWriter, r *http.Request) (int, int64, bool) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok || userID <= 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": false, "message": "authentication required"})
		return 0, 0, false
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid group id"})
		return 0, 0, false
	}
	return userID, groupID, true
}

// same as groupRequestIdentity but also reads the post id
func groupPostRequestIdentity(w http.ResponseWriter, r *http.Request) (int, int64, int64, bool) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok {
		return 0, 0, 0, false
	}
	postID, err := strconv.ParseInt(r.PathValue("postID"), 10, 64)
	if err != nil || postID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid post id"})
		return 0, 0, 0, false
	}
	return userID, groupID, postID, true
}

// requireGroupMember sends 403 if the user is not in the group
func (app App) requireGroupMember(w http.ResponseWriter, groupID int64, userID int) bool {
	isMember, err := groupposts.IsMember(app.DB, groupID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not verify group membership"})
		return false
	}
	if !isMember {
		writeJSON(w, http.StatusForbidden, map[string]any{"status": false, "message": "group membership is required"})
		return false
	}
	return true
}

// requireGroupPost sends 404 if the post is not in this group
func (app App) requireGroupPost(w http.ResponseWriter, groupID, postID int64) bool {
	exists, err := groupposts.PostBelongsToGroup(app.DB, groupID, postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not verify group post"})
		return false
	}
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "group post not found"})
		return false
	}
	return true
}

// what we got from the post form: the text and maybe an image file
type groupContentForm struct {
	content string
	file    multipart.File
	header  *multipart.FileHeader
}

// parseGroupContentForm reads a multipart form and checks it:
// - body not too big
// - image (if any) is jpeg/png/gif and under the size limit
// - there is text or an image, and the text is not too long
// if it fails we close the file so it doesnt stay open
func parseGroupContentForm(w http.ResponseWriter, r *http.Request, maxBody, maxImage int64, maxText int, kind string) (groupContentForm, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseMultipartForm(maxBody); err != nil {
		return groupContentForm{}, errors.New(kind + " body is too large or invalid")
	}

	form := groupContentForm{content: strings.TrimSpace(r.FormValue("content"))}
	file, header, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		return groupContentForm{}, errors.New("invalid " + kind + " image upload")
	}
	if err == nil {
		form.file = file
		form.header = header
		if header.Size > maxImage {
			file.Close()
			return groupContentForm{}, errors.New(kind + " image must be smaller than 5 MB")
		}

		// check the real type from the first bytes of the file
		buffer := make([]byte, 512)
		read, readErr := file.Read(buffer)
		if readErr != nil && readErr != io.EOF {
			file.Close()
			return groupContentForm{}, errors.New("could not read " + kind + " image")
		}
		mimeType := http.DetectContentType(buffer[:read])
		if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
			file.Close()
			return groupContentForm{}, errors.New("only JPEG, PNG, and GIF images are allowed")
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			file.Close()
			return groupContentForm{}, errors.New("could not reset " + kind + " image")
		}
	}

	if (form.content == "" && form.file == nil) || len([]rune(form.content)) > maxText {
		if form.file != nil {
			form.file.Close()
		}
		return groupContentForm{}, errors.New(kind + " needs text or an image, with text limited to " + strconv.Itoa(maxText) + " characters")
	}
	return form, nil
}

// parseTextComment reads a json comment like {"content": "..."}
// and makes sure its 1 to 200 characters
func parseTextComment(w http.ResponseWriter, r *http.Request) (string, error) {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		return "", errors.New("comment images are not supported; send text only")
	}

	var input struct {
		Content string `json:"content"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCommentBodySize))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("invalid comment body")
	}

	content := strings.TrimSpace(input.Content)
	if content == "" || len([]rune(content)) > 200 {
		return "", errors.New("comment text must contain 1 to 200 characters")
	}
	return content, nil
}
