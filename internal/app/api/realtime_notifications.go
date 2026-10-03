package api

import (
	"log"

	"social/database/notifications"
	"social/internal/models"
)

func (app *App) deliverNotification(notification models.Notification) {
	if app.Realtime == nil {
		return
	}
	event := realtimeEvent{
		Type:         "notification",
		Notification: &notification,
	}
	if err := app.Realtime.SendToUser(notification.UserID, event); err != nil {
		log.Printf("deliver realtime notification: %v", err)
	}
}

func (app *App) deliverNotificationByID(userID int, notificationID int64) {
	notification, err := notifications.GetByID(app.DB, userID, notificationID)
	if err != nil {
		log.Printf("load realtime notification %d: %v", notificationID, err)
		return
	}
	app.deliverNotification(notification)
}

func (app *App) deliverNotificationsByRelatedID(category, notificationType string, relatedID int64) {
	items, err := notifications.ListByRelatedID(app.DB, category, notificationType, relatedID)
	if err != nil {
		log.Printf("load realtime notifications for %s/%s/%d: %v", category, notificationType, relatedID, err)
		return
	}
	for _, notification := range items {
		app.deliverNotification(notification)
	}
}

// notify saves a notification for one user and pushes it live to their open tabs.
// a failed notification is only logged, it should never fail the action that caused it
func (app *App) notify(userID, actorID int, category, notificationType, message string, relatedID *int64) {
	if userID <= 0 || userID == actorID {
		return
	}
	request := models.CreateNotificationRequest{
		Category:  category,
		Type:      notificationType,
		Message:   message,
		RelatedID: relatedID,
	}
	if actorID > 0 {
		request.ActorID = &actorID
	}
	notification, err := notifications.Create(app.DB, userID, request)
	if err != nil {
		log.Printf("create %s/%s notification: %v", category, notificationType, err)
		return
	}
	app.deliverNotification(notification)
}

// userFullName gives "First Last" for messages like "Bob Jones liked your post"
func (app *App) userFullName(userID int) string {
	var name string
	err := app.DB.QueryRow(`SELECT first_name || ' ' || last_name FROM user WHERE id = ?`, userID).Scan(&name)
	if err != nil || name == "" {
		return "Someone"
	}
	return name
}

// groupTitle gives the name of a group for notification messages
func (app *App) groupTitle(groupID int64) string {
	var title string
	if err := app.DB.QueryRow(`SELECT title FROM groups WHERE id = ?`, groupID).Scan(&title); err != nil {
		return "the group"
	}
	return title
}

// int64Ptr is a small helper for the optional related id
func int64Ptr(value int64) *int64 {
	return &value
}
