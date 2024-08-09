package pool

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	minBitSize              = 6
	steps                   = 20
	maxSize                 = 1 << 20
	minSize                 = 1 << minBitSize
	calibrateCallsThreshold = 100000
	maxPercentile           = 0.95
)

type Pool[T any] interface {
	Get() T
	Put(T)
	Count() int64
}

type pool[T any] struct {
	internalPool *sync.Pool
	count        int64
	calibrating  uint32
	calls        [steps]uint64
	defaultSize  uint64
	maxSize      uint64
	_            [64]byte
}

func NewPool[T any](constructor func() T) Pool[T] {
	return &pool[T]{
		internalPool: &sync.Pool{

			New: func() any {
				return constructor()
			},
		},
		defaultSize: minSize,
		maxSize:     maxSize,
	}
}

func (p *pool[T]) Get() T {
	return p.internalPool.Get().(T)
}

func (p *pool[T]) Put(value T) {
	p.internalPool.Put(value)
}

func (p *pool[T]) Count() int64 {
	return atomic.LoadInt64(&p.count)
}

func (p *pool[T]) calibrate() {
	callsSum, maxSize, defaultSize := uint64(0), uint64(minSize), uint64(minSize)
	maxSum := uint64(float64(calibrateCallsThreshold) * maxPercentile)

	ptr := uintptr(unsafe.Pointer(&p.calls[0]))
	stepSize := unsafe.Sizeof(p.calls[0])

	for i := 0; i < steps; i++ {
		calls := *(*uint64)(unsafe.Pointer(ptr))
		ptr += stepSize

		if calls > 0 {
			size := uint64(minSize << i)
			callsSum += calls
			if size > maxSize {
				maxSize = size
			}
			if callsSum > maxSum {
				break
			}
			if size < defaultSize {
				defaultSize = size
			}
		}
	}

	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)
	atomic.StoreUint32(&p.calibrating, 0)
}

func index(n int) int {
	n--
	n >>= minBitSize
	idx := 0
	for n > 0 {
		n >>= 1
		idx++
	}
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}
