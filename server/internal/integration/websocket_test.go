//go:build integration

package integration

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialWS connects to the real /api/v1/ws endpoint as actor, then completes a
// heartbeat round-trip before returning. The server registers the connection
// in the hub as the first statement of its per-connection goroutine — before
// it can read and ack our heartbeat — so seeing the ack back guarantees
// registration has already happened and hub notifications will reach us.
func dialWS(t *testing.T, h *harness, actor *user) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(h.server.URL, "http") + "/api/v1/ws?token=" + url.QueryEscape(actor.IDToken)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		status := "<no response>"
		if resp != nil {
			status = resp.Status
		}
		t.Fatalf("dial websocket for %s: %v (%s)", actor.Subject, err, status)
	}
	t.Cleanup(func() { conn.Close() })

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"heartbeat"}`)); err != nil {
		t.Fatalf("send heartbeat: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read heartbeat ack: %v", err)
	}
	var ack struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &ack); err != nil || ack.Type != "heartbeat.ack" {
		t.Fatalf("expected heartbeat.ack, got %q (err %v)", payload, err)
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("clear read deadline: %v", err)
	}
	return conn
}

func readWSEvent(t *testing.T, conn *websocket.Conn, into any) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket event: %v", err)
	}
	if err := json.Unmarshal(payload, into); err != nil {
		t.Fatalf("decode websocket event %q: %v", payload, err)
	}
}

func TestWebSocket_ChatAddedEvent(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")

	bobConn := dialWS(t, h, bob)

	chat, err := createChat(h, alice, "Alice & Bob", []*user{alice, bob}, randomUUID(t))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	var event struct {
		Type   string `json:"type"`
		ChatID string `json:"chatId"`
	}
	readWSEvent(t, bobConn, &event)

	if event.Type != "chat.added" {
		t.Fatalf("event type = %q, want chat.added", event.Type)
	}
	if event.ChatID != chat.ID {
		t.Fatalf("event chatId = %q, want %q", event.ChatID, chat.ID)
	}
}

func TestWebSocket_ChatUnreadEvent(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")

	chat, err := createChat(h, alice, "Alice & Bob", []*user{alice, bob}, randomUUID(t))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	bobConn := dialWS(t, h, bob)

	var sent apiMessage
	resp := h.do("POST", "/api/v1/message", map[string]any{
		"chat_id": chat.ID,
		"user_id": alice.ID,
		"data":    "hello from alice",
	}, alice, &sent)
	if resp.StatusCode != 201 {
		t.Fatalf("POST /api/v1/message: status %d", resp.StatusCode)
	}

	var event struct {
		Type  string `json:"type"`
		Chats []struct {
			ChatID             string `json:"chatId"`
			UnreadMessageCount int    `json:"unreadMessageCount"`
		} `json:"chats"`
	}
	readWSEvent(t, bobConn, &event)

	if event.Type != "chat.unread" {
		t.Fatalf("event type = %q, want chat.unread", event.Type)
	}

	found := false
	for _, item := range event.Chats {
		if item.ChatID == chat.ID {
			found = true
			if item.UnreadMessageCount < 1 {
				t.Fatalf("chat %q unreadMessageCount = %d, want >= 1", item.ChatID, item.UnreadMessageCount)
			}
		}
	}
	if !found {
		t.Fatalf("chat.unread event %+v did not include chat %q", event.Chats, chat.ID)
	}
}
