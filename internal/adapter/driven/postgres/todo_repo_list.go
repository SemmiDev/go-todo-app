package postgres

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// List retrieves todos based on the provided filter and pagination parameters.
func (r *TodoRepo) List(ctx context.Context, f output.TodoFilter) ([]*todo.Todo, int, error) {
	f.Validate()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// ── Build Select ─────────────────────────────────────────────────────────
	baseSelect := psql.Select("id, user_id, title, description, status, priority, due_date, created_at, updated_at, reminders, triggered_reminders").
		From("todos").
		Where(squirrel.Eq{"user_id": f.UserID})

	if f.Status != nil {
		baseSelect = baseSelect.Where(squirrel.Eq{"status": string(*f.Status)})
	}
	if f.TagID != nil {
		baseSelect = baseSelect.Where("id IN (SELECT todo_id FROM todo_tags WHERE tag_id = ?)", f.TagID)
	}
	if f.HasKeyword() {
		baseSelect = baseSelect.Where("title ILIKE ?", "%"+f.Keyword+"%")
	}

	// Date range filter on due_date
	if f.StartDate != nil {
		baseSelect = baseSelect.Where("due_date >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		baseSelect = baseSelect.Where("due_date <= ?", f.EndDate)
	}

	// ── Count query ───────────────────────────────────────────────────────────
	countQuery, countArgs, err := psql.Select("COUNT(*)").
		From("todos").
		Where(squirrel.Eq{"user_id": f.UserID}). // basic filters for count
		ToSql()
	
	// Re-building the count query with all filters to be accurate
	countBuilder := psql.Select("COUNT(*)").From("todos").Where(squirrel.Eq{"user_id": f.UserID})
	if f.Status != nil { countBuilder = countBuilder.Where(squirrel.Eq{"status": string(*f.Status)}) }
	if f.TagID != nil { countBuilder = countBuilder.Where("id IN (SELECT todo_id FROM todo_tags WHERE tag_id = ?)", f.TagID) }
	if f.HasKeyword() { countBuilder = countBuilder.Where("title ILIKE ?", "%"+f.Keyword+"%") }
	if f.StartDate != nil { countBuilder = countBuilder.Where("due_date >= ?", f.StartDate) }
	if f.EndDate != nil { countBuilder = countBuilder.Where("due_date <= ?", f.EndDate) }
	
	countQuery, countArgs, err = countBuilder.ToSql()
	if err != nil {
		return nil, 0, wrapErr(err, "build count query")
	}

	var total int
	if err := r.db.GetQuerier(ctx).GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, wrapErr(err, "count todos")
	}

	// ── Order, Limit, Offset ──────────────────────────────────────────────────
	if f.HasSort() && allowedSortColumns[f.SortBy] {
		dir := "ASC"
		if f.IsDesc() {
			dir = "DESC"
		}
		baseSelect = baseSelect.OrderBy(fmt.Sprintf("%s %s", f.SortBy, dir))
	} else {
		baseSelect = baseSelect.OrderBy("created_at DESC")
	}

	limit := f.GetLimit()
	if limit != filter.UnlimitedPage {
		baseSelect = baseSelect.Limit(uint64(limit)).Offset(uint64(f.GetOffset()))
	}

	dataQuery, dataArgs, err := baseSelect.ToSql()
	if err != nil {
		return nil, 0, wrapErr(err, "build data query")
	}

	var rows []todoModel
	if err := r.db.GetQuerier(ctx).SelectContext(ctx, &rows, dataQuery, dataArgs...); err != nil {
		return nil, 0, wrapErr(err, "list todos")
	}

	todos := make([]*todo.Todo, len(rows))
	for i, row := range rows {
		todos[i] = row.toDomain()
	}
	return todos, total, nil
}
