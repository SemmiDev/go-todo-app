// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// FindDueSoon returns todos that:
//   - have a due_date within the next `within` duration from now
//   - are not yet done
//   - have not yet received a reminder email
//
// The partial index on todos (due_date, user_id WHERE status != 'done' AND reminder_sent = FALSE)
// makes this query O(1) in practice.
func (r *TodoRepo) FindDueSoon(ctx context.Context, within time.Duration) ([]*todo.Todo, error) {
	now := time.Now().UTC()
	cutoff := now.Add(within)

	const q = `
		SELECT *
		FROM todos
		WHERE due_date IS NOT NULL
		  AND due_date > $1
		  AND due_date <= $2
		  AND status != 'done'
		  AND reminder_sent = FALSE
		ORDER BY due_date ASC`

	var rows []todoModel
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows, q, now, cutoff); err != nil {
		return nil, wrapErr(err, "find due soon todos")
	}

	todos := make([]*todo.Todo, len(rows))
	for i, row := range rows {
		todos[i] = row.toDomain()
	}
	return todos, nil
}

// MarkReminderSent atomically sets reminder_sent = TRUE for a single todo.
func (r *TodoRepo) MarkReminderSent(ctx context.Context, todoID uuid.UUID) error {
	const q = `UPDATE todos SET reminder_sent = TRUE WHERE id = $1`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q, todoID)
	return wrapErr(err, "mark reminder sent")
}
