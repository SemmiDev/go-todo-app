package output

import "context"

// Transactor defines the interface for context-based database transaction orchestration.
// The underlying adapter will automatically wrap the lambda execution in a transaction,
// passing down the localized pgxpool.Tx through the context map.
type Transactor interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
