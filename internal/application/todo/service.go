// Package todo provides the application logic for managing todos and tags.
// It coordinates interactions between repositories, cache, and transactor.
package todo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
	"github.com/semmidev/go-todo-app/internal/port/input"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// Service implements both input.TagUseCase and input.TodoUseCase to handle todo-related workflows.
type Service struct {
	todoRepo    output.TodoRepository
	tagRepo     output.TagRepository
	todoTagRepo output.TodoTagRepository
	cacheRepo   output.CacheRepository
	uow         output.UnitOfWork
}

// Compile-time interface conformance checks.
var (
	_ input.TagUseCase  = (*Service)(nil)
	_ input.TodoUseCase = (*Service)(nil)
)

// NewService creates a new todo service with the necessary output ports.
func NewService(
	todoRepo output.TodoRepository,
	tagRepo output.TagRepository,
	todoTagRepo output.TodoTagRepository,
	cacheRepo output.CacheRepository,
	uow output.UnitOfWork,
) *Service {
	return &Service{
		todoRepo:    todoRepo,
		tagRepo:     tagRepo,
		todoTagRepo: todoTagRepo,
		cacheRepo:   cacheRepo,
		uow:         uow,
	}
}

// ─── Tag operations ───────────────────────────────────────────────────────────

// CreateTag persists a new tag for the specified user.
func (s *Service) CreateTag(ctx context.Context, p input.CreateTagParams) (*todo.Tag, error) {
	t, err := todo.NewTag(p.UserID, p.Name, p.Color)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", apperr.ErrValidation, err)
	}
	if err := s.tagRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// GetTag retrieves a tag by its ID, utilizing cache when available.
func (s *Service) GetTag(ctx context.Context, p input.GetTagParams) (*todo.Tag, error) {
	cacheKey := fmt.Sprintf("tag:%s", p.TagID.String())
	if data, err := s.cacheRepo.Get(ctx, cacheKey); err == nil {
		var c cachedTag
		if json.Unmarshal(data, &c) == nil {
			t := mapCacheToTag(&c)
			if t.UserID() == p.UserID {
				return t, nil
			}
		}
	}

	t, err := s.tagRepo.GetByID(ctx, p.TagID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if t.UserID() != p.UserID {
		return nil, apperr.ErrForbidden
	}

	if cData, err := json.Marshal(mapTagToCache(t)); err == nil {
		_ = s.cacheRepo.Set(ctx, cacheKey, cData, time.Hour)
	}

	return t, nil
}

// UpdateTag modifies an existing tag and invalidates its cache.
func (s *Service) UpdateTag(ctx context.Context, p input.UpdateTagParams) (*todo.Tag, error) {
	t, err := s.tagRepo.GetByID(ctx, p.TagID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if t.UserID() != p.UserID {
		return nil, apperr.ErrForbidden
	}
	if err := t.Update(p.Name, p.Color); err != nil {
		return nil, fmt.Errorf("%w: %s", apperr.ErrValidation, err)
	}
	if err := s.tagRepo.Update(ctx, t); err != nil {
		return nil, err
	}
	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("tag:%s", t.ID().String()))
	return t, nil
}

// DeleteTag removes a tag and cleans up related cache entries.
func (s *Service) DeleteTag(ctx context.Context, p input.DeleteTagParams) error {
	t, err := s.tagRepo.GetByID(ctx, p.TagID)
	if err != nil {
		return apperr.ErrNotFound
	}
	if t.UserID() != p.UserID {
		return apperr.ErrForbidden
	}
	if err := s.tagRepo.Delete(ctx, p.TagID); err != nil {
		return err
	}
	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("tag:%s", p.TagID.String()))
	return nil
}

// ListTags returns all tags belonging to a specific user.
func (s *Service) ListTags(ctx context.Context, p input.ListTagsParams) ([]*todo.Tag, error) {
	return s.tagRepo.ListByUserID(ctx, p.UserID)
}

// ─── Todo operations ──────────────────────────────────────────────────────────

