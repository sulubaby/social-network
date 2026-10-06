package users

import (
	"database/sql"
	"social/internal/models"
)

func RegisterUser(db *sql.DB, userData *models.UserRegistration) error {
	_, err := db.Exec(`
		INSERT INTO user (email, first_name, last_name, password, dob, username)
		VALUES (?,?,?,?,?,?)
	`, userData.Email, userData.FirstName, userData.LastName, userData.Password, userData.DOB, userData.UserName)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO profile (user_id, avatar_path, about)
		VALUES ((select id from user where email = ?),?,?)
	`, userData.Email, userData.Avatar, userData.About)

	return err
}
