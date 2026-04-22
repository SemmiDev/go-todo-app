// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/todo-app/internal/common/apperr"
	"github.com/semmidev/todo-app/internal/domain/user"
)

// userModel represents the database schema for a user.
type userModel struct {
	ID        uuid.UUID `db:"id"`
	Email     string    `db:"email"`
	FullName  string    `db:"full_name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (m *userModel) toDomain() *user.User {
	return user.Reconstitute(m.ID, m.Email, m.FullName, m.CreatedAt, m.UpdatedAt)
}

// UserRepo implements output.UserRepository using PostgreSQL.
type UserRepo struct{ db *DB }

// NewUserRepo returns a new UserRepo instance.
func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

// GetOrCreateByEmail retrieves a user by email or creates a new one if it doesn't exist.
func (r *UserRepo) GetOrCreateByEmail(ctx context.Context, email, fullName string) (*user.User, error) {
	const q = `
		INSERT INTO users (id, email, full_name, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET full_name = EXCLUDED.full_name, updated_at = NOW()
		RETURNING id, email, full_name, created_at, updated_at`
	id, _ := uuid.NewV7()
	var m userModel
	if err := r.db.GetQuerier(ctx).GetContext(ctx, &m, q, id, email, fullName); err != nil {
		return nil, wrapErr(err, "get or create user")
	}
	return m.toDomain(), nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var m userModel
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, `SELECT * FROM users WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return m.toDomain(), wrapErr(err, "get user by id")
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var m userModel
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, `SELECT * FROM users WHERE email = $1`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return m.toDomain(), wrapErr(err, "get user by email")
}
