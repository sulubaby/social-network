package chats

import (
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"social/internal/models"
)

const MaxMessageLength = 2000

var (
	ErrChatNotFound     = errors.New("chat not found")
	ErrForbidden        = errors.New("chat access forbidden")
	ErrGroupNotFound    = errors.New("group not found")
	ErrInvalidMessage   = errors.New("message content is required")
	ErrMessageTooLong   = errors.New("message content is too long")
	ErrNoFollowRelation = errors.New("an accepted follow relationship is required")
	ErrSelfChat         = errors.New("cannot chat with yourself")
	ErrUserNotFound     = errors.New("user not found")
)

func OpenPrivateChat(db *sql.DB, userID, otherUserID int) (models.ChatConversation, error) {
	if userID == otherUserID {
		return models.ChatConversation{}, ErrSelfChat
	}

	tx, err := db.Begin()
	if err != nil {
		return models.ChatConversation{}, err
	}
	defer tx.Rollback()

	var exists bool
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM user WHERE id = ?)`, otherUserID).Scan(&exists); err != nil {
		return models.ChatConversation{}, err
	}
	if !exists {
		return models.ChatConversation{}, ErrUserNotFound
	}
	if err = requireFollowRelationship(tx, userID, otherUserID); err != nil {
		return models.ChatConversation{}, err
	}

	lowID, highID := normalizedPair(userID, otherUserID)
	if _, err = tx.Exec(`
		INSERT OR IGNORE INTO chats (type, private_user_low_id, private_user_high_id, num_of_members)
		VALUES ('private', ?, ?, 2)
	`, lowID, highID); err != nil {
		return models.ChatConversation{}, err
	}

	var chatID int64
	if err = tx.QueryRow(`
		SELECT id FROM chats
		WHERE type = 'private'
		  AND private_user_low_id = ?
		  AND private_user_high_id = ?
	`, lowID, highID).Scan(&chatID); err != nil {
		return models.ChatConversation{}, err
	}
	if _, err = tx.Exec(`
		INSERT OR IGNORE INTO chat_users (user_id, chat_id)
		VALUES (?, ?), (?, ?)
	`, userID, chatID, otherUserID, chatID); err != nil {
		return models.ChatConversation{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.ChatConversation{}, err
	}

	return getPrivateConversation(db, chatID, userID)
}

func ListPrivateChats(db *sql.DB, userID int) ([]models.ChatConversation, error) {
	rows, err := db.Query(`
		SELECT
			c.id,
			u.id,
			COALESCE(u.username, ''),
			u.first_name,
			u.last_name,
			COALESCE(p.avatar_path, ''),
			COALESCE(last_message.content, ''),
			COALESCE(last_message.created_at, '')
		FROM chats c
		JOIN user u ON u.id = CASE
			WHEN c.private_user_low_id = ? THEN c.private_user_high_id
			ELSE c.private_user_low_id
		END
		LEFT JOIN profile p ON p.user_id = u.id
		LEFT JOIN messages last_message ON last_message.id = (
			SELECT m.id
			FROM messages m
			WHERE m.chat_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		)
		WHERE c.type = 'private'
		  AND (? = c.private_user_low_id OR ? = c.private_user_high_id)
		ORDER BY
			CASE WHEN last_message.id IS NULL THEN 1 ELSE 0 END,
			last_message.created_at DESC,
			last_message.id DESC,
			c.created_at DESC,
			c.id DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]models.ChatConversation, 0)
	for rows.Next() {
		var conversation models.ChatConversation
		conversation.Type = "private"
		if err = rows.Scan(
			&conversation.ID,
			&conversation.OtherUser.ID,
			&conversation.OtherUser.Username,
			&conversation.OtherUser.FirstName,
			&conversation.OtherUser.LastName,
			&conversation.OtherUser.AvatarPath,
			&conversation.LatestMessage,
			&conversation.LatestMessageTime,
		); err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func ListPrivateCandidates(db *sql.DB, userID int) ([]models.ChatCandidate, error) {
	rows, err := db.Query(`
		SELECT
			u.id,
			COALESCE(u.username, ''),
			u.first_name,
			u.last_name,
			COALESCE(p.avatar_path, ''),
			c.id
		FROM user u
		LEFT JOIN profile p ON p.user_id = u.id
		LEFT JOIN chats c
		  ON c.type = 'private'
		 AND c.private_user_low_id = MIN(?, u.id)
		 AND c.private_user_high_id = MAX(?, u.id)
		WHERE u.id <> ?
		  AND EXISTS (
			SELECT 1
			FROM user_followers relationship
			WHERE relationship.status = 1
			  AND (
				(relationship.follower_id = ? AND relationship.target_id = u.id)
				OR (relationship.follower_id = u.id AND relationship.target_id = ?)
			  )
		  )
		ORDER BY LOWER(u.first_name), LOWER(u.last_name), LOWER(COALESCE(u.username, '')), u.id
	`, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]models.ChatCandidate, 0)
	for rows.Next() {
		var candidate models.ChatCandidate
		var chatID sql.NullInt64
		if err = rows.Scan(
			&candidate.ID,
			&candidate.Username,
			&candidate.FirstName,
			&candidate.LastName,
			&candidate.AvatarPath,
			&chatID,
		); err != nil {
			return nil, err
		}
		if chatID.Valid {
			candidate.ChatID = &chatID.Int64
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func ListPrivateMessages(db *sql.DB, chatID int64, userID int, pagination ...int) ([]models.ChatMessage, error) {
	if err := requirePrivateChatAccess(db, chatID, userID); err != nil {
		return nil, err
	}
	return listMessages(db, chatID, userID, pagination...)
}

func SendPrivateMessage(db *sql.DB, chatID int64, userID int, content string) (models.ChatMessage, error) {
	return sendMessage(db, chatID, userID, content, "private")
}

func ListGroupMessages(db *sql.DB, groupID int64, userID int, pagination ...int) ([]models.ChatMessage, error) {
	messages, _, err := ListGroupMessagesWithChatID(db, groupID, userID, pagination...)
	return messages, err
}

func ListGroupMessagesWithChatID(db *sql.DB, groupID int64, userID int, pagination ...int) ([]models.ChatMessage, int64, error) {
	chatID, err := EnsureGroupChat(db, groupID, userID)
	if err != nil {
		return nil, 0, err
	}
	messages, err := listMessages(db, chatID, userID, pagination...)
	return messages, chatID, err
}

func SendGroupMessage(db *sql.DB, groupID int64, userID int, content string) (models.ChatMessage, error) {
	chatID, err := EnsureGroupChat(db, groupID, userID)
	if err != nil {
		return models.ChatMessage{}, err
	}
	return sendMessage(db, chatID, userID, content, "group")
}

// SendMessage is the single realtime insertion path for private and group
// chats. Authorization is derived from the stored chat type, never from a
// client-provided sender or chat kind.
func SendMessage(db *sql.DB, chatID int64, userID int, content string) (models.ChatMessage, error) {
	return sendMessage(db, chatID, userID, content, "")
}

func sendMessage(db *sql.DB, chatID int64, userID int, content, expectedType string) (models.ChatMessage, error) {
	content, err := normalizeMessage(content)
	if err != nil {
		return models.ChatMessage{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return models.ChatMessage{}, err
	}
	defer tx.Rollback()

	chatType, err := requireChatAccess(tx, chatID, userID)
	if err != nil {
		return models.ChatMessage{}, err
	}
	if expectedType != "" && chatType != expectedType {
		return models.ChatMessage{}, ErrChatNotFound
	}

	message, err := insertMessage(tx, chatID, userID, content)
	if err != nil {
		return models.ChatMessage{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.ChatMessage{}, err
	}
	return message, nil
}

func GetChatParticipants(db *sql.DB, chatID int64) ([]int, error) {
	var chatType string
	var groupID sql.NullInt64
	err := db.QueryRow(`SELECT type, group_id FROM chats WHERE id = ?`, chatID).Scan(&chatType, &groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrChatNotFound
	}
	if err != nil {
		return nil, err
	}

	query := `SELECT user_id FROM chat_users WHERE chat_id = ? ORDER BY user_id`
	args := []any{chatID}
	if chatType == "group" {
		if !groupID.Valid {
			return nil, ErrChatNotFound
		}
		query = `SELECT user_id FROM group_members WHERE group_id = ? ORDER BY user_id`
		args = []any{groupID.Int64}
	} else if chatType != "private" {
		return nil, ErrChatNotFound
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := make([]int, 0)
	for rows.Next() {
		var userID int
		if err = rows.Scan(&userID); err != nil {
			return nil, err
		}
		participants = append(participants, userID)
	}
	return participants, rows.Err()
}

func GetAuthorizedChatParticipants(db *sql.DB, chatID int64, userID int) ([]int, error) {
	if _, err := requireChatAccess(db, chatID, userID); err != nil {
		return nil, err
	}
	return GetChatParticipants(db, chatID)
}

func EnsureGroupChat(db *sql.DB, groupID int64, userID int) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	chatID, err := ensureGroupChat(tx, groupID, userID)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return chatID, nil
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

type messageQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func ensureGroupChat(tx *sql.Tx, groupID int64, userID int) (int64, error) {
	var groupExists, isMember bool
	if err := tx.QueryRow(`
		SELECT
			EXISTS(SELECT 1 FROM groups WHERE id = ?),
			EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)
	`, groupID, groupID, userID).Scan(&groupExists, &isMember); err != nil {
		return 0, err
	}
	if !groupExists {
		return 0, ErrGroupNotFound
	}
	if !isMember {
		return 0, ErrForbidden
	}
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO chats (type, group_id, num_of_members)
		VALUES ('group', ?, (SELECT COUNT(*) FROM group_members WHERE group_id = ?))
	`, groupID, groupID); err != nil {
		return 0, err
	}
	var chatID int64
	if err := tx.QueryRow(`SELECT id FROM chats WHERE type = 'group' AND group_id = ?`, groupID).Scan(&chatID); err != nil {
		return 0, err
	}
	return chatID, nil
}

func requirePrivateChatAccess(db queryRower, chatID int64, userID int) error {
	var lowID, highID int
	err := db.QueryRow(`
		SELECT private_user_low_id, private_user_high_id
		FROM chats
		WHERE id = ? AND type = 'private'
	`, chatID).Scan(&lowID, &highID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrChatNotFound
	}
	if err != nil {
		return err
	}
	if userID != lowID && userID != highID {
		return ErrForbidden
	}
	return requireFollowRelationship(db, lowID, highID)
}

func requireChatAccess(db queryRower, chatID int64, userID int) (string, error) {
	var chatType string
	var groupID sql.NullInt64
	err := db.QueryRow(`SELECT type, group_id FROM chats WHERE id = ?`, chatID).Scan(&chatType, &groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrChatNotFound
	}
	if err != nil {
		return "", err
	}

	switch chatType {
	case "private":
		if err = requirePrivateChatAccess(db, chatID, userID); err != nil {
			return "", err
		}
	case "group":
		if !groupID.Valid {
			return "", ErrChatNotFound
		}
		var isMember bool
		if err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?
			)
		`, groupID.Int64, userID).Scan(&isMember); err != nil {
			return "", err
		}
		if !isMember {
			return "", ErrForbidden
		}
	default:
		return "", ErrChatNotFound
	}

	return chatType, nil
}

func requireFollowRelationship(db queryRower, firstUserID, secondUserID int) error {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM user_followers
			WHERE status = 1
			  AND (
				(follower_id = ? AND target_id = ?)
				OR (follower_id = ? AND target_id = ?)
			  )
		)
	`, firstUserID, secondUserID, secondUserID, firstUserID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNoFollowRelation
	}
	return nil
}

func listMessages(db messageQuerier, chatID int64, userID int, pagination ...int) ([]models.ChatMessage, error) {
	query := `
		SELECT
			m.id,
			m.chat_id,
			m.sender_id,
			COALESCE(u.username, ''),
			u.first_name,
			u.last_name,
			COALESCE(p.avatar_path, ''),
			m.content,
			m.created_at,
			m.sender_id = ?
		FROM messages m
		JOIN user u ON u.id = m.sender_id
		LEFT JOIN profile p ON p.user_id = u.id
		WHERE m.chat_id = ?
		ORDER BY m.created_at DESC, m.id DESC
	`
	args := []any{userID, chatID}
	if len(pagination) >= 2 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pagination[0], pagination[1])
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]models.ChatMessage, 0)
	for rows.Next() {
		var message models.ChatMessage
		if err = rows.Scan(
			&message.ID,
			&message.ChatID,
			&message.SenderID,
			&message.Username,
			&message.FirstName,
			&message.LastName,
			&message.AvatarPath,
			&message.Content,
			&message.CreatedAt,
			&message.IsOwn,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Paginated chat history is fetched newest-first so SQLite can stop at the
	// requested window. Reverse it before returning so the UI stays chronological.
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func insertMessage(tx *sql.Tx, chatID int64, userID int, content string) (models.ChatMessage, error) {
	result, err := tx.Exec(`INSERT INTO messages (sender_id, chat_id, content) VALUES (?, ?, ?)`, userID, chatID, content)
	if err != nil {
		return models.ChatMessage{}, err
	}
	messageID, err := result.LastInsertId()
	if err != nil {
		return models.ChatMessage{}, err
	}
	var message models.ChatMessage
	err = tx.QueryRow(`
		SELECT
			m.id, m.chat_id, m.sender_id, COALESCE(u.username, ''),
			u.first_name, u.last_name, COALESCE(p.avatar_path, ''),
			m.content, m.created_at
		FROM messages m
		JOIN user u ON u.id = m.sender_id
		LEFT JOIN profile p ON p.user_id = u.id
		WHERE m.id = ?
	`, messageID).Scan(
		&message.ID,
		&message.ChatID,
		&message.SenderID,
		&message.Username,
		&message.FirstName,
		&message.LastName,
		&message.AvatarPath,
		&message.Content,
		&message.CreatedAt,
	)
	message.IsOwn = true
	return message, err
}

func getPrivateConversation(db *sql.DB, chatID int64, userID int) (models.ChatConversation, error) {
	var conversation models.ChatConversation
	conversation.Type = "private"
	err := db.QueryRow(`
		SELECT
			c.id,
			u.id,
			COALESCE(u.username, ''),
			u.first_name,
			u.last_name,
			COALESCE(p.avatar_path, ''),
			COALESCE(last_message.content, ''),
			COALESCE(last_message.created_at, '')
		FROM chats c
		JOIN user u ON u.id = CASE
			WHEN c.private_user_low_id = ? THEN c.private_user_high_id
			ELSE c.private_user_low_id
		END
		LEFT JOIN profile p ON p.user_id = u.id
		LEFT JOIN messages last_message ON last_message.id = (
			SELECT m.id FROM messages m
			WHERE m.chat_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		)
		WHERE c.id = ?
		  AND c.type = 'private'
		  AND (? = c.private_user_low_id OR ? = c.private_user_high_id)
	`, userID, chatID, userID, userID).Scan(
		&conversation.ID,
		&conversation.OtherUser.ID,
		&conversation.OtherUser.Username,
		&conversation.OtherUser.FirstName,
		&conversation.OtherUser.LastName,
		&conversation.OtherUser.AvatarPath,
		&conversation.LatestMessage,
		&conversation.LatestMessageTime,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ChatConversation{}, ErrChatNotFound
	}
	return conversation, err
}

func normalizeMessage(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", ErrInvalidMessage
	}
	if utf8.RuneCountInString(content) > MaxMessageLength {
		return "", ErrMessageTooLong
	}
	return content, nil
}

func normalizedPair(firstUserID, secondUserID int) (int, int) {
	if firstUserID < secondUserID {
		return firstUserID, secondUserID
	}
	return secondUserID, firstUserID
}
