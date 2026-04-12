// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
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
