package models

import (
	"time"
)

type UserRegistration struct {
	ID        int       `json:"ID"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	UserName  string    `json:"username"`
	Email     string    `json:"email"`
	About     string    `json:"about"`
	DOB       time.Time `json:"DOB"`
	Password  string    `json:"password"`
	Avatar    string    `json:"avatar"`
	IsPrivate int       `json:"isPrivate"`
}

type UserLogger struct {
	Identifier string
	Pass       string
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
