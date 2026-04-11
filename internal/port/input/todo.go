package input

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// ─── Tag Params ───────────────────────────────────────────────────────────────

// CreateTagParams holds parameters for creating a tag.
type CreateTagParams struct {
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	Name   string    `json:"name" validate:"required,max=50"`
	Color  string    `json:"color" validate:"omitempty,iscolor"`
}

// GetTagParams holds parameters for retrieving a tag.
type GetTagParams struct {
	TagID  uuid.UUID `json:"tag_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// UpdateTagParams holds parameters for updating a tag.
type UpdateTagParams struct {
	TagID  uuid.UUID `json:"tag_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	Name   string    `json:"name" validate:"required,max=50"`
	Color  string    `json:"color" validate:"omitempty,iscolor"`
}

// DeleteTagParams holds parameters for deleting a tag.
type DeleteTagParams struct {
	TagID  uuid.UUID `json:"tag_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ListTagsParams holds parameters for listing tags.
type ListTagsParams struct {
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ─── Todo Params ──────────────────────────────────────────────────────────────

// CreateTodoParams holds parameters for creating a todo.
type CreateTodoParams struct {
	UserID      uuid.UUID     `json:"user_id" validate:"required,uuid"`
	Title       string        `json:"title" validate:"required,max=200"`
	Description string        `json:"description" validate:"max=1000"`
	Priority    todo.Priority `json:"priority" validate:"omitempty,oneof=low medium high"`
	DueDate     *time.Time    `json:"due_date" validate:"omitempty"`
	TagIDs      []uuid.UUID   `json:"tag_ids" validate:"omitempty,dive,uuid"`
}

// GetTodoParams holds parameters for retrieving a todo.
type GetTodoParams struct {
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// UpdateTodoParams holds parameters for updating a todo.
type UpdateTodoParams struct {
	TodoID      uuid.UUID     `json:"todo_id" validate:"required,uuid"`
	UserID      uuid.UUID     `json:"user_id" validate:"required,uuid"`
	Title       string        `json:"title" validate:"required,max=200"`
	Description string        `json:"description" validate:"max=1000"`
	Status      todo.Status   `json:"status" validate:"omitempty,oneof=pending in_progress done"`
	Priority    todo.Priority `json:"priority" validate:"omitempty,oneof=low medium high"`
	DueDate     *time.Time    `json:"due_date" validate:"omitempty"`
	TagIDs      []uuid.UUID   `json:"tag_ids" validate:"omitempty,dive,uuid"`
}

// DeleteTodoParams holds parameters for deleting a todo.
type DeleteTodoParams struct {
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ListTodosParams holds filtering and pagination parameters.
type ListTodosParams struct {
	filter.Filter

	UserID uuid.UUID    `json:"user_id" validate:"required,uuid"`
	Status *todo.Status `json:"status" validate:"omitempty,oneof=pending in_progress done"`
	TagID  *uuid.UUID   `json:"tag_id" validate:"omitempty,uuid"`
}

// AddTagToTodoParams holds parameters for adding a tag to a todo.
type AddTagToTodoParams struct {
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	TagID  uuid.UUID `json:"tag_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// RemoveTagFromTodoParams holds parameters for removing a tag from a todo.
type RemoveTagFromTodoParams struct {
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	TagID  uuid.UUID `json:"tag_id" validate:"required,uuid"`
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ─── Todo Results ─────────────────────────────────────────────────────────────

// ListTodosResult holds paginated todos and their rich Paging metadata.
type ListTodosResult struct {
	Todos  []*todo.Todo
	Paging *filter.Paging
}

// ─── Use-case interfaces ─────────────────────────────────────────────────────

// TagUseCase is the driving port for tag operations.
type TagUseCase interface {
	CreateTag(ctx context.Context, p CreateTagParams) (*todo.Tag, error)
	GetTag(ctx context.Context, p GetTagParams) (*todo.Tag, error)
	UpdateTag(ctx context.Context, p UpdateTagParams) (*todo.Tag, error)
	DeleteTag(ctx context.Context, p DeleteTagParams) error
	ListTags(ctx context.Context, p ListTagsParams) ([]*todo.Tag, error)
}

// TodoUseCase is the driving port for todo operations.
type TodoUseCase interface {
	CreateTodo(ctx context.Context, p CreateTodoParams) (*todo.Todo, error)
	GetTodo(ctx context.Context, p GetTodoParams) (*todo.Todo, error)
	UpdateTodo(ctx context.Context, p UpdateTodoParams) (*todo.Todo, error)
	DeleteTodo(ctx context.Context, p DeleteTodoParams) error
	ListTodos(ctx context.Context, p ListTodosParams) (*ListTodosResult, error)
	AddTagToTodo(ctx context.Context, p AddTagToTodoParams) (*todo.Todo, error)
	RemoveTagFromTodo(ctx context.Context, p RemoveTagFromTodoParams) (*todo.Todo, error)
}
