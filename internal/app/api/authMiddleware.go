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
