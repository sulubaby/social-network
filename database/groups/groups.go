package groups

import (
	"database/sql"
	"errors"
)

var (
	ErrGroupNotFound      = errors.New("group not found")
	ErrInvitationNotFound = errors.New("pending group invitation not found")
)

// AcceptInvitation accepts one specific pending invitation. If the group has
// a chat, the same user is added there too. It gives back the group and the
// person who sent the invitation so they can be told.
func AcceptInvitation(db *sql.DB, userID int, invitationID int64) (groupID int64, inviterID int, err error) {
	if userID <= 0 || invitationID <= 0 {
		return 0, 0, ErrInvitationNotFound
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	err = tx.QueryRow(`
		SELECT group_id, inviter_id
		FROM group_invitations
		WHERE id = ?
		  AND user_id = ?
		  AND status = 'pending'
	`, invitationID, userID).Scan(&groupID, &inviterID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrInvitationNotFound
	}

	if err != nil {
		return 0, 0, err
	}

	result, err := tx.Exec(`
		UPDATE group_invitations
		SET status = 'accepted'
		WHERE id = ?
		  AND user_id = ?
		  AND status = 'pending'
	`, invitationID, userID)

	if err != nil {
		return 0, 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, 0, err
	}

	if affected == 0 {
		return 0, 0, ErrInvitationNotFound
	}

	// joining through an invitation also answers my own pending join request
	if _, err = tx.Exec(`
		UPDATE group_join_requests
		SET status = 'accepted'
		WHERE group_id = ? AND user_id = ? AND status = 'pending'
	`, groupID, userID); err != nil {
		return 0, 0, err
	}

	// Add user as group member
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO group_members (group_id, user_id)
		VALUES (?, ?)
	`, groupID, userID)

	if err != nil {
		return 0, 0, err
	}

	// Find group chat
	var chatID int64

	err = tx.QueryRow(`
		SELECT id
		FROM chats
		WHERE group_id = ?
		ORDER BY id
		LIMIT 1
	`, groupID).Scan(&chatID)

	if err == sql.ErrNoRows {
		return groupID, inviterID, tx.Commit()
	}

	if err != nil {
		return 0, 0, err
	}

	// Add member to group chat
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO chat_users (user_id, chat_id)
		VALUES (?, ?)
	`, userID, chatID)

	if err != nil {
		return 0, 0, err
	}

	return groupID, inviterID, tx.Commit()
}

func DeclineInvitation(db *sql.DB, userID int, invitationID int64) error {
	if userID <= 0 || invitationID <= 0 {
		return ErrInvitationNotFound
	}

	result, err := db.Exec(`
		UPDATE group_invitations
		SET status = 'declined'
		WHERE id = ?
		  AND user_id = ?
		  AND status = 'pending'
	`, invitationID, userID)

	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrInvitationNotFound
	}

	return nil
}

func AcceptJoinRequest(db *sql.DB, creatorID int, requesterID int, groupID int64) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// verify this user is the group creator
	var ownerID int
	err = tx.QueryRow(`
        SELECT creator_id
        FROM groups
        WHERE id = ?
    `, groupID).Scan(&ownerID)

	if err != nil {
		return err
	}

	if ownerID != creatorID {
		return errors.New("only the group creator can accept join requests")
	}

	// verify pending request exists
	result, err := tx.Exec(`
        UPDATE group_join_requests
        SET status = 'accepted'
        WHERE group_id = ?
          AND user_id = ?
          AND status = 'pending'
    `, groupID, requesterID)

	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("pending join request not found")
	}

	// add requester as member
	_, err = tx.Exec(`
        INSERT OR IGNORE INTO group_members (group_id, user_id)
        VALUES (?, ?)
    `, groupID, requesterID)

	if err != nil {
		return err
	}

	// any invitation still waiting for this person is answered now too
	if _, err = tx.Exec(`
		UPDATE group_invitations
		SET status = 'accepted'
		WHERE group_id = ? AND user_id = ? AND status = 'pending'
	`, groupID, requesterID); err != nil {
		return err
	}

	return tx.Commit()
}

func RejectJoinRequest(db *sql.DB, creatorID int, requesterID int, groupID int64) error {

	var ownerID int

	err := db.QueryRow(`
        SELECT creator_id
        FROM groups
        WHERE id = ?
    `, groupID).Scan(&ownerID)

	if err != nil {
		return err
	}

	if ownerID != creatorID {
		return errors.New("only the group creator can reject join requests")
	}

	result, err := db.Exec(`
        UPDATE group_join_requests
        SET status = 'rejected'
        WHERE group_id = ?
          AND user_id = ?
          AND status = 'pending'
    `, groupID, requesterID)

	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("pending join request not found")
	}

	return nil
}

// LeaveGroup removes a member from a group. The creator owns the group, so they
// delete it instead of leaving it.
var ErrCreatorCannotLeave = errors.New("the group creator cannot leave, delete the group instead")
var ErrNotMember = errors.New("you are not a member of this group")

func LeaveGroup(db *sql.DB, userID int, groupID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var creatorID int
	err = tx.QueryRow(`SELECT creator_id FROM groups WHERE id = ?`, groupID).Scan(&creatorID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrGroupNotFound
	}
	if err != nil {
		return err
	}
	if creatorID == userID {
		return ErrCreatorCannotLeave
	}

	result, err := tx.Exec(`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotMember
	}

	// leave the group chat and drop my answers to its events
	if _, err = tx.Exec(`
		DELETE FROM chat_users
		WHERE user_id = ? AND chat_id IN (SELECT id FROM chats WHERE type = 'group' AND group_id = ?)
	`, userID, groupID); err != nil {
		return err
	}
	if _, err = tx.Exec(`
		DELETE FROM event_rsvps
		WHERE user_id = ? AND event_id IN (SELECT id FROM events WHERE group_id = ?)
	`, userID, groupID); err != nil {
		return err
	}

	return tx.Commit()
}
