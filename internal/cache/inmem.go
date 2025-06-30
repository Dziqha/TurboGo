package cache

import (
	"sync"
	"time"
)

type entry struct {
	Value     []byte
	ExpiresAt *time.Time
}

type InMemCache struct {
	mu      sync.RWMutex
	store   map[string]entry
	cleaner *time.Ticker
	done    chan struct{}
}

func NewInMem() *InMemCache {
	r := &InMemCache{
		store: make(map[string]entry),
		done:  make(chan struct{}),
	}
	
	r.cleaner = time.NewTicker(time.Minute)
	go r.cleanup()
	
	return r
}

func (r *InMemCache) Set(key string, value []byte, ttl time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var expiresAt *time.Time
	if ttl > 0 {
		expiry := time.Now().Add(ttl)
		expiresAt = &expiry
	}
	
	r.store[key] = entry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

func (r *InMemCache) Get(key string) ([]byte, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	e, ok := r.store[key]
	if !ok {
		return nil, false
	}
	
	if e.ExpiresAt != nil && time.Now().After(*e.ExpiresAt) {
		go func() {
			r.mu.Lock()
			delete(r.store, key)
			r.mu.Unlock()
		}()
		return nil, false
	}
	
	return e.Value, true
}

func (r *InMemCache) Delete(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	_, ok := r.store[key]
	if ok {
		delete(r.store, key)
	}
	return ok
}

func (r *InMemCache) Exists(key string) bool {
	_, exists := r.Get(key)
	return exists
}

func (r *InMemCache) SetEx(key string, value []byte, seconds int) {
	r.Set(key, value, time.Duration(seconds)*time.Second)
}

func (r *InMemCache) SetNX(key string, value []byte, ttl time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if e, ok := r.store[key]; ok {
		if e.ExpiresAt == nil || time.Now().Before(*e.ExpiresAt) {
			return false
		}
	}
	
	var expiresAt *time.Time
	if ttl > 0 {
		expiry := time.Now().Add(ttl)
		expiresAt = &expiry
	}
	
	r.store[key] = entry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	return true
}

func (r *InMemCache) cleanup() {
	for {
		select {
		case <-r.cleaner.C:
			r.cleanupExpired()
		case <-r.done:
			return
		}
	}
}

func (r *InMemCache) cleanupExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	for key, entry := range r.store {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			delete(r.store, key)
		}
	}
}

func (r *InMemCache) Close() {
	close(r.done)
	if r.cleaner != nil {
		r.cleaner.Stop()
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store = make(map[string]entry)
}

func (r *InMemCache) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.store)
}


func (r *InMemCache) TTL(key string) time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.store[key]
	if !ok {
		return -2 * time.Second // -2s = key not found
	}

	if e.ExpiresAt == nil {
		return -1 * time.Second // -1s = no expiration (infinite)
	}

	ttl := time.Until(*e.ExpiresAt)
	if ttl <= 0 {
		return -2 * time.Second // key expired
	}

	return ttl
}


func (r *InMemCache) Range(fn func(key string, value []byte)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.store {
		fn(k, v.Value)
	}
}

