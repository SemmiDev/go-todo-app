// Package input defines the input ports (use cases) of the application.
// These interfaces are implemented by the application layer and called by the driving adapters.
package input

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/common/filter"
	"github.com/semmidev/todo-app/internal/domain/todo"
)

// ─── Tag Params ───────────────────────────────────────────────────────────────

// CreateTagParams holds parameters for creating a tag.
type CreateTagParams struct {
	// UserID is the unique identifier of the user who owns the tag.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	// Name is the display name of the tag.
	Name string `json:"name" validate:"required,max=50"`
	// Color is the hex color code or name for the tag.
	Color string `json:"color" validate:"omitempty,iscolor"`
}

// GetTagParams holds parameters for retrieving a tag.
type GetTagParams struct {
	// TagID is the unique identifier of the tag to retrieve.
	TagID uuid.UUID `json:"tag_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the tag.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// UpdateTagParams holds parameters for updating a tag.
type UpdateTagParams struct {
	// TagID is the unique identifier of the tag to update.
	TagID uuid.UUID `json:"tag_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the tag.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	// Name is the new display name of the tag.
	Name string `json:"name" validate:"required,max=50"`
	// Color is the new hex color code or name for the tag.
	Color string `json:"color" validate:"omitempty,iscolor"`
}

// DeleteTagParams holds parameters for deleting a tag.
type DeleteTagParams struct {
	// TagID is the unique identifier of the tag to delete.
	TagID uuid.UUID `json:"tag_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the tag.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ListTagsParams holds parameters for listing tags.
type ListTagsParams struct {
	// UserID is the unique identifier of the user whose tags are being listed.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ─── Todo Params ──────────────────────────────────────────────────────────────

// CreateTodoParams holds parameters for creating a todo.
type CreateTodoParams struct {
	// UserID is the unique identifier of the user who owns the todo.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	// Title is the brief summary of the todo item.
	Title string `json:"title" validate:"required,max=200"`
	// Description is the detailed explanation of the todo item.
	Description string `json:"description" validate:"max=1000"`
	// Priority is the importance level of the todo (low, medium, high).
	Priority todo.Priority `json:"priority" validate:"omitempty,oneof=low medium high"`
	// DueDate is the optional deadline for the todo item.
	DueDate *time.Time `json:"due_date" validate:"omitempty"`
	// TagIDs is a list of tag identifiers to associate with the todo.
	TagIDs []uuid.UUID `json:"tag_ids" validate:"omitempty,dive,uuid"`
	// Reminders is a list of durations before the due date to send reminders (e.g., "1h", "15m").
	Reminders []string `json:"reminders" validate:"omitempty,dive,duration"`
}

// GetTodoParams holds parameters for retrieving a todo.
type GetTodoParams struct {
	// TodoID is the unique identifier of the todo to retrieve.
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the todo.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// UpdateTodoParams holds parameters for updating a todo.
type UpdateTodoParams struct {
	// TodoID is the unique identifier of the todo to update.
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the todo.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	// Title is the new brief summary of the todo item.
	Title string `json:"title" validate:"required,max=200"`
	// Description is the new detailed explanation of the todo item.
	Description string `json:"description" validate:"max=1000"`
	// Status is the current state of the todo (pending, in_progress, done).
	Status todo.Status `json:"status" validate:"omitempty,oneof=pending in_progress done"`
	// Priority is the new importance level of the todo.
	Priority todo.Priority `json:"priority" validate:"omitempty,oneof=low medium high"`
	// DueDate is the new optional deadline for the todo item.
	DueDate *time.Time `json:"due_date" validate:"omitempty"`
	// TagIDs is the new list of tag identifiers for the todo.
	TagIDs []uuid.UUID `json:"tag_ids" validate:"omitempty,dive,uuid"`
	// Reminders is the new list of durations before the due date to send reminders.
	Reminders []string `json:"reminders" validate:"omitempty,dive,duration"`
}

// UpdateTodoStatusParams holds parameters for updating a todo's status.
type UpdateTodoStatusParams struct {
	// TodoID is the unique identifier of the todo to update.
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the todo.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	// Status is the new current state of the todo (pending, in_progress, done).
	Status todo.Status `json:"status" validate:"required,oneof=pending in_progress done"`
}

// DeleteTodoParams holds parameters for deleting a todo.
type DeleteTodoParams struct {
	// TodoID is the unique identifier of the todo to delete.
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the todo.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ListTodosParams holds filtering and pagination parameters.
type ListTodosParams struct {
	filter.Filter

	// UserID is the unique identifier of the user whose todos are being listed.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	// Status filters todos by their current state.
	Status *todo.Status `json:"status" validate:"omitempty,oneof=pending in_progress done"`
	// TagID filters todos that have a specific tag.
	TagID *uuid.UUID `json:"tag_id" validate:"omitempty,uuid"`
}

// AddTagToTodoParams holds parameters for adding a tag to a todo.
type AddTagToTodoParams struct {
	// TodoID is the unique identifier of the todo.
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	// TagID is the unique identifier of the tag to add.
	TagID uuid.UUID `json:"tag_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns both.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// RemoveTagFromTodoParams holds parameters for removing a tag from a todo.
type RemoveTagFromTodoParams struct {
	// TodoID is the unique identifier of the todo.
	TodoID uuid.UUID `json:"todo_id" validate:"required,uuid"`
	// TagID is the unique identifier of the tag to remove.
	TagID uuid.UUID `json:"tag_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns both.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ─── Todo Results ─────────────────────────────────────────────────────────────

// ListTodosResult holds paginated todos and their rich Paging metadata.
type ListTodosResult struct {
	// Todos is the slice of todo items for the current page.
	Todos []*todo.Todo
	// Paging contains pagination metadata like total count and next/prev page info.
	Paging *filter.Paging
}

// ─── Use-case interfaces ─────────────────────────────────────────────────────

// TagUseCase is the driving port for tag operations.
type TagUseCase interface {
	// CreateTag persists a new tag for a user.
	CreateTag(ctx context.Context, p CreateTagParams) (*todo.Tag, error)
	// GetTag retrieves a specific tag by its ID and user ID.
	GetTag(ctx context.Context, p GetTagParams) (*todo.Tag, error)
	// UpdateTag modifies an existing tag's details.
	UpdateTag(ctx context.Context, p UpdateTagParams) (*todo.Tag, error)
	// DeleteTag removes a tag from the system.
	DeleteTag(ctx context.Context, p DeleteTagParams) error
	// ListTags returns all tags belonging to a specific user.
	ListTags(ctx context.Context, p ListTagsParams) ([]*todo.Tag, error)
}

// TodoUseCase is the driving port for todo operations.
type TodoUseCase interface {
	// CreateTodo persists a new todo item for a user.
	CreateTodo(ctx context.Context, p CreateTodoParams) (*todo.Todo, error)
	// GetTodo retrieves a specific todo by its ID and user ID.
	GetTodo(ctx context.Context, p GetTodoParams) (*todo.Todo, error)
	// UpdateTodo modifies an existing todo's details.
	UpdateTodo(ctx context.Context, p UpdateTodoParams) (*todo.Todo, error)
	// UpdateTodoStatus modifies only the status of an existing todo.
	UpdateTodoStatus(ctx context.Context, p UpdateTodoStatusParams) (*todo.Todo, error)
	// DeleteTodo removes a todo item from the system.
	DeleteTodo(ctx context.Context, p DeleteTodoParams) error
	// ListTodos returns a paginated list of todos based on filters.
	ListTodos(ctx context.Context, p ListTodosParams) (*ListTodosResult, error)
	// AddTagToTodo associates a tag with a todo item.
	AddTagToTodo(ctx context.Context, p AddTagToTodoParams) (*todo.Todo, error)
	// RemoveTagFromTodo disassociates a tag from a todo item.
	RemoveTagFromTodo(ctx context.Context, p RemoveTagFromTodoParams) (*todo.Todo, error)
}
