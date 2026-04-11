package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// TodoTagRepo implements output.TodoTagRepository.
type TodoTagRepo struct{ db *DB }

func NewTodoTagRepo(db *DB) *TodoTagRepo { return &TodoTagRepo{db: db} }

func (r *TodoTagRepo) SetTags(ctx context.Context, todoID uuid.UUID, tagIDs []uuid.UUID) error {
	if _, err := r.db.GetQuerier(ctx).ExecContext(ctx,
		`DELETE FROM todo_tags WHERE todo_id = $1`, todoID); err != nil {
		return wrapErr(err, "clear todo tags")
	}
	if len(tagIDs) == 0 {
		return nil
	}
	args := []interface{}{todoID}
	vals := make([]string, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		args = append(args, tagID)
		vals = append(vals, fmt.Sprintf(`($1, $%d)`, len(args)))
	}
	q := fmt.Sprintf(`INSERT INTO todo_tags (todo_id, tag_id) VALUES %s ON CONFLICT DO NOTHING`,
		strings.Join(vals, ","))
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx, q, args...)
	return wrapErr(err, "set todo tags")
}

func (r *TodoTagRepo) AddTag(ctx context.Context, todoID, tagID uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx,
		`INSERT INTO todo_tags (todo_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, todoID, tagID)
	return wrapErr(err, "add tag to todo")
}

func (r *TodoTagRepo) RemoveTag(ctx context.Context, todoID, tagID uuid.UUID) error {
	_, err := r.db.GetQuerier(ctx).ExecContext(ctx,
		`DELETE FROM todo_tags WHERE todo_id = $1 AND tag_id = $2`, todoID, tagID)
	return wrapErr(err, "remove tag from todo")
}

func (r *TodoTagRepo) GetTagsForTodo(ctx context.Context, todoID uuid.UUID) ([]*todo.Tag, error) {
	const q = `
		SELECT t.* FROM tags t
		JOIN todo_tags tt ON tt.tag_id = t.id
		WHERE tt.todo_id = $1
		ORDER BY t.name`
	var rows []tagModel
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows, q, todoID); err != nil {
		return nil, wrapErr(err, "get tags for todo")
	}
	tags := make([]*todo.Tag, len(rows))
	for i, row := range rows {
		tags[i] = row.toDomain()
	}
	return tags, nil
}

func (r *TodoTagRepo) GetTagsForTodos(ctx context.Context, todoIDs []uuid.UUID) (map[uuid.UUID][]*todo.Tag, error) {
	if len(todoIDs) == 0 {
		return nil, nil
	}
	args := make([]interface{}, len(todoIDs))
	placeholders := make([]string, len(todoIDs))
	for i, id := range todoIDs {
		args[i] = id
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	q := fmt.Sprintf(`
		SELECT tt.todo_id, t.id, t.user_id, t.name, t.color, t.created_at, t.updated_at
		FROM tags t
		JOIN todo_tags tt ON tt.tag_id = t.id
		WHERE tt.todo_id IN (%s)
		ORDER BY t.name`, strings.Join(placeholders, ","))

	type row struct {
		TodoID    uuid.UUID `db:"todo_id"`
		ID        uuid.UUID `db:"id"`
		UserID    uuid.UUID `db:"user_id"`
		Name      string    `db:"name"`
		Color     string    `db:"color"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	var rows []row
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, wrapErr(err, "get tags for todos")
	}
	result := make(map[uuid.UUID][]*todo.Tag)
	for _, rw := range rows {
		result[rw.TodoID] = append(result[rw.TodoID],
			todo.ReconstituteTag(rw.ID, rw.UserID, rw.Name, rw.Color, rw.CreatedAt, rw.UpdatedAt))
	}
	return result, nil
}
