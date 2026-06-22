CREATE TABLE unread_messages (
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    chat_id    UUID NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX unread_messages_user_id_chat_id_idx ON unread_messages (user_id, chat_id);
