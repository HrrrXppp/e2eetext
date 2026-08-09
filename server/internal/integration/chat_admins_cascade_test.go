//go:build integration

package integration

import (
	"strings"
	"testing"
)

// TestGoldenPath_ChatAdmins_DeleteSoleAdminProhibited covers the fix for
// #36: chat_admins.user_id is `ON DELETE RESTRICT` (not CASCADE), so
// deleting a user who is still an admin of a chat is rejected by the
// database rather than silently cascading their chat_admins row away and
// leaving their (still-a-member) chat partner with no admin at all. The
// account can only be deleted once it's no longer an admin of any chat
// (reassigning/demoting first) — that reassignment path is deferred to a
// future change.
func TestGoldenPath_ChatAdmins_DeleteSoleAdminProhibited(t *testing.T) {
	h := newHarness(t)

	alice := registerUser(t, h, "alice")
	bob := registerUser(t, h, "bob")

	keyID := randomUUID(t)
	chat, err := createChat(h, alice, "Alice & Bob", []*user{alice, bob}, keyID)
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}
	if !chat.IsAdmin {
		t.Fatal("creator alice should be admin on the chat she just created")
	}

	aliceLocalID := localIDOf(alice.ID)

	if _, err := h.db.Exec(`DELETE FROM users WHERE id = $1`, aliceLocalID); err == nil {
		t.Fatal("expected deleting alice (sole admin) to be prohibited, but it succeeded")
	}
}

// localIDOf strips the leading "<nodeId>/" from a scoped resource ID,
// returning the raw local UUID as stored in Postgres — needed when a test
// drops to direct SQL (e.g. asserting ON DELETE CASCADE behavior) that the
// API layer's node-scoped IDs don't apply to.
func localIDOf(scopedID string) string {
	_, localID, ok := strings.Cut(scopedID, "/")
	if !ok {
		return scopedID
	}
	return localID
}
