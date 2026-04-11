package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

type todoModel struct {
	ID           uuid.UUID  `db:"id"`
	UserID       uuid.UUID  `db:"user_id"`
	Title        string     `db:"title"`
	Description  string     `db:"description"`
	Status       string     `db:"status"`
	Priority     string     `db:"priority"`
	DueDate      *time.Time `db:"due_date"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	ReminderSent bool       `db:"reminder_sent"`
}

func (m *todoModel) toDomain() *todo.Todo {
	return todo.Reconstitute(
		m.ID, m.UserID, m.Title, m.Description,
		todo.Status(m.Status), todo.Priority(m.Priority),
		m.DueDate, m.CreatedAt, m.UpdatedAt, nil,
		m.ReminderSent,
	)
}

// TodoRepo implements output.TodoRepository.
type TodoRepo struct{ db *DB }

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

func (r *TodoRepo) Create(ctx context.Context, t *todo.Todo) error {
	const q = `
		INSERT INTO todos (id, user_id, title, description, status, priority, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q,
		t.ID(), t.UserID(), t.Title(), t.Description(),
		string(t.Status()), string(t.Priority()), t.DueDate(),
		t.CreatedAt(), t.UpdatedAt())
	return wrapErr(err, "create todo")
}

func (r *TodoRepo) GetByID(ctx context.Context, id uuid.UUID) (*todo.Todo, error) {
	var m todoModel
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, `SELECT * FROM todos WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return m.toDomain(), wrapErr(err, "get todo")
}

func (r *TodoRepo) Update(ctx context.Context, t *todo.Todo) error {
	const q = `
		UPDATE todos SET title=$1, description=$2, status=$3, priority=$4, due_date=$5, updated_at=$6
		WHERE id=$7`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q,
		t.Title(), t.Description(), string(t.Status()), string(t.Priority()),
		t.DueDate(), t.UpdatedAt(), t.ID())
	return wrapErr(err, "update todo")
}

func (r *TodoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM todos WHERE id = $1`, id)
	return wrapErr(err, "delete todo")
}

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
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows,
		`SELECT * FROM todos `+where+orderClause+limitClause, args...); err != nil {
		return nil, 0, wrapErr(err, "list todos")
	}

	todos := make([]*todo.Todo, len(rows))
	for i, row := range rows {
		todos[i] = row.toDomain()
	}
	return todos, total, nil
}
