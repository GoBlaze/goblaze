package pool

import (
	"sync"
	"unsafe"
)

type No struct{}

func (*No) Lock()   {}
func (*No) Unlock() {}

// Pool represents a pool of objects with type T.
type Pool[T any] struct {
	_ No // nolint:structcheck,unused

	items *sync.Pool
	_     [cacheLinePadSize - unsafe.Sizeof(&sync.Pool{})%cacheLinePadSize]byte
}

// NewPool creates a new Pool[T] with a function that creates new objects.
func NewPool[T any](newFunc func() T) *Pool[T] {
	p := &Pool[T]{
		items: &sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
	}
	return p
}

// Get returns an object from the pool, creating a new one if necessary.
func (p *Pool[T]) Get() T {
	return p.items.Get().(T)
}

// Put adds an object to the pool.
func (p *Pool[T]) Put(x T) {
	p.items.Put(x)
}
