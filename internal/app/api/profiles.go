package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"social/database/chats"
	"social/database/preferences"
	"social/database/profiles"
	"social/internal/helpers"
	"social/internal/models"
	"strconv"
)

/*
function of type handler used to get the profile of another user. the id of the profile is taken from the
query (?id=). if the profile is public, or it is private but the logged in user follows it, the full profile
is returned with the visibility settings applied, the follow status and if the logged in user can message
this user. otherwise only the limited private profile data is returned with showProfile set to false.
viewing your own profile with this handler is not allowed.

followStatus: 1 -> following, 0 -> request pending, -1 -> not following

@METHOD
	-> GET
*/
func (app *App) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	requestedID := r.URL.Query().Get("id")
	profileID, err := strconv.Atoi(requestedID)
	if err != nil || profileID <= 0 {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid user id",
		})
		return
	}

	if profileID == userID {
		log.Println("same id")
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "cannot view your own profile this way",
		})
		return
	}

	isPrivate, err := profiles.IsPrivate(app.DB, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusNotFound, map[string]any{
				"status":  false,
				"message": "no user found",
			})
			return
		}

		log.Println("here5", err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not check profile privacy",
		})
		return
	}

	isFollowing, err := profiles.CheckFollower(app.DB, userID, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			isFollowing = -1
		} else {
			log.Println("here3", err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not check follow status",
			})
			return
		}
	}

	if !isPrivate || isFollowing == 1 {
		userData, err := profiles.GetUserData(app.DB, profileID)
		if err != nil {
			if err == sql.ErrNoRows {
				helpers.WriteJson(w, http.StatusNotFound, map[string]any{
					"status":       false,
					"showProfile":  false,
					"followStatus": -1,
					"message":      "no user found",
				})
				return
			}
			log.Println("here1", err)

			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":       false,
				"showProfile":  false,
				"followStatus": -1,
				"message":      "could not get profile data",
			})
			return
		}

		visibility, err := preferences.ApplyProfileVisibility(app.DB, userID, profileID, &userData)
		if err != nil {
			log.Println(err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":       false,
				"showProfile":  false,
				"followStatus": -1,
				"message":      "could not get profile data",
			})
			return
		}

		canMessage, err := chats.CanSendMessage(app.DB, userID, profileID)
		if err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":       false,
				"showProfile":  false,
				"followStatus": -1,
				"message":      "could not get profile data",
			})
			return
		}
		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":       true,
			"showProfile":  true,
			"followStatus": isFollowing,
			"data":         userData,
			"canMessage":   canMessage,
			"visibility":   visibility,
		})
		return
	}

	userData, err := profiles.GetPrivateProfileData(app.DB, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusNotFound, map[string]any{
				"status":       false,
				"showProfile":  false,
				"followStatus": -1,
				"message":      "no user found",
			})
			return
		}

		log.Println("here0", err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":       false,
			"showProfile":  false,
			"followStatus": -1,
			"message":      "could not get profile data",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":       true,
		"showProfile":  false,
		"followStatus": isFollowing,
		"data":         userData,
	})
}

/*
function of type handler used to follow another user. the id of the user to follow is taken from the
query (?targetid=). if the target profile is public the follow is done directly (followStatus 1) and the
target is notified that someone started following them. if it is private, a follow request is sent
instead (followStatus 0) and the target is notified of the request. following yourself is not allowed.

@METHOD
	-> POST
*/
func (app *App) RequestFollow(w http.ResponseWriter, r *http.Request) {
	followerID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	queryID := r.URL.Query().Get("targetid")

	targetID, err := strconv.Atoi(queryID)

	if err != nil || targetID == followerID {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid target ID",
		})
		return
	}

	isPrivate, err := profiles.IsPrivate(app.DB, targetID)

	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to get user data",
		})
		return
	}

	var requestCode int

	if isPrivate {
		requestCode = 0
	} else {
		requestCode = 1
	}

	if err := profiles.SendFollowRequest(app.DB, targetID, followerID, requestCode); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "request to follow failed",
		})
		return
	}

	followerName := app.actorName(followerID)

	if requestCode == 1 {
		app.notify(followerID, models.NewNotification{
			UserID:       targetID,
			Message:      fmt.Sprintf("%s started following you", followerName),
			FollowUserID: &followerID,
		})
	} else {
		app.notify(followerID, models.NewNotification{
			UserID:              targetID,
			Message:             fmt.Sprintf("%s sent you a follow request", followerName),
			FollowRequestUserID: &followerID,
		})
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":       true,
		"followStatus": requestCode,
		"message":      "request sent",
	})
}

/*
function of type handler used to cancel a follow request or unfollow a user. the id of the target user is
taken from the query (?targetid=). it removes the relation between the logged in user (the follower) and the
target, then returns followStatus -1 (not following).

@METHOD
	-> DELETE
*/
func (app *App) CancelRequest(w http.ResponseWriter, r *http.Request) {
	followerID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	queryID := r.URL.Query().Get("targetid")
	targetID, err := strconv.Atoi(queryID)
	if err != nil || targetID == followerID {
		log.Println(err, "here")
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid target ID",
		})
		return
	}

	if err := profiles.SendFollowRequest(app.DB, targetID, followerID, -1); err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid target ID",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":       true,
		"followStatus": -1,
		"message":      "request removed",
	})
	return
}

