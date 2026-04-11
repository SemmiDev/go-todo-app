package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/domain/session"
)

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

// SessionRepo implements output.SessionRepository.
type SessionRepo struct{ db *DB }

func NewSessionRepo(db *DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(ctx context.Context, s *session.Session) error {
	const q = `INSERT INTO sessions (id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q,
		s.ID, s.UserID, s.RefreshToken, s.UserAgent, s.ClientIP, s.IsBlocked, s.ExpiresAt, s.CreatedAt,
	)
	return wrapErr(err, "create session")
}

func (r *SessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	var m sessionModel
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, `SELECT * FROM sessions WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, wrapErr(err, "get session")
	}
	return &session.Session{
		ID:           m.ID,
		UserID:       m.UserID,
		RefreshToken: m.RefreshToken,
		UserAgent:    m.UserAgent,
		ClientIP:     m.ClientIP,
		IsBlocked:    m.IsBlocked,
		ExpiresAt:    m.ExpiresAt,
		CreatedAt:    m.CreatedAt,
	}, nil
}

func (r *SessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return wrapErr(err, "delete session")
}

func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return wrapErr(err, "delete sessions by user")
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return wrapErr(err, "delete expired sessions")
}
