package api

import (
	"errors"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"social/database/posts"
	"social/database/profiles"
	"social/internal/helpers"
	"social/internal/models"
	"social/internal/validation"
)

// AddPost makes a new post from the create post form (text, privacy, image or video, tags, location)
func (app *App) AddPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	// cap the body size so nobody can send a huge file and fill the server
	r.Body = http.MaxBytesReader(w, r.Body, validation.MaxPostMediaSize+(1<<20))

	if err := r.ParseMultipartForm(validation.MaxFormMemory); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not read form",
		})
		return
	}

	// everything comes as form values, if allowComments is missing we just allow comments
	allowComments, err := strconv.Atoi(r.FormValue("allowComments"))
	if err != nil {
		allowComments = 1
	}

	// groupID is the privacy: 0 public, -1 followers only, bigger than 0 is a chosen list of followers
	groupID, err := strconv.Atoi(r.FormValue("groupID"))
	if err != nil {
		groupID = 0
	}

	var taggedPeople []int

	// tagged people come as a json list of user ids
	taggedPeopleData := r.FormValue("taggedPeople")

	if taggedPeopleData != "" {
		if err := json.Unmarshal([]byte(taggedPeopleData), &taggedPeople); err != nil {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid tagged people",
			})
			return
		}
	}

	post := models.RegsiterPost{
		UserID:        userID,
		Content:       r.FormValue("content"),
		AllowComments: allowComments,
		GroupID:       groupID,
		Location:      r.FormValue("location"),
		PeopleTagged:  taggedPeople,
	}

	// the media is optional, so a missing file is fine and only other errors matter
	file, header, err := r.FormFile("image")

	// check the text, privacy and the file type before saving anything
	res := validation.ValidatePost(&post, header)
	if res.Field != "" {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"field":   res.Field,
			"message": res.Message,
		})
		return
	}

	if err == nil {
		defer file.Close()

		// save the image or video in uploads and keep its path for the db
		imagePath, err := helpers.SaveUploads(file, header, "post")
		if err != nil {
		if errors.Is(err, helpers.ErrUploadType) {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": err.Error(),
			})
			return
		}
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not save image",
			})
			return
		}

		post.Image_path = imagePath
	} else if err != http.ErrMissingFile {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid image",
		})
		return
	}

	// a chosen list of followers has to exist and belong to this user
	if err := posts.GroupExists(app.DB, post.GroupID, userID); err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "group does not exists",
			})
			return
		}
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get group post",
		})
		return
	}

	postID, err := posts.AddPost(app.DB, post)

	if err != nil {
		log.Println("here2", err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not upload post",
		})
		return
	}

	// send notifications to tagged and mentioned people, the map stops someone getting two
	actorName := app.actorName(userID)
	notified := map[int]bool{userID: true}

	for _, taggedUserID := range taggedPeople {
		if notified[taggedUserID] {
			continue
		}

		notified[taggedUserID] = true

		app.notify(userID, models.NewNotification{
			UserID:            taggedUserID,
			Message:           fmt.Sprintf("%s tagged you in a post", actorName),
			PostIDTag:         &postID,
			PostMentionUserID: &userID,
		})
	}

	// @mentions in the text get their own notification
	for _, mentionedID := range app.mentionedUserIDs(post.Content, notified) {
		notified[mentionedID] = true

		app.notify(userID, models.NewNotification{
			UserID:            mentionedID,
			Message:           fmt.Sprintf("%s mentioned you in a post", actorName),
			PostIDTag:         &postID,
			PostMentionUserID: &userID,
		})
	}

	helpers.WriteJson(w, http.StatusCreated, map[string]any{
		"status":  true,
		"message": "post uploaded successfully",
	})
}

// GetHomePosts gives the main feed, or only the videos when type=videos
func (app *App) GetHomePosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	offset := 0

	if value := r.URL.Query().Get("offset"); value != "" {
		var err error

		offset, err = strconv.Atoi(value)

		if err != nil || offset < 0 {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid offset",
			})
			return
		}
	}

	// the videos tab uses offset and limit paging, limit is capped at 50
	if r.URL.Query().Get("type") == "videos" {
		limit := 5

		if value := r.URL.Query().Get("limit"); value != "" {
			var err error

			limit, err = strconv.Atoi(value)

			if err != nil || limit < 1 {
				helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
					"status":  false,
					"message": "invalid limit",
				})
				return
			}

			if limit > 50 {
				limit = 50
			}
		}

		videos, err := posts.GetHomeVideos(app.DB, userID, offset, limit)

		if err != nil {
			log.Println(err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not get home videos",
			})
			return
		}

		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":  true,
			"posts":   videos,
			"hasMore": len(videos) == limit,
		})
		return
	}

	// ids of the posts the feed already shows, "1,2,3"
	var shownIDs []int
	for _, part := range strings.Split(r.URL.Query().Get("exclude"), ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && id > 0 {
			shownIDs = append(shownIDs, id)
		}
		if len(shownIDs) >= 1000 {
			break
		}
	}

	// normal feed, the query skips posts the user already has on screen
	homePosts, err := posts.GetHomePosts(app.DB, userID, shownIDs)

	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get home posts",
		})
		return
	}

	// adds how i am connected to each author so the card shows the right follow button
	posts.FillRelationships(app.DB, userID, homePosts)

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"posts":  homePosts,
	})
}

