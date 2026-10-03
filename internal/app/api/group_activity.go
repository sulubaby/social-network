package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"social/database/events"
	"social/internal/helpers"
	"strconv"
	"strings"
	"time"
)

type eventVoter struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	AvatarPath string `json:"avatarPath"`
}

type groupEvent struct {
	ID            int64        `json:"id"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	StartsAt      string       `json:"startsAt"`
	Response      string       `json:"response"`
	IsCreator     bool         `json:"isCreator"`
	GoingCount    int          `json:"goingCount"`
	NotGoingCount int          `json:"notGoingCount"`
	GoingUsers    []eventVoter `json:"goingUsers"`
	NotGoingUsers []eventVoter `json:"notGoingUsers"`
}

func (app App) EventRSVP(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok || userID <= 0 {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{"status": false, "message": "authentication required"})
		return
	}
	eventID, err := strconv.ParseInt(r.PathValue("eventID"), 10, 64)
	if err != nil || eventID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid event id"})
		return
	}

	var input struct {
		Response string `json:"response"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&input) != nil {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid response"})
		return
	}
	if err := events.SetRSVP(app.DB, userID, eventID, input.Response); err != nil {
		writeEventError(w, err, "could not save response")
		return
	}
	voter, err := getEventVoter(app.DB, userID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "response saved but user details could not be loaded"})
		return
	}
	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true, "response": input.Response, "user": voter})
}

func (app App) RemoveEventRSVP(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	eventID, err := strconv.ParseInt(r.PathValue("eventID"), 10, 64)
	if err != nil || eventID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid event id"})
		return
	}
	if err := events.RemoveRSVP(app.DB, userID, groupID, eventID); err != nil {
		writeEventError(w, err, "could not remove response")
		return
	}
	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true, "userId": userID})
}

func (app App) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	eventID, err := strconv.ParseInt(r.PathValue("eventID"), 10, 64)
	if err != nil || eventID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid event id"})
		return
	}
	if err := events.Delete(app.DB, userID, groupID, eventID); err != nil {
		writeEventError(w, err, "could not delete event")
		return
	}
	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true})
}

func (app App) InviteGroupMember(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	var input struct {
		UserID int `json:"userId"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&input) != nil || input.UserID <= 0 || input.UserID == userID {
		helpers.WriteJson(w, 400, map[string]any{"message": "choose another user"})
		return
	}
	tx, err := app.DB.Begin()
	if err != nil {
		helpers.WriteJson(w, 500, map[string]any{"message": "could not invite user"})
		return
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM user WHERE id = ?)`, input.UserID).Scan(&exists); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not verify invited user"})
		return
	}
	if exists == 0 {
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{"status": false, "message": "user not found"})
		return
	}
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, input.UserID).Scan(&exists); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not verify group membership"})
		return
	}
	if exists == 1 {
		helpers.WriteJson(w, http.StatusConflict, map[string]any{"status": false, "message": "user is already a member"})
		return
	}
	result, err := tx.Exec(`INSERT INTO group_invitations (group_id,user_id,inviter_id,status) VALUES (?,?,?,'pending')`, groupID, input.UserID, userID)
	if err != nil {
		status, message := helpers.NormalizeSQLError(err)
		if strings.Contains(err.Error(), "UNIQUE constraint failed: group_invitations.group_id, group_invitations.user_id") {
			status = http.StatusConflict
			message = "invitation is already pending"
		}
		helpers.WriteJson(w, status, map[string]any{"status": false, "message": message})
		return
	}
	invitationID, err := result.LastInsertId()
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not save invitation"})
		return
	}
	notificationResult, err := tx.Exec(`INSERT INTO notifications (user_id,actor_id,category,type,message,related_id) SELECT ?,?,'groups','invitation',? || ' invited you to ' || title,? FROM groups WHERE id=?`, input.UserID, userID, app.userFullName(userID), invitationID, groupID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not save notification"})
		return
	}
	notificationsCreated, err := notificationResult.RowsAffected()
	if err != nil || notificationsCreated != 1 {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not save notification"})
		return
	}
	notificationID, err := notificationResult.LastInsertId()
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not save notification"})
		return
	}
	if tx.Commit() != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not finish invitation"})
		return
	}
	app.deliverNotificationByID(input.UserID, notificationID)
	helpers.WriteJson(w, http.StatusCreated, map[string]any{"status": true, "invitationId": invitationID})
}

