package models

import "time"

// Comment is one comment on a post, the way the frontend receives it.
// Own is true when the logged in user wrote it (so we show the delete option)
type Comment struct {
	ID         int64     `json:"id"`
	PostID     int64     `json:"postId"`
	UserID     int       `json:"userId"`
	Author     string    `json:"author"`
	AvatarPath string    `json:"avatarPath"`
	Content    string    `json:"content"`
	ImagePath  string    `json:"imagePath"`
	Own        bool      `json:"own"`
	CreatedAt  time.Time `json:"createdAt"`
}

// body of a text only comment request (json)
type CreateCommentRequest struct {
	Content string `json:"content"`
}