/*
function of type handler used to get the followers of a user, 20 at a time. the query takes ?offset= (the
number of followers to skip, used for pagination) and optionally ?targetid= (the user to get the followers
of). if targetid is missing (or "null" / "undefined") the followers of the logged in user are returned. for a
private profile only the owner and the people following it are allowed to see the list.

@METHOD
	-> GET
*/
func (app *App) GetFollowers(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	queryID := r.URL.Query().Get("targetid")
	queryOffset := r.URL.Query().Get("offset")

	offset, err := strconv.Atoi(queryOffset)
	if err != nil || offset < 0 {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid offset",
		})
		return
	}

	targetID := userID

	if queryID != "" && queryID != "null" && queryID != "undefined" {
		targetID, err = strconv.Atoi(queryID)
		if err != nil || targetID <= 0 {
			log.Println(err)
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid target id",
			})
			return
		}
	}

	if !app.canSeeConnections(w, userID, targetID) {
		return
	}

	followers, err := profiles.GetFollowers(app.DB, targetID, 20, offset)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusOK, map[string]any{
				"status": false,
				"data":   nil,
			})
			return
		}

		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get followers",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"data":   followers,
	})
}

/*
function of type handler used to get the users that a user is following, 20 at a time. the query takes
?offset= (the number of users to skip, used for pagination) and optionally ?targetid= (the user to get the
following list of). if targetid is missing (or "null" / "undefined") the list of the logged in user is
returned. for a private profile only the owner and the people following it are allowed to see the list.

@METHOD
	-> GET
*/
func (app *App) GetFollowing(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	queryID := r.URL.Query().Get("targetid")
	queryOffset := r.URL.Query().Get("offset")

	offset, err := strconv.Atoi(queryOffset)
	if err != nil || offset < 0 {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid offset",
		})
		return
	}

	targetID := userID
	log.Println(queryID)
	if queryID != "" && queryID != "null" && queryID != "undefined" {
		targetID, err = strconv.Atoi(queryID)
		if err != nil || targetID <= 0 {
			log.Println(err)
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid target id",
			})
			return
		}
	}

	if !app.canSeeConnections(w, userID, targetID) {
		return
	}

	following, err := profiles.GetFollowing(app.DB, targetID, 20, offset)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusOK, map[string]any{
				"status": false,
				"data":   nil,
			})
			return
		}

		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get following",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"data":   following,
	})
}

/*
function of type handler used to remove a follower of the logged in user. the id of the follower to remove
is taken from the query (?targetid=). it deletes the relation where the target follows the logged in user,
so the target stops following them. removing yourself is not allowed.

@METHOD
	-> DELETE
*/
func (app *App) RemoveFollower(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	queryID := r.URL.Query().Get("targetid")
	followerID, err := strconv.Atoi(queryID)
	if err != nil || followerID == userID {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid target ID",
		})
		return
	}

	if err := profiles.SendFollowRequest(app.DB, userID, followerID, -1); err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not remove follower",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "follower removed",
	})
}

/*
function of type handler used to accept a pending follow request sent to the logged in user. the id of the
user who sent the request is taken from the query (?targetid=). if the accept succeeds, the requester is
notified that their request was accepted.

@METHOD
	-> POST
*/
func (app *App) AcceptFollowRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	queryID := r.URL.Query().Get("targetid")

	requesterID, err := strconv.Atoi(queryID)

	if err != nil || requesterID == userID {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid target ID",
		})
		return
	}

	err = profiles.AcceptFollowRequest(app.DB, requesterID, userID)

	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not accept follow request",
		})
		return
	}

	app.notify(userID, models.NewNotification{
		UserID:                    requesterID,
		Message:                   fmt.Sprintf("%s accepted your follow request", app.actorName(userID)),
		FollowRequestAcceptUserID: &userID,
	})

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "follow request accepted",
	})
}

/*
function of type handler used to reject a pending follow request sent to the logged in user. the id of the
user who sent the request is taken from the query (?targetid=). if there is no such request it returns
404 (follow request not found).

@METHOD
	-> POST
*/
func (app *App) RejectFollowRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	requesterID, err := strconv.Atoi(r.URL.Query().Get("targetid"))

	if err != nil || requesterID == userID {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid target ID",
		})
		return
	}

	err = profiles.RejectFollowRequest(app.DB, requesterID, userID)

	if err == sql.ErrNoRows {
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "follow request not found",
		})
		return
	}

	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not reject follow request",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "follow request rejected",
	})
}

// canSeeConnections: the followers and following lists of a private profile are
// only for the owner and the people that follow it, like the rest of the profile
func (app *App) canSeeConnections(w http.ResponseWriter, viewerID, targetID int) bool {
	if viewerID == targetID {
		return true
	}

	isPrivate, err := profiles.IsPrivate(app.DB, targetID)
	if err == sql.ErrNoRows {
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "no user found",
		})
		return false
	}
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not check profile privacy",
		})
		return false
	}
	if !isPrivate {
		return true
	}

	status, err := profiles.CheckFollower(app.DB, viewerID, targetID)
	if err == nil && status == 1 {
		return true
	}

	helpers.WriteJson(w, http.StatusForbidden, map[string]any{
		"status":  false,
		"message": "this profile is private",
	})
	return false
}