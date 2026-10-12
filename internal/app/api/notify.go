package api

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"social/database/chats"
	"social/database/groups"
	"social/database/notifications"
	"social/database/posts"
	"social/database/preferences"
	"social/database/users"
	"social/internal/helpers"
	"social/internal/models"
)

// finds @username in posts and comments, the char before @ cannot be a letter so emails dont count
var mentionPattern = regexp.MustCompile(`(?:^|[^\w@])@([A-Za-z0-9_.]{2,30})`)

// chat usernames follow different rules so they get their own pattern
var chatMentionPattern = regexp.MustCompile(`(?:^|[^\w@])@([A-Za-z0-9_-]{3,12})`)

// skipsSpamCheck is for notifications the user really has to get, like invites
func skipsSpamCheck(n models.NewNotification) bool {
	if n.GroupInviteUserID != nil || n.EventInviteUserID != nil {
		return true
	}

	if n.PostMentionUserID != nil && n.GroupID != nil && n.PostIDTag == nil {
		return true
	}

	return n.CommentMentionUserID != nil && n.GroupID != nil && n.CommentIDTag == nil
}

// notify is the one place every notification goes through
// it skips yourself, respects the settings, blocks spam, saves it, then sends it live on the websocket
func (app *App) notify(actorID int, n models.NewNotification) bool {
	if n.UserID <= 0 || n.UserID == actorID {
		return false
	}

	// the user can turn some types off in settings
	allowed, err := preferences.ShouldNotify(
		app.DB,
		n.UserID,
		actorID,
		helpers.GetNotificationPreferenceType(n),
	)

	if err != nil {
		log.Println("failed to check notification preference:", err)
	} else if !allowed {
		return false
	}

	if !skipsSpamCheck(n) {
		spam, err := notifications.IsSpam(app.DB, actorID, n)

		if err != nil {
			log.Println("failed to check notification spam:", err)
		} else if spam {
			return false
		}
	}

	id, err := notifications.InsertNotification(app.DB, n)

	if err != nil {
		log.Println("failed to insert notification:", err)
		return false
	}

	// send the new unread count too so the bell badge updates without a fetch
	unread, err := notifications.GetUnreadCount(app.DB, n.UserID)

	if err != nil {
		log.Println("failed to count unread notifications:", err)
		unread = -1
	}

	app.sendNotification(actorID, n.UserID, n, id, unread)

	return true
}

// actorName is the name shown in the message, full name first, then username, then "Someone"
func (app *App) actorName(userID int) string {
	user, err := users.GetUserSimpleData(app.DB, userID)

	if err != nil {
		return "Someone"
	}

	name := strings.TrimSpace(user.FirstName + " " + user.LastName)

	if name == "" {
		name = user.UserName
	}

	if name == "" {
		return "Someone"
	}

	return name
}

// mentionedUserIDs turns the @names in a text into user ids, skip has people we already notified
func (app *App) mentionedUserIDs(content string, skip map[int]bool) []int {
	var ids []int

	seen := map[int]bool{}

	for _, match := range mentionPattern.FindAllStringSubmatch(content, -1) {
		// a dot at the end is the end of the sentence, not part of the name
		username := strings.TrimRight(match[1], ".")

		if len(username) < 2 {
			continue
		}

		id := users.GetUserID(app.DB, username)

		if id <= 0 || skip[id] || seen[id] {
			continue
		}

		seen[id] = true
		ids = append(ids, id)
	}

	return ids
}

// groupMentionedUserIDs is the same but only keeps people inside the group
func (app *App) groupMentionedUserIDs(content string, groupID int, skip map[int]bool) []int {
	if groupID <= 0 {
		return nil
	}

	var ids []int

	for _, id := range app.mentionedUserIDs(content, skip) {
		member, err := groups.IsMember(app.DB, groupID, id)

		if err != nil || !member {
			continue
		}

		ids = append(ids, id)
	}

	return ids
}

// optionalGroupID gives nil for 0 so the db column stays empty
func optionalGroupID(groupID int) *int {
	if groupID <= 0 {
		return nil
	}

	return &groupID
}

