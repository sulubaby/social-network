package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	chatsdb "social/database/chats"
	"social/internal/helpers"
	"social/internal/models"
	"social/internal/realtime"

	"github.com/gorilla/websocket"
)

type incomingRealtimeEvent struct {
	Type    string `json:"type"`
	ChatID  int64  `json:"chat_id"`
	Content string `json:"content"`
}

type realtimeEvent struct {
	Type         string               `json:"type"`
	Message      *models.ChatMessage  `json:"message,omitempty"`
	Notification *models.Notification `json:"notification,omitempty"`
	Code         string               `json:"code,omitempty"`
}

type realtimeTypingEvent struct {
	Type   string `json:"type"`
	ChatID int64  `json:"chatId"`
	UserID int    `json:"userId"`
}

type realtimeErrorEvent struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: websocketOriginAllowed,
}

func (app *App) WsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}
	if app.Realtime == nil {
		helpers.WriteJson(w, http.StatusServiceUnavailable, map[string]any{
			"status":  false,
			"message": "realtime service unavailable",
		})
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := app.Realtime.Register(userID, conn)
	client.ReadPump(func(payload []byte) {
		app.handleRealtimeEvent(client, payload)
	})
}

func (app *App) handleRealtimeEvent(client *realtime.Client, payload []byte) {
	var event incomingRealtimeEvent
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		app.sendRealtimeError(client, "invalid_event", "Invalid realtime event.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		app.sendRealtimeError(client, "invalid_event", "Invalid realtime event.")
		return
	}

	if event.ChatID <= 0 {
		app.sendRealtimeError(client, "invalid_chat", "A valid chat is required.")
		return
	}

	switch event.Type {
	case "message":
		app.handleRealtimeMessage(client, event)
	case "typing_start", "typing_stop":
		app.handleRealtimeTyping(client, event)
	default:
		app.sendRealtimeError(client, "unsupported_type", "Unsupported realtime event type.")
	}
}

func (app *App) handleRealtimeMessage(client *realtime.Client, event incomingRealtimeEvent) {
	message, err := chatsdb.SendMessage(app.DB, event.ChatID, client.UserID, event.Content)
	if err != nil {
		code, safeMessage := realtimeChatError(err)
		app.sendRealtimeError(client, code, safeMessage)
		return
	}

	participants, err := chatsdb.GetChatParticipants(app.DB, event.ChatID)
	if err != nil {
		log.Printf("load realtime chat participants: %v", err)
		app.sendRealtimeError(client, "delivery_failed", "Message saved but realtime delivery failed.")
		return
	}

	err = app.Realtime.SendToUsers(participants, func(recipientID int) any {
		outgoing := message
		outgoing.IsOwn = int64(recipientID) == outgoing.SenderID
		return realtimeEvent{Type: "message", Message: &outgoing}
	})
	if err != nil {
		log.Printf("marshal realtime message: %v", err)
	}

	app.notifyPrivateMessage(event.ChatID, client.UserID, message)
}

func (app *App) handleRealtimeTyping(client *realtime.Client, event incomingRealtimeEvent) {
	participants, err := chatsdb.GetAuthorizedChatParticipants(app.DB, event.ChatID, client.UserID)
	if err != nil {
		code, safeMessage := realtimeChatError(err)
		app.sendRealtimeError(client, code, safeMessage)
		return
	}

	recipients := make([]int, 0, len(participants))
	for _, participantID := range participants {
		if participantID != client.UserID {
			recipients = append(recipients, participantID)
		}
	}

	err = app.Realtime.SendToUsers(recipients, func(int) any {
		return realtimeTypingEvent{
			Type:   event.Type,
			ChatID: event.ChatID,
			UserID: client.UserID,
		}
	})
	if err != nil {
		log.Printf("marshal realtime typing event: %v", err)
	}
}

func (app *App) sendRealtimeError(client *realtime.Client, code, message string) {
	if err := client.Send(realtimeErrorEvent{
		Type:    "error",
		Code:    code,
		Message: message,
	}); err != nil {
		log.Printf("send realtime error: %v", err)
	}
}

func realtimeChatError(err error) (string, string) {
	switch {
	case errors.Is(err, chatsdb.ErrInvalidMessage):
		return "message_empty", "Message content is required."
	case errors.Is(err, chatsdb.ErrMessageTooLong):
		return "message_too_long", "Message must be 2000 characters or fewer."
	case errors.Is(err, chatsdb.ErrForbidden), errors.Is(err, chatsdb.ErrNoFollowRelation):
		return "chat_forbidden", "You do not have access to this chat."
	case errors.Is(err, chatsdb.ErrChatNotFound):
		return "chat_not_found", "Chat not found."
	default:
		log.Printf("save realtime chat message: %v", err)
		return "message_failed", "Could not send message."
	}
}

func websocketOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	for _, allowed := range strings.Split(os.Getenv("ORBIT_ALLOWED_ORIGINS"), ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}

	requestHost := r.Host
	if strings.EqualFold(parsed.Host, requestHost) {
		return true
	}

	originHost, _, originErr := net.SplitHostPort(parsed.Host)
	requestName, _, requestErr := net.SplitHostPort(requestHost)
	if originErr == nil && requestErr == nil {
		return isLoopbackHost(originHost) && isLoopbackHost(requestName)
	}
	return false
}

func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
}
