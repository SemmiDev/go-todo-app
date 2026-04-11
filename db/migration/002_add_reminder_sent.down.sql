DROP INDEX IF EXISTS todos_reminder_idx;
ALTER TABLE todos DROP COLUMN IF EXISTS reminder_sent;
