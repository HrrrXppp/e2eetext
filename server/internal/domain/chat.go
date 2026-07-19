package domain

import "time"

type Chat struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name,omitempty"`
	DisappearAfterMinutes  int       `json:"disappearAfterMinutes"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
	UnreadMessageCount     int       `json:"unreadMessageCount"`
}
