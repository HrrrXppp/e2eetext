package ws

import "time"

const (
	EventTypeChatUnread   = "chat.unread"
	EventTypeChatAdded    = "chat.added"
	EventTypeChatRemoved  = "chat.removed"
	EventTypeHeartbeat    = "heartbeat"
	EventTypeHeartbeatAck = "heartbeat.ack"
)

type ChatUnreadItem struct {
	ChatID             string    `json:"chatId"`
	UnreadMessageCount int       `json:"unreadMessageCount"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type ChatUnreadEvent struct {
	Type  string           `json:"type"`
	Chats []ChatUnreadItem `json:"chats"`
}

type ChatAddedEvent struct {
	Type   string `json:"type"`
	ChatID string `json:"chatId"`
}

type ChatRemovedEvent struct {
	Type   string `json:"type"`
	ChatID string `json:"chatId"`
}
