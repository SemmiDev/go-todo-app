// Package postgres provides the PostgreSQL implementation of the driven adapters.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// tagModel represents the database schema for a tag.
type tagModel struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Name      string    `db:"name"`
	Color     string    `db:"color"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (m *tagModel) toDomain() *todo.Tag {
	return todo.ReconstituteTag(m.ID, m.UserID, m.Name, m.Color, m.CreatedAt, m.UpdatedAt)
}

// TagRepo implements output.TagRepository using PostgreSQL.
type TagRepo struct{ db *DB }

// NewTagRepo returns a new TagRepo instance.
func NewTagRepo(db *DB) *TagRepo { return &TagRepo{db: db} }

// Create inserts a new tag into the database.
func (r *TagRepo) Create(ctx context.Context, t *todo.Tag) error {
	const q = `INSERT INTO tags (id, user_id, name, color, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q, t.ID(), t.UserID(), t.Name(), t.Color(), t.CreatedAt(), t.UpdatedAt())
	return wrapErr(err, "create tag")
}

// GetByID retrieves a tag by its ID.
func (r *TagRepo) GetByID(ctx context.Context, id uuid.UUID) (*todo.Tag, error) {
	var m tagModel
	err := r.db.GetQuerier(ctx).GetContext(ctx, &m, `SELECT * FROM tags WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return m.toDomain(), wrapErr(err, "get tag")
}

// Update modifies an existing tag.
func (r *TagRepo) Update(ctx context.Context, t *todo.Tag) error {
	const q = `UPDATE tags SET name = $1, color = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q, t.Name(), t.Color(), t.UpdatedAt(), t.ID())
	return wrapErr(err, "update tag")
}

// Delete removes a tag by its ID.
func (r *TagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, `DELETE FROM tags WHERE id = $1`, id)
	return wrapErr(err, "delete tag")
}

// ListByUserID retrieves all tags associated with a user ID.
func (r *TagRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*todo.Tag, error) {
	var rows []tagModel
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows,
		`SELECT * FROM tags WHERE user_id = $1 ORDER BY name`, userID); err != nil {
		return nil, wrapErr(err, "list tags")
	}
	tags := make([]*todo.Tag, len(rows))
	for i, row := range rows {
		tags[i] = row.toDomain()
	}
	return tags, nil
}

// GetByIDs retrieves multiple tags by their IDs.
func (r *TagRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*todo.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]interface{}, len(ids))
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args[i] = id
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	q := fmt.Sprintf(`SELECT * FROM tags WHERE id IN (%s)`, strings.Join(placeholders, ","))
	var rows []tagModel
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, wrapErr(err, "get tags by ids")
	}
	tags := make([]*todo.Tag, len(rows))
	for i, row := range rows {
		tags[i] = row.toDomain()
	}
	return tags, nil
}
