package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"social/database/users"
	"social/internal/app/tokens"
	"social/internal/helpers"
	"social/internal/models"
	"time"
)

/*
Handler used to log a user into their account.

Method:
    POST

-> Data provided must match the json format provided in models.UserLogger

-> The identifier can be the user's username or email.

-> The password provided will be compared with the hashed password stored in the database.

-> in case of error there will be a respond written back and can me checked by
 - status boolean
 - message string

-> in case of success a respond will be written back
 - status must be true
 - message: success message

-> a authentication token will be created and stored in a HTTP-only cookie
 - cookie name: token
 - cookie expires after 30 days
*/
func (app *App) LoggingUser(w http.ResponseWriter, r *http.Request) {
	var logger models.UserLogger

	if err := json.NewDecoder(r.Body).Decode(&logger); err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "could not get data",
		})
		return
	}

	hashedPassword, err := users.GetHashedPassowrd(app.DB, logger.Identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
				"status":  false,
				"message": "user identifier or password is incorrect",
			})
			return
		} else {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not get user data",
			})
			return
		}
	}

	if match := helpers.AuthonticateUser(logger.Pass, hashedPassword); !match {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "user identifier or password is incorrect",
		})
		return
	}

	userID := users.GetUserID(app.DB, logger.Identifier)
	if userID == -1 {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get user data",
		})
		return
	}

	token, err := tokens.GenerateToken(userID)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not create authentication token",
		})
		return
	}

	cookie := http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	// "keep me signed in" keeps the cookie for 30 days, without it the cookie
	// is gone when the browser closes (the token itself still expires on its own)
	if logger.Remember == nil || *logger.Remember {
		cookie.Expires = time.Now().Add(24 * 30 * time.Hour)
	}

	http.SetCookie(w, &cookie)

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "logged in",
	})
}

// this function should be deleted, it is used but wrongly
func (app *App) AuthorizeSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"message": "valid session",
	})
}

/*
Handler used to authorize the current user's session.

Method:
    GET

-> no data should be provided

-> this handler should only be called after the authentication middleware
   has verified the user's session.

-> in case of error there will be a respond written back and can me checked by
 - status boolean
 - message string

-> in case of success a respond will be written back
 - status must be true
 - message: valid session

-> This handler should not be used directly for authentication.
   The authentication middleware should be responsible for validating
   the session.
*/
func (app *App) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("token"); err == nil {
		if payload, verifyErr := tokens.VerifyToken(cookie.Value); verifyErr == nil && app.Realtime != nil {
			app.Realtime.DisconnectUser(payload.UserID)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "logged out",
	})
}
