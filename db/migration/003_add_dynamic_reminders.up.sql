ALTER TABLE todos ADD COLUMN IF NOT EXISTS reminders JSONB DEFAULT '[]';
ALTER TABLE todos ADD COLUMN IF NOT EXISTS triggered_reminders JSONB DEFAULT '[]';
ALTER TABLE todos DROP COLUMN IF EXISTS reminder_sent;

DROP INDEX IF EXISTS todos_reminder_idx;
CREATE INDEX IF NOT EXISTS todos_dynamic_reminder_idx
    ON todos (due_date)
    WHERE status != 'done'
      AND due_date IS NOT NULL
      AND jsonb_array_length(reminders) > 0;
