// handlers for the notifications page:
// GET the list, mark one/all as read, and do the actions (accept, decline, join...)
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"social/database/events"
	"social/database/groups"
	"social/database/notifications"
	"social/database/profiles"
	"social/internal/helpers"
	"social/internal/models"
	"strconv"
)

// Notifications handles GET /api/notifications
// returns one page of the users notifications + the unread count
func (app App) Notifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		helpers.WriteJson(w, http.StatusMethodNotAllowed, map[string]any{
			"status":  false,
			"message": "method not allowed",
		})
		return
	}

	// read limit and offset from the url
	page, err := parsePage(r)
	if err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "limit must be between 1 and 50 and offset cannot be negative",
		})
		return
	}

	category := r.URL.Query().Get("category")
	// we ask for one extra row so we know if there is another page (hasMore)
	result, err := notifications.List(app.DB, userID, category, page.Limit+1, page.Offset)
	if errors.Is(err, notifications.ErrInvalidCategory) {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": err.Error(),
		})
		return
	}
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not load notifications",
		})
		return
	}
	result, hasMore := trimPage(result, page, true)

	unreadCount, err := notifications.UnreadCount(app.DB, userID, category)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not load notification count",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":        true,
		"notifications": result,
		"unreadCount":   unreadCount,
		"hasMore":       hasMore,
		"nextOffset":    page.Offset + len(result),
	})
}

// MarkNotificationRead handles PATCH /api/notifications/{id}/read
func (app App) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", http.MethodPatch)
		helpers.WriteJson(w, http.StatusMethodNotAllowed, map[string]any{
			"status":  false,
			"message": "method not allowed",
		})
		return
	}

	notificationID, err := strconv.ParseInt(r.PathValue("notificationID"), 10, 64)
	if err != nil || notificationID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid notification id",
		})
		return
	}

	if err := notifications.MarkRead(app.DB, userID, notificationID); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not mark notification as read",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true})
}

// MarkAllNotificationsRead handles PATCH /api/notifications/read-all
func (app App) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", http.MethodPatch)
		helpers.WriteJson(w, http.StatusMethodNotAllowed, map[string]any{
			"status":  false,
			"message": "method not allowed",
		})
		return
	}

	if err := notifications.MarkAllRead(app.DB, userID); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not mark notifications as read",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true})
}

// body for the action request, like {"action": "accept"}
type notificationActionRequest struct {
	Action string `json:"action"`
}

// ApplyNotificationAction handles PATCH /api/notifications/{id}/action
// this is when the user clicks a button inside a notification (accept, decline, join, rsvp)
func (app App) ApplyNotificationAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", http.MethodPatch)
		helpers.WriteJson(w, http.StatusMethodNotAllowed, map[string]any{
			"status":  false,
			"message": "method not allowed",
		})
		return
	}

	notificationID, err := strconv.ParseInt(
		r.PathValue("notificationID"),
		10,
		64,
	)
	if err != nil || notificationID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid notification id",
		})
		return
	}

	var request notificationActionRequest
	// read the body, small size limit and no unknown fields
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid notification action",
		})
		return
	}

	// make sure the notification exists and is mine
	notification, err := notifications.GetByID(app.DB, userID, notificationID)
	if errors.Is(err, sql.ErrNoRows) {
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "notification not found",
		})
		return
	}
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not load notification",
		})
		return
	}

	// do the actual action, not found errors become 404
	if err := app.applyNotificationAction(userID, notification, request.Action); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, groups.ErrGroupNotFound) || errors.Is(err, groups.ErrInvitationNotFound) || errors.Is(err, events.ErrEventNotFound) {
			status = http.StatusNotFound
		}
		helpers.WriteJson(w, status, map[string]any{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	// after the action is done the notification counts as read
	if err := notifications.MarkRead(app.DB, userID, notificationID); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "action completed but notification could not be marked read",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"action": request.Action,
	})
}

// applyNotificationAction decides what to do based on the category and type
// of the notification and the button the user clicked
func (app App) applyNotificationAction(userID int, notification models.Notification, action string) error {

	switch notification.Category {

	// follow requests: accept or decline
	case "requests":
		if notification.Type != "follow_request" || notification.ActorID == nil {
			return errors.New("this request cannot be acted on")
		}

		switch action {
		case "accept":
			if err := profiles.DecideFollowRequest(
				app.DB,
				userID,
				*notification.ActorID,
				true,
			); err != nil {
				return err
			}
			app.notify(*notification.ActorID, userID, "requests", "follow_accepted", app.userFullName(userID)+" accepted your follow request", nil)
			return nil

		case "decline":
			return profiles.DecideFollowRequest(
				app.DB,
				userID,
				*notification.ActorID,
				false,
			)

		default:
			return errors.New("request action must be accept or decline")
		}

	// group stuff: join requests (for the group owner) and invitations (for the invited user)
	case "groups":
		if notification.RelatedID == nil {
			return errors.New("group notification is missing its group")
		}

		switch notification.Type {

		// find which group the request is for, only if its still pending
		case "join_request":
			if notification.ActorID == nil {
				return errors.New("join request is missing requester")
			}

			var groupID int64
			err := app.DB.QueryRow(`
				SELECT group_id
				FROM group_join_requests
				WHERE id = ?
				  AND user_id = ?
				  AND status = 'pending'
			`, *notification.RelatedID, *notification.ActorID).Scan(&groupID)
			if err != nil {
				return err
			}

			switch action {
			case "accept":
				if err := groups.AcceptJoinRequest(
					app.DB,
					userID,
					*notification.ActorID,
					groupID,
				); err != nil {
					return err
				}
				app.notify(*notification.ActorID, userID, "groups", "join_accepted", "Your request to join "+app.groupTitle(groupID)+" was accepted", int64Ptr(groupID))
				return nil

			case "reject":
				if err := groups.RejectJoinRequest(
					app.DB,
					userID,
					*notification.ActorID,
					groupID,
				); err != nil {
					return err
				}
				app.notify(*notification.ActorID, userID, "groups", "join_rejected", "Your request to join "+app.groupTitle(groupID)+" was declined", int64Ptr(groupID))
				return nil

			default:
				return errors.New(
					"join request action must be accept or reject",
				)
			}

		// someone invited me to a group
		case "invitation":
			switch action {
			case "join":
				groupID, inviterID, err := groups.AcceptInvitation(
					app.DB,
					userID,
					*notification.RelatedID,
				)
				if err != nil {
					return err
				}
				app.notify(inviterID, userID, "groups", "invitation_accepted", app.userFullName(userID)+" accepted your invitation to "+app.groupTitle(groupID), int64Ptr(groupID))
				return nil

			case "decline":
				return groups.DeclineInvitation(
					app.DB,
					userID,
					*notification.RelatedID,
				)

			default:
				return errors.New(
					"invitation action must be join or decline",
				)
			}

		default:
			return errors.New("unsupported group notification type")
		}

	// new event in my group: going or not going
	case "events":
		if notification.RelatedID == nil {
			return errors.New("event notification is missing its event")
		}

		switch action {
		case "rsvp":
			return events.SetRSVP(
				app.DB,
				userID,
				*notification.RelatedID,
				"going",
			)

		case "decline":
			return events.SetRSVP(
				app.DB,
				userID,
				*notification.RelatedID,
				"declined",
			)

		default:
			return errors.New("event action must be rsvp or decline")
		}

	default:
		return errors.New("unsupported notification category")
	}
}
