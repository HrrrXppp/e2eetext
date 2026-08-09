CREATE TABLE user_identity_keys (
    user_id    UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    device_id  TEXT NOT NULL DEFAULT 'default',
    public_key JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_key_versions (
    id         UUID PRIMARY KEY,
    chat_id    UUID NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX chat_key_versions_chat_id_idx ON chat_key_versions (chat_id);

CREATE TABLE user_chat_key_wraps (
    chat_id    UUID NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    key_id     UUID NOT NULL REFERENCES chat_key_versions (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    wrap       JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chat_id, key_id, user_id)
);

CREATE INDEX user_chat_key_wraps_user_chat_idx ON user_chat_key_wraps (user_id, chat_id);
