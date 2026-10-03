package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"social/database/users"
)

/*
A middle ware so simply takes a handler and returns a handler

In this handler we check if the user is authoniticated by reading the session id from the cookie and finding it in the
sessions table (it must not be expired). then write the user ID in the request Context for the callback to use it.

Paramters:

	handler http.HandleFunc

Returns:

	http.HandleFunc
*/
func (app *App) AuthMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  false,
				"message": "not authenticated",
			})
			return
		}

		userID, err := users.SessionUser(app.DB, cookie.Value)
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  false,
				"message": "invalid or expired session",
			})
			return
		}

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  false,
				"message": "could not verify user",
			})
			return
		}

		ctx := context.WithValue(r.Context(), "userID", userID)

		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}
