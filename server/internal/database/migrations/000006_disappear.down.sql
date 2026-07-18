DO $$
BEGIN
  PERFORM cron.unschedule(j.jobid)
  FROM cron.job j
  WHERE j.jobname = 'purge-expired-messages';
EXCEPTION
  WHEN undefined_table THEN
    NULL;
END $$;

DROP EXTENSION IF EXISTS pg_cron;

DROP FUNCTION IF EXISTS purge_expired_messages();

DROP INDEX IF EXISTS messages_chat_id_created_at_idx;

ALTER TABLE chats DROP COLUMN IF EXISTS disappear_after_minutes;
