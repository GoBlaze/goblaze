//go:build !race

package pool

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type GoPool[T any] struct {
	currSize uint64
	_        [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte //nolint:unused
	maxSize  uint64
	alloc    func() any
	free     func(any)
	task     func(T)
	_        [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte //nolint:unused
	top      unsafe.Pointer
	_        [cacheLinePadSize - unsafe.Sizeof(atomic.Pointer[dataItem[T]]{})]byte //nolint:unused
}

func NewGoPool[T any](size uint64, task func(T)) *GoPool[T] {
	dataPool := sync.Pool{New: func() any { return new(dataItem[T]) }}
	return &GoPool[T]{maxSize: size, task: task, alloc: dataPool.Get, free: dataPool.Put}
}

func (gp *GoPool[T]) Invoke(value T) {
	var s *slotFunc[T]
	for {
		if s = gp.pop(); s != nil {
			s.data = value
			safeReady(s.threadPtr)
			return
		} else if atomic.AddUint64(&gp.currSize, 1) <= gp.maxSize {
			s = &slotFunc[T]{data: value}
			go gp.loopQ(s)
			return
		} else {
			atomic.AddUint64(&gp.currSize, uint64SubtractionConstant)
			mCall(goSchedM)
		}
	}
}

// represents the infinite loop for a worker goroutine
func (gp *GoPool[T]) loopQ(d *slotFunc[T]) {
	d.threadPtr = GetG()
	for {
		gp.task(d.data)
		gp.push(d)
		mCall(fastPark)
	}
}

// pop value from the top of the stack
func (gp *GoPool[T]) pop() (value *slotFunc[T]) {
	var top, next unsafe.Pointer
	for {
		top = atomic.LoadPointer(&gp.top)
		if top == nil {
			return
		}
		next = atomic.LoadPointer(&(*dataItem[T])(top).next)
		if atomic.CompareAndSwapPointer(&gp.top, top, next) {
			value = (*dataItem[T])(top).value
			(*dataItem[T])(top).next, (*dataItem[T])(top).value = nil, nil
			gp.free((*dataItem[T])(top))
			return
		}
	}
}

// push a value on top of the stack
func (gp *GoPool[T]) push(v *slotFunc[T]) {
	var (
		top  unsafe.Pointer
		item = gp.alloc().(*dataItem[T])
	)
	item.value = v
	for {
		top = atomic.LoadPointer(&gp.top)
		item.next = top
		if atomic.CompareAndSwapPointer(&gp.top, top, unsafe.Pointer(item)) {
			return
		}
	}
}

// a single node in the stack
type dataItem[T any] struct {
	next  unsafe.Pointer
	value *slotFunc[T]
}

// a single slot for a worker in GoPool
type slotFunc[T any] struct {
	threadPtr unsafe.Pointer
	data      T
}
