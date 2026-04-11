package output

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// TodoFilter holds the filtering/pagination criteria for listing todos.
// It embeds filter.Filter for canonical pagination/sorting and adds
// todo-specific filter fields.
type TodoFilter struct {
	filter.Filter

	UserID uuid.UUID
	Status *todo.Status
	TagID  *uuid.UUID
}

// TodoRepository is the driven port for todo persistence.
type TodoRepository interface {
	Create(ctx context.Context, t *todo.Todo) error
	GetByID(ctx context.Context, id uuid.UUID) (*todo.Todo, error)
	Update(ctx context.Context, t *todo.Todo) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f TodoFilter) ([]*todo.Todo, int, error)

	// Reminder methods
	FindDueSoon(ctx context.Context, within time.Duration) ([]*todo.Todo, error)
	MarkReminderSent(ctx context.Context, todoID uuid.UUID) error
}

// TodoTagRepository is the driven port for the todo-tag join table.
type TodoTagRepository interface {
	SetTags(ctx context.Context, todoID uuid.UUID, tagIDs []uuid.UUID) error
	AddTag(ctx context.Context, todoID, tagID uuid.UUID) error
	RemoveTag(ctx context.Context, todoID, tagID uuid.UUID) error
	GetTagsForTodo(ctx context.Context, todoID uuid.UUID) ([]*todo.Tag, error)
	GetTagsForTodos(ctx context.Context, todoIDs []uuid.UUID) (map[uuid.UUID][]*todo.Tag, error)
}
