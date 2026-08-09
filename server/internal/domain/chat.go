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
