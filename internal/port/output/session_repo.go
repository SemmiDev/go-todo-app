package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/session"
)

// SessionRepository is the driven port for session persistence.
type SessionRepository interface {
	Create(ctx context.Context, s *session.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
