package users

import "database/sql"

/*
function to get hashed password from the database by searching identifier (email or username)

Parameters:
	db *sql.DB, identifier string

Returns:
	string -> hashed password
	error -> nil if success
*/
func GetHashedPassowrd(db *sql.DB, identifier string) (string, error) {
	var password string
	err := db.QueryRow(`select password from user where email = LOWER(?) OR username = LOWER(?)`, identifier, identifier).Scan(&password)
	if err != nil {
		return "", err
	}
	return password, nil
}