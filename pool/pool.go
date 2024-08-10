package pool

import (
	"sync"
)

type No struct{}

func (*No) Lock()   {}
func (*No) Unlock() {}

type Pool[T any] struct {
	noCopy No // nolint:structcheck,unused

	items *sync.Pool
	_     cacheLinePadding
}

// New creates a new Pool[T] with the given function to create new items.

func NewPool[T any](item func() T) Pool[T] {
	return Pool[T]{
		items: &sync.Pool{
			New: func() interface{} {
				return item()
			},
		},
	}
}

// Get returns an item from the pool, creating a new one if necessary.

func (p *Pool[T]) Get() T {
	return p.items.Get().(T)
}

// Put adds an item to the pool.
func (p *Pool[T]) Put(item T) {
	p.items.Put(item)
}
