package goblaze

import (
	"sync"

	"github.com/GoBlaze/goblaze/pool"
)

var waitGroupPool = pool.NewPool[*sync.WaitGroup](func() *sync.WaitGroup {
	return new(sync.WaitGroup)
})

func AcquireWaitGroup() *sync.WaitGroup {
	return waitGroupPool.Get()
}

// ReleaseWaitGroup возвращает WaitGroup в пул
func ReleaseWaitGroup(wg *sync.WaitGroup) {
	Reset()
	waitGroupPool.Put(wg)
}

func Reset() *sync.WaitGroup {
	return &sync.WaitGroup{}
}
