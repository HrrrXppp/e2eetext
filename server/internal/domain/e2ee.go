package domain

import (
	"encoding/json"
	"time"
)

type IdentityPublicKey struct {
	V         int    `json:"v"`
	Alg       string `json:"alg"`
	PublicKey string `json:"publicKey"`
}

type ChatKeyWrap struct {
	V             int    `json:"v"`
	Alg           string `json:"alg"`
	KeyID         string `json:"keyId"`
	KEMCiphertext string `json:"kemCiphertext"`
	Nonce         string `json:"nonce"`
	Ciphertext    string `json:"ciphertext"`
}

type UserIdentityKey struct {
	UserID    string
	DeviceID  string
	PublicKey json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ChatKeyVersion struct {
	ID        string
	ChatID    string
	CreatedBy string
	CreatedAt time.Time
}

type UserChatKeyWrap struct {
	ChatID    string
	KeyID     string
	UserID    string
	Wrap      json.RawMessage
	CreatedAt time.Time
}
