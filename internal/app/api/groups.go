package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"social/database/groups"
	"social/internal/helpers"
	"strconv"
	"strings"
)

func (app App) GetGroups(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	rows, err := app.DB.Query(`
	SELECT
		g.id,
		g.title,
		g.description,
		g.creator_id,
		COUNT(gm.user_id) AS member_count,

		EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = g.id
			  AND user_id = ?
		) AS is_member,

		EXISTS (
			SELECT 1
			FROM group_join_requests
			WHERE group_id = g.id
			  AND user_id = ?
			  AND status = 'pending'
		) AS is_requested

	FROM groups g

	LEFT JOIN group_members gm
		ON gm.group_id = g.id

	GROUP BY
		g.id,
		g.title,
		g.description,
		g.creator_id

	ORDER BY g.id DESC
	`, userID, userID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to get groups",
		})
		return
	}
	defer rows.Close()

	type Group struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		CreatorID   int    `json:"creatorId"`
		MemberCount int    `json:"memberCount"`
		IsMember    bool   `json:"isMember"`
		IsRequested bool   `json:"isRequested"`
	}

	groups := []Group{}

	for rows.Next() {
		var group Group

		err := rows.Scan(
			&group.ID,
			&group.Title,
			&group.Description,
			&group.CreatorID,
			&group.MemberCount,
			&group.IsMember,
			&group.IsRequested,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "failed to read groups",
			})
			return
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to read groups",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": true,
		"groups": groups,
	})
}