// CreateTodo handles the creation of a new todo and associates it with tags within a transaction.
func (s *Service) CreateTodo(ctx context.Context, p input.CreateTodoParams) (*todo.Todo, error) {
	t, err := todo.New(p.UserID, p.Title, p.Description, p.Priority, p.DueDate, p.Reminders)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", apperr.ErrValidation, err)
	}

	err = s.uow.Do(ctx, func(store output.UnitOfWorkStore) error {
		if err := store.Todos().Create(ctx, t); err != nil {
			return err
		}
		if len(p.TagIDs) > 0 {
			if err := store.TodoTags().SetTags(ctx, t.ID(), p.TagIDs); err != nil {
				return err
			}
			tags, _ := store.TodoTags().GetTagsForTodo(ctx, t.ID())
			t.SetTags(tags)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return t, nil
}

// GetTodo retrieves a todo by its ID, including its tags, utilizing cache when available.
func (s *Service) GetTodo(ctx context.Context, p input.GetTodoParams) (*todo.Todo, error) {
	cacheKey := fmt.Sprintf("todo:%s", p.TodoID.String())
	if data, err := s.cacheRepo.Get(ctx, cacheKey); err == nil {
		var c cachedTodo
		if json.Unmarshal(data, &c) == nil {
			t := mapCacheToTodo(&c)
			if t.IsOwnedBy(p.UserID) {
				return t, nil
			}
		}
	}

	t, err := s.todoRepo.GetByID(ctx, p.TodoID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if !t.IsOwnedBy(p.UserID) {
		return nil, apperr.ErrForbidden
	}
	tags, _ := s.todoTagRepo.GetTagsForTodo(ctx, t.ID())
	t.SetTags(tags)

	if cData, err := json.Marshal(mapTodoToCache(t)); err == nil {
		_ = s.cacheRepo.Set(ctx, cacheKey, cData, time.Hour)
	}

	return t, nil
}

// UpdateTodo modifies a todo's properties and its tag associations within a transaction.
func (s *Service) UpdateTodo(ctx context.Context, p input.UpdateTodoParams) (*todo.Todo, error) {
	t, err := s.todoRepo.GetByID(ctx, p.TodoID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if !t.IsOwnedBy(p.UserID) {
		return nil, apperr.ErrForbidden
	}
	if err := t.Update(p.Title, p.Description, p.Status, p.Priority, p.DueDate, p.Reminders); err != nil {
		return nil, fmt.Errorf("%w: %s", apperr.ErrValidation, err)
	}

	err = s.uow.Do(ctx, func(store output.UnitOfWorkStore) error {
		if err := store.Todos().Update(ctx, t); err != nil {
			return err
		}
		if p.TagIDs != nil {
			if err := store.TodoTags().SetTags(ctx, t.ID(), p.TagIDs); err != nil {
				return err
			}
		}
		tags, _ := store.TodoTags().GetTagsForTodo(ctx, t.ID())
		t.SetTags(tags)
		return nil
	})
	if err != nil {
		return nil, err
	}

	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("todo:%s", t.ID().String()))
	return t, nil
}

// UpdateTodoStatus modifies only the status of a todo, enforcing state machine rules.
func (s *Service) UpdateTodoStatus(ctx context.Context, p input.UpdateTodoStatusParams) (*todo.Todo, error) {
	t, err := s.todoRepo.GetByID(ctx, p.TodoID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if !t.IsOwnedBy(p.UserID) {
		return nil, apperr.ErrForbidden
	}

	if err := t.TransitionStatus(p.Status); err != nil {
		return nil, fmt.Errorf("%w: %s", apperr.ErrValidation, err)
	}

	err = s.uow.Do(ctx, func(store output.UnitOfWorkStore) error {
		if err := store.Todos().Update(ctx, t); err != nil {
			return err
		}
		tags, _ := store.TodoTags().GetTagsForTodo(ctx, t.ID())
		t.SetTags(tags)
		return nil
	})
	if err != nil {
		return nil, err
	}

	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("todo:%s", t.ID().String()))
	return t, nil
}

// DeleteTodo removes a todo and cleans up related cache entries.
func (s *Service) DeleteTodo(ctx context.Context, p input.DeleteTodoParams) error {
	t, err := s.todoRepo.GetByID(ctx, p.TodoID)
	if err != nil {
		return apperr.ErrNotFound
	}
	if !t.IsOwnedBy(p.UserID) {
		return apperr.ErrForbidden
	}
	if err := s.todoRepo.Delete(ctx, p.TodoID); err != nil {
		return err
	}
	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("todo:%s", p.TodoID.String()))
	return nil
}

// ListTodos returns a paginated list of todos based on the provided filters.
func (s *Service) ListTodos(ctx context.Context, p input.ListTodosParams) (*input.ListTodosResult, error) {
	// Enforce safe defaults (clamping, sorting direction, etc.)
	p.Validate()

	todos, total, err := s.todoRepo.List(ctx, output.TodoFilter{
		Filter: p.Filter,
		UserID: p.UserID,
		Status: p.Status,
		TagID:  p.TagID,
	})
	if err != nil {
		return nil, err
	}

	// Enrich with tags (batch query)
	if len(todos) > 0 {
		ids := make([]uuid.UUID, len(todos))
		for i, t := range todos {
			ids[i] = t.ID()
		}
		tagMap, _ := s.todoTagRepo.GetTagsForTodos(ctx, ids)
		for _, t := range todos {
			if tags, ok := tagMap[t.ID()]; ok {
				t.SetTags(tags)
			}
		}
	}

	paging, err := filter.NewPaging(p.CurrentPage, p.PerPage, total)
	if err != nil {
		return nil, err
	}
	return &input.ListTodosResult{Todos: todos, Paging: paging}, nil
}

// AddTagToTodo associates a tag with a todo, ensuring proper ownership.
func (s *Service) AddTagToTodo(ctx context.Context, p input.AddTagToTodoParams) (*todo.Todo, error) {
	t, err := s.todoRepo.GetByID(ctx, p.TodoID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if !t.IsOwnedBy(p.UserID) {
		return nil, apperr.ErrForbidden
	}
	tag, err := s.tagRepo.GetByID(ctx, p.TagID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if tag.UserID() != p.UserID {
		return nil, apperr.ErrForbidden
	}
	if err := s.todoTagRepo.AddTag(ctx, p.TodoID, p.TagID); err != nil {
		return nil, err
	}
	tags, _ := s.todoTagRepo.GetTagsForTodo(ctx, t.ID())
	t.SetTags(tags)
	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("todo:%s", t.ID().String()))
	return t, nil
}

// RemoveTagFromTodo disassociates a tag from a todo.
func (s *Service) RemoveTagFromTodo(ctx context.Context, p input.RemoveTagFromTodoParams) (*todo.Todo, error) {
	t, err := s.todoRepo.GetByID(ctx, p.TodoID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if !t.IsOwnedBy(p.UserID) {
		return nil, apperr.ErrForbidden
	}
	if err := s.todoTagRepo.RemoveTag(ctx, p.TodoID, p.TagID); err != nil {
		return nil, err
	}
	tags, _ := s.todoTagRepo.GetTagsForTodo(ctx, t.ID())
	t.SetTags(tags)
	_ = s.cacheRepo.Delete(ctx, fmt.Sprintf("todo:%s", t.ID().String()))
	return t, nil
}

// ─── Caching Helpers ──────────────────────────────────────────────────────────

type cachedTag struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type cachedTodo struct {
	ID                 uuid.UUID     `json:"id"`
	UserID             uuid.UUID     `json:"user_id"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	Status             todo.Status   `json:"status"`
	Priority           todo.Priority `json:"priority"`
	DueDate            *time.Time    `json:"due_date"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	Tags               []*cachedTag  `json:"tags"`
	Reminders          []string      `json:"reminders"`
	TriggeredReminders []string      `json:"triggered_reminders"`
}

func mapTagToCache(t *todo.Tag) *cachedTag {
	if t == nil {
		return nil
	}
	return &cachedTag{
		ID: t.ID(), UserID: t.UserID(), Name: t.Name(), Color: t.Color(),
		CreatedAt: t.CreatedAt(), UpdatedAt: t.UpdatedAt(),
	}
}

func mapCacheToTag(c *cachedTag) *todo.Tag {
	if c == nil {
		return nil
	}
	return todo.ReconstituteTag(c.ID, c.UserID, c.Name, c.Color, c.CreatedAt, c.UpdatedAt)
}

func mapTodoToCache(t *todo.Todo) *cachedTodo {
	if t == nil {
		return nil
	}
	var cTags []*cachedTag
	for _, tg := range t.Tags() {
		cTags = append(cTags, mapTagToCache(tg))
	}
	return &cachedTodo{
		ID: t.ID(), UserID: t.UserID(), Title: t.Title(), Description: t.Description(),
		Status: t.Status(), Priority: t.Priority(), DueDate: t.DueDate(),
		CreatedAt: t.CreatedAt(), UpdatedAt: t.UpdatedAt(), Tags: cTags,
		Reminders: t.Reminders(), TriggeredReminders: t.TriggeredReminders(),
	}
}

func mapCacheToTodo(c *cachedTodo) *todo.Todo {
	if c == nil {
		return nil
	}
	var tags []*todo.Tag
	for _, ct := range c.Tags {
		tags = append(tags, mapCacheToTag(ct))
	}
	return todo.Reconstitute(c.ID, c.UserID, c.Title, c.Description, c.Status, c.Priority, c.DueDate, c.CreatedAt, c.UpdatedAt, tags, c.Reminders, c.TriggeredReminders)
}
