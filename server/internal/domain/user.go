package domain

import "time"

type User struct {
	ID             string    `json:"id"`
	OIDCProviderID string    `json:"oidcProviderId"`
	Subject        string    `json:"subject"`
	Name           string    `json:"name,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
