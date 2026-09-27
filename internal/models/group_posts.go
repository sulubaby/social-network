package models

import "time"

// GroupPost is a post inside a group. IsOwner means i wrote it
type GroupPost struct {
	ID           int64     `json:"id"`
	GroupID      int64     `json:"groupId"`
	UserID       int       `json:"userId"`
	Username     string    `json:"username"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	AvatarPath   string    `json:"avatarPath"`
	Content      string    `json:"content"`
	ImagePath    string    `json:"imagePath"`
	CreatedAt    time.Time `json:"createdAt"`
	CommentCount int       `json:"commentCount"`
	IsOwner      bool      `json:"isOwner"`
}

// GroupPostComment is a comment on a group post.
// ImagePath has json:"-" so the real file path never goes to the browser
type GroupPostComment struct {
	ID         int64     `json:"id"`
	PostID     int64     `json:"postId"`
	UserID     int       `json:"userId"`
	Username   string    `json:"username"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	AvatarPath string    `json:"avatarPath"`
	Content    string    `json:"content"`
	ImagePath  string    `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	IsOwner    bool      `json:"isOwner"`
}
