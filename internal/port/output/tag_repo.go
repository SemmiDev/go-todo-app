package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// TagRepository is the driven port for tag persistence.
type TagRepository interface {
	Create(ctx context.Context, t *todo.Tag) error
	GetByID(ctx context.Context, id uuid.UUID) (*todo.Tag, error)
	Update(ctx context.Context, t *todo.Tag) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*todo.Tag, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*todo.Tag, error)
}
