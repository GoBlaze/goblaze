package pool

import (
	"sync"
	"sync/atomic"
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
	val := p.internalPool.Get()
	if val == nil {
		var zeroVal T
		return zeroVal
	}

	atomic.AddInt64(&p.count, -1)
	return val.(T)
}

func (p *pool[T]) Put(value T) {
	p.internalPool.Put(value)
	atomic.AddInt64(&p.count, 1)

	if atomic.CompareAndSwapUint32(&p.calibrating, 0, 1) {
		go p.calibrate()
	}
}

func (p *pool[T]) Count() int64 {
	return atomic.LoadInt64(&p.count)
}

func (p *pool[T]) calibrate() {
	var callsSum uint64
	var maxSize uint64
	var defaultSize uint64 = minSize

	var maxSum uint64 = uint64(float64(calibrateCallsThreshold) * maxPercentile)

	for i := 0; i < steps; i++ {
		calls := atomic.SwapUint64(&p.calls[i], 0)
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