func (app App) GetInviteUsers(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}

	rows, err := app.DB.Query(`
		SELECT
			u.id,
			COALESCE(u.username, ''),
			u.first_name,
			u.last_name,
			COALESCE(p.avatar_path, ''),
			gi.id
		FROM user u
		LEFT JOIN profile p ON p.user_id = u.id
		LEFT JOIN group_invitations gi
			ON gi.group_id = ?
			AND gi.user_id = u.id
			AND gi.status = 'pending'
		WHERE NOT EXISTS (
			SELECT 1 FROM group_members gm
			WHERE gm.group_id = ? AND gm.user_id = u.id
		)
		ORDER BY COALESCE(u.username, ''), u.id
	`, groupID, groupID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load invite users"})
		return
	}
	defer rows.Close()

	type inviteUser struct {
		ID           int    `json:"id"`
		Username     string `json:"username"`
		FirstName    string `json:"firstName"`
		LastName     string `json:"lastName"`
		AvatarPath   string `json:"avatarPath"`
		IsInvited    bool   `json:"isInvited"`
		InvitationID *int64 `json:"invitationId"`
	}

	users := []inviteUser{}
	for rows.Next() {
		var user inviteUser
		var invitationID sql.NullInt64
		if err := rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.AvatarPath, &invitationID); err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not read invite users"})
			return
		}
		if invitationID.Valid {
			id := invitationID.Int64
			user.IsInvited = true
			user.InvitationID = &id
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load invite users"})
		return
	}
	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true, "users": users})
}