func (app App) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid request body",
		})
		return
	}

	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "group title is required",
		})
		return
	}

	if input.Description == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "group description is required",
		})
		return
	}

	if len([]rune(input.Title)) > 45 || len([]rune(input.Description)) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "the title can have 45 characters and the description 500",
		})
		return
	}

	tx, err := app.DB.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to create group",
		})
		return
	}

	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO groups (creator_id, title, description)
		VALUES (?, ?, ?)
	`, userID, input.Title, input.Description)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to create group",
		})
		return
	}

	groupID, err := result.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to get created group",
		})
		return
	}

	// The creator should automatically be a member of the group.
	_, err = tx.Exec(`
		INSERT INTO group_members (group_id, user_id)
		VALUES (?, ?)
	`, groupID, userID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to add creator to group",
		})
		return
	}
	err = tx.Commit()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to create group",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  true,
		"message": "group created successfully",
		"group": map[string]any{
			"id":          groupID,
			"title":       input.Title,
			"description": input.Description,
			"creatorId":   userID,
			"memberCount": 1,
			"isMember":    true,
			"isRequested": false,
		},
	})
}

func (app App) GetGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid group id",
		})
		return
	}

	type Group struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		CreatorID   int    `json:"creatorId"`
		MemberCount int    `json:"memberCount"`
		IsMember    bool   `json:"isMember"`
		IsRequested bool   `json:"isRequested"`
		IsCreator   bool   `json:"isCreator"`
		// set when someone invited me and i did not answer yet
		InvitationID *int64 `json:"invitationId"`
	}

	var group Group
	var invitationID sql.NullInt64

	err = app.DB.QueryRow(`
		SELECT
			g.id,
			g.title,
			g.description,
			g.creator_id,

			COUNT(gm.user_id) AS member_count,

			EXISTS (
				SELECT 1
				FROM group_members
				WHERE group_id = g.id
				  AND user_id = ?
			) AS is_member,

			EXISTS (
				SELECT 1
				FROM group_join_requests
				WHERE group_id = g.id
				  AND user_id = ?
				  AND status = 'pending'
			) AS is_requested,

			g.creator_id = ? AS is_creator,

			(
				SELECT id
				FROM group_invitations
				WHERE group_id = g.id
				  AND user_id = ?
				  AND status = 'pending'
			) AS invitation_id

		FROM groups g

		LEFT JOIN group_members gm
			ON gm.group_id = g.id

		WHERE g.id = ?

		GROUP BY
			g.id,
			g.title,
			g.description,
			g.creator_id
	`, userID, userID, userID, userID, groupID).Scan(
		&group.ID,
		&group.Title,
		&group.Description,
		&group.CreatorID,
		&group.MemberCount,
		&group.IsMember,
		&group.IsRequested,
		&group.IsCreator,
		&invitationID,
	)
	if invitationID.Valid {
		group.InvitationID = &invitationID.Int64
	}

	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status":  false,
			"message": "group not found",
		})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to get group",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": true,
		"group":  group,
	})
}

func (app App) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid group id",
		})
		return
	}

	result, err := app.DB.Exec(`
		DELETE FROM groups
		WHERE id = ?
		  AND creator_id = ?
	`, groupID, userID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to delete group",
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to delete group",
		})
		return
	}

	if rowsAffected == 0 {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"status":  false,
			"message": "group not found or you are not the creator",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "group deleted successfully",
	})
}

func (app App) JoinRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid group id"})
		return
	}

	var isMember bool
	err = app.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = ?
			  AND user_id = ?
		)
	`, groupID, userID).Scan(&isMember)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to check membership",
		})
		return
	}

	if isMember {
		writeJSON(w, http.StatusConflict, map[string]any{
			"status":  false,
			"message": "you are already a member",
		})
		return
	}

	tx, err := app.DB.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to send join request",
		})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO group_join_requests (group_id, user_id, status)
		VALUES (?, ?, 'pending')
	`, groupID, userID)

	if err != nil {
		status, message := helpers.NormalizeSQLError(err)
		if strings.Contains(err.Error(), "UNIQUE constraint failed: group_join_requests.group_id, group_join_requests.user_id") {
			status = http.StatusConflict
			message = "join request is already pending"
		}
		writeJSON(w, status, map[string]any{
			"status":  false,
			"message": message,
		})
		return
	}

	requestID, err := result.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to send join request",
		})
		return
	}

	var creatorID int

	err = tx.QueryRow(`
    SELECT creator_id
    FROM groups
    WHERE id = ?
	`, groupID).Scan(&creatorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to notify group creator",
		})
		return
	}

	notificationResult, err := tx.Exec(`
	INSERT INTO notifications (
		user_id,
		actor_id,
		category,
		type,
		message,
		related_id,
		is_read
		)
		VALUES (?, ?, ?, ?, ?, ?, 0)
		`,
		creatorID,
		userID,
		"groups",
		"join_request",
		app.userFullName(userID)+" wants to join "+app.groupTitle(groupID),
		requestID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to notify group creator",
		})
		return
	}
	notificationID, err := notificationResult.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to notify group creator",
		})
		return
	}

	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to send join request",
		})
		return
	}
	app.deliverNotificationByID(creatorID, notificationID)

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  true,
		"message": "join request sent",
	})
}

func (app App) UndoJoinRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "authentication required",
		})
		return
	}

	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid group id"})
		return
	}

	tx, err := app.DB.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to undo join request",
		})
		return
	}
	defer tx.Rollback()

	var requestID int64
	err = tx.QueryRow(`
		SELECT id
		FROM group_join_requests
		WHERE group_id = ?
		  AND user_id = ?
		  AND status = 'pending'
	`, groupID, userID).Scan(&requestID)
	if err == sql.ErrNoRows {
		if err = tx.Commit(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "failed to undo join request",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": true,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to undo join request",
		})
		return
	}

	_, err = tx.Exec(`
		DELETE FROM notifications
		WHERE category = 'groups'
		  AND type = 'join_request'
		  AND related_id = ?
	`, requestID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to undo join request",
		})
		return
	}

	_, err = tx.Exec(`
		DELETE FROM group_join_requests
		WHERE id = ?
		  AND group_id = ?
		  AND user_id = ?
		  AND status = 'pending'
	`, requestID, groupID, userID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to undo join request",
		})
		return
	}

	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to undo join request",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": true,
	})
}

// LeaveGroup handles DELETE /api/groups/{id}/members/me
func (app App) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok {
		return
	}

	err := groups.LeaveGroup(app.DB, userID, groupID)
	switch {
	case errors.Is(err, groups.ErrGroupNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "group not found"})
	case errors.Is(err, groups.ErrCreatorCannotLeave), errors.Is(err, groups.ErrNotMember):
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": err.Error()})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not leave group"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "you left the group"})
	}
}

// AnswerGroupInvitation handles PATCH /api/groups/{id}/invitation with {"action": "join" | "decline"}
// so an invited person can answer from the group page, not only from the notification
func (app *App) AnswerGroupInvitation(w http.ResponseWriter, r *http.Request) {
	userID, groupID, ok := groupRequestIdentity(w, r)
	if !ok {
		return
	}

	var input struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "invalid action"})
		return
	}

	var invitationID int64
	err := app.DB.QueryRow(`
		SELECT id FROM group_invitations
		WHERE group_id = ? AND user_id = ? AND status = 'pending'
	`, groupID, userID).Scan(&invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]any{"status": false, "message": "no pending invitation"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not load invitation"})
		return
	}

	switch input.Action {
	case "join":
		_, inviterID, err := groups.AcceptInvitation(app.DB, userID, invitationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not join group"})
			return
		}
		app.notify(inviterID, userID, "groups", "invitation_accepted", app.userFullName(userID)+" accepted your invitation to "+app.groupTitle(groupID), int64Ptr(groupID))
	case "decline":
		if err := groups.DeclineInvitation(app.DB, userID, invitationID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "message": "could not decline invitation"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "message": "action must be join or decline"})
		return
	}

	// the invitation notification now shows the answer
	if _, err := app.DB.Exec(`
		UPDATE notifications SET is_read = 1
		WHERE user_id = ? AND category = 'groups' AND type = 'invitation' AND related_id = ?
	`, userID, invitationID); err != nil {
		log.Printf("mark invitation notification read: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": true, "action": input.Action})
}
