DROP TABLE chat_admins;

ALTER TABLE chats
    DROP COLUMN created_by;
