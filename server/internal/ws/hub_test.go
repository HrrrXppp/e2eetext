package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHub_NotifyChatUnread_NoConnections(t *testing.T) {
	hub := NewHub()
	hub.NotifyChatUnread("missing-user", ChatUnreadEvent{
		Type:  EventTypeChatUnread,
		Chats: []ChatUnreadItem{},
	})
}

func TestHub_NotifyChatAdded_SendsEvent(t *testing.T) {
	hub := NewHub()
	registered := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}

		hub.Register("user-1", conn)
		close(registered)
		<-r.Context().Done()
		hub.Unregister("user-1", conn)
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer clientConn.Close()

	select {
	case <-registered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket registration")
	}

	hub.NotifyChatAdded("user-1", ChatAddedEvent{
		Type:   EventTypeChatAdded,
		ChatID: "node/chat-1",
	})

	_, payload, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}

	var event ChatAddedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("decode websocket payload: %v", err)
	}
	if event.Type != EventTypeChatAdded {
		t.Fatalf("type = %q, want %q", event.Type, EventTypeChatAdded)
	}
	if event.ChatID != "node/chat-1" {
		t.Fatalf("chatId = %q, want node/chat-1", event.ChatID)
	}
}
