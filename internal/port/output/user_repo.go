package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/user"
)

// UserRepository is the driven port for user persistence.
type UserRepository interface {
	GetOrCreateByEmail(ctx context.Context, email, fullName string) (*user.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
}