// notifyComment tells the comment owner about a reply, or the post owner about a new comment
// then the people mentioned in it
func (app *App) notifyComment(actorID int, comment models.Comment) {
	name := app.actorName(actorID)
	actor := actorID
	postID := comment.PostID
	commentID := comment.ID

	notified := map[int]bool{actorID: true}

	if comment.RepltTo != nil {
		ownerID, _, err := posts.GetCommentOwner(app.DB, *comment.RepltTo)

		if err != nil {
			log.Println("could not find comment owner:", err)
		} else if !notified[ownerID] {
			notified[ownerID] = true

			app.notify(actorID, models.NewNotification{
				UserID:             ownerID,
				Message:            fmt.Sprintf("%s replied to your comment", name),
				PostIDTag:          &postID,
				CommentIDTag:       &commentID,
				CommentReplyUserID: &actor,
			})
		}
	} else {
		ownerID, err := posts.GetPostOwnerID(app.DB, postID)

		if err != nil {
			log.Println("could not find post owner:", err)
		} else if !notified[ownerID] {
			notified[ownerID] = true

			app.notify(actorID, models.NewNotification{
				UserID:             ownerID,
				Message:            fmt.Sprintf("%s commented on your post", name),
				PostIDTag:          &postID,
				CommentIDTag:       &commentID,
				CommentReplyUserID: &actor,
			})
		}
	}

	for _, id := range app.mentionedUserIDs(comment.Content, notified) {
		notified[id] = true

		app.notify(actorID, models.NewNotification{
			UserID:               id,
			Message:              fmt.Sprintf("%s mentioned you in a comment", name),
			PostIDTag:            &postID,
			CommentIDTag:         &commentID,
			CommentMentionUserID: &actor,
		})
	}
}

// notifyCommentLike only sends when the vote is still a like after the toggle
func (app *App) notifyCommentLike(actorID, commentID int) {
	vote, err := posts.GetUserCommentVote(app.DB, commentID, actorID)

	if err != nil || vote != 1 {
		return
	}

	ownerID, postID, err := posts.GetCommentOwner(app.DB, commentID)

	if err != nil {
		log.Println("could not find comment owner:", err)
		return
	}

	actor := actorID

	app.notify(actorID, models.NewNotification{
		UserID:            ownerID,
		Message:           fmt.Sprintf("%s liked your comment", app.actorName(actorID)),
		PostIDTag:         &postID,
		CommentIDTag:      &commentID,
		CommentLikeUserID: &actor,
	})
}

// the group versions below work the same way but keep the group id so the click opens the group
func (app *App) notifyGroupComment(actorID int, comment models.GroupComment) {
	postOwnerID, groupID, err := groups.GetGroupPostMeta(app.DB, comment.GroupPostID)

	if err != nil {
		log.Println("could not find group post:", err)
		return
	}

	name := app.actorName(actorID)
	actor := actorID

	notified := map[int]bool{actorID: true}

	if comment.ReplyTo != nil {
		ownerID, _, _, err := groups.GetGroupCommentMeta(app.DB, *comment.ReplyTo)

		if err != nil {
			log.Println("could not find group comment owner:", err)
		} else if !notified[ownerID] {
			notified[ownerID] = true

			app.notify(actorID, models.NewNotification{
				UserID:             ownerID,
				Message:            fmt.Sprintf("%s replied to your comment in a group", name),
				CommentReplyUserID: &actor,
				GroupID:            optionalGroupID(groupID),
			})
		}
	} else if !notified[postOwnerID] {
		notified[postOwnerID] = true

		app.notify(actorID, models.NewNotification{
			UserID:             postOwnerID,
			Message:            fmt.Sprintf("%s commented on your group post", name),
			CommentReplyUserID: &actor,
			GroupID:            optionalGroupID(groupID),
		})
	}

	for _, id := range app.groupMentionedUserIDs(comment.Content, groupID, notified) {
		notified[id] = true

		app.notify(actorID, models.NewNotification{
			UserID:               id,
			Message:              fmt.Sprintf("%s mentioned you in a group comment", name),
			CommentMentionUserID: &actor,
			GroupID:              optionalGroupID(groupID),
		})
	}
}

func (app *App) notifyGroupCommentLike(actorID, commentID int) {
	vote, err := groups.GetGroupCommentVote(app.DB, commentID, actorID)

	if err != nil || vote != 1 {
		return
	}

	ownerID, _, groupID, err := groups.GetGroupCommentMeta(app.DB, commentID)

	if err != nil {
		log.Println("could not find group comment owner:", err)
		return
	}

	actor := actorID

	app.notify(actorID, models.NewNotification{
		UserID:            ownerID,
		Message:           fmt.Sprintf("%s liked your comment in a group", app.actorName(actorID)),
		CommentLikeUserID: &actor,
		GroupID:           optionalGroupID(groupID),
	})
}

// the stored value has to match, if they removed the reaction there is nothing to say
func (app *App) notifyGroupPostReaction(actorID, postID, value int) {
	stored, err := groups.GetGroupPostReactionValue(app.DB, postID, actorID)

	if err != nil || stored != value {
		return
	}

	ownerID, groupID, err := groups.GetGroupPostMeta(app.DB, postID)

	if err != nil {
		log.Println("could not find group post owner:", err)
		return
	}

	actor := actorID
	name := app.actorName(actorID)

	notification := models.NewNotification{
		UserID:  ownerID,
		GroupID: optionalGroupID(groupID),
	}

	if value == 1 {
		notification.PostLikeUserID = &actor
		notification.Message = fmt.Sprintf("%s liked your group post", name)
	} else {
		notification.PostDislikeUserID = &actor
		notification.Message = fmt.Sprintf("%s disliked your group post", name)
	}

	app.notify(actorID, notification)
}

