-- Add reminder tracking to todos.
-- The partial index makes FindDueSoon queries extremely fast even on large tables.
ALTER TABLE todos ADD COLUMN IF NOT EXISTS reminder_sent BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS todos_reminder_idx
    ON todos (due_date, user_id)
    WHERE status != 'done'
      AND reminder_sent = FALSE
      AND due_date IS NOT NULL;
