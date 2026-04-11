package output

import (
	"context"
	"time"
)

// CacheRepository is the driven port for caching.
type CacheRepository interface {
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
