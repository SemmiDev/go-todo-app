package postgres

import (
	"context"

	"github.com/semmidev/go-todo-app/internal/port/output"
)

type unitOfWork struct {
	db *DB
}

func NewUnitOfWork(db *DB) output.UnitOfWork {
	return &unitOfWork{db: db}
}

func (u *unitOfWork) Do(ctx context.Context, fn func(store output.UnitOfWorkStore) error) error {
	return u.db.RunInTx(ctx, func(txCtx context.Context) error {
		return fn(&unitOfWorkStore{db: u.db, ctx: txCtx})
	})
}

type unitOfWorkStore struct {
	db  *DB
	ctx context.Context
}

func (s *unitOfWorkStore) Todos() output.TodoRepository {
	return NewTodoRepo(s.db)
}

func (s *unitOfWorkStore) Tags() output.TagRepository {
	return NewTagRepo(s.db)
}

func (s *unitOfWorkStore) TodoTags() output.TodoTagRepository {
	return NewTodoTagRepo(s.db)
}
