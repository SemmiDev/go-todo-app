package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/common/apperr"
	"github.com/semmidev/todo-app/internal/domain/todo"
)

// Create inserts a new todo into the database.
func (r *TodoRepo) Create(ctx context.Context, t *todo.Todo) error {
	const q = `
		INSERT INTO todos (id, user_id, title, description, status, priority, due_date, created_at, updated_at, reminders, triggered_reminders)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q,
		t.ID(), t.UserID(), t.Title(), t.Description(),
		string(t.Status()), string(t.Priority()), t.DueDate(),
		t.CreatedAt(), t.UpdatedAt(), JSONBStringArray(t.Reminders()), JSONBStringArray(t.TriggeredReminders()))
	return wrapErr(err, "create todo")
}

// GetByID retrieves a todo by its ID.
func (r *TodoRepo) GetByID(ctx context.Context, id uuid.UUID) (*todo.Todo, error) {
	var m todoModel
	const q = `
		SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at, reminders, triggered_reminders
		FROM todos WHERE id = $1`
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return m.toDomain(), wrapErr(err, "get todo")
}

// Update modifies an existing todo.
func (r *TodoRepo) Update(ctx context.Context, t *todo.Todo) error {
	const q = `
		UPDATE todos SET title=$1, description=$2, status=$3, priority=$4, due_date=$5, updated_at=$6, reminders=$7, triggered_reminders=$8
		WHERE id=$9`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q,
		t.Title(), t.Description(), string(t.Status()), string(t.Priority()),
		t.DueDate(), t.UpdatedAt(), JSONBStringArray(t.Reminders()), JSONBStringArray(t.TriggeredReminders()), t.ID())
	return wrapErr(err, "update todo")
}

// Delete removes a todo by its ID.
func (r *TodoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM todos WHERE id = $1`, id)
	return wrapErr(err, "delete todo")
}
