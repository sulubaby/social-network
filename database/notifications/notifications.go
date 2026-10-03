// this file has all the database functions for notifications
// (making them, listing them, counting unread ones, marking them read)
package notifications

import (
	"database/sql"
	"errors"
	"social/internal/models"
	"strings"
)

// we return this when someone asks for a category that doesnt exist
var ErrInvalidCategory = errors.New("notification category must be requests, groups, events, messages, or posts")

// Create saves a new notification for a user and gives back the saved row.
// it checks the category and makes sure type and message are not empty
func Create(db *sql.DB, userID int, request models.CreateNotificationRequest) (models.Notification, error) {
	if userID <= 0 || !IsCategory(request.Category) {
		return models.Notification{}, ErrInvalidCategory
	}

	request.Category = strings.TrimSpace(request.Category)
	request.Type = strings.TrimSpace(request.Type)
	request.Message = strings.TrimSpace(request.Message)
	if request.Type == "" || request.Message == "" || len([]rune(request.Message)) > 500 {
		return models.Notification{}, errors.New("notification type and message are required")
	}

	// save it in the table
	result, err := db.Exec(`
		INSERT INTO notifications (user_id, actor_id, category, type, message, related_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, request.ActorID, request.Category, request.Type, request.Message, request.RelatedID)
	if err != nil {
		return models.Notification{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Notification{}, err
	}

	return GetByID(db, userID, id)
}

// List gets the notifications of one user, newest first.
// category can be empty or "all" to get everything.
// pagination is optional: [limit, offset]
func List(db *sql.DB, userID int, category string, pagination ...int) ([]models.Notification, error) {
	query := notificationSelect + ` WHERE n.user_id = ?`
	args := []any{userID}

	// only filter by category if one was picked.
	// "alerts" is everything except chat messages (the bell in the top bar)
	if category == "alerts" {
		query += ` AND n.category <> 'messages'`
	} else if category != "" && category != "all" {
		if !IsCategory(category) {
			return nil, ErrInvalidCategory
		}
		query += ` AND n.category = ?`
		args = append(args, category)
	}

	// newest first, and if no page was given we just cap it at 100
	query += ` ORDER BY n.created_at DESC, n.id DESC`
	if len(pagination) >= 2 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pagination[0], pagination[1])
	} else {
		query += ` LIMIT 100`
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.Notification{}
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, notification)
	}

	return result, rows.Err()
}

// UnreadCount counts how many notifications the user didnt read yet.
// used for the red number badge in the nav
func UnreadCount(db *sql.DB, userID int, categories ...string) (int, error) {
	category := ""
	if len(categories) > 0 {
		category = categories[0]
	}
	if category != "" && category != "all" && category != "alerts" && !IsCategory(category) {
		return 0, ErrInvalidCategory
	}

	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = ? AND is_read = 0
	`
	args := []any{userID}
	if category == "alerts" {
		query += ` AND category <> 'messages'`
	} else if category != "" && category != "all" {
		query += ` AND category = ?`
		args = append(args, category)
	}

	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// MarkRead marks one notification as read.
// we also check user_id so nobody can mark someone elses notification
func MarkRead(db *sql.DB, userID int, notificationID int64) error {
	_, err := db.Exec(`
		UPDATE notifications
		SET is_read = 1
		WHERE id = ? AND user_id = ?
	`, notificationID, userID)
	return err
}

// MarkAllRead is for the "mark all as read" button
func MarkAllRead(db *sql.DB, userID int) error {
	_, err := db.Exec(`
		UPDATE notifications
		SET is_read = 1
		WHERE user_id = ? AND is_read = 0
	`, userID)
	return err
}

// MarkMessageNotificationsRead clears message alerts for one private chat.
// Other notification categories and other conversations stay unread.
func MarkMessageNotificationsRead(db *sql.DB, userID int, chatID int64) error {
	_, err := db.Exec(`
		UPDATE notifications
		SET is_read = 1
		WHERE user_id = ?
		  AND category = 'messages'
		  AND type = 'new_message'
		  AND related_id = ?
	`, userID, chatID)
	return err
}

// DeleteFromActor removes the notifications one person caused about one thing,
// for example the "liked your post" alert after the like is taken back
func DeleteFromActor(db *sql.DB, userID int, category, notificationType string, actorID int, relatedID *int64) error {
	query := `
		DELETE FROM notifications
		WHERE user_id = ? AND category = ? AND type = ? AND actor_id = ?
	`
	args := []any{userID, category, notificationType, actorID}
	if relatedID != nil {
		query += ` AND related_id = ?`
		args = append(args, *relatedID)
	}
	_, err := db.Exec(query, args...)
	return err
}

// UpsertMessage keeps one unread "new message" alert per chat. a new message
// refreshes that alert (text, sender and time) instead of adding another row,
// so a busy chat does not flood the notifications page
func UpsertMessage(db *sql.DB, userID, actorID int, chatID int64, message string) (models.Notification, error) {
	var existingID int64
	err := db.QueryRow(`
		SELECT id FROM notifications
		WHERE user_id = ? AND category = 'messages' AND type = 'new_message'
		  AND related_id = ? AND is_read = 0
		ORDER BY id DESC
		LIMIT 1
	`, userID, chatID).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return Create(db, userID, models.CreateNotificationRequest{
			ActorID:   &actorID,
			Category:  "messages",
			Type:      "new_message",
			Message:   message,
			RelatedID: &chatID,
		})
	}
	if err != nil {
		return models.Notification{}, err
	}

	if _, err := db.Exec(`
		UPDATE notifications
		SET message = ?, actor_id = ?, created_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, message, actorID, existingID); err != nil {
		return models.Notification{}, err
	}
	return GetByID(db, userID, existingID)
}

// GetByID gets one notification, only if it belongs to this user
func GetByID(db *sql.DB, userID int, notificationID int64) (models.Notification, error) {
	return scanNotification(
		db.QueryRow(
			notificationSelect+` WHERE n.user_id = ? AND n.id = ?`,
			userID,
			notificationID,
		),
	)
}

// GetLatestForActor finds the newest notification of a type that came from a
// specific person (actor). the follow request notification is made by a
// database trigger, so we use this to find it and push it to the user live
func GetLatestForActor(db *sql.DB, userID int, category, notificationType string, actorID int) (models.Notification, error) {
	return scanNotification(
		db.QueryRow(
			notificationSelect+`
				WHERE n.user_id = ?
				  AND n.category = ?
				  AND n.type = ?
				  AND n.actor_id = ?
				ORDER BY n.id DESC
				LIMIT 1
			`,
			userID,
			category,
			notificationType,
			actorID,
		),
	)
}

// ListByRelatedID gets all notifications that point to the same thing
// (like the same join request or the same event), so we can push them all live
func ListByRelatedID(db *sql.DB, category, notificationType string, relatedID int64) ([]models.Notification, error) {
	rows, err := db.Query(
		notificationSelect+`
			WHERE n.category = ?
			  AND n.type = ?
			  AND n.related_id = ?
			ORDER BY n.id
		`,
		category,
		notificationType,
		relatedID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, notification)
	}
	return result, rows.Err()
}

// IsCategory checks the category is one of the 5 we support
func IsCategory(category string) bool {
	switch category {
	case "requests", "groups", "events", "messages", "posts":
		return true
	default:
		return false
	}
}

// the base SELECT we use everywhere for notifications.
// the joins add the current status of the thing the notification is about
// (was the join request accepted? did i answer the invite? did i rsvp?)
// so the page knows if it should still show the buttons or not
const notificationSelect = `
	SELECT
		n.id,
		n.user_id,
		n.actor_id,
		n.category,
		n.type,
		n.message,
		n.related_id,
		n.is_read,
		n.created_at,

		gjr.status AS request_status,
		gi.status AS invitation_status,
		(SELECT status FROM user_followers f WHERE n.type = 'follow_request'
		 AND f.target_id = n.user_id AND f.follower_id = n.actor_id) AS follow_status,
		(SELECT response FROM event_rsvps v WHERE n.type = 'event_created'
		 AND v.event_id = n.related_id AND v.user_id = n.user_id) AS event_response,

		-- the group a notification is about, so the page can link to it
		CASE
			WHEN n.category = 'events' THEN (SELECT e.group_id FROM events e WHERE e.id = n.related_id)
			WHEN n.type = 'join_request' THEN gjr.group_id
			WHEN n.type = 'invitation' THEN gi.group_id
			WHEN n.category = 'groups' THEN n.related_id
		END AS group_id,
		COALESCE(actor.first_name || ' ' || actor.last_name, '') AS actor_name,
		COALESCE(actor_profile.avatar_path, '') AS actor_avatar

	FROM notifications n

	LEFT JOIN group_join_requests gjr
		ON n.category = 'groups'
		AND n.type = 'join_request'
		AND gjr.id = n.related_id
		AND gjr.user_id = n.actor_id

	LEFT JOIN group_invitations gi
		ON n.category = 'groups'
		AND n.type = 'invitation'
		AND gi.id = n.related_id
		AND gi.user_id = n.user_id

	LEFT JOIN user actor ON actor.id = n.actor_id
	LEFT JOIN profile actor_profile ON actor_profile.user_id = n.actor_id
`

// small interface so scanNotification works with both QueryRow and Query rows
type rowScanner interface {
	Scan(dest ...any) error
}

// scanNotification reads one row into a Notification struct.
// sqlite keeps is_read as 0/1 so we turn it into a bool at the end
func scanNotification(row rowScanner) (models.Notification, error) {
	var notification models.Notification
	var isRead int
	if err := row.Scan(
		&notification.ID,
		&notification.UserID,
		&notification.ActorID,
		&notification.Category,
		&notification.Type,
		&notification.Message,
		&notification.RelatedID,
		&isRead,
		&notification.CreatedAt,
		&notification.RequestStatus,
		&notification.InvitationStatus,
		&notification.FollowStatus,
		&notification.EventResponse,
		&notification.GroupID,
		&notification.ActorName,
		&notification.ActorAvatar,
	); err != nil {
		return models.Notification{}, err
	}
	notification.IsRead = isRead == 1
	return notification, nil
}
