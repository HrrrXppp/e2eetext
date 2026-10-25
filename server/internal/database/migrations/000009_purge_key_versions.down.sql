CREATE OR REPLACE FUNCTION purge_expired_messages()
RETURNS bigint
LANGUAGE sql
AS $$
  WITH deleted AS (
    DELETE FROM messages m
    USING chats c
    WHERE m.chat_id = c.id
      AND m.created_at < NOW() - (c.disappear_after_minutes * INTERVAL '1 minute')
    RETURNING m.id
  )
  SELECT COUNT(*) FROM deleted;
$$;

COMMENT ON FUNCTION purge_expired_messages() IS
    'Hard-deletes messages past chat TTL (created_at + disappear_after_minutes). Scheduled by pg_cron.';
