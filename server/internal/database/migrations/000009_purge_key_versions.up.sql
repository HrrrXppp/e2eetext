-- Expiring messages also expires the chat key versions nothing uses anymore.
-- A version (and, by cascade, its wraps) is removed once no message names it
-- in its envelope keyId. The newest version of every chat is always kept, so
-- members can still send into a chat whose messages have all expired.
CREATE OR REPLACE FUNCTION purge_expired_messages()
RETURNS bigint
LANGUAGE plpgsql
AS $$
DECLARE
  purged bigint;
BEGIN
  WITH deleted AS (
    DELETE FROM messages m
    USING chats c
    WHERE m.chat_id = c.id
      AND m.created_at < NOW() - (c.disappear_after_minutes * INTERVAL '1 minute')
    RETURNING m.id
  )
  SELECT COUNT(*) INTO purged FROM deleted;

  DELETE FROM chat_key_versions v
  WHERE EXISTS (
      SELECT 1 FROM chat_key_versions newer
      WHERE newer.chat_id = v.chat_id
        AND (newer.created_at, newer.id) > (v.created_at, v.id)
    )
    AND NOT EXISTS (
      SELECT 1 FROM messages m
      WHERE m.chat_id = v.chat_id
        AND strpos(m.data, '"keyId":"' || v.id::text || '"') > 0
    );

  RETURN purged;
END;
$$;

COMMENT ON FUNCTION purge_expired_messages() IS
    'Hard-deletes messages past chat TTL (created_at + disappear_after_minutes), then chat key versions (and their wraps) that no message uses, except each chat''s newest. Scheduled by pg_cron.';
