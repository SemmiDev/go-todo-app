// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import "context"

// Transactor defines the driven port for managing atomic database transactions.
type Transactor interface {
	// RunInTx executes the provided function within a single database transaction.
	// It handles the lifecycle of the transaction (begin, commit, rollback) and
	// propagates the transactional context.
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
