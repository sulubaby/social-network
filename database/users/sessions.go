package users

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

/*
CreateSession saves a new session for the user and gives back its id.
the id is a random uuid, it is the only thing we put in the cookie.

Parameters:
	db *sql.DB, userID int, ttl time.Duration -> how long the session lives

Returns:
	string -> session id
	error -> nil if success
*/
func CreateSession(db *sql.DB, userID int, ttl time.Duration) (string, error) {
	id := uuid.NewString()

	// clean old sessions of everyone while we are here
	if _, err := db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC()); err != nil {
		return "", err
	}

	_, err := db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`, id, userID, time.Now().UTC().Add(ttl))
	if err != nil {
		return "", err
	}

	return id, nil
}

/*
SessionUser finds the user of a session that did not expire yet.

Returns:
	int -> user id
	error -> sql.ErrNoRows if the session is missing or expired
*/
func SessionUser(db *sql.DB, sessionID string) (int, error) {
	var userID int
	err := db.QueryRow(`
		SELECT user_id
		FROM sessions
		WHERE id = ? AND expires_at > ?
	`, sessionID, time.Now().UTC()).Scan(&userID)
	return userID, err
}

// DeleteSession removes one session (used by log out)
func DeleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}
