// Package sqlite opens the database and applies the migrations in
// pkg/db/migrations/sqlite when the server starts.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// MigrationsPath is where the migration files live (relative to where the server runs)
const MigrationsPath = "file://pkg/db/migrations/sqlite"

// ConnectToDB opens the database and checks that it answers
func ConnectToDB(driverName, driverSource string) (*sql.DB, error) {
	db, err := sql.Open(driverName, driverSource)
	if err != nil {
		return nil, fmt.Errorf("FAILED TO OPEN THE DATABASE: %s", err.Error())
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("FAILED TO CONNECT TO THE DATABASE: %s", err.Error())
	}

	log.Println("CONNECTED TO THE DATABASE SUCCESSFULLY")
	return db, nil
}

// RunMigrations brings the database at path up to the newest migration
func RunMigrations(path string) error {
	m, err := migrate.New(MigrationsPath, "sqlite3://"+path)
	if err != nil {
		return err
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	fmt.Println("Migrations applied successfully")
	return nil
}
