package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		conns: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*websocket.Conn]struct{})
	}
	h.conns[userID][conn] = struct{}{}
}

func (h *Hub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	userConns, ok := h.conns[userID]
	if !ok {
		return
	}

	delete(userConns, conn)
	if len(userConns) == 0 {
		delete(h.conns, userID)
	}
}

func (h *Hub) NotifyChatUnread(userID string, event ChatUnreadEvent) {
	h.notify(userID, event)
}

func (h *Hub) NotifyChatAdded(userID string, event ChatAddedEvent) {
	h.notify(userID, event)
}

func (h *Hub) notify(userID string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.conns[userID] {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}
