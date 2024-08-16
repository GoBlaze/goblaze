package mutex

import (
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/GoBlaze/goblaze/chan/constants"
)

type MutexExp struct {
	i int32
	_ [constants.CacheLinePadSize - unsafe.Sizeof(int32(0))]byte
}

func (m *MutexExp) get() int32 {
	return atomic.LoadInt32(&m.i)
}

func (m *MutexExp) set(i int32) {
	atomic.StoreInt32(&m.i, i)
}

func (m *MutexExp) Lock() {
	for !atomic.CompareAndSwapInt32(&m.i, 0, 1) {

		runtime.Gosched()

	}
	return
}

func (m *MutexExp) Unlock() {
	if m.get() == 0 {
		panic("BUG: Unlock of unlocked Mutex")

	}

	m.set(0)
}