// PostReaction is the like and dislike button, 1 is like and -1 is dislike
func (app *App) PostReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	var rect models.Reaction
	r.Body = http.MaxBytesReader(w, r.Body, validation.MaxJSONBody)

	if err := json.NewDecoder(r.Body).Decode(&rect); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid reaction ",
		})
		return
	}

	if rect.Value != 1 && rect.Value != -1 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid reaction",
		})
		return
	}

	if !app.requirePostVisible(w, userID, rect.PostID) {
		return
	}

	rect.UserID = userID
	// pressing the same reaction again removes it, then deletion is true and we dont notify
	err, deletion := posts.InsertReaction(app.DB, rect)
	if err != nil {

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not insert reaction",
		})
		return
	}

	if deletion {
		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":  true,
			"message": "reaction inserted!",
		})
		return
	}

	// tell the post owner who liked or disliked it
	targetID, err := posts.GetPostOwnerID(app.DB, rect.PostID)
	if err != nil {
		log.Println("could not find post owner:", err)
	} else {
		notification := models.NewNotification{
			UserID:    targetID,
			PostIDTag: &rect.PostID,
		}

		name := app.actorName(userID)

		if rect.Value == 1 {
			notification.PostLikeUserID = &userID
			notification.Message = fmt.Sprintf("%s liked your post", name)
		} else {
			notification.PostDislikeUserID = &userID
			notification.Message = fmt.Sprintf("%s disliked your post", name)
		}

		app.notify(userID, notification)
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "reaction inserted!",
	})
}

// GetUserPosts gives the posts on a profile, targetID is whose profile, empty means my own
func (app *App) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	targetIDStr := r.URL.Query().Get("targetID")
	targetID := 0
	log.Println(targetIDStr, "str")

	if targetIDStr != "" {
		var err error

		targetID, err = strconv.Atoi(targetIDStr)
		if err != nil {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid targetID",
			})
			return
		}
	} else {
		log.Println("dont need filtering")
		targetID = userID
	}

	offsetStr := r.URL.Query().Get("offset")

	offset := 0

	if offsetStr != "" {
		var err error

		offset, err = strconv.Atoi(offsetStr)

		if err != nil || offset < 0 {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid offset",
			})
			return
		}
	}

	limit := 9
	limitStr := r.URL.Query().Get("limit")

	if limitStr != "" {
		var err error

		limit, err = strconv.Atoi(limitStr)

		if err != nil || limit < 1 {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid limit",
			})
			return
		}

		if limit > 50 {
			limit = 50
		}
	}

	// the profile has tabs for all posts, videos and tagged posts
	postType := r.URL.Query().Get("type")
	videosOnly := postType == "videos"

	if postType == "tagged" {
		app.writeTaggedPosts(w, userID, targetID, offset, limit)
		return
	}

	userPosts, err := posts.GetUserPosts(app.DB, targetID, offset, limit, videosOnly)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusOK, map[string]any{
				"status":  true,
				"message": "no posts",
				"hasMore": false,
			})
			return
		}

		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get posts",
		})
		return
	}

	hasMore := len(userPosts) == limit

	// when looking at someone else, drop the posts their privacy hides from me
	if targetID != userID {
		log.Println("need filtering")
		userPosts, err = posts.FilterPosts(app.DB, &userPosts, userID, targetID)

		if err != nil {
			log.Println(err, "filtering")
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not filter posts",
			})
			return
		}
	}

	posts.FillRelationships(app.DB, userID, userPosts)

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"data":    userPosts,
		"hasMore": hasMore,
	})
}

// DeletePost only works for the owner, the query returns no rows otherwise
func (app *App) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	postID, err := strconv.Atoi(r.URL.Query().Get("postId"))
	if err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid post id",
		})
		return
	}

	err = posts.DeletePost(app.DB, postID, userID)

	if err == sql.ErrNoRows {
		helpers.WriteJson(w, http.StatusForbidden, map[string]any{
			"status":  false,
			"message": "you cannot delete this post",
		})
		return
	}

	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not delete post",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "post deleted",
	})
}

// ViewPost saves that the user saw a post, used for the view count
func (app *App) ViewPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	var postID int
	r.Body = http.MaxBytesReader(w, r.Body, validation.MaxJSONBody)

	if err := json.NewDecoder(r.Body).Decode(&postID); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid post id",
		})
		return
	}

	if !app.requirePostVisible(w, userID, postID) {
		return
	}

	// seeing the same post twice hits the unique rule, that is not a real error
	if err := posts.ViewPost(app.DB, postID, userID); err != nil && err.Error() != "UNIQUE constraint failed: post_views.user_id, post_views.post_id" {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not add post",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "all good",
	})
}

// writeTaggedPosts sends the posts someone was tagged in
func (app *App) writeTaggedPosts(w http.ResponseWriter, userID, targetID, offset, limit int) {
	if targetID != userID {
		// private profile and i dont follow them, so i get an empty list
		isPrivate, err := posts.IsPrivateProfile(app.DB, targetID)
		if err != nil {
			log.Println(err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not get posts",
			})
			return
		}

		if isPrivate {
			status, err := profiles.CheckFollower(app.DB, userID, targetID)
			if err != nil && err != sql.ErrNoRows {
				log.Println(err)
				helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
					"status":  false,
					"message": "could not get posts",
				})
				return
			}

			if err == sql.ErrNoRows || status != 1 {
				helpers.WriteJson(w, http.StatusOK, map[string]any{
					"status":  true,
					"data":    []models.Post{},
					"hasMore": false,
				})
				return
			}
		}
	}

	taggedPosts, err := posts.GetTaggedPosts(app.DB, targetID, offset, limit)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get posts",
		})
		return
	}

	hasMore := len(taggedPosts) == limit

	// the posts are from other people so their privacy has to be checked too
	visiblePosts, err := posts.FilterTaggedPosts(app.DB, taggedPosts, userID, targetID)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not filter posts",
		})
		return
	}

	posts.FillRelationships(app.DB, userID, visiblePosts)

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"data":    visiblePosts,
		"hasMore": hasMore,
	})
}
