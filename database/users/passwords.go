package users

import "database/sql"

func GetHashedPassowrd(db *sql.DB, identifier string) (string, error) {
	var password string
	err := db.QueryRow(`select password from user where email = ? OR username = ?`, identifier, identifier).Scan(&password)
	if err != nil {
		return "", err
	}
	return password, nil
}