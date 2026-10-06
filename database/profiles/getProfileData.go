package profiles

import (
	"database/sql"
	"errors"

	database "social/database/users"
	"social/internal/models"
)

func GetUserAbout(db *sql.DB, userID int) (models.UserAbout, error) {
	var userProfile models.UserAbout

	var work sql.NullString
	var hobbies sql.NullString
	var education sql.NullString
	var intrests sql.NullString
	var travel sql.NullString
	var website sql.NullString
	var linkedin sql.NullString
	var instgram sql.NullString
	var twitter sql.NullString

	err := db.QueryRow(`
		SELECT work, hobbies, education, intrests, travel, website, linkedin, instgram, twitter
		FROM user_about
		WHERE user_id = ?
	`, userID).Scan(
		&work,
		&hobbies,
		&education,
		&intrests,
		&travel,
		&website,
		&linkedin,
		&instgram,
		&twitter,
	)

	if err != nil {
		return models.UserAbout{}, err
	}

	if work.Valid {
		userProfile.Work = work.String
	}
	if hobbies.Valid {
		userProfile.Hobbies = hobbies.String
	}
	if education.Valid {
		userProfile.Education = education.String
	}
	if intrests.Valid {
		userProfile.Intrests = intrests.String
	}
	if travel.Valid {
		userProfile.Travel = travel.String
	}
	if website.Valid {
		userProfile.Website = website.String
	}
	if linkedin.Valid {
		userProfile.Linkedin = linkedin.String
	}
	if instgram.Valid {
		userProfile.Instgram = instgram.String
	}
	if twitter.Valid {
		userProfile.Twitter = twitter.String
	}

	var bio sql.NullString

	err = db.QueryRow(`
		SELECT about
		FROM profile
		WHERE user_id = ?
	`, userID).Scan(&bio)

	if err != nil {
		return models.UserAbout{}, err
	}

	if bio.Valid {
		userProfile.Bio = bio.String
	}

	return userProfile, nil
}

func UpdateUserAbout(db *sql.DB, userID int, userAbout *models.UserAbout) error {
	result, err := db.Exec(`
		UPDATE user_about
		SET work = ?,
			hobbies = ?,
			education = ?,
			intrests = ?,
			travel = ?,
			website = ?,
			linkedin = ?,
			instgram = ?,
			twitter = ?
		WHERE user_id = ?
	`,
		userAbout.Work,
		userAbout.Hobbies,
		userAbout.Education,
		userAbout.Intrests,
		userAbout.Travel,
		userAbout.Website,
		userAbout.Linkedin,
		userAbout.Instgram,
		userAbout.Twitter,
		userID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func IsPrivate(db *sql.DB, userID int) (bool, error) {
	var isPrivate sql.NullInt64

	err := db.QueryRow(`
		SELECT is_private
		FROM profile
		WHERE user_id = ?
	`, userID).Scan(&isPrivate)

	if err != nil {
		return false, err
	}

	if !isPrivate.Valid {
		return false, nil
	}

	if isPrivate.Int64 == 1 {
		return true, nil
	}

	if isPrivate.Int64 == 0 {
		return false, nil
	}

	return false, errors.New("invalid data")
}

func CheckFollower(db *sql.DB, followerID, targetID int) (int, error) {
	var status sql.NullInt64

	err := db.QueryRow(`
		SELECT status
		FROM user_followers
		WHERE follower_id = ? AND target_id = ?
	`, followerID, targetID).Scan(&status)

	if err != nil {
		return 0, err
	}

	if !status.Valid {
		return 0, nil
	}

	return int(status.Int64), nil
}

func GetUserData(db *sql.DB, userID int) (models.UserData, error) {
	userData, err := database.GetUserData(db, userID)
	if err != nil {
		return models.UserData{}, err
	}

	userAbout, err := GetUserAbout(db, userID)
	if err != nil {
		return models.UserData{}, err
	}

	userData.About = userAbout

	return userData, nil
}

func GetPrivateProfileData(db *sql.DB, userID int) (models.UserData, error) {
	var userData models.UserData

	var firstName sql.NullString
	var lastName sql.NullString

	err := db.QueryRow(`
		SELECT first_name, last_name
		FROM user
		WHERE id = ?
	`, userID).Scan(
		&firstName,
		&lastName,
	)

	if err != nil {
		return models.UserData{}, err
	}

	if firstName.Valid {
		userData.UserInfo.FirstName = firstName.String
	}

	if lastName.Valid {
		userData.UserInfo.LastName = lastName.String
	}

	var numOfFollowing sql.NullInt64
	var numOfFollowers sql.NullInt64
	var numOfPosts sql.NullInt64
	var about sql.NullString
	var avatar sql.NullString

	err = db.QueryRow(`
		SELECT
			num_of_following,
			num_of_followers,
			num_of_posts,
			about,
			avatar_path
		FROM profile
		WHERE user_id = ?
	`, userID).Scan(
		&numOfFollowing,
		&numOfFollowers,
		&numOfPosts,
		&about,
		&avatar,
	)

	if err != nil {
		return models.UserData{}, err
	}

	if numOfFollowing.Valid {
		userData.NumOfFollowing = int(numOfFollowing.Int64)
	}

	if numOfFollowers.Valid {
		userData.NumOfFollowers = int(numOfFollowers.Int64)
	}

	if numOfPosts.Valid {
		userData.NumOfPosts = int(numOfPosts.Int64)
	}

	if about.Valid {
		userData.About.Bio = about.String
	}

	if avatar.Valid {
		userData.UserInfo.Avatar = avatar.String
	}

	return userData, nil
}