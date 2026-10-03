package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	chatsdb "social/database/chats"
	"social/database/notifications"
	"social/internal/models"
)

func (app App) PrivateChats(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": false, "message": "authentication required"})
		return
	}

	conversations, err := chatsdb.ListPrivateChats(app.DB, userID)
	if err != nil {
		log.Printf("list private chats: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load chats"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "chats": conversations})
}

func (app App) PrivateChatUsers(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": false, "message": "authentication required"})
		return
	}

	users, err := chatsdb.ListPrivateCandidates(app.DB, userID)
	if err != nil {
		log.Printf("list private chat users: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load chat contacts"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "users": users})
}

func (app App) OpenPrivateChat(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": false, "message": "authentication required"})
		return
	}

	var input struct {
		UserID int `json:"userId"`
	}
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil || input.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "valid userId is required"})
		return
	}

	conversation, err := chatsdb.OpenPrivateChat(app.DB, userID, input.UserID)
	if err != nil {
		writeChatError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "chat": conversation})
}

func (app App) PrivateChatMessages(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": false, "message": "authentication required"})
		return
	}
	chatID, err := parseChatPathID(r, "chatID")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid chat id"})
		return
	}

	page, err := parsePage(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "limit must be between 1 and 50 and offset cannot be negative",
		})
		return
	}

	messages, listErr := chatsdb.ListPrivateMessages(app.DB, chatID, userID, page.Limit+1, page.Offset)
	if listErr != nil {
		writeChatError(w, listErr)
		return
	}
	messages, hasMore := trimPage(messages, page, false)
	if readErr := notifications.MarkMessageNotificationsRead(app.DB, userID, chatID); readErr != nil {
		log.Printf("mark private message notifications read: %v", readErr)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"messages":   messages,
		"hasMore":    hasMore,
		"nextOffset": page.Offset + len(messages),
	})
}

func (app *App) notifyPrivateMessage(chatID int64, senderID int, message models.ChatMessage) {
	var recipientID int
	err := app.DB.QueryRow(`
		SELECT CASE
			WHEN private_user_low_id = ? THEN private_user_high_id
			ELSE private_user_low_id
		END
		FROM chats
		WHERE id = ? AND type = 'private'
	`, senderID, chatID).Scan(&recipientID)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("find private message recipient: %v", err)
		return
	}
	if recipientID <= 0 || recipientID == senderID {
		return
	}

	senderName := strings.TrimSpace(strings.TrimSpace(message.FirstName) + " " + strings.TrimSpace(message.LastName))
	if senderName == "" {
		senderName = "Someone"
	}
	notification, err := notifications.UpsertMessage(app.DB, recipientID, senderID, chatID, senderName+": "+messagePreview(message.Content))
	if err != nil {
		// A notification failure should not make a successfully sent message look failed.
		log.Printf("create private message notification: %v", err)
		return
	}
	app.deliverNotification(notification)
}

func (app App) GroupChatMessages(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok {
		return
	}

	page, err := parsePage(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "limit must be between 1 and 50 and offset cannot be negative",
		})
		return
	}

	messages, chatID, err := chatsdb.ListGroupMessagesWithChatID(app.DB, groupID, userID, page.Limit+1, page.Offset)
	if err != nil {
		writeChatError(w, err)
		return
	}
	messages, hasMore := trimPage(messages, page, false)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"chatId":     chatID,
		"messages":   messages,
		"hasMore":    hasMore,
		"nextOffset": page.Offset + len(messages),
	})
}

// messagePreview keeps the notification text short
func messagePreview(content string) string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) > 80 {
		return string(runes[:80]) + "..."
	}
	return string(runes)
}

func parseChatPathID(r *http.Request, key string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func writeChatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chatsdb.ErrInvalidMessage):
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "message content is required"})
	case errors.Is(err, chatsdb.ErrMessageTooLong):
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "message must be 2000 characters or fewer"})
	case errors.Is(err, chatsdb.ErrSelfChat):
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "you cannot chat with yourself"})
	case errors.Is(err, chatsdb.ErrNoFollowRelation):
		writeJSON(w, http.StatusForbidden, map[string]any{"status": false, "message": "an accepted follow relationship is required"})
	case errors.Is(err, chatsdb.ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]any{"status": false, "message": "you do not have access to this chat"})
	case errors.Is(err, chatsdb.ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "user not found"})
	case errors.Is(err, chatsdb.ErrChatNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "chat not found"})
	case errors.Is(err, chatsdb.ErrGroupNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "group not found"})
	default:
		log.Printf("chat operation: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not complete chat operation"})
	}
}
