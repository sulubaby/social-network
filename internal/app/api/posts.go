package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"social/database/posts"
	"social/internal/helpers"
	"social/internal/models"
	"strconv"
	"strings"
)

// max size for the whole post request and for the image
const maxPostBodySize = 6 << 20
const maxPostImageSize = 5 * 1024 * 1024

// CreatePost handles POST /api/posts
// the frontend sends a multipart form: content, privacy, location,
// selectedFollowerIds (json list) and maybe an image
func (app App) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPostBodySize)
	if err = r.ParseMultipartForm(maxPostBodySize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "request body too large",
		})
		return
	}

	// read the text fields from the form
	request := models.CreatePostRequest{
		Content:  r.FormValue("content"),
		Privacy:  r.FormValue("privacy"),
		Location: strings.TrimSpace(r.FormValue("location")),
	}
	selectedIDs := r.FormValue("selectedFollowerIds")
	if selectedIDs != "" {
		if err := json.Unmarshal([]byte(selectedIDs), &request.SelectedFollowerIDs); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid selected followers",
			})
			return
		}
	}

	// the image is optional
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

		if imageHeader.Size > maxPostImageSize {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "image must be smaller than 5 MB",
			})
			return
		}

		// check the real file type from the first bytes (only jpeg, png, gif)
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

		if _, seekErr := imageFile.Seek(0, io.SeekStart); seekErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "could not reset image",
			})
			return
		}
	}

	// check the text, location, privacy and selected followers before saving anything
	// a post needs text, a picture or both. text is max 500 characters
	request.Content = strings.TrimSpace(request.Content)
	if (request.Content == "" && fileErr != nil) || len([]rune(request.Content)) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "a post needs text or an image, and text can have up to 500 characters",
		})
		return
	}
	if !posts.IsValidLocation(request.Location) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid location",
		})
		return
	}
	if !posts.IsPostPrivacy(request.Privacy) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "privacy must be public, followers, or selected",
		})
		return
	}
	if err = posts.ValidateSelectedIDs(request.Privacy, request.SelectedFollowerIDs); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	// everything is valid, now we can save the image
	if fileErr == nil {
		imagePath, saveErr := helpers.SaveUploads(imageFile, imageHeader, "post")
		if saveErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not save image",
			})
			return
		}

		request.ImagePath = imagePath
	}

	// save the post in the database
	post, err := posts.CreatePost(app.DB, userID, request)
	if err != nil && request.ImagePath != "" {
		// the post was not saved, so dont keep its picture on the disk
		helpers.DeleteUpload(request.ImagePath)
	}
	if errors.Is(err, posts.ErrSelectedFollowersRequired) || errors.Is(err, posts.ErrInvalidPostViewer) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": err.Error(),
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not create post",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status": true,
		"post":   post,
	})
}

// ListPosts handles GET /api/posts
// with ?userID= it returns that users posts (for the profile page),
// without it, it returns the home feed
func (app App) ListPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	page, err := parsePage(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "limit must be between 1 and 50 and offset cannot be negative",
		})
		return
	}

	// profile page posts (?userID=), only the ones i am allowed to see
	queryUserID := r.URL.Query().Get("userID")
	if queryUserID != "" {
		requestUserID, err := strconv.Atoi(queryUserID)
		if err != nil {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid userID",
			})
			return
		}

		// ask for one extra post to know if there is a next page, same as the feed
		userPosts, err := posts.ListProfilePosts(app.DB, userID, requestUserID, page.Limit+1, page.Offset)
		if err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not get posts",
			})
			return
		}
		userPosts, hasMore := trimPage(userPosts, page, true)

		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":     true,
			"posts":      userPosts,
			"hasMore":    hasMore,
			"nextOffset": page.Offset + len(userPosts),
		})
		return
	}

	// home feed. we ask for one extra post to know if there is a next page
	feedPosts, err := posts.ListFeedPosts(app.DB, userID, page.Limit+1, page.Offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not load feed",
		})
		return
	}
	feedPosts, hasMore := trimPage(feedPosts, page, true)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"posts":      feedPosts,
		"hasMore":    hasMore,
		"nextOffset": page.Offset + len(feedPosts),
	})
}

// authenticatedUserID gives back the user id that AuthMiddleware put in the request
func authenticatedUserID(r *http.Request) (int, error) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok || userID <= 0 {
		return 0, errors.New("missing authenticated user")
	}

	return userID, nil
}

// writeJSON sends a json response with a status code
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// DeletePost handles DELETE /api/posts with a body like {"postID": 5}
// the database part makes sure only the owner can delete it
func (app *App) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	var request struct {
		PostID int `json:"postID"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.PostID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid post ID",
		})
		return
	}

	imagePaths, err := posts.DeletePost(app.DB, request.PostID, userID)
	if errors.Is(err, posts.ErrPostNotFound) {
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "post not found or it is not yours",
		})
		return
	}
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not delete post",
		})
		return
	}

	// the post is gone, remove its picture and the comment pictures from the disk too
	for _, path := range imagePaths {
		helpers.DeleteUpload(path)
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "post deleted",
	})
}
