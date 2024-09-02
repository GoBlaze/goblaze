package goblaze

import (
	"sync"

	"github.com/GoBlaze/goblaze/pool"
)

var waitGroupPool = pool.NewPool[sync.WaitGroup](func() *sync.WaitGroup {
	return &sync.WaitGroup{}
}, nil)

func Get() *sync.WaitGroup {
	return waitGroupPool.Get()
}

func Put(wg *sync.WaitGroup) {
	Reset()
	waitGroupPool.Put(wg)

}

func Reset() *sync.WaitGroup {
	return &sync.WaitGroup{}
}
