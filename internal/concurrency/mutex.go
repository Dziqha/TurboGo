package concurrency

import "sync"

// SafeMap wraps a map with a RWMutex for safe concurrent access
type SafeMap[K comparable, V any] struct {
	mu      sync.RWMutex
	mapData map[K]V
}

// NewSafeMap creates a new concurrent-safe map
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		mapData: make(map[K]V),
	}
}

// Set sets the key to the value
func (s *SafeMap[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mapData[key] = value
}

// Get returns the value for a key and a boolean if it exists
func (s *SafeMap[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.mapData[key]
	return val, ok
}

// Delete removes a key from the map
func (s *SafeMap[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.mapData, key)
}

// Exists returns true if the key exists
func (s *SafeMap[K, V]) Exists(key K) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.mapData[key]
	return ok
}
