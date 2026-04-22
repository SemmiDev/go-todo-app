// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/common/apperr"
	"github.com/semmidev/todo-app/internal/domain/session"
)

// sessionModel represents the database schema for a session.
type sessionModel struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	RefreshToken string    `db:"refresh_token"`
	UserAgent    string    `db:"user_agent"`
	ClientIP     string    `db:"client_ip"`
	IsBlocked    bool      `db:"is_blocked"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}

func (m *sessionModel) toDomain() *session.Session {
	return session.Reconstitute(
		m.ID, m.UserID, m.RefreshToken, m.UserAgent, m.ClientIP,
		m.IsBlocked, m.ExpiresAt, m.CreatedAt,
	)
}

// SessionRepo implements output.SessionRepository using PostgreSQL.
type SessionRepo struct{ db *DB }

// NewSessionRepo returns a new SessionRepo instance.
func NewSessionRepo(db *DB) *SessionRepo { return &SessionRepo{db: db} }

// Create inserts a new session into the database.
func (r *SessionRepo) Create(ctx context.Context, s *session.Session) error {
	const q = `INSERT INTO sessions (id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q,
		s.ID(), s.UserID(), s.RefreshToken(), s.UserAgent(), s.ClientIP(), s.IsBlocked(), s.ExpiresAt(), s.CreatedAt(),
	)
	return wrapErr(err, "create session")
}

// GetByID retrieves a session by its ID.
func (r *SessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	var m sessionModel
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, `SELECT * FROM sessions WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, wrapErr(err, "get session")
	}
	return m.toDomain(), nil
}

// ListByUserID retrieves all sessions associated with a user ID.
func (r *SessionRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error) {
	var models []sessionModel
	err := r.db.GetQuerier(ctx).SelectContext(ctx, &models, `SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, wrapErr(err, "list sessions by user")
	}

	sessions := make([]*session.Session, 0, len(models))
	for _, m := range models {
		sessions = append(sessions, m.toDomain())
	}
	return sessions, nil
}

// Delete removes a session by its ID.
func (r *SessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return wrapErr(err, "delete session")
}

// DeleteByUserID removes all sessions associated with a user ID.
func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return wrapErr(err, "delete sessions by user")
}

// DeleteExpired removes all sessions that have passed their expiration time.
func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return wrapErr(err, "delete expired sessions")
}
