// Package output defines the driven ports for persistence and external integrations.
// Every repository interface must be defined here and implemented in internal/adapter/driven.
package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/user"
)

// UserRepository is the driven port for user persistence.
// It abstracts all database operations related to the user domain.
type UserRepository interface {
	// GetOrCreateByEmail retrieves a user by their email or creates a new one if it doesn't exist.
	GetOrCreateByEmail(ctx context.Context, email, fullName string) (*user.User, error)
	// GetByID retrieves a user by their unique identifier.
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	// GetByEmail retrieves a user by their email address.
	GetByEmail(ctx context.Context, email string) (*user.User, error)
}
