//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

type apiChatMember struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"isAdmin"`
}

func listMembers(t *testing.T, h *harness, chatID string, actor *user) ([]apiChatMember, int) {
	t.Helper()

	var raw json.RawMessage
	resp := h.do(http.MethodGet, "/api/v1/chat/"+chatID+"/members", nil, actor, &raw)
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode
	}

	var members []apiChatMember
	if err := json.Unmarshal(raw, &members); err != nil {
		t.Fatalf("decode members %s: %v", raw, err)
	}
	return members, resp.StatusCode
}

func TestGoldenPath_RemoveMember(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")
	carol := registerUser(t, h, "carol")
	dave := registerUser(t, h, "dave") // never a member

	chat, err := createChat(h, alice, "Trio", []*user{alice, bob, carol}, randomUUID(t))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	members, status := listMembers(t, h, chat.ID, bob)
	if status != http.StatusOK || len(members) != 3 {
		t.Fatalf("GET members as member bob: status %d, members %+v", status, members)
	}
	admins := 0
	for _, m := range members {
		if m.IsAdmin {
			admins++
			if m.ID != alice.ID {
				t.Fatalf("admin = %q, want alice %q", m.ID, alice.ID)
			}
		}
	}
	if admins != 1 {
		t.Fatalf("admins = %d, want 1", admins)
	}

	if _, status := listMembers(t, h, chat.ID, dave); status != http.StatusForbidden {
		t.Fatalf("GET members as non-member: status %d, want 403", status)
	}

	bobConn := dialWS(t, h, bob)

	removePath := func(target *user) string {
		return "/api/v1/chat/" + chat.ID + "/members/" + target.ID
	}

	if resp := h.do(http.MethodDelete, removePath(carol), nil, bob, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("DELETE member as non-admin bob: status %d, want 403", resp.StatusCode)
	}
	if resp := h.do(http.MethodDelete, removePath(alice), nil, alice, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("DELETE self as admin: status %d, want 400", resp.StatusCode)
	}
	if resp := h.do(http.MethodDelete, removePath(dave), nil, alice, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE non-member dave: status %d, want 404", resp.StatusCode)
	}

	if resp := h.do(http.MethodDelete, removePath(bob), nil, alice, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE bob as admin: status %d, want 204", resp.StatusCode)
	}

	var event struct {
		Type   string `json:"type"`
		ChatID string `json:"chatId"`
	}
	readWSEvent(t, bobConn, &event)
	if event.Type != "chat.removed" || event.ChatID != chat.ID {
		t.Fatalf("ws event = %+v, want chat.removed for %q", event, chat.ID)
	}

	// Bob lost access: membership, key wraps and the chat list entry are gone.
	if _, status := listMembers(t, h, chat.ID, bob); status != http.StatusForbidden {
		t.Fatalf("GET members as removed bob: status %d, want 403", status)
	}
	if resp := h.do(http.MethodGet, "/api/v1/chat/"+chat.ID+"/key-wraps", nil, bob, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("GET key-wraps as removed bob: status %d, want 403", resp.StatusCode)
	}
	var wraps int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM user_chat_key_wraps WHERE chat_id = $1 AND user_id = $2`, localIDOf(chat.ID), localIDOf(bob.ID)).Scan(&wraps); err != nil || wraps != 0 {
		t.Fatalf("bob key wraps rows = %d (err %v), want 0", wraps, err)
	}

	members, status = listMembers(t, h, chat.ID, alice)
	if status != http.StatusOK || len(members) != 2 {
		t.Fatalf("GET members after removal: status %d, members %+v", status, members)
	}
}
