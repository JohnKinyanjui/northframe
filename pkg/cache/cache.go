// Package cache defines an adapter-friendly byte cache and a concurrency-safe
// in-memory implementation for development and single-instance deployments.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var ErrMiss = errors.New("cache miss")

type Store interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	Delete(context.Context, string) error
}

type entry struct {
	value   []byte
	expires time.Time
}
type Memory struct {
	mu    sync.RWMutex
	items map[string]entry
	now   func() time.Time
}

func NewMemory() *Memory { return &Memory{items: map[string]entry{}, now: time.Now} }
func (store *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store.mu.RLock()
	item, ok := store.items[key]
	store.mu.RUnlock()
	if !ok || !item.expires.IsZero() && !store.now().Before(item.expires) {
		if ok {
			_ = store.Delete(context.Background(), key)
		}
		return nil, ErrMiss
	}
	return append([]byte(nil), item.value...), nil
}
func (store *Memory) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item := entry{value: append([]byte(nil), value...)}
	if ttl > 0 {
		item.expires = store.now().Add(ttl)
	}
	store.mu.Lock()
	store.items[key] = item
	store.mu.Unlock()
	return nil
}
func (store *Memory) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	delete(store.items, key)
	store.mu.Unlock()
	return nil
}

func SetJSON[T any](ctx context.Context, store Store, key string, value T, ttl time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return store.Set(ctx, key, encoded, ttl)
}
func GetJSON[T any](ctx context.Context, store Store, key string) (T, error) {
	var value T
	encoded, err := store.Get(ctx, key)
	if err != nil {
		return value, err
	}
	err = json.Unmarshal(encoded, &value)
	return value, err
}
