// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/domain/todo"
)

// FindDueSoon returns todos that have configured reminders that are now due
// but have not yet been triggered.
func (r *TodoRepo) FindDueSoon(ctx context.Context) ([]*todo.Todo, error) {
	const q = `
		WITH reminder_offsets AS (
			SELECT
				id, user_id, title, description, status, priority, due_date, created_at, updated_at, reminders, triggered_reminders,
				jsonb_array_elements_text(COALESCE(reminders, '[]'::jsonb)) AS offset_text
			FROM todos
			WHERE status != 'done'
			  AND due_date IS NOT NULL
			  AND jsonb_array_length(COALESCE(reminders, '[]'::jsonb)) > 0
		)
		SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at, reminders, triggered_reminders
		FROM reminder_offsets
		WHERE (due_date - (offset_text::interval)) <= NOW()
		  AND NOT (COALESCE(triggered_reminders, '[]'::jsonb) ? offset_text)
		ORDER BY due_date ASC`

	var rows []todoModel
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows, q); err != nil {
		return nil, wrapErr(err, "find due soon todos")
	}

	todos := make([]*todo.Todo, len(rows))
	for i, row := range rows {
		todos[i] = row.toDomain()
	}
	return todos, nil
}

// MarkReminderTriggered records that a specific reminder offset has been sent for a todo.
func (r *TodoRepo) MarkReminderTriggered(ctx context.Context, todoID uuid.UUID, offset string) error {
	const q = `
		UPDATE todos
		SET triggered_reminders = triggered_reminders || jsonb_build_array($1::text)
		WHERE id = $2 AND NOT (triggered_reminders ? $1)`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q, offset, todoID)
	return wrapErr(err, "mark reminder triggered")
}
