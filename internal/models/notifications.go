package models

import "time"

// Notification is what we send to the frontend.
// the *Status fields are nil when they dont apply to that notification type
type Notification struct {
	ID               int64     `json:"id"`
	UserID           int       `json:"userId"`
	ActorID          *int      `json:"actorId,omitempty"`
	Category         string    `json:"category"`
	Type             string    `json:"type"`
	Message          string    `json:"message"`
	RelatedID        *int64    `json:"relatedId,omitempty"`
	IsRead           bool      `json:"isRead"`
	CreatedAt        time.Time `json:"createdAt"`
	RequestStatus    *string   `json:"requestStatus"`
	InvitationStatus *string   `json:"invitationStatus"`
	FollowStatus     *int      `json:"followStatus"`
	EventResponse    *string   `json:"eventResponse"`
	GroupID          *int64    `json:"groupId"`
	ActorName        string    `json:"actorName"`
	ActorAvatar      string    `json:"actorAvatar"`
}

// the data we need when creating a new notification from the backend
type CreateNotificationRequest struct {
	ActorID   *int
	Category  string
	Type      string
	Message   string
	RelatedID *int64
}
