package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"social/database/users"
	"social/internal/app/tokens"

	"golang.org/x/net/websocket"
)

// AuthMiddleware authenticates the current user before passing the request
// to the provided handler by verifying the token cookie, checking whether
// the session has been revoked, and confirming that the user exists.
//
// @Parameters:
//   - handler: http.HandlerFunc
//     The handler to execute after successful authentication.
//
// @Returns:
//   - http.HandlerFunc
//     The wrapped handler that authenticates the user before executing
//     the original handler.
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

		payload, err := tokens.VerifyToken(cookie.Value)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  false,
				"message": "invalid or expired session",
			})
			return
		}

		
		if revoked, revokeErr := users.IsTokenRevoked(app.DB, cookie.Value); revokeErr != nil || revoked {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  false,
				"message": "invalid or expired session",
			})
			return
		}

		err = users.UserExists(app.DB, payload.UserID)
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  false,
				"message": "invalid user",
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

		ctx := context.WithValue(r.Context(), "userID", payload.UserID)

		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

// WebSocket connection to the provided handler by verifying the token
// cookie, checking whether the session has been revoked, and confirming
// that the user exists.
//
// @Parameters:
//   - handler: websocket.Handler
//     The WebSocket handler to execute after successful authentication.
//
// @Returns:
//   - websocket.Handler
//     The wrapped WebSocket handler that authenticates the user before
//     executing the original handler.
func (app *App) WSAuthMiddleware(handler websocket.Handler) websocket.Handler {
	return func(ws *websocket.Conn) {
		cookie, err := ws.Request().Cookie("token")
		if err != nil {
			log.Println("websocket: not authenticated")
			ws.Close()
			return
		}

		payload, err := tokens.VerifyToken(cookie.Value)
		if err != nil {
			log.Println("websocket: invalid or expired session")
			ws.Close()
			return
		}

		if revoked, revokeErr := users.IsTokenRevoked(app.DB, cookie.Value); revokeErr != nil || revoked {
			log.Println("websocket: logged out session")
			ws.Close()
			return
		}

		err = users.UserExists(app.DB, payload.UserID)
		if err != nil {
			log.Println("websocket: invalid user")
			ws.Close()
			return
		}

		ctx := context.WithValue(
			ws.Request().Context(),
			"userID",
			payload.UserID,
		)

		*ws.Request() = *ws.Request().WithContext(ctx)

		handler(ws)
	}
}
