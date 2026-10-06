package helpers

import (
	"database/sql"
	database "social/database/users"
	"social/internal/models"
	"strconv"
	"strings"
)

func GetPostGroupUsers(db *sql.DB, usersIDs []int) ([]models.UserRegistration, error) {
	var users []models.UserRegistration
	for _, id := range usersIDs {
		user, err := database.GetUserSimpleData(db, id)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func JoinUserIDs(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ":")
}
