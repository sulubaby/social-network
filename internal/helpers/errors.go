package helpers

import (
	"encoding/json"
	"net/http"
	"strings"
)

func WriteJson(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func NormalizeSQLError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}

	message := err.Error()

	switch {
	case strings.Contains(message, "UNIQUE constraint failed: user.email"):
		return http.StatusConflict, "This email is already registered."

	case strings.Contains(message, "UNIQUE constraint failed: user.username"):
		return http.StatusConflict, "This username is already taken."

	case strings.Contains(message, "UNIQUE constraint failed"):
		return http.StatusConflict, "This information already exists."

	case strings.Contains(message, "FOREIGN KEY constraint failed"):
		return http.StatusBadRequest, "The requested data could not be found."

	case strings.Contains(message, "NOT NULL constraint failed"):
		return http.StatusBadRequest, "Required information is missing."

	default:
		return http.StatusInternalServerError, "Something went wrong. Please try again."
	}
}
