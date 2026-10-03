package models

type ChatUser struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	AvatarPath string `json:"avatarPath"`
}

type ChatConversation struct {
	ID                int64    `json:"id"`
	Type              string   `json:"type"`
	OtherUser         ChatUser `json:"otherUser"`
	LatestMessage     string   `json:"latestMessage"`
	LatestMessageTime string   `json:"latestMessageTime"`
}

type ChatCandidate struct {
	ChatUser
	ChatID *int64 `json:"chatId"`
}

type ChatMessage struct {
	ID         int64  `json:"id"`
	ChatID     int64  `json:"chatId"`
	SenderID   int64  `json:"senderId"`
	Username   string `json:"username"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	AvatarPath string `json:"avatarPath"`
	Content    string `json:"content"`
	CreatedAt  string `json:"createdAt"`
	IsOwn      bool   `json:"isOwn"`
	// only filled on live events so the page knows where a message belongs
	ChatType   string `json:"chatType,omitempty"`
	GroupID    int64  `json:"groupId,omitempty"`
	GroupTitle string `json:"groupTitle,omitempty"`
}
