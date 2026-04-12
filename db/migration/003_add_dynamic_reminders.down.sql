DROP INDEX IF EXISTS todos_dynamic_reminder_idx;
ALTER TABLE todos DROP COLUMN IF EXISTS reminders;
ALTER TABLE todos DROP COLUMN IF EXISTS triggered_reminders;
ALTER TABLE todos ADD COLUMN IF NOT EXISTS reminder_sent BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS todos_reminder_idx
    ON todos (due_date, user_id)
    WHERE status != 'done'
      AND reminder_sent = FALSE
      AND due_date IS NOT NULL;
