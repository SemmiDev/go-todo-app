// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import (
	"context"
	"time"
)

// CacheRepository is the driven port for temporary data storage and retrieval.
type CacheRepository interface {
	// Set stores a byte slice value in the cache with a specified expiration.
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	// Get retrieves a value from the cache by its key.
	Get(ctx context.Context, key string) ([]byte, error)
	// Delete removes a value from the cache by its key.
	Delete(ctx context.Context, key string) error
}
