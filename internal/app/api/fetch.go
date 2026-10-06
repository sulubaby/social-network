package api

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"social/database/profiles"
	"social/database/users"
	"social/internal/helpers"
	"strconv"
)

func (app *App) SearchLocation(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if query == "" {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "no data provided",
		})
		return
	}

	url := "https://nominatim.openstreetmap.org/search" +
		"?format=jsonv2" +
		"&addressdetails=1" +
		"&limit=5" +
		"&q=" + url.QueryEscape(query)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to create location request",
		})
		return
	}

	req.Header.Set("User-Agent", "MySocialNetwork/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to search location",
		})
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Nominatim returned status:", resp.StatusCode)

		helpers.WriteJson(w, http.StatusBadGateway, map[string]any{
			"status":  false,
			"message": "location API returned an error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	io.Copy(w, resp.Body)
}

func (app *App) GetFriends(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}
	
	targetIDParam := r.URL.Query().Get("targetid")
	targetID := userID 
	if targetIDParam != "" && targetIDParam != "null" && targetIDParam != "undefined" {
		parsedTargetID, err := strconv.Atoi(targetIDParam)
		if err != nil {
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid targetid",
			})
			return
		}
		targetID = parsedTargetID
	}

	searchValue := r.URL.Query().Get("search")
	queryOffset := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(queryOffset)

	if err != nil || offset < 0 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "invalid offset",
		})
		return
	}

	if searchValue == "" {
		friends, err := users.GetFriends(app.DB, targetID, offset)
		if err != nil {
			log.Println(err)
			helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
				"status":  false,
				"message": "could not get user friends",
			})
			return
		}
		helpers.WriteJson(w, http.StatusOK, map[string]any{
			"status":  true,
			"message": "friends fetched",
			"data":    friends,
		})
		return
	}

	friends, err := users.SearchFriends(app.DB, targetID, searchValue)
	if err != nil {
		log.Println(err)
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not get friends",
			"data":    friends,
		})
		return
	}
	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "friends fetched",
		"data":    friends,
	})
}

func (app *App) SearchFollows(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	targetID := userID
	queryID := r.URL.Query().Get("targetid")

	if queryID != "" && queryID != "null" && queryID != "undefined" {
		parsedID, err := strconv.Atoi(queryID)

		if err != nil || parsedID <= 0 {
			log.Println(err)
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid targetid",
			})
			return
		}

		targetID = parsedID
	}

	searchValue := r.URL.Query().Get("search")

	follows, err := profiles.SearchFollows(app.DB, targetID, searchValue)

	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not find follows",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"data":   follows,
	})
}

func (app *App) SearchFollowing(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)

	if !ok {
		helpers.WriteJson(w, http.StatusUnauthorized, map[string]any{
			"status":  false,
			"message": "could not authorize user",
		})
		return
	}

	targetID := userID
	queryID := r.URL.Query().Get("targetid")

	if queryID != "" && queryID != "null" && queryID != "undefined" {
		parsedID, err := strconv.Atoi(queryID)

		if err != nil || parsedID <= 0 {
			log.Println(err)
			helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
				"status":  false,
				"message": "invalid targetid",
			})
			return
		}

		targetID = parsedID
	}

	searchValue := r.URL.Query().Get("search")

	following, err := profiles.SearchFollowing(app.DB, targetID, searchValue)

	if err != nil {
		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "could not find following",
		})
		return
	}

	helpers.WriteJson(w, http.StatusOK, map[string]any{
		"status": true,
		"data":   following,
	})
}
