// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/domain/todo"
)

// TagRepository is the driven port for managing tags in the database.
type TagRepository interface {
	// Create persists a new tag.
	Create(ctx context.Context, t *todo.Tag) error
	// GetByID retrieves a specific tag by its unique ID.
	GetByID(ctx context.Context, id uuid.UUID) (*todo.Tag, error)
	// Update modifies an existing tag's details.
	Update(ctx context.Context, t *todo.Tag) error
	// Delete removes a tag from the database.
	Delete(ctx context.Context, id uuid.UUID) error
	// ListByUserID retrieves all tags belonging to a specific user.
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*todo.Tag, error)
	// GetByIDs retrieves a batch of tags by their identifiers.
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*todo.Tag, error)
}
