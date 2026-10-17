package ws

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]struct{}

	// total mirrors the number of registered connections so ConnectionCount
	// can read it without taking mu.
	total atomic.Int64
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
	if _, exists := h.conns[userID][conn]; !exists {
		h.conns[userID][conn] = struct{}{}
		h.total.Add(1)
	}
}

func (h *Hub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	userConns, ok := h.conns[userID]
	if !ok {
		return
	}

	if _, exists := userConns[conn]; exists {
		delete(userConns, conn)
		h.total.Add(-1)
	}
	if len(userConns) == 0 {
		delete(h.conns, userID)
	}
}

// ConnectionCount returns the number of WebSocket connections currently
// registered on this node (all users combined).
func (h *Hub) ConnectionCount() int {
	return int(h.total.Load())
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