func (app App) UndoInvitation(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	invitationID, err := strconv.ParseInt(r.PathValue("invitationID"), 10, 64)
	if err != nil || invitationID <= 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid invitation id"})
		return
	}

	tx, err := app.DB.Begin()
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not undo invitation"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		DELETE FROM group_invitations
		WHERE id = ? AND group_id = ? AND status = 'pending'
	`, invitationID, groupID)
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not undo invitation"})
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not undo invitation"})
		return
	}
	if affected == 0 {
		helpers.WriteJson(w, http.StatusNotFound, map[string]any{"status": false, "message": "pending invitation not found"})
		return
	}
	if err := tx.Commit(); err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not undo invitation"})
		return
	}
	helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true})
}

func (app App) GroupEvents(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok || !app.requireGroupMember(w, groupID, userID) {
		return
	}
	if r.Method == http.MethodGet {
		items, err := listGroupEvents(app.DB, userID, groupID)
		if err != nil {
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load events"})
			return
		}
		helpers.WriteJson(w, http.StatusOK, map[string]any{"status": true, "events": items})
		return
	}
	var input struct {
		Title       string
		Description string
		StartsAt    string
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil {
		helpers.WriteJson(w, 400, map[string]any{"message": "invalid event"})
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	startsAt, err := time.Parse(time.RFC3339, input.StartsAt)
	if err != nil || !startsAt.After(time.Now()) || len([]rune(input.Title)) == 0 || len([]rune(input.Title)) > 50 || len([]rune(input.Description)) == 0 || len([]rune(input.Description)) > 500 {
		helpers.WriteJson(w, 400, map[string]any{"message": "provide a title, description and future event time"})
		return
	}
	tx, err := app.DB.Begin()
	if err != nil {
		helpers.WriteJson(w, 500, map[string]any{"message": "could not create event"})
		return
	}
	defer tx.Rollback()
	startsAtValue := startsAt.UTC().Format(time.RFC3339)
	result, err := tx.Exec(`INSERT INTO events (title,content,group_id,creator_id,starts_at) VALUES (?,?,?,?,?)`, input.Title, input.Description, groupID, userID, startsAtValue)
	if err != nil {
		helpers.WriteJson(w, 500, map[string]any{"message": "could not save event"})
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		helpers.WriteJson(w, 500, map[string]any{"message": "could not save event"})
		return
	}
	_, err = tx.Exec(`INSERT INTO notifications (user_id,actor_id,category,type,message,related_id) SELECT user_id,?,'events','event_created',?,? FROM group_members WHERE group_id=? AND user_id<>?`, userID, "New event in "+app.groupTitle(groupID)+": "+input.Title, id, groupID, userID)
	if err != nil {
		helpers.WriteJson(w, 500, map[string]any{"message": "could not notify members"})
		return
	}
	if tx.Commit() != nil {
		helpers.WriteJson(w, 500, map[string]any{"message": "could not finish event"})
		return
	}
	app.deliverNotificationsByRelatedID("events", "event_created", id)
	helpers.WriteJson(w, http.StatusCreated, map[string]any{
		"status": true,
		"id":     id,
		"event": groupEvent{
			ID:            id,
			Title:         input.Title,
			Description:   input.Description,
			StartsAt:      startsAtValue,
			IsCreator:     true,
			GoingUsers:    []eventVoter{},
			NotGoingUsers: []eventVoter{},
		},
	})
}

func listGroupEvents(db *sql.DB, userID int, groupID int64) ([]groupEvent, error) {
	rows, err := db.Query(`
		SELECT id, title, content, COALESCE(starts_at, ''), creator_id = ?
		FROM events
		WHERE group_id = ?
		ORDER BY starts_at, id
	`, userID, groupID)
	if err != nil {
		return nil, err
	}

	items := []groupEvent{}
	itemIndexes := map[int64]int{}
	for rows.Next() {
		var item groupEvent
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.StartsAt, &item.IsCreator); err != nil {
			rows.Close()
			return nil, err
		}
		item.GoingUsers = []eventVoter{}
		item.NotGoingUsers = []eventVoter{}
		itemIndexes[item.ID] = len(items)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = db.Query(`
		SELECT er.event_id, er.response, u.id, COALESCE(u.username, ''), u.first_name, u.last_name, COALESCE(p.avatar_path, '')
		FROM event_rsvps er
		JOIN events e ON e.id = er.event_id
		JOIN group_members gm ON gm.group_id = e.group_id AND gm.user_id = er.user_id
		JOIN user u ON u.id = er.user_id
		LEFT JOIN profile p ON p.user_id = u.id
		WHERE e.group_id = ?
		ORDER BY er.created_at, er.user_id
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID int64
		var response string
		var voter eventVoter
		if err := rows.Scan(&eventID, &response, &voter.ID, &voter.Username, &voter.FirstName, &voter.LastName, &voter.AvatarPath); err != nil {
			return nil, err
		}
		index, found := itemIndexes[eventID]
		if !found {
			continue
		}
		if voter.ID == userID {
			items[index].Response = response
		}
		if response == "going" {
			items[index].GoingUsers = append(items[index].GoingUsers, voter)
			items[index].GoingCount++
		} else if response == "declined" {
			items[index].NotGoingUsers = append(items[index].NotGoingUsers, voter)
			items[index].NotGoingCount++
		}
	}
	return items, rows.Err()
}

func getEventVoter(db *sql.DB, userID int) (eventVoter, error) {
	var voter eventVoter
	err := db.QueryRow(`
		SELECT u.id, COALESCE(u.username, ''), u.first_name, u.last_name, COALESCE(p.avatar_path, '')
		FROM user u
		LEFT JOIN profile p ON p.user_id = u.id
		WHERE u.id = ?
	`, userID).Scan(&voter.ID, &voter.Username, &voter.FirstName, &voter.LastName, &voter.AvatarPath)
	return voter, err
}

func writeEventError(w http.ResponseWriter, err error, fallback string) {
	status := http.StatusInternalServerError
	message := fallback
	if errors.Is(err, events.ErrEventNotFound) {
		status = http.StatusNotFound
		message = err.Error()
	} else if errors.Is(err, events.ErrNotCreator) {
		status = http.StatusForbidden
		message = err.Error()
	} else if strings.Contains(err.Error(), "event response must") {
		status = http.StatusBadRequest
		message = err.Error()
	}
	helpers.WriteJson(w, status, map[string]any{"status": false, "message": message})
}
