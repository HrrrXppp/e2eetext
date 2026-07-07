package domain

import "time"

type Message struct {
	ID            string    `json:"id"`
	ChatID        string    `json:"chatId"`
	UserID        string    `json:"userId"`
	UserName      string    `json:"userName,omitempty"`
	Data          string    `json:"data"`
	KemCiphertext []byte    `json:"kemCiphertext"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Unread        bool      `json:"unread,omitempty"`
}
