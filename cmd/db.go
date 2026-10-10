package server

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func ConnectToDB(driverName, driverSource string) (*sql.DB, error) {
	db, err := sql.Open(driverName, driverSource)
	if err != nil {
		return nil, fmt.Errorf("FAILED TO OPEN THE DATABASE: %s",err.Error())
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("FAILED TO CONNECT TO THE DATABASE: %s", err.Error())
	} else {
		log.Println("CONNECTED TO THE DATABASE SUCCESSFULLY")
	}

	// same as migration 030. if someone runs the server on a db that never got
	// that migration, the login check would fail on every request and kick
	// everyone out, so make sure the table is there
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS revoked_tokens (
			token_hash TEXT PRIMARY KEY,
			expires_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens (expires_at);
	`); err != nil {
		return nil, fmt.Errorf("FAILED TO MAKE THE revoked_tokens TABLE: %s", err.Error())
	}

	return db, nil
}
