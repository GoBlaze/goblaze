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

	data atomic.Value
	_    [cacheLinePadSize - unsafe.Sizeof(atomic.Value{})]byte
}

func New[K comparable, V any]() *SafeMap[K, V] {
	sm := &SafeMap[K, V]{}
	m := make(map[K]V)
	sm.data.Store(&m)
	return sm
}

func (s *SafeMap[K, V]) Set(k K, v V) {
	for {

		m := s.data.Load().(*map[K]V)

		newMap := make(map[K]V, len(*m)+1)
		for key, value := range *m {
			newMap[key] = value
		}

		newMap[k] = v

		if s.data.CompareAndSwap(m, &newMap) {
			break
		}
	}
}

func (s *SafeMap[K, V]) Store(k K, v V) {
	s.Set(k, v)
}

func (s *SafeMap[K, V]) Get(k K) (V, bool) {
	m := s.data.Load().(*map[K]V)
	val, ok := (*m)[k]
	return val, ok
}

func (s *SafeMap[K, V]) Load(k K) (V, bool) {
	return s.Get(k)
}

func (s *SafeMap[K, V]) Delete(k K) {
	for {

		m := s.data.Load().(*map[K]V)

		newMap := make(map[K]V, len(*m))
		for key, value := range *m {
			newMap[key] = value
		}

		delete(newMap, k)

		if s.data.CompareAndSwap(m, &newMap) {
			break
		}
	}
}

func (s *SafeMap[K, V]) Len() int {
	m := s.data.Load().(*map[K]V)
	return len(*m)
}

func (s *SafeMap[K, V]) ForEach(f func(K, V)) {
	m := s.data.Load().(*map[K]V)
	for k, v := range *m {
		f(k, v)
	}
}
