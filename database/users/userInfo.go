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

	// emails and usernames are saved in lower case, so the login is not case sensitive
	err := db.QueryRow(`
		SELECT id
		FROM user
		WHERE username = LOWER(?) OR email = LOWER(?)
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
func UpdateUserInfo(db *sql.DB, userID int, userData *models.UserRegistration) ([]int, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE user
		SET first_name = ?, last_name = ?, email = ?, username = NULLIF(?, ''), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userData.FirstName, userData.LastName, userData.Email, userData.UserName, userID)
	if err != nil {
		return nil, err
	}

	// the password only changes when a new one was typed (it is already hashed here)
	if userData.Password != "" {
		if _, err = tx.Exec(`UPDATE user SET password = ? WHERE id = ?`, userData.Password, userID); err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(`
		UPDATE profile
		SET about = ?, is_private = ?
		WHERE user_id = ?
	`, userData.About, userData.IsPrivate, userID)
	if err != nil {
		return nil, err
	}

	// a public profile has no follow requests, everyone who was waiting is accepted
	var accepted []int
	if userData.IsPrivate == 0 {
		rows, err := tx.Query(`SELECT follower_id FROM user_followers WHERE target_id = ? AND status = 0`, userID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var followerID int
			if err := rows.Scan(&followerID); err != nil {
				rows.Close()
				return nil, err
			}
			accepted = append(accepted, followerID)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}

		if _, err = tx.Exec(`UPDATE user_followers SET status = 1 WHERE target_id = ? AND status = 0`, userID); err != nil {
			return nil, err
		}
	}

	return accepted, tx.Commit()
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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// follows have no foreign key, remove them here so the other people's
	// follower and following numbers go down (the triggers do that)
	if _, err := tx.Exec(`DELETE FROM user_followers WHERE follower_id = ? OR target_id = ?`, userID, userID); err != nil {
		return err
	}

	// groups i created cannot live without their owner
	if _, err := tx.Exec(`DELETE FROM groups WHERE creator_id = ?`, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM user WHERE id = ?`, userID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// the account is gone, a missing avatar file should not matter anymore
	if avatar != "avatars/default.png" {
		_ = helpers.DeleteAvatar(avatar)
	}

	return nil
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
