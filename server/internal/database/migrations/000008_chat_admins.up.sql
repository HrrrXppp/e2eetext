-- Chat admins: a real multi-admin membership model instead of a single
-- implicit "owner". chats.created_by records who created the chat;
-- chat_admins is the authorization primitive future privileged chat actions
-- (#12-#15) will gate on.
--
-- No backfill: this predates any deployed production data, so there are no
-- pre-existing chats to migrate. Every chat created from here on gets its
-- creator recorded in created_by and inserted into chat_admins atomically
-- at creation time (see ChatRepository.Create / E2EERepository.CreateChatWithKeys).
ALTER TABLE chats
    ADD COLUMN created_by UUID REFERENCES users (id) ON DELETE SET NULL;

-- user_id is ON DELETE RESTRICT (not CASCADE): a user who is still an admin
-- of any chat cannot be deleted, so a chat can never be left with a member
-- but no admin as a side effect of deleting an admin's account (#36).
-- Reassigning/demoting the admin first (so their chat_admins row is removed
-- through the normal admin-management path) is required before the account
-- can be deleted; that reassignment path is deferred to a future change.
CREATE TABLE chat_admins (
    chat_id    UUID NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX chat_admins_user_id_idx ON chat_admins (user_id);
