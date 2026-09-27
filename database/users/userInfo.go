package users

import (
	"database/sql"
	"errors"
	"social/internal/helpers"
	"social/internal/models"
)

/*
GetUserID retrieves a user's ID using their username or email as the identifier.

Parameters:

	db *sql.DB, identifier string

Returns:

	int
		-> User ID if successful
		-> -1 if the user cannot be found or a database error occurs
*/
func GetUserID(db *sql.DB, identifier string) int {
	var id int

	err := db.QueryRow(`
		SELECT id
		FROM user
		WHERE username = ? OR email = ?
	`, identifier, identifier).Scan(&id)

	if err != nil {
		return -1
	}

	return id
}

/*
GetUserData retrieves detailed information about a user, including their
personal information and profile statistics.

Parameters:

	db *sql.DB, userID int

Returns:

	models.UserData
		-> Struct containing the user's personal and profile information

	error
		-> nil if successful
		-> Error if the user or profile cannot be retrieved
*/
func GetUserData(db *sql.DB, userID int) (models.UserData, error) {
	var userData models.UserData
	var firstName sql.NullString
	var lastName sql.NullString
	var email sql.NullString
	var username sql.NullString
	var dob sql.NullTime

	err := db.QueryRow(`
		SELECT first_name, last_name, email, username, dob
		FROM user
		WHERE id = ?
	`, userID).Scan(
		&firstName,
		&lastName,
		&email,
		&username,
		&dob,
	)

	if err != nil {
		return userData, err
	}

	// the frontend needs my id (for example to load my own posts on my profile)
	userData.UserInfo.ID = userID

	if firstName.Valid {
		userData.UserInfo.FirstName = firstName.String
	} else {
		userData.UserInfo.FirstName = ""
	}

	if lastName.Valid {
		userData.UserInfo.LastName = lastName.String
	} else {
		userData.UserInfo.LastName = ""
	}

	if email.Valid {
		userData.UserInfo.Email = email.String
	} else {
		userData.UserInfo.Email = ""
	}

	if username.Valid {
		userData.UserInfo.UserName = username.String
	} else {
		userData.UserInfo.UserName = ""
	}

	if dob.Valid {
		userData.UserInfo.DOB = dob.Time
	}

	var followers sql.NullInt64
	var following sql.NullInt64
	var posts sql.NullInt64
	var avatar sql.NullString
	var about sql.NullString
	var isPrivate sql.NullInt64

	if err := db.QueryRow(`
		SELECT num_of_followers, num_of_following, num_of_posts, avatar_path, about, is_private
		FROM profile
		WHERE user_id = ?
	`, userID).Scan(
		&followers,
		&following,
		&posts,
		&avatar,
		&about,
		&isPrivate,
	); err != nil {
		return userData, err
	}

	if followers.Valid {
		userData.NumOfFollowers = int(followers.Int64)
	} else {
		userData.NumOfFollowers = 0
	}

	if following.Valid {
		userData.NumOfFollowing = int(following.Int64)
	} else {
		userData.NumOfFollowing = 0
	}

	if posts.Valid {
		userData.NumOfPosts = int(posts.Int64)
	} else {
		userData.NumOfPosts = 0
	}

	if avatar.Valid {
		userData.UserInfo.Avatar = avatar.String
	} else {
		userData.UserInfo.Avatar = ""
	}

	if about.Valid {
		userData.About.Bio = about.String
	} else {
		userData.About.Bio = ""
	}

	if isPrivate.Valid {
		userData.IsPrivate = int(isPrivate.Int64)
	} else {
		userData.IsPrivate = 0
	}

	return userData, nil
}

