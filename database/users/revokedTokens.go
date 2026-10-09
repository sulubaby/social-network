package users

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
)

// only a hash of the token is saved, never the token itself
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RevokeToken is called on log out, the token stops working right away
func RevokeToken(db *sql.DB, token string, expiresAt int64) error {
	// old entries are useless once their token expired anyway
	if _, err := db.Exec(`DELETE FROM revoked_tokens WHERE expires_at < ?`, time.Now().Unix()); err != nil {
		return err
	}

	_, err := db.Exec(`
		INSERT OR IGNORE INTO revoked_tokens (token_hash, expires_at)
		VALUES (?, ?)
	`, tokenHash(token), expiresAt)
	return err
}

// IsTokenRevoked says if this token was logged out
func IsTokenRevoked(db *sql.DB, token string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM revoked_tokens WHERE token_hash = ?)
	`, tokenHash(token)).Scan(&exists)
	return exists, err
}
