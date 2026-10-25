//go:build integration

package integration

import (
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
