//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGoldenPath_AddKeyVersion(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")
	carol := registerUser(t, h, "carol") // not a member

	firstKey := randomUUID(t)
	chat, err := createChat(h, alice, "Rotating", []*user{alice, bob}, firstKey)
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	path := "/api/v1/chat/" + chat.ID + "/key-wraps"
	body := func(keyID string, members ...*user) map[string]any {
		wraps := make([]map[string]any, len(members))
		for i, m := range members {
			wraps[i] = map[string]any{"user_id": m.ID, "wrap": randomWrap(t, keyID)}
		}
		return map[string]any{"key_id": keyID, "wraps": wraps}
	}

	secondKey := randomUUID(t)
	if resp := h.do(http.MethodPost, path, body(secondKey, alice, bob), carol, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST key-wraps as non-member: status %d, want 403", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, alice), bob, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST key-wraps missing a member: status %d, want 400", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, alice, bob, carol), bob, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST key-wraps with a non-member wrap: status %d, want 400", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, alice, bob), bob, nil); resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST key-wraps as member: status %d, want 201", resp.StatusCode)
	}
	if resp := h.do(http.MethodPost, path, body(secondKey, alice, bob), bob, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("POST key-wraps with a reused key id: status %d, want 409", resp.StatusCode)
	}

	// Alice keeps the old wrap and gets the new one, oldest first.
	var wrapsResp struct {
		Wraps []struct {
			KeyID string `json:"keyId"`
		} `json:"wraps"`
	}
	if resp := h.do(http.MethodGet, path, nil, alice, &wrapsResp); resp.StatusCode != http.StatusOK {
		t.Fatalf("GET key-wraps: status %d", resp.StatusCode)
	}
	if len(wrapsResp.Wraps) != 2 || wrapsResp.Wraps[0].KeyID != firstKey || wrapsResp.Wraps[1].KeyID != secondKey {
		t.Fatalf("alice key wraps = %+v, want [%s %s]", wrapsResp.Wraps, firstKey, secondKey)
	}
}

func TestGoldenPath_PurgeDropsUnusedKeyVersions(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")

	keys := []string{randomUUID(t), randomUUID(t), randomUUID(t)}
	chat, err := createChat(h, alice, "Purging", []*user{alice, bob}, keys[0])
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}
	for _, keyID := range keys[1:] {
		wraps := []map[string]any{
			{"user_id": alice.ID, "wrap": randomWrap(t, keyID)},
			{"user_id": bob.ID, "wrap": randomWrap(t, keyID)},
		}
		body := map[string]any{"key_id": keyID, "wraps": wraps}
		if resp := h.do(http.MethodPost, "/api/v1/chat/"+chat.ID+"/key-wraps", body, alice, nil); resp.StatusCode != http.StatusCreated {
			t.Fatalf("POST key-wraps %s: status %d, want 201", keyID, resp.StatusCode)
		}
	}

	envelope := fmt.Sprintf(`{"v":1,"alg":"aes256gcm-chat-key","keyId":%q,"nonce":"AAAA","ciphertext":"hi"}`, keys[1])
	if resp := h.do(http.MethodPost, "/api/v1/message", map[string]any{"chat_id": chat.ID, "user_id": alice.ID, "data": envelope}, alice, nil); resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST message: status %d, want 201", resp.StatusCode)
	}

	remaining := func() []string {
		t.Helper()
		rows, err := h.db.Query(`SELECT id FROM chat_key_versions ORDER BY created_at, id`)
		if err != nil {
			t.Fatalf("list key versions: %v", err)
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatalf("scan key version: %v", err)
			}
			ids = append(ids, id)
		}
		return ids
	}
	wrapCount := func() int {
		t.Helper()
		var n int
		if err := h.db.QueryRow(`SELECT COUNT(*) FROM user_chat_key_wraps`).Scan(&n); err != nil {
			t.Fatalf("count wraps: %v", err)
		}
		return n
	}

	// The first version has no messages and is not the newest: it goes,
	// together with its wraps. The used and the newest versions stay.
	if _, err := h.db.Exec(`SELECT purge_expired_messages()`); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if got := remaining(); len(got) != 2 || got[0] != keys[1] || got[1] != keys[2] {
		t.Fatalf("key versions after first purge = %v, want [%s %s]", got, keys[1], keys[2])
	}
	if n := wrapCount(); n != 4 {
		t.Fatalf("wraps after first purge = %d, want 4", n)
	}

	// Once the message expires its version is unused; the newest still stays.
	if _, err := h.db.Exec(`UPDATE messages SET created_at = NOW() - INTERVAL '1000 days'`); err != nil {
		t.Fatalf("age message: %v", err)
	}
	if _, err := h.db.Exec(`SELECT purge_expired_messages()`); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if got := remaining(); len(got) != 1 || got[0] != keys[2] {
		t.Fatalf("key versions after second purge = %v, want [%s]", got, keys[2])
	}
	if n := wrapCount(); n != 2 {
		t.Fatalf("wraps after second purge = %d, want 2", n)
	}
}
