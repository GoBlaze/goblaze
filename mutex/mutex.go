package mutex

import (
	"sync/atomic"
	"unsafe"

	"github.com/GoBlaze/goblaze/chan/constants"
)

type Mutex struct {
	i int32
	_ [constants.CacheLinePadSize - unsafe.Sizeof(int32(0))]byte
}

func (m *Mutex) get() int32 {
	return atomic.LoadInt32(&m.i)
}

func (m *Mutex) set(i int32) {
	atomic.StoreInt32(&m.i, i)
}

func (m *Mutex) Lock() {
	for atomic.CompareAndSwapInt32(&m.i, 0, 1) != true {

	}
	return
}

func (m *Mutex) Unlock() {
	if m.get() == 0 {
		panic("BUG: Unlock of unlocked Mutex")

	}

	m.set(0)
}
