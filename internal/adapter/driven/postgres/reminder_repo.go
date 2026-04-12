// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// FindDueSoon returns todos that have configured reminders that are now due
// but have not yet been triggered.
func (r *TodoRepo) FindDueSoon(ctx context.Context, _ time.Duration) ([]*todo.Todo, error) {
	const q = `
		WITH reminder_offsets AS (
			SELECT t.*, jsonb_array_elements_text(COALESCE(t.reminders, '[]'::jsonb)) AS offset_text
			FROM todos t
			WHERE t.status != 'done' 
			  AND t.due_date IS NOT NULL 
			  AND jsonb_array_length(COALESCE(t.reminders, '[]'::jsonb)) > 0
		)
		SELECT *
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
