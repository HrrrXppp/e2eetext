//go:build integration

package integration

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// randomUUID returns a random RFC 4122 v4-formatted UUID string. chat_key_versions.id
// is a Postgres UUID column, so key IDs used in these tests must be valid UUIDs —
// a real client would generate one client-side the same way when minting a new chat key.
func randomUUID(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate random uuid: %v", err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

type apiUser struct {
	ID string `json:"id"`
}

type apiChat struct {
	ID string `json:"id"`
}

type apiMessage struct {
	ID     string `json:"id"`
	Data   string `json:"data"`
	Unread bool   `json:"unread"`
}

// randomIdentityPublicKey returns a base64url (no padding) string of exactly
// e2ee.HybridIdentityPublicKeyBytes random bytes. The server only validates
// shape/length for identity keys and chat-key wraps — it can never validate
// their cryptographic content, since it never holds a private key — so
// random bytes of the right size exercise the exact same code paths a real
// client's ML-KEM public key would.
func randomIdentityPublicKey(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 1216)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate random identity key: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func randomWrap(t *testing.T, keyID string) json.RawMessage {
	t.Helper()
	buf := make([]byte, 64)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate random wrap bytes: %v", err)
	}
	wrap := map[string]any{
		"v":             1,
		"alg":           "hybrid-kem-mlkem768-x25519-aes256gcm",
		"keyId":         keyID,
		"kemCiphertext": base64.RawURLEncoding.EncodeToString(buf),
		"nonce":         base64.RawURLEncoding.EncodeToString(buf[:12]),
		"ciphertext":    base64.RawURLEncoding.EncodeToString(buf),
	}
	raw, err := json.Marshal(wrap)
	if err != nil {
		t.Fatalf("marshal wrap: %v", err)
	}
	return raw
}

func putIdentityKey(t *testing.T, h *harness, actor *user) {
	t.Helper()
	publicKey := map[string]any{
		"v":         1,
		"alg":       "hybrid-kem-mlkem768-x25519",
		"publicKey": randomIdentityPublicKey(t),
	}
	resp := h.do(http.MethodPut, "/api/v1/user/"+actor.ID+"/identity-key",
		map[string]any{"publicKey": publicKey}, actor, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT identity-key for %s: status %d", actor.Subject, resp.StatusCode)
	}
}

// registerUser signs a fresh mock identity in, registers it as a server-side
// user via the real auth+user flow, and uploads an identity key — the exact
// sequence a real client performs on first sign-in.
func registerUser(t *testing.T, h *harness, name string) *user {
	t.Helper()

	u := h.newUser(name)

	var created apiUser
	resp := h.do(http.MethodPost, "/api/v1/user", nil, u, &created)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/v1/user for %s: status %d", name, resp.StatusCode)
	}
	u.ID = created.ID

	putIdentityKey(t, h, u)

	return u
}

func TestGoldenPath_ChatAndMessageRoundTrip(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")

	keyID := randomUUID(t)
	chat, err := createChat(h, alice, "Alice & Bob", []*user{alice, bob}, keyID)
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	plaintextEnvelope := fmt.Sprintf(`{"v":1,"alg":"aes256gcm-chat-key","keyId":%q,"nonce":"AAAA","ciphertext":"hello from alice"}`, keyID)

	var sent apiMessage
	resp := h.do(http.MethodPost, "/api/v1/message", map[string]any{
		"chat_id": chat.ID,
		"user_id": alice.ID,
		"data":    plaintextEnvelope,
	}, alice, &sent)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/v1/message: status %d", resp.StatusCode)
	}
	if sent.Data != plaintextEnvelope {
		t.Fatalf("stored message data = %q, want the exact opaque envelope we sent (server must never transform ciphertext)", sent.Data)
	}

	// Bob lists messages: the ciphertext must round-trip byte-for-byte, and
	// his copy must be marked unread (Alice's own message never is).
	var bobMessages []apiMessage
	resp = h.do(http.MethodGet, "/api/v1/message?chat_id="+chat.ID, nil, bob, &bobMessages)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/message as bob: status %d", resp.StatusCode)
	}
	if len(bobMessages) != 1 {
		t.Fatalf("bob sees %d messages, want 1", len(bobMessages))
	}
	if bobMessages[0].Data != plaintextEnvelope {
		t.Fatalf("bob's copy of the ciphertext = %q, want it unchanged from what alice sent", bobMessages[0].Data)
	}
	if !bobMessages[0].Unread {
		t.Fatalf("bob's message should be unread before he marks it read")
	}

	// Mark read, then confirm it clears.
	resp = h.do(http.MethodPatch, "/api/v1/message/"+bobMessages[0].ID, map[string]any{"unread": false}, bob, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH mark-read: status %d", resp.StatusCode)
	}

	// Fresh slice: apiMessage.Unread has `json:",omitempty"`, so a read message
	// omits the field entirely — decoding into the reused bobMessages slice
	// would silently keep its stale `true` value instead of zeroing it.
	var bobMessagesAfterRead []apiMessage
	resp = h.do(http.MethodGet, "/api/v1/message?chat_id="+chat.ID, nil, bob, &bobMessagesAfterRead)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/message as bob (after mark-read): status %d", resp.StatusCode)
	}
	if bobMessagesAfterRead[0].Unread {
		t.Fatalf("bob's message should be read after PATCH unread=false")
	}
}

func TestGoldenPath_KeyWrapsOnlyVisibleToMembers(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")
	carol := registerUser(t, h, "carol") // not a member of the chat below

	keyID := randomUUID(t)
	chat, err := createChat(h, alice, "Alice & Bob only", []*user{alice, bob}, keyID)
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	var wrapsResp struct {
		Wraps []struct {
			KeyID string `json:"keyId"`
		} `json:"wraps"`
	}

	resp := h.do(http.MethodGet, "/api/v1/chat/"+chat.ID+"/key-wraps", nil, bob, &wrapsResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET key-wraps as member bob: status %d", resp.StatusCode)
	}
	if len(wrapsResp.Wraps) != 1 || wrapsResp.Wraps[0].KeyID != keyID {
		t.Fatalf("bob's key wraps = %+v, want exactly one wrap for keyId %q", wrapsResp.Wraps, keyID)
	}

	resp = h.do(http.MethodGet, "/api/v1/chat/"+chat.ID+"/key-wraps", nil, carol, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("GET key-wraps as non-member carol: status %d, want 403", resp.StatusCode)
	}
}

// createChat performs the real client-side chat-creation sequence: build one
// wrap per member and POST them all in the e2ee payload.
func createChat(h *harness, creator *user, name string, members []*user, keyID string) (apiChat, error) {
	userIDs := make([]string, len(members))
	wraps := make([]map[string]any, len(members))
	for i, m := range members {
		userIDs[i] = m.ID
		wraps[i] = map[string]any{"user_id": m.ID, "wrap": randomWrap(h.t, keyID)}
	}

	var chat apiChat
	resp := h.do(http.MethodPost, "/api/v1/chat", map[string]any{
		"name":       name,
		"users_uids": userIDs,
		"e2ee": map[string]any{
			"key_id": keyID,
			"wraps":  wraps,
		},
	}, creator, &chat)
	if resp.StatusCode != http.StatusCreated {
		return apiChat{}, fmt.Errorf("POST /api/v1/chat: status %d", resp.StatusCode)
	}
	return chat, nil
}
