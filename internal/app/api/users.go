package api

import (
	"encoding/json"
	"log"
	"net/http"
	"social/database/profiles"
	"social/database/users"
	"social/internal/helpers"
	"social/internal/models"
	"social/internal/validation"
)

/*
Handler used to get current session user data. no data should be provided

Method:

	GET

-> in case of error there will be a respond written back and can me checked by
  - status boolean
  - message string

-> in case of success a respond will be written back
  - status must be true to get the data
  - data : data provided
*/
func (app *App) GetUserData(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	userData, err := users.GetUserData(app.DB, userID)
	if err != nil {
		log.Println(err, "here")
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user data",
		})
		return
	}

	userAbout, err := profiles.GetUserAbout(app.DB, userID)
	if err != nil {
		log.Println(err, "here1")
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user data",
		})
		return
	}
	userData.About = userAbout

	followers, err := profiles.GetFollowers(app.DB, userID, 10, 0)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user data",
		})
		return
	}

	userData.Followers = followers

	following, err := profiles.GetFollowing(app.DB, userID, 10, 0)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user data",
		})
		return
	}
	userData.Following = following

	friends, err := users.GetFriends(app.DB, userID, 0)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user data",
		})
		return
	}

	userData.Friends = friends
	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"data":   userData,
	})
}

/*
Handler used to update user personal information AND the about (BIO). every time the data provided should be all the data. updated or not
This Handler also validate the data before inserting.

METHOD:

	PATCH

-> Data provided must match the json format provided in models.UserRegistraion

-> in case of error there will be a respond written back and can me checked by
  - status boolean
  - message string

-> in case of success a respond will be written back
  - status must be true to get the data
  - message: success message
*/
func (app *App) UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	var userData models.UserRegistration
	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not get data",
		})
		return
	}

	if err := validation.ValidateUpdateInfo(&userData); err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid data:" + err.Error(),
		})
		return
	}

	if len(userData.Password) != 0 {
		hashedPassword, err := helpers.HashPassword(userData.Password)
		if err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not hash password:" + err.Error(),
			})
			return
		}
		userData.Password = hashedPassword
	}

	acceptedFollowers, err := users.UpdateUserInfo(app.DB, userID, &userData)
	if err != nil {
		log.Println(err)

		status, message := helpers.NormalizeSQLError(err)

		helpers.WriteJson(w, status, map[string]any{
			"status":  false,
			"message": message,
		})
		return
	}

	// going public accepted everyone who was waiting, let them know
	if len(acceptedFollowers) > 0 {
		name := app.userFullName(userID)
		for _, followerID := range acceptedFollowers {
			if _, err := app.DB.Exec(`
				UPDATE notifications SET is_read = 1
				WHERE user_id = ? AND actor_id = ? AND category = 'requests' AND type = 'follow_request'
			`, userID, followerID); err != nil {
				log.Println(err)
			}
			app.notify(followerID, userID, "requests", "follow_accepted", name+" accepted your follow request", nil)
		}
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "data updated!",
	})
}

/*
Handler used to Update or DELETE current user avatars. It uses PATCH method because even if the avatar is deleted it will be just updated to the default avatars.

METHOD:

	PATCH

-> TO DELETE: provide parameter named delete and give it value true
-> TO UPDATE: provide the multiheader

-> in case of error there will be a respond written back and can me checked by
  - status boolean
  - message string

-> in case of success a respond will be written back
  - status must be true to get the data
  - message: success message
*/
func (app *App) UpdateUserAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	delete := r.URL.Query().Get("delete")
	if delete == "true" {
		path, err := users.DeleteUserAvatar(app.DB, userID)
		if err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not delete avatar",
			})
			return
		}

		if err := helpers.DeleteAvatar(path); err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not delete user avatar",
			})
			return
		}
		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":  true,
			"message": "avatar deleted",
		})
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not get avatar",
		})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "avatar must be smaller than 5MB",
		})
		return
	}

	contentType := header.Header.Get("Content-Type")

	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "avatar must be JPG or PNG",
		})
		return
	}

	avatarPath, err := helpers.SaveUploads(file, header, "avatar")
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not save avatar",
		})
		return
	}

	err = users.UpdateUserAvatar(app.DB, userID, avatarPath)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not update avatar",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "avatar updated successfully",
		"avatar":  avatarPath,
	})
}

/*
Handler used to get the current user's About/BIO information. no data should be provided

Method:
    GET

-> in case of error there will be a respond written back and can me checked by
 - status boolean
 - message string

-> in case of success a respond will be written back
 - status must be true to get the data
 - data: user about data
*/
func (app *App) GetUserAbout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	userProfile, err := profiles.GetUserAbout(app.DB, userID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user profile" + err.Error(),
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"data":   userProfile,
	})
}

/*
Handler used to update the current user's About/BIO information.

Method:
    PATCH

-> Data provided must match the json format provided in models.UserAbout

-> in case of error there will be a respond written back and can me checked by
 - status boolean
 - message string

-> in case of success a respond will be written back
 - status must be true
 - message: success message
*/
func (app *App) UpdateUserAbout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	var userAbout models.UserAbout
	if err := json.NewDecoder(r.Body).Decode(&userAbout); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "error happened fetching data",
		})
		return
	}

	// every about field is a short text (the columns are 200 characters)
	for _, value := range []string{userAbout.Work, userAbout.Hobbies, userAbout.Education, userAbout.Intrests,
		userAbout.Travel, userAbout.Website, userAbout.Linkedin, userAbout.Instgram, userAbout.Twitter} {
		if len([]rune(value)) > 200 {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "each field can have up to 200 characters",
			})
			return
		}
	}

	if err := profiles.UpdateUserAbout(app.DB, userID, &userAbout); err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not update user about",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "user updated!",
	})
}

/*
Handler used to check if a username or email is already registered.

Method:
    POST

-> Data provided must match the following JSON format:
 - type string -> "name" or "email"
 - input string -> username or email to check

-> type must be:
 - "name" to check username availability
 - "email" to check email availability

-> in case of error there will be a respond written back and can me checked by
 - status boolean
 - message string

-> in case of success a respond will be written back
 - status must be true
 - avilable boolean -> true if the username/email is available
*/
func (app *App) CheckRegistration(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Type  string `json:"type"`
		Input string `json:"input"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid data",
		})
		return
	}

	var exists bool
	var err error

	switch req.Type {
	case "name":
		exists, err = users.CheckUserName(app.DB, req.Input)

	case "email":
		exists, err = users.CheckUserEmail(app.DB, req.Input)

	default:
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid type",
		})
		return
	}

	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not check availability",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":   true,
		"avilable": !exists,
	})
}

/*
Handler used to permanently delete the current user's account.

Method:
    DELETE

-> no data should be provided

-> the current user's account will be deleted

-> the user's token cookie will also be removed after successful deletion

-> in case of error there will be a respond written back and can me checked by
 - status boolean
 - message string

-> in case of success a respond will be written back
 - status must be true
 - message: success message
*/
func (app *App) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	err := users.DeleteUser(app.DB, userID)
	if err == nil && app.Realtime != nil {
		app.Realtime.DisconnectUser(userID)
	}
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not delete user",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "user deleted",
	})

}
