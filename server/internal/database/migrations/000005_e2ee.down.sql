ALTER TABLE messages DROP COLUMN kem_ciphertext;

ALTER TABLE user_chats DROP COLUMN kem_ciphertext;
ALTER TABLE user_chats DROP COLUMN wrapped_chat_private_key;

ALTER TABLE chats DROP COLUMN kem_public_key;
ALTER TABLE chats DROP COLUMN admin_user_id;

ALTER TABLE users DROP COLUMN kem_public_key;
