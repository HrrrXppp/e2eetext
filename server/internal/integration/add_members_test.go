//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
)

func TestGoldenPath_AddMembers(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")
	carol := registerUser(t, h, "carol")
	dave := registerUser(t, h, "dave")
	eve := registerUser(t, h, "eve") // outsider, never added

	firstKey := randomUUID(t)
	chat, err := createChat(h, alice, "Growing", []*user{alice, bob}, firstKey)
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	path := "/api/v1/chat/" + chat.ID + "/members"
	body := func(keyID string, newUsers []*user, wrapFor ...*user) map[string]any {
		ids := make([]string, len(newUsers))
		for i, u := range newUsers {
			ids[i] = u.ID
		}
		wraps := make([]map[string]any, len(wrapFor))
		for i, u := range wrapFor {
			wraps[i] = map[string]any{"user_id": u.ID, "wrap": randomWrap(t, keyID)}
		}
		return map[string]any{"user_ids": ids, "key_id": keyID, "wraps": wraps}
	}
	keyIDsFor := func(actor *user) []string {
		t.Helper()
		var out struct {
			Wraps []struct {
				KeyID string `json:"keyId"`
			} `json:"wraps"`
		}
		if resp := h.do(http.MethodGet, "/api/v1/chat/"+chat.ID+"/key-wraps", nil, actor, &out); resp.StatusCode != http.StatusOK {
			t.Fatalf("GET key-wraps: status %d", resp.StatusCode)
		}
		ids := make([]string, len(out.Wraps))
		for i, w := range out.Wraps {
			ids[i] = w.KeyID
		}
		return ids
	}

	secondKey := randomUUID(t)
	all := []*user{alice, bob, carol, dave}

	// Rejections leave the chat untouched.
	if resp := h.do(http.MethodPost, path, body(secondKey, []*user{carol, dave}, all...), bob, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST members as non-admin: status %d, want 403", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, []*user{carol, dave}, all...), eve, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST members as non-member: status %d, want 403", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, nil, alice, bob), alice, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST members with no users: status %d, want 400", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, []*user{carol, dave}, alice, bob, carol), alice, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST members missing a wrap: status %d, want 400", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, []*user{carol, bob}, alice, bob, carol), alice, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("POST members with an existing member: status %d, want 409", resp.StatusCode)
	}
	if members, _ := listMembers(t, h, chat.ID, alice); len(members) != 2 {
		t.Fatalf("members after rejected batches = %d, want 2", len(members))
	}
	if got := keyIDsFor(alice); len(got) != 1 {
		t.Fatalf("alice key versions after rejected batches = %v, want only the first", got)
	}

	carolConn := dialWS(t, h, carol)
	daveConn := dialWS(t, h, dave)

	if resp := h.do(http.MethodPost, path, body(secondKey, []*user{carol, dave}, all...), alice, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST members as admin: status %d, want 204", resp.StatusCode)
	}

	for _, conn := range []*websocket.Conn{carolConn, daveConn} {
		var event struct {
			Type   string `json:"type"`
			ChatID string `json:"chatId"`
		}
		readWSEvent(t, conn, &event)
		if event.Type != "chat.added" || event.ChatID != chat.ID {
			t.Fatalf("ws event = %+v, want chat.added for %q", event, chat.ID)
		}
	}

	if members, _ := listMembers(t, h, chat.ID, alice); len(members) != 4 {
		t.Fatalf("members after add = %d, want 4", len(members))
	}

	// Original members keep every version; new members only get the new one.
	for _, u := range []*user{alice, bob} {
		if got := keyIDsFor(u); len(got) != 2 || got[0] != firstKey || got[1] != secondKey {
			t.Fatalf("original member key versions = %v, want [%s %s]", got, firstKey, secondKey)
		}
	}
	for _, u := range []*user{carol, dave} {
		if got := keyIDsFor(u); len(got) != 1 || got[0] != secondKey {
			t.Fatalf("new member key versions = %v, want [%s]", got, secondKey)
		}
	}

	// Reusing the key id is a conflict and adds nobody.
	if resp := h.do(http.MethodPost, path, body(secondKey, []*user{eve}, append(all, eve)...), alice, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("POST members with a reused key id: status %d, want 409", resp.StatusCode)
	}
	if members, _ := listMembers(t, h, chat.ID, alice); len(members) != 4 {
		t.Fatalf("members after reused key id = %d, want 4", len(members))
	}
}
