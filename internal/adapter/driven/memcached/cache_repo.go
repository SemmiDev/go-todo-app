package memcached

import (
	"context"
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
)

// CacheRepo implements output.CacheRepository using Memcached.
type CacheRepo struct {
	client *memcache.Client
}

// NewCacheRepo creates a new Memcached adapter.
func NewCacheRepo(serverList ...string) *CacheRepo {
	return &CacheRepo{
		client: memcache.New(serverList...),
	}
}

func (r *CacheRepo) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	item := &memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: int32(expiration.Seconds()),
	}
	err := r.client.Set(item)
	if err != nil {
		return err
	}
	return nil
}

func (r *CacheRepo) Get(ctx context.Context, key string) ([]byte, error) {
	item, err := r.client.Get(key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

func (r *CacheRepo) Delete(ctx context.Context, key string) error {
	err := r.client.Delete(key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil // idempotent delete
	}
	if err != nil {
		return err
	}
	return nil
}
