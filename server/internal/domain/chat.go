package domain

import "time"

type Chat struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name,omitempty"`
	DisappearAfterMinutes int       `json:"disappearAfterMinutes"`
	CreatedBy             string    `json:"createdBy,omitempty"`
	IsAdmin               bool      `json:"isAdmin"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
	UnreadMessageCount    int       `json:"unreadMessageCount"`
}

// ChatMember is one member of a chat, as returned by the members-list
// endpoint: their admin status and when they joined.
type ChatMember struct {
	UserID   string    `json:"userId"`
	IsAdmin  bool      `json:"isAdmin"`
	JoinedAt time.Time `json:"joinedAt"`
}
