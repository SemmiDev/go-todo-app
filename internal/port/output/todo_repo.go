// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// TodoFilter holds the canonical criteria for querying todos.
type TodoFilter struct {
	filter.Filter

	// UserID filters todos by their owner.
	UserID uuid.UUID
	// Status optionally filters todos by their current state.
	Status *todo.Status
	// TagID optionally filters todos that are associated with a specific tag.
	TagID *uuid.UUID
}

// TodoRepository is the driven port for managing todos in the database.
type TodoRepository interface {
	// Create persists a new todo item.
	Create(ctx context.Context, t *todo.Todo) error
	// GetByID retrieves a specific todo by its unique ID.
	GetByID(ctx context.Context, id uuid.UUID) (*todo.Todo, error)
	// Update modifies an existing todo's details.
	Update(ctx context.Context, t *todo.Todo) error
	// Delete removes a todo from the database.
	Delete(ctx context.Context, id uuid.UUID) error
	// List retrieves a paginated and filtered list of todos, including total count.
	List(ctx context.Context, f TodoFilter) ([]*todo.Todo, int, error)

	// FindDueSoon retrieves todos with deadlines approaching based on their configured reminders.
	FindDueSoon(ctx context.Context) ([]*todo.Todo, error)
	// MarkReminderTriggered records that a specific reminder offset for the given todo has been dispatched.
	MarkReminderTriggered(ctx context.Context, todoID uuid.UUID, offset string) error
}

// TodoTagRepository is the driven port for managing the relationship between todos and tags.
type TodoTagRepository interface {
	// SetTags replaces all existing tag associations for a todo with a new set.
	SetTags(ctx context.Context, todoID uuid.UUID, tagIDs []uuid.UUID) error
	// AddTag creates a new association between a todo and a tag.
	AddTag(ctx context.Context, todoID, tagID uuid.UUID) error
	// RemoveTag deletes an association between a todo and a tag.
	RemoveTag(ctx context.Context, todoID, tagID uuid.UUID) error
	// GetTagsForTodo retrieves all tags associated with a specific todo.
	GetTagsForTodo(ctx context.Context, todoID uuid.UUID) ([]*todo.Tag, error)
	// GetTagsForTodos retrieves tags for multiple todos, returning a map keyed by todo ID.
	GetTagsForTodos(ctx context.Context, todoIDs []uuid.UUID) (map[uuid.UUID][]*todo.Tag, error)
}
