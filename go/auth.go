package main

import "sync"

type AuthStore struct {
	mu   sync.RWMutex
	keys map[string]struct{}
}

func NewAuthStore(keys []string) *AuthStore {
	a := &AuthStore{}
	a.Reload(keys)
	return a
}

func (a *AuthStore) Reload(keys []string) {
	next := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key != "" {
			next[key] = struct{}{}
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.keys = next
}

func (a *AuthStore) Valid(key string) bool {
	if key == "" {
		return false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.keys[key]
	return ok
}

func (a *AuthStore) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.keys)
}
