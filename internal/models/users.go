package models

import (
	"time"
)

type UserRegistration struct {
	ID        int
	FirstName string
	LastName  string
	UserName  string
	Email     string
	About     string
	DOB       time.Time
	Password  string
	Avatar    string
	IsPrivate int
}

type UserLogger struct {
	Identifier string
	Pass       string
	// false means the cookie only lives until the browser is closed
	Remember *bool
}

type UserData struct {
	UserInfo       UserRegistration
	NumOfFollowers int
	NumOfFollowing int
	NumOfPosts     int
	About          UserAbout
	Followers      map[int]UserRegistration
	Following      map[int]UserRegistration
	Friends        map[int]UserRegistration
	IsPrivate      int
}

type UserAbout struct {
	Bio       string
	Work      string `json:"Work"`
	Education string `json:"Education"`
	Travel    string `json:"Travel"`
	Intrests  string `json:"interests"`
	Hobbies   string `json:"Hobbies"`
	Website   string `json:"Website"`
	Linkedin  string `json:"Linkedin"`
	Instgram  string `json:"instagram"`
	Twitter   string `json:"Twitter"`
}

type UserProfile struct {
	About    UserAbout
	UserInfo UserRegistration
}