func (app *App) notifyGroupPostPeople(actorID, groupID int, content string, tagged []int) {
	name := app.actorName(actorID)
	actor := actorID

	notified := map[int]bool{actorID: true}

	for _, id := range tagged {
		if notified[id] {
			continue
		}

		notified[id] = true

		app.notify(actorID, models.NewNotification{
			UserID:               id,
			Message:              fmt.Sprintf("%s tagged you in a group post", name),
			CommentMentionUserID: &actor,
			GroupID:              optionalGroupID(groupID),
		})
	}

	for _, id := range app.groupMentionedUserIDs(content, groupID, notified) {
		notified[id] = true

		app.notify(actorID, models.NewNotification{
			UserID:               id,
			Message:              fmt.Sprintf("%s mentioned you in a group post", name),
			CommentMentionUserID: &actor,
			GroupID:              optionalGroupID(groupID),
		})
	}
}

func (app *App) notifyGroupRequestAccepted(ownerID, groupID, requesterID int) {
	group, err := groups.GetGroupData(app.DB, groupID)

	if err != nil {
		log.Println("could not get group data:", err)
		return
	}

	actor := ownerID
	gid := groupID

	app.notify(ownerID, models.NewNotification{
		UserID:            requesterID,
		Message:           fmt.Sprintf("%s accepted your request to join the group \"%s\"", app.actorName(ownerID), group.Title),
		GroupAcceptUserID: &actor,
		GroupID:           &gid,
	})
}

func (app *App) notifyGroupInviteAccepted(actorID, groupID, inviterID int) {
	if inviterID <= 0 {
		return
	}

	group, err := groups.GetGroupData(app.DB, groupID)

	if err != nil {
		log.Println("could not get group data:", err)
		return
	}

	actor := actorID
	gid := groupID

	app.notify(actorID, models.NewNotification{
		UserID:            inviterID,
		Message:           fmt.Sprintf("%s accepted your invitation to join the group \"%s\"", app.actorName(actorID), group.Title),
		GroupAcceptUserID: &actor,
		GroupID:           &gid,
	})
}

// notifyEventInvite tells every member of the group about a new event
func (app *App) notifyEventInvite(actorID int, event models.GroupEvent) {
	memberIDs, err := groups.GetActiveMemberIDs(app.DB, event.GroupID)

	if err != nil {
		log.Println("could not get group members:", err)
		return
	}

	name := app.actorName(actorID)
	actor := actorID
	eventID := event.ID
	groupID := event.GroupID

	for _, id := range memberIDs {
		app.notify(actorID, models.NewNotification{
			UserID:            id,
			Message:           fmt.Sprintf("%s invited you to the event \"%s\"", name, event.Title),
			EventInviteUserID: &actor,
			EventID:           &eventID,
			GroupID:           &groupID,
		})
	}
}

// notifyChatMentions is for @names in group chats, private chats dont send it
func (app *App) notifyChatMentions(actorID, groupID int, content string) {
	matches := chatMentionPattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return
	}

	isPrivate, groupName, err := chats.GetChatMeta(app.DB, groupID)

	if err != nil {
		log.Println("could not get chat meta:", err)
		return
	}

	if isPrivate {
		return
	}

	name := app.actorName(actorID)
	actor := actorID
	gid := groupID

	message := fmt.Sprintf("%s mentioned you in the group chat", name)

	if groupName != "" {
		message = fmt.Sprintf("%s mentioned you in the group chat \"%s\"", name, groupName)
	}

	seen := map[int]bool{actorID: true}

	for _, match := range matches {
		id := users.GetUserID(app.DB, strings.ToLower(match[1]))

		if id <= 0 || seen[id] {
			continue
		}

		seen[id] = true

		member, err := chats.UserInGroup(app.DB, id, groupID)

		if err != nil || !member {
			continue
		}

		app.notify(actorID, models.NewNotification{
			UserID:            id,
			Message:           message,
			PostMentionUserID: &actor,
			GroupID:           &gid,
		})
	}
}

// notifyEventResponse tells the event creator if someone is going or not
func (app *App) notifyEventResponse(actorID int, event models.GroupEvent) {
	if event.UserResponse == nil {
		return
	}

	verb := "can't go to"

	if *event.UserResponse == 1 {
		verb = "is going to"
	}

	actor := actorID
	eventID := event.ID

	app.notify(actorID, models.NewNotification{
		UserID:              event.Creator.ID,
		Message:             fmt.Sprintf("%s %s your event \"%s\"", app.actorName(actorID), verb, event.Title),
		EventResponseUserID: &actor,
		EventID:             &eventID,
		GroupID:             optionalGroupID(event.GroupID),
	})
}
