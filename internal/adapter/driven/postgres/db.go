package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// DB wraps a postgres connection pool with sqlx capabilities and context-aware
// transaction orchestration.
type DB struct {
	pool *pgxpool.Pool
	*sqlx.DB
}

// New creates a DB instance connected via pgxpool and wrapped by sqlx.
func New(ctx context.Context, dsn string) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("new pgxpool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	dbx := sqlx.NewDb(sqlDB, "pgx")

	dbx.SetMaxOpenConns(25)
	dbx.SetMaxIdleConns(10)

	return &DB{pool: pool, DB: dbx}, nil
}

// Close shuts down both the sqlx wrapper and the underlying pgxpool.
func (db *DB) Close() error {
	db.DB.Close()
	db.pool.Close()
	return nil
}

// txKey is an unexported context key for the active transaction.
type txKey struct{}

// RunInTx executes fn within a database transaction.
// If fn returns an error the transaction is rolled back; otherwise it is committed.
// Nested calls (ctx already carries a tx) reuse the existing transaction.
func (db *DB) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		// Already in a transaction — reuse it
		return fn(ctx)
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return fmt.Errorf("fn error: %v; rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Querier is satisfied by both *sqlx.DB and *sqlx.Tx, enabling query functions
// to work transparently inside or outside a transaction.
type Querier interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}

// GetQuerier returns the active transaction from ctx if one exists, otherwise
// returns the underlying *sqlx.DB.
func (db *DB) GetQuerier(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db.DB
}

// wrapErr wraps a non-nil error with an operation prefix.
func wrapErr(err error, op string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
