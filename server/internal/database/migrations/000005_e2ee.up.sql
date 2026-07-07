ALTER TABLE users ADD COLUMN kem_public_key BYTEA NOT NULL;

ALTER TABLE chats ADD COLUMN admin_user_id UUID NOT NULL REFERENCES users (id);
ALTER TABLE chats ADD COLUMN kem_public_key BYTEA NOT NULL;

ALTER TABLE user_chats ADD COLUMN wrapped_chat_private_key BYTEA NOT NULL;
ALTER TABLE user_chats ADD COLUMN kem_ciphertext BYTEA NOT NULL;

ALTER TABLE messages ADD COLUMN kem_ciphertext BYTEA NOT NULL;
