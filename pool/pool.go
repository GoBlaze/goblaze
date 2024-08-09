package pool

import (
	"sync"
)

type No struct{}

func (*No) Lock()   {}
func (*No) Unlock() {}

type Pool[T any] struct {
	noCopy No // nolint:structcheck,unused
	items  sync.Pool

	pointers sync.Pool
}

// New creates a new Pool[T] with the given function to create new items.

func New[T any](item func() T) Pool[T] {
	return Pool[T]{
		items: sync.Pool{
			New: func() interface{} {
				val := item()
				return &val
			},
		},
	}
}

// Get returns an item from the pool, creating a new one if necessary.

func (p *Pool[T]) Get() T {
	pooled := p.items.Get()
	if pooled == nil {

		var zero T
		return zero
	}

	ptr := pooled.(*T)
	item := *ptr
	var zero T

	*ptr = zero
	p.pointers.Put(ptr)
	return item
}

// Put adds an item to the pool.
func (p *Pool[T]) Put(item T) {
	var ptr *T
	if pooled := p.pointers.Get(); pooled != nil {
		ptr = pooled.(*T)
	} else {
		ptr = new(T)
	}
	*ptr = item
	p.items.Put(ptr)
}
