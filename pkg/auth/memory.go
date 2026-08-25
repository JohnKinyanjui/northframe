package auth

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is a concurrency-safe development and test session store.
// Production applications should implement Store with their shared database or
// cache so sessions survive restarts and work across instances.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	clock    func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]Session), clock: time.Now}
}

func (store *MemoryStore) Load(_ context.Context, id string) (Session, error) {
	store.mu.RLock()
	session, ok := store.sessions[id]
	store.mu.RUnlock()
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	if !session.ExpiresAt.After(store.clock()) {
		store.mu.Lock()
		delete(store.sessions, id)
		store.mu.Unlock()
		return Session{}, ErrSessionNotFound
	}
	return cloneSession(session), nil
}

func (store *MemoryStore) Save(_ context.Context, session Session) error {
	store.mu.Lock()
	store.sessions[session.ID] = cloneSession(session)
	store.mu.Unlock()
	return nil
}

func (store *MemoryStore) Delete(_ context.Context, id string) error {
	store.mu.Lock()
	delete(store.sessions, id)
	store.mu.Unlock()
	return nil
}
