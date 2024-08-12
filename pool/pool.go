package pool

import (
	"sync"
)

type No struct{}

func (*No) Lock()   {}
func (*No) Unlock() {}

type Pool[T any] struct {
	_     No
	_     cacheLinePadding //nolint:unused
	pools *sync.Pool
	put   func(*T)
}

func (p *Pool[T]) Get() *T  { return p.pools.Get().(*T) }
func (p *Pool[T]) Put(t *T) { p.put(t); p.pools.Put(t) }

func NewPool[T any](get func() *T, put func(*T)) *Pool[T] {
	if get == nil {
		get = func() *T { return new(T) }
	}
	if put == nil {
		put = func(t *T) {}
	}

	return &Pool[T]{
		pools: &sync.Pool{New: func() any { return get() }},
		put:   put,
	}
}
