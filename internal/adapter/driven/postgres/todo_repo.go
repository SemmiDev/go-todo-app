// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// JSONBStringArray is a wrapper for []string to support PostgreSQL JSONB scanning/valuing.
type JSONBStringArray []string

func (a *JSONBStringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("type assertion to []byte or string failed: %T", value)
	}
	return json.Unmarshal(bytes, a)
}

func (a JSONBStringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

// todoModel represents the database schema for a todo.
type todoModel struct {
	ID                 uuid.UUID        `db:"id"`
	UserID             uuid.UUID        `db:"user_id"`
	Title              string           `db:"title"`
	Description        string           `db:"description"`
	Status             string           `db:"status"`
	Priority           string           `db:"priority"`
	DueDate            *time.Time       `db:"due_date"`
	CreatedAt          time.Time        `db:"created_at"`
	UpdatedAt          time.Time        `db:"updated_at"`
	Reminders          JSONBStringArray `db:"reminders"`
	TriggeredReminders JSONBStringArray `db:"triggered_reminders"`
}

func (m *todoModel) toDomain() *todo.Todo {
	return todo.Reconstitute(
		m.ID, m.UserID, m.Title, m.Description,
		todo.Status(m.Status), todo.Priority(m.Priority),
		m.DueDate, m.CreatedAt, m.UpdatedAt, nil,
		[]string(m.Reminders), []string(m.TriggeredReminders),
	)
}

// TodoRepo implements output.TodoRepository using PostgreSQL.
type TodoRepo struct{ db *DB }

// NewTodoRepo returns a new TodoRepo instance.
func NewTodoRepo(db *DB) *TodoRepo { return &TodoRepo{db: db} }

// allowedSortColumns whitelists columns for ORDER BY to prevent SQL injection.
var allowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"title":      true,
	"priority":   true,
	"status":     true,
	"due_date":   true,
}

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

// List retrieves todos based on the provided filter and pagination parameters.
func (r *TodoRepo) List(ctx context.Context, f output.TodoFilter) ([]*todo.Todo, int, error) {
	// Enforce safe defaults on the embedded filter
	f.Validate()

	args := []interface{}{f.UserID}
	where := `WHERE user_id = $1`

	if f.Status != nil {
		args = append(args, string(*f.Status))
		where += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	if f.TagID != nil {
		args = append(args, *f.TagID)
		where += fmt.Sprintf(` AND id IN (SELECT todo_id FROM todo_tags WHERE tag_id = $%d)`, len(args))
	}
	if f.HasKeyword() {
		args = append(args, "%"+f.Keyword+"%")
		where += fmt.Sprintf(` AND title ILIKE $%d`, len(args))
	}

	// ── Count query ───────────────────────────────────────────────────────────
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	var total int
	if err := r.db.GetQuerier(ctx).GetContext(ctx, &total,
		`SELECT COUNT(*) FROM todos `+where, countArgs...); err != nil {
		return nil, 0, wrapErr(err, "count todos")
	}

	// ── ORDER BY (whitelisted) ─────────────────────────────────────────────────
	orderClause := ` ORDER BY created_at DESC` // safe default
	if f.HasSort() && allowedSortColumns[f.SortBy] {
		dir := "ASC"
		if f.IsDesc() {
			dir = "DESC"
		}
		orderClause = fmt.Sprintf(` ORDER BY %s %s`, f.SortBy, dir)
	}

	// ── LIMIT / OFFSET ────────────────────────────────────────────────────────
	limitClause := ""
	limit := f.GetLimit()
	if limit != filter.UnlimitedPage {
		args = append(args, limit, f.GetOffset())
		limitClause = fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	}

	// ── Data query ────────────────────────────────────────────────────────────
	var rows []todoModel
	const dataQ = `
		SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at, reminders, triggered_reminders
		FROM todos `
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows,
		dataQ+where+orderClause+limitClause, args...); err != nil {
		return nil, 0, wrapErr(err, "list todos")
	}

	todos := make([]*todo.Todo, len(rows))
	for i, row := range rows {
		todos[i] = row.toDomain()
	}
	return todos, total, nil
}
