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

	return db, nil
}
