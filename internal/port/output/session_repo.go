// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/domain/session"
)

// SessionRepository is the driven port for managing authentication sessions in the database.
type SessionRepository interface {
	// Create persists a new user session.
	Create(ctx context.Context, s *session.Session) error
	// GetByID retrieves a specific session by its unique ID.
	GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error)
	// ListByUserID retrieves all active sessions for a specific user.
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)
	// Delete removes a session by its unique ID.
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteByUserID removes all sessions belonging to a specific user.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	// DeleteExpired removes all sessions that have passed their expiration date.
	DeleteExpired(ctx context.Context) error
}
