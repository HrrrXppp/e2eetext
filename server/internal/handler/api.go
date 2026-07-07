package handler

import (
	"time"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/service"
)

type apiUser struct {
	ID             string    `json:"id"`
	OIDCProviderID string    `json:"oidcProviderId"`
	Subject        string    `json:"subject"`
	Name           string    `json:"name,omitempty"`
	KemPublicKey   []byte    `json:"kemPublicKey"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type apiChat struct {
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

type apiMessage struct {
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

func toAPIUser(n node.Registry, user domain.User) apiUser {
	return apiUser{
		ID:             n.ScopeID(user.ID),
		OIDCProviderID: n.ScopeID(user.OIDCProviderID),
		Subject:        user.Subject,
		Name:           user.Name,
		KemPublicKey:   user.KemPublicKey,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func toAPIUsers(n node.Registry, users []domain.User) []apiUser {
	if len(users) == 0 {
		return []apiUser{}
	}

	out := make([]apiUser, len(users))
	for i, user := range users {
		out[i] = toAPIUser(n, user)
	}
	return out
}

func toAPIChat(n node.Registry, chat domain.Chat) apiChat {
	return apiChat{
		ID:                    n.ScopeID(chat.ID),
		Name:                  chat.Name,
		AdminUserID:           n.ScopeID(chat.AdminUserID),
		KemPublicKey:          chat.KemPublicKey,
		WrappedChatPrivateKey: chat.WrappedChatPrivateKey,
		KemCiphertext:         chat.KemCiphertext,
		CreatedAt:             chat.CreatedAt,
		UpdatedAt:             chat.UpdatedAt,
		UnreadMessageCount:    chat.UnreadMessageCount,
	}
}

func toAPIChats(n node.Registry, chats []domain.Chat) []apiChat {
	if len(chats) == 0 {
		return []apiChat{}
	}

	out := make([]apiChat, len(chats))
	for i, chat := range chats {
		out[i] = toAPIChat(n, chat)
	}
	return out
}

func toAPIMessage(n node.Registry, message domain.Message) apiMessage {
	return apiMessage{
		ID:            n.ScopeID(message.ID),
		ChatID:        n.ScopeID(message.ChatID),
		UserID:        n.ScopeID(message.UserID),
		UserName:      message.UserName,
		Data:          message.Data,
		KemCiphertext: message.KemCiphertext,
		CreatedAt:     message.CreatedAt,
		UpdatedAt:     message.UpdatedAt,
		Unread:        message.Unread,
	}
}

func toAPIMessages(n node.Registry, messages []domain.Message) []apiMessage {
	if len(messages) == 0 {
		return []apiMessage{}
	}

	out := make([]apiMessage, len(messages))
	for i, message := range messages {
		out[i] = toAPIMessage(n, message)
	}
	return out
}

func toAPIProviderView(n node.Registry, provider service.ProviderView) service.ProviderView {
	provider.ID = n.ScopeID(provider.ID)
	return provider
}

func toAPIProviderViews(n node.Registry, providers []service.ProviderView) []service.ProviderView {
	if len(providers) == 0 {
		return []service.ProviderView{}
	}

	out := make([]service.ProviderView, len(providers))
	for i, provider := range providers {
		out[i] = toAPIProviderView(n, provider)
	}
	return out
}
