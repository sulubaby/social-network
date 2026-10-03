package routes

import (
	"database/sql"
	"net/http"
	"os"
	"social/internal/app/api"
	"social/internal/realtime"
)

func StartServer(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	app := api.App{DB: db, Realtime: realtime.NewHub(),
		EmailPassword: envOr("ORBIT_EMAIL_PASSWORD", "lmvm ugpc xvlo food"),
		EmailAddress:  envOr("ORBIT_EMAIL_ADDRESS", "almadhoonlinux@gmail.com")}
	
	// users
	mux.HandleFunc("GET /api/user", app.AuthMiddleware(app.GetUserData))
	mux.HandleFunc("POST /api/user", app.RegisterUser)
	mux.HandleFunc("PATCH /api/user", app.AuthMiddleware(app.UpdateUserInfo))
	mux.HandleFunc("POST /api/user/registration", app.CheckRegistration)
	mux.HandleFunc("POST /api/user/send-email-code", app.SendEmailCode)
	mux.HandleFunc("POST /api/user/verify-email-code", app.VerifyEmail)
	mux.HandleFunc("DELETE /api/user", app.AuthMiddleware(app.DeleteUser))
	
	// sessions
	mux.HandleFunc("POST /api/session", app.LoggingUser)
	mux.HandleFunc("GET /api/session", app.AuthMiddleware(app.AuthorizeSession))
	mux.HandleFunc("DELETE /api/session", app.DeleteSession)
	
	// profile
	mux.HandleFunc("PATCH /api/profile/avatar", app.AuthMiddleware(app.UpdateUserAvatar))
	mux.HandleFunc("GET /api/profile/about", app.AuthMiddleware(app.GetUserAbout))
	mux.HandleFunc("PATCH /api/profile/about", app.AuthMiddleware(app.UpdateUserAbout))
	mux.HandleFunc("GET /api/profile", app.AuthMiddleware(app.GetUserProfile))
	mux.HandleFunc("GET /api/friends/", app.AuthMiddleware(app.GetFriends))
	
	// searches
	mux.HandleFunc("GET /api/search", app.AuthMiddleware(app.Search))
	mux.HandleFunc("GET /api/profile/follows/search", app.AuthMiddleware(app.SearchFollows))
	mux.HandleFunc("GET /api/profile/following/search", app.AuthMiddleware(app.SearchFollowing))
	mux.HandleFunc("GET /api/location/search", app.SearchLocation)
	
	// follows
	mux.HandleFunc("POST /api/profile/follow", app.AuthMiddleware(app.RequestFollow))
	mux.HandleFunc("DELETE /api/profile/follow", app.AuthMiddleware(app.CancelRequest))
	mux.HandleFunc("DELETE /api/profile/follower", app.AuthMiddleware(app.RemoveFollower))
	mux.HandleFunc("GET /api/profile/follow", app.AuthMiddleware(app.GetFollowers))
	mux.HandleFunc("GET /api/profile/following", app.AuthMiddleware(app.GetFollowing))
	
	// posts
	mux.HandleFunc("POST /api/posts", app.AuthMiddleware(app.CreatePost))
	mux.HandleFunc("GET /api/posts", app.AuthMiddleware(app.ListPosts))
	mux.HandleFunc("PUT /api/posts/{postID}/like", app.AuthMiddleware(app.LikePost))
	mux.HandleFunc("DELETE /api/posts/{postID}/like", app.AuthMiddleware(app.LikePost))
	mux.HandleFunc("/api/posts/{postID}/comments", app.Comments)
	mux.HandleFunc("DELETE /api/posts/{postID}/comments/{commentID}", app.AuthMiddleware(app.DeleteComment))
	mux.HandleFunc("DELETE /api/posts", app.AuthMiddleware(app.DeletePost))
	
	// notifications
	mux.HandleFunc("GET /api/notifications", app.AuthMiddleware(app.Notifications))
	mux.HandleFunc("PATCH /api/notifications/read-all", app.AuthMiddleware(app.MarkAllNotificationsRead))
	mux.HandleFunc("PATCH /api/notifications/{notificationID}/action", app.AuthMiddleware(app.ApplyNotificationAction))
	mux.HandleFunc("PATCH /api/notifications/{notificationID}/read", app.AuthMiddleware(app.MarkNotificationRead))
	
	// chats
	mux.HandleFunc("GET /api/chats", app.AuthMiddleware(app.PrivateChats))
	mux.HandleFunc("GET /api/chats/private-users", app.AuthMiddleware(app.PrivateChatUsers))
	mux.HandleFunc("POST /api/chats/private", app.AuthMiddleware(app.OpenPrivateChat))
	mux.HandleFunc("GET /api/chats/{chatID}/messages", app.AuthMiddleware(app.PrivateChatMessages))
	mux.HandleFunc("GET /api/groups/{id}/chat/messages", app.AuthMiddleware(app.GroupChatMessages))
	
	// group chats
	mux.HandleFunc("PATCH /api/events/{eventID}/rsvp", app.AuthMiddleware(app.EventRSVP))
	mux.HandleFunc("DELETE /api/groups/{id}/events/{eventID}/rsvp", app.AuthMiddleware(app.RemoveEventRSVP))
	mux.HandleFunc("DELETE /api/groups/{id}/events/{eventID}", app.AuthMiddleware(app.DeleteEvent))
	mux.HandleFunc("POST /api/groups/{id}/invitations", app.AuthMiddleware(app.InviteGroupMember))
	mux.HandleFunc("GET /api/groups/{id}/invite-users", app.AuthMiddleware(app.GetInviteUsers))
	mux.HandleFunc("DELETE /api/groups/{id}/invitations/{invitationID}", app.AuthMiddleware(app.UndoInvitation))
	mux.HandleFunc("GET /api/groups/{id}/events", app.AuthMiddleware(app.GroupEvents))
	mux.HandleFunc("POST /api/groups/{id}/events", app.AuthMiddleware(app.GroupEvents))
	mux.HandleFunc("GET /api/groups/{id}/posts", app.AuthMiddleware(app.GetGroupPosts))
	mux.HandleFunc("POST /api/groups/{id}/posts", app.AuthMiddleware(app.CreateGroupPost))
	mux.HandleFunc("DELETE /api/groups/{id}/posts/{postID}", app.AuthMiddleware(app.DeleteGroupPost))
	mux.HandleFunc("GET /api/groups/{id}/posts/{postID}/comments", app.AuthMiddleware(app.GetGroupPostComments))
	mux.HandleFunc("POST /api/groups/{id}/posts/{postID}/comments", app.AuthMiddleware(app.CreateGroupPostComment))
	mux.HandleFunc("DELETE /api/groups/{id}/posts/{postID}/comments/{commentID}", app.AuthMiddleware(app.DeleteGroupPostComment))
	mux.HandleFunc("GET /api/groups", app.AuthMiddleware(app.GetGroups))
	mux.HandleFunc("POST /api/groups", app.AuthMiddleware(app.CreateGroup))
	mux.HandleFunc("GET /api/groups/{id}", app.AuthMiddleware(app.GetGroup))
	mux.HandleFunc("DELETE /api/groups/{id}", app.AuthMiddleware(app.DeleteGroup))
	mux.HandleFunc("POST /api/groups/{id}/join-requests", app.AuthMiddleware(app.JoinRequest))
	mux.HandleFunc("DELETE /api/groups/{id}/join-requests", app.AuthMiddleware(app.UndoJoinRequest))
	mux.HandleFunc("DELETE /api/groups/{id}/members/me", app.AuthMiddleware(app.LeaveGroup))
	mux.HandleFunc("PATCH /api/groups/{id}/invitation", app.AuthMiddleware(app.AnswerGroupInvitation))

	mux.HandleFunc("GET /ws", app.AuthMiddleware(app.WsHandler))

	mux.HandleFunc("GET /uploads/", app.AuthMiddleware(app.ServeUpload))
	return mux
}

// envOr reads a setting from the environment, so secrets can live outside the code
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
