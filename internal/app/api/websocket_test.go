package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"social/internal/realtime"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

func TestTypingEventsAreAuthorizedEphemeralAndExcludeSender(t *testing.T) {
	db := newRealtimeTestDB(t)
	app := &App{DB: db, Realtime: realtime.NewHub()}

	mux := http.NewServeMux()
	for userID := 1; userID <= 5; userID++ {
		userID := userID
		mux.HandleFunc(fmt.Sprintf("/ws/%d", userID), func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "userID", userID)
			app.WsHandler(w, r.WithContext(ctx))
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	connections := make(map[int]*websocket.Conn)
	for userID := 1; userID <= 5; userID++ {
		url := "ws" + strings.TrimPrefix(server.URL, "http") + fmt.Sprintf("/ws/%d", userID)
		connection, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatal(err)
		}
		connections[userID] = connection
		t.Cleanup(func() { connection.Close() })
	}

	if err := connections[1].WriteJSON(map[string]any{
		"type": "typing_start", "chat_id": 10,
	}); err != nil {
		t.Fatal(err)
	}
	assertTypingEvent(t, connections[2], "typing_start", 10, 1)
	assertNoRealtimeEvent(t, connections[5])

	if err := connections[1].WriteJSON(map[string]any{
		"type": "typing_stop", "chat_id": 20,
	}); err != nil {
		t.Fatal(err)
	}
	assertTypingEvent(t, connections[2], "typing_stop", 20, 1)
	assertTypingEvent(t, connections[3], "typing_stop", 20, 1)

	if err := connections[4].WriteJSON(map[string]any{
		"type": "typing_start", "chat_id": 20,
	}); err != nil {
		t.Fatal(err)
	}
	var errorEvent realtimeErrorEvent
	if err := connections[4].ReadJSON(&errorEvent); err != nil {
		t.Fatal(err)
	}
	if errorEvent.Type != "error" || errorEvent.Code != "chat_forbidden" {
		t.Fatalf("unauthorized typing response = %+v", errorEvent)
	}

	assertNoRealtimeEvent(t, connections[1])
	var messageCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&messageCount); err != nil {
		t.Fatal(err)
	}
	if messageCount != 0 {
		t.Fatalf("typing events persisted %d messages", messageCount)
	}
}

func assertTypingEvent(t *testing.T, connection *websocket.Conn, eventType string, chatID int64, userID int) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var event realtimeTypingEvent
	if err := connection.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	if event.Type != eventType || event.ChatID != chatID || event.UserID != userID {
		t.Fatalf("typing event = %+v, expected type=%s chat=%d user=%d", event, eventType, chatID, userID)
	}
	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
}

func assertNoRealtimeEvent(t *testing.T, connection *websocket.Conn) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	var event json.RawMessage
	if err := connection.ReadJSON(&event); err == nil {
		t.Fatalf("unexpected realtime event: %s", event)
	}
}

func newRealtimeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE user_followers (follower_id INTEGER, target_id INTEGER, status INTEGER);
		CREATE TABLE group_members (group_id INTEGER, user_id INTEGER);
		CREATE TABLE chats (
			id INTEGER PRIMARY KEY,
			type TEXT NOT NULL,
			group_id INTEGER,
			private_user_low_id INTEGER,
			private_user_high_id INTEGER
		);
		CREATE TABLE chat_users (user_id INTEGER, chat_id INTEGER);
		CREATE TABLE messages (id INTEGER PRIMARY KEY, sender_id INTEGER, chat_id INTEGER, content TEXT);
		INSERT INTO user_followers VALUES (1, 2, 1);
		INSERT INTO group_members VALUES (7, 1), (7, 2), (7, 3);
		INSERT INTO chats VALUES (10, 'private', NULL, 1, 2), (20, 'group', 7, NULL, NULL);
		INSERT INTO chat_users VALUES (1, 10), (2, 10);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
