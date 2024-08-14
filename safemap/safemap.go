package safemap

import (
	"sync/atomic"
	"unsafe"
)

type No struct{}

func (*No) Lock()   {}
func (*No) Unlock() {}

type SafeMap[K comparable, V any] struct {
	_ cacheLinePadding

	data atomic.Pointer[map[K]V]
	_    [cacheLinePadSize - unsafe.Sizeof(atomic.Pointer[map[K]V]{})]byte
}

func New[K comparable, V any]() *SafeMap[K, V] {

	m := make(map[K]V)
	sm := &SafeMap[K, V]{}
	sm.data.Store(&m)
	return sm
}

func (s *SafeMap[K, V]) Set(k K, v V) {

	m := s.data.Load()
	(*m)[k] = v
}

func (s *SafeMap[K, V]) Store(k K, v V) {
	s.Set(k, v)
}

func (s *SafeMap[K, V]) Get(k K) (V, bool) {

	m := s.data.Load()
	val, ok := (*m)[k]
	return val, ok
}

// Load retrieves the value for the given key and returns whether it exists.
func (s *SafeMap[K, V]) Load(k K) (V, bool) {
	return s.Get(k)
}

// Delete removes the key-value pair from the map.
func (s *SafeMap[K, V]) Delete(k K) {

	m := s.data.Load()
	delete(*m, k)
}

// Len returns the number of key-value pairs in the map.
func (s *SafeMap[K, V]) Len() int {

	m := s.data.Load()
	return len(*m)
}

// ForEach iterates over all key-value pairs in the map and applies the function f.
func (s *SafeMap[K, V]) ForEach(f func(K, V)) {

	m := s.data.Load()
	for k, v := range *m {
		f(k, v)
	}
}
