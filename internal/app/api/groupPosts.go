package api

import (
	"errors"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"social/database/groups"
	"social/database/users"
	"social/internal/helpers"
	"social/internal/models"
	"social/internal/validation"
	"strconv"
)

func (app *App) AddGroupPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, validation.MaxPostMediaSize+(1<<20))

	if err := r.ParseMultipartForm(validation.MaxFormMemory); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not read form",
		})
		return
	}

	allowComments, err := strconv.Atoi(r.FormValue("allowComments"))
	if err != nil {
		allowComments = 1
	}

	groupID, err := strconv.Atoi(r.FormValue("groupID"))
	if err != nil || groupID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid group",
		})
		return
	}

	var taggedPeople []int

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

	post := models.Post{
		UserId:        userID,
		Content:       r.FormValue("content"),
		AllowComments: allowComments == 1,
		GroupId:       &groupID,
		Location:      nil,
		TaggedPeople:  nil,
	}

	location := r.FormValue("location")

	if location != "" {
		post.Location = &location
	}

	file, header, err := r.FormFile("image")

	check := models.RegsiterPost{
		Content:       post.Content,
		AllowComments: allowComments,
		GroupID:       groupID,
		Location:      location,
		PeopleTagged:  taggedPeople,
	}

	res := validation.ValidatePost(&check, header)

	if res.Field != "" {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"field":   res.Field,
			"message": res.Message,
		})
		return
	}

	post.Content = check.Content

	if post.Location != nil {
		cleanLocation := check.Location
		post.Location = &cleanLocation
	}

	if err == nil {
		defer file.Close()

		imagePath, err := helpers.SaveUploads(file, header, "post")

		if err != nil {
		if errors.Is(err, helpers.ErrUploadType) {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": err.Error(),
			})
			return
		}
			log.Println(err)

			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not save image",
			})
			return
		}

		post.ImagePath = &imagePath
	}

	exists, err := groups.GroupExists(app.DB, groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "group does not exist or user is not a member",
			})
			return
		}

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not verify group",
		})
		return
	}

	if !exists {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "group does not exist or user is not a member",
		})
		return
	}

	// accepted members only, a pending invite or join request is not enough
	userIN, err := groups.IsMember(app.DB, groupID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "group does not exist or user is not a member",
			})
			return
		}

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not verify group",
		})
		return
	}

	if !userIN {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "group does not exist or user is not a member",
		})
		return
	}

	if len(taggedPeople) > 0 {
		members, err := groups.GroupMemberIDsAmong(app.DB, groupID, taggedPeople)
		if err != nil {
			log.Println(err)

			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not verify tagged people",
			})
			return
		}

		for _, id := range taggedPeople {
			if !members[id] {
				helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
					"status":  false,
					"field":   "tags",
					"message": "you can only tag members of this group",
				})
				return
			}
		}
	}

	postID, err := groups.AddPost(app.DB, post, groupID, taggedPeople)

	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not upload post",
		})
		return
	}

	app.notifyGroupPostPeople(userID, groupID, post.Content, taggedPeople)

	user, err := users.GetUserSimpleData(app.DB, userID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not upload post",
		})
		return
	}

	msg := map[string]any{
		"type": "postGroup",
		"data": map[string]any{
			"content":   post.Content,
			"imagePath": post.ImagePath,
			"groupID":   groupID,
			"postID":    postID,
			"user": map[string]any{
				"ID":        user.ID,
				"firstName": user.FirstName,
				"lastName":  user.LastName,
				"avatar":    user.Avatar,
			},
		},
	}

	helpers.WriteJson(w, http.StatusCreated, map[string]any{
		"status": true,
		"data":   msg,
	})
}

func (app *App) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	groupIDParam := r.URL.Query().Get("targetid")
	if groupIDParam == "" {
		groupIDParam = r.URL.Query().Get("groupID")
	}

	groupID, err := strconv.Atoi(groupIDParam)
	if err != nil || groupID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid group",
		})
		return
	}

	searchValue, searchOK := readSearch(w, r)
	if !searchOK {
		return
	}

	offset := 0

	if queryOffset := r.URL.Query().Get("offset"); queryOffset != "" {
		offset, err = strconv.Atoi(queryOffset)

		if err != nil || offset < 0 {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid offset",
			})
			return
		}
	}

	userIN, err := groups.IsMember(app.DB, groupID, userID)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not verify group",
		})
		return
	}

	if !userIN {
		helpers.WriteJson(w, http.StatusForbidden, map[string]any{
			"status":  false,
			"message": "you are not a member of this group",
		})
		return
	}

	if searchValue == "" {
		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":  true,
			"message": "members fetched",
			"data":    map[int]models.UserRegistration{},
		})
		return
	}

	ids, err := groups.SearchGroupMembers(app.DB, groupID, searchValue, offset, 10)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get members",
		})
		return
	}

	members := make(map[int]models.UserRegistration)

	for _, id := range ids {
		if id == userID {
			continue
		}

		member, err := users.GetUserSimpleData(app.DB, id)
		if err != nil {
			log.Println(err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not get members",
			})
			return
		}

		members[id] = member
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "members fetched",
		"data":    members,
	})
}

func (app *App) GetGroupPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	groupID, err := strconv.Atoi(r.URL.Query().Get("groupID"))
	if err != nil {
		helpers.WriteJson(w, http.StatusBadGateway, map[string]any{
			"status":  false,
			"message": "could not find post",
		})
		return
	}

	postID, err := strconv.Atoi(r.URL.Query().Get("postID"))
	if err != nil {
		helpers.WriteJson(w, http.StatusBadGateway, map[string]any{
			"status":  false,
			"message": "could not find post",
		})
		return
	}

	// accepted members only, a pending invite or join request is not enough
	userIN, err := groups.IsMember(app.DB, groupID, userID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get post data",
		})
		return
	}

	if !userIN {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not find post",
		})
		return
	}

	post, err := groups.GetGroupPost(app.DB, postID, userID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get post data",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": false,
		"data":   post,
	})
	return
}

func (app *App) InsertGroupPostReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	var react models.Reaction
	r.Body = http.MaxBytesReader(w, r.Body, validation.MaxJSONBody)

	if err := json.NewDecoder(r.Body).Decode(&react); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not insert reaction",
		})
		return
	}

	if react.Value != 1 && react.Value != -1 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid reaction",
		})
		return
	}

	// only members of the group can like or dislike its posts
	allowed, err := groups.UserInPostGroup(app.DB, react.PostID, userID)
	if err != nil || !allowed {
		helpers.WriteJson(w, http.StatusForbidden, map[string]any{
			"status":  false,
			"message": "you are not a member of this group",
		})
		return
	}

	react.UserID = userID
	err = groups.InsertReaction(app.DB, react)

	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not insert reaction",
		})
		return
	}

	app.notifyGroupPostReaction(userID, react.PostID, react.Value)

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "reaction inserted",
	})
}

func (app *App) DeleteGroupPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	postID, err := strconv.Atoi(r.URL.Query().Get("postId"))
	if err != nil || postID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid post id",
		})
		return
	}

	err = groups.DeleteGroupPost(app.DB, postID, userID)

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
