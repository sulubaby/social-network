package models

// RegsiterPost is what the create post form sends
type RegsiterPost struct {
	UserID        int
	GroupID       int    `json:"groupID"`
	Content       string `json:"content"`
	Image_path    string
	AllowComments int    `json:"allowComments"`
	Location      string `json:"location"`
	PeopleTagged  []int  `json:"taggedPeople"`
}

type TaggedPerson struct {
	Id         int    `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	AvatarPath string `json:"avatarPath"`
}

// Post is one post card in the feed or on a profile
type Post struct {
	Public         int
	Private        int
	Id             int     `json:"id"`
	UserId         int     `json:"userId"`
	FirstName      string  `json:"firstName"`
	LastName       string  `json:"lastName"`
	Username       *string `json:"username"`
	AvatarPath     string  `json:"avatarPath"`
	Content        string  `json:"content"`
	ImagePath      *string `json:"imagePath"`
	AllowComments  bool    `json:"allowComments"`
	Location       *string `json:"location"`
	GroupId        *int    `json:"groupId"`
	GroupName      string
	CreatedAt      string         `json:"createdAt"`
	// how i am connected to the author, used for the follow button
	Relationship   string         `json:"relationship"`
	Visibility     string         `json:"visibility"`
	VisibilityUser string         `json:"visibilityUser"`
	TaggedPeople   []TaggedPerson `json:"taggedPeople"`
	LikeCount      int            `json:"likeCount"`
	DisLikeCount   int            `json:"disLikeCount"`
	CommentCount   int            `json:"commentCount"`
	// my own reaction, 1 like, -1 dislike, 0 nothing
	ReactionValue  int
}

// Comment is a comment or a reply, RepltTo is the parent comment id
type Comment struct {
	ID        int              `json:"id"`
	Content   string           `json:"content"`
	ImagePath string           `json:"imagePath"`
	User      UserRegistration `json:"user"`
	PostID    int              `json:"postId"`
	RepltTo   *int             `json:"replyTo"`
	Votes     int              `json:"votes"`
	CreatedAt string           `json:"createdAt"`
	Replies   int              `json:"replies"`
	IsLiked   int
}

type Reaction struct {
	UserID int `json:"userID"`
	PostID int `json:"postID"`
	Value  int `json:"value"`
}
