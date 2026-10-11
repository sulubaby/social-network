package api

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"social/database/profiles"
	"social/database/users"
	"social/internal/helpers"
	"social/internal/validation"
	"strconv"
	"strings"
)

/*
function of type handler used to search for a location (used when the user fills in places like city or
hometown). the search text is taken from the query (?q=) and must be between 2 and 150 characters. the
language is taken from the Accept-Language header (default "en"). it first looks in the in-memory cache,
if there is no cached result it sends the search to the OpenStreetMap Nominatim API (after waiting for the
throttle so the API rate limit is respected), caches the response and returns it as it is (json, up to 8
results with the address details). this handler does not need the user to be logged in.

@METHOD
	-> GET
*/
func (app *App) SearchLocation(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	if query == "" {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "no data provided",
		})
		return
	}

	if len([]rune(query)) > validation.MaxSearch {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "search text is too long",
		})
		return
	}

	if len([]rune(query)) < 2 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "search text is too short",
		})
		return
	}

	if len([]rune(query)) > 150 {
		helpers.WriteJson(w, http.StatusBadRequest, map[string]any{
			"status":  false,
			"message": "search text is too long",
		})
		return
	}

	lang := strings.TrimSpace(r.Header.Get("Accept-Language"))

	if lang == "" || len(lang) > 100 {
		lang = "en"
	}

	cacheKey := strings.ToLower(query) + "|" + lang

	if cached, ok := locationCacheGet(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	endpoint := "https://nominatim.openstreetmap.org/search" +
		"?format=jsonv2" +
		"&addressdetails=1" +
		"&dedupe=1" +
		"&limit=8" +
		"&accept-language=" + url.QueryEscape(lang) +
		"&q=" + url.QueryEscape(query)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "failed to create location request",
		})
		return
	}

	req.Header.Set("User-Agent", "MySocialNetwork/1.0")

	locationThrottle()

	resp, err := locationClient.Do(req)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Println(err)

		helpers.WriteJson(w, http.StatusBadGateway, map[string]any{
			"status":  false,
			"message": "could not read location response",
		})
		return
	}

	locationCacheSet(cacheKey, body)

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

/*
function of type handler used to get the friends of a user. the query takes ?offset= (the number of friends
to skip, used for pagination, required) and optionally ?targetid= (the user to get the friends of). if
targetid is missing (or "null" / "undefined") the friends of the logged in user are returned. the search
text is read with readSearch, if it is empty the friends are returned page by page using the offset, if it
is not empty the friends matching the search are returned instead.

@METHOD
	-> GET
*/
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

	searchValue, searchOK := readSearch(w, r)
	if !searchOK {
		return
	}
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

/*
function of type handler used to search in the followers of a user. the search text is read with readSearch
and optionally ?targetid= is used to choose the user (if it is missing, or "null" / "undefined", the
followers of the logged in user are searched). for a private profile only the owner and the people
following it are allowed to search.

@METHOD
	-> GET
*/
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

	searchValue, searchOK := readSearch(w, r)
	if !searchOK {
		return
	}

	if !app.canSeeConnections(w, userID, targetID) {
		return
	}

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

/*
function of type handler used to search in the users that a user is following. the search text is read with
readSearch and optionally ?targetid= is used to choose the user (if it is missing, or "null" / "undefined",
the following list of the logged in user is searched). for a private profile only the owner and the people
following it are allowed to search.

@METHOD
	-> GET
*/
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

	searchValue, searchOK := readSearch(w, r)
	if !searchOK {
		return
	}

	if !app.canSeeConnections(w, userID, targetID) {
		return
	}

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