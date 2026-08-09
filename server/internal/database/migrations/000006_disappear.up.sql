ALTER TABLE chats
    ADD COLUMN disappear_after_minutes INTEGER NOT NULL DEFAULT 86400
        CHECK (disappear_after_minutes > 0);

COMMENT ON COLUMN chats.disappear_after_minutes IS
    'Chat-level message TTL in minutes; expiry is messages.created_at + this value (default 60 days = 86400).';

CREATE INDEX messages_chat_id_created_at_idx ON messages (chat_id, created_at);

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

-- Requires cron.database_name to be this database (local Compose: messenger;
-- RDS: Terraform modules/rds parameter group + out-of-band reboot).
CREATE EXTENSION IF NOT EXISTS pg_cron;

DO $$
BEGIN
  PERFORM cron.unschedule(j.jobid)
  FROM cron.job j
  WHERE j.jobname = 'purge-expired-messages';
EXCEPTION
  WHEN undefined_table THEN
    NULL;
END $$;

SELECT cron.schedule(
  'purge-expired-messages',
  '* * * * *',
  $$SELECT purge_expired_messages()$$
);
