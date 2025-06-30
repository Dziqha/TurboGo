package concurrency

import "sync"

// SafeValue adalah wrapper thread-safe untuk menyimpan nilai generic
type SafeValue[T any] struct {
	mu  sync.RWMutex
	val T
}

// Set mengatur nilai secara eksklusif
func (s *SafeValue[T]) Set(val T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.val = val
}

// Get mengambil salinan nilai secara aman
func (s *SafeValue[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.val
}

// LockFn menjalankan fungsi dengan akses eksklusif ke val
func (s *SafeValue[T]) LockFn(fn func(val *T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.val)
}

// RLockFn menjalankan fungsi dengan akses baca ke val
func (s *SafeValue[T]) RLockFn(fn func(val T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.val)
}
