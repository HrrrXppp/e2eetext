package domain

import "time"

type Chat struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name,omitempty"`
	AdminUserID           string    `json:"adminUserId"`
	KemPublicKey          []byte    `json:"kemPublicKey"`
	WrappedChatPrivateKey []byte    `json:"wrappedChatPrivateKey"`
	KemCiphertext         []byte    `json:"kemCiphertext"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
	UnreadMessageCount    int       `json:"unreadMessageCount"`
}