/*
UpdateUserInfo updates a user's personal information and profile settings.

Parameters:

	db *sql.DB, userID int, userData *models.UserRegistration

Returns:

	error
	-> nil if successful
	-> Error if the user or profile cannot be updated
*/
func UpdateUserInfo(db *sql.DB, userID int, userData *models.UserRegistration) error {
	_, err := db.Exec(`
		UPDATE user
		SET first_name = ?, last_name = ?, email = ?, username = ?
		WHERE id = ?
	`, userData.FirstName, userData.LastName, userData.Email, userData.UserName, userID)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
		UPDATE profile
		SET about = ?, is_private = ?
		WHERE user_id = ?
	`, userData.About, userData.IsPrivate, userID)

	return err
}

/*
UpdateUserAvatar updates the avatar path for a user.

Parameters:

	db *sql.DB, userID int, avatar_path string

Returns:

	error
	-> nil if successful
	-> Error if the avatar path cannot be updated
*/
func UpdateUserAvatar(db *sql.DB, userID int, avatar_path string) error {
	_, err := db.Exec(`
		UPDATE profile
		SET avatar_path = ?
		WHERE user_id = ?
	`, avatar_path, userID)

	return err
}

/*
UserExists checks whether a user with the specified ID exists.

Parameters:

	db *sql.DB, userID int

Returns:

	error
	-> nil if the user exists
	-> Error if the user does not exist or the database query fails
*/
func UserExists(db *sql.DB, userID int) error {
	var id int

	return db.QueryRow(
		`SELECT id FROM user WHERE id = ?`,
		userID,
	).Scan(&id)
}

/*
GetUserSimpleData retrieves basic user information, including their
ID, first name, last name, and avatar.

Parameters:

	db *sql.DB, userID int

Returns:

	models.UserRegistration
	-> Struct containing the user's basic information

	error
	-> nil if successful
	-> Error if the user or profile cannot be retrieved
*/
func GetUserSimpleData(db *sql.DB, userID int) (models.UserRegistration, error) {
	var user models.UserRegistration
	var firstName sql.NullString
	var lastName sql.NullString

	err := db.QueryRow(`
		SELECT id, first_name, last_name
		FROM user
		WHERE id = ?
	`, userID).Scan(
		&user.ID,
		&firstName,
		&lastName,
	)

	if err != nil {
		return models.UserRegistration{}, err
	}

	if firstName.Valid {
		user.FirstName = firstName.String
	} else {
		user.FirstName = ""
	}

	if lastName.Valid {
		user.LastName = lastName.String
	} else {
		user.LastName = ""
	}

	var avatar sql.NullString

	err = db.QueryRow(`
		SELECT avatar_path
		FROM profile
		WHERE user_id = ?
	`, userID).Scan(&avatar)

	if err != nil {
		return user, err
	}

	if avatar.Valid {
		user.Avatar = avatar.String
	} else {
		user.Avatar = ""
	}

	return user, nil
}

/*
DeleteUser permanently deletes a user from the database. also delete the avatar if its not the default.

Parameters:

	db *sql.DB, userID int

Returns:

	error
	-> nil if successful
	-> Error if the user cannot be deleted
*/
func DeleteUser(db *sql.DB, userID int) error {
	var avatar string
	if err := db.QueryRow(`select avatar_path from profile where user_id = ?`, userID).Scan(&avatar); err != nil {
		return err
	}

	if avatar != "avatars/default.png" {
		if err := helpers.DeleteAvatar(avatar); err != nil {
			return err
		}
	}

	_, err := db.Exec(`DELETE FROM user WHERE id = ?`, userID)

	return err
}

/*
DeleteUserAvatar replaces the user's current avatar with the default avatar.

Parameters:

	db *sql.DB, userID int

Returns:

	string
	-> Path of the previous avatar

	error
	-> nil if successful
	-> Error if the avatar cannot be retrieved or updated
	-> Error if the user is already using the default avatar
*/
func DeleteUserAvatar(db *sql.DB, userID int) (string, error) {
	var path string

	if err := db.QueryRow(`
		SELECT avatar_path
		FROM profile
		WHERE user_id = ?
	`, userID).Scan(&path); err != nil {
		return "", err
	}

	if path == "avatars/default.png" {
		return "", errors.New("user does not have an avatar")
	}

	_, err := db.Exec(`
		UPDATE profile
		SET avatar_path = 'avatars/default.png'
		WHERE user_id = ?
	`, userID)

	return path, err
}
