package fasthttp

import (
	"errors"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	zenq "github.com/GoBlaze/goblaze/chan"
)

// workerPool serves incoming connections via a pool of workers
// in FILO order, i.e. the most recently stopped worker will serve the next
// incoming connection.
//
// Such a scheme keeps CPU caches hot (in theory).
type workerPool struct {
	// _ cacheLinePadding //nolint:unused

	ready workerChanStack
	_     [cacheLinePadSize - unsafe.Sizeof(workerChanStack{})]byte //nolint:unused

	workerChanPool sync.Pool
	_              [cacheLinePadSize - unsafe.Sizeof(sync.Pool{})]byte //nolint:unused

	Logger     Logger
	_          [cacheLinePadSize - unsafe.Sizeof(Logger(nil))]byte //nolint:unused
	WorkerFunc ServeHandler
	_          [cacheLinePadSize - unsafe.Sizeof(ServeHandler(nil))]byte //nolint:structcheck,unused
	connState  func(net.Conn, ConnState)
	_          [cacheLinePadSize - 8]byte //nolint:unused

	MaxIdleWorkerDuration time.Duration
	_                     [cacheLinePadSize - unsafe.Sizeof(time.Duration(0))]byte //nolint:unused
	MaxWorkersCount       int32
	_                     [cacheLinePadSize - unsafe.Sizeof(int32(0))]byte //nolint:structcheck,unused
	workersCount          int32
	_                     [cacheLinePadSize - unsafe.Sizeof(int32(0))]byte //nolint:unused

	stopSignal atomic.Bool                                           // Use atomic.Bool for stop signal
	_          [cacheLinePadSize - unsafe.Sizeof(atomic.Bool{})]byte //nolint:unused
	mustStop   atomic.Bool
	_          [cacheLinePadSize - unsafe.Sizeof(atomic.Bool{})]byte //nolint:unused

	LogAllErrors bool
	_            [cacheLinePadSize - 1]byte //nolint:unused

}

type workerChan struct {
	_    cacheLinePadding
	next *workerChan
	_    [cacheLinePadSize - 8]byte

	ch *zenq.ZenQ[net.Conn]
	_  [cacheLinePadSize - 8]byte

	lastUseTime int64
	_           [cacheLinePadSize - unsafe.Sizeof(int64(0))]byte
}

type workerChanStack struct {
	head, tail atomic.Pointer[workerChan]
	_          [cacheLinePadSize - (2 * unsafe.Sizeof(uintptr(0)))]byte //nolint:unused
}

func (s *workerChanStack) push(ch *workerChan) {
	for {
		oldHead := s.head.Load()
		ch.next = oldHead
		if s.head.CompareAndSwap(oldHead, ch) {
			break
		}
	}

	if s.tail.Load() == nil {
		s.tail.Store(ch)
	}
}
func (s *workerChanStack) pop() *workerChan {
	for {
		oldHead := s.head.Load()
		if oldHead == nil {
			return nil
		}

		if s.head.CompareAndSwap(oldHead, oldHead.next) {
			if s.head.Load() == nil {
				s.tail.Store(nil)
			}
			return oldHead
		}
	}
}

func (wp *workerPool) Start() {
	if wp.stopSignal.Load() {
		return
	}

	wp.workerChanPool.New = func() any {
		return &workerChan{
			ch: zenq.New[net.Conn](workerChanCap),
		}
	}

	go func() {
		for {
			wp.clean()
			if wp.stopSignal.Load() {
				return
			}
			time.Sleep(wp.getMaxIdleWorkerDuration())
		}
	}()
}

func (wp *workerPool) Stop() {
	if wp.mustStop.Load() {
		return
	}
	wp.stopSignal.Store(true)

	// Close all worker channels in the ready stack
	for {
		ch := wp.ready.pop()
		if ch == nil {
			break
		}
		ch.ch.Write(nil)
	}

	wp.mustStop.Store(true)
}
func (wp *workerPool) getMaxIdleWorkerDuration() time.Duration {
	if wp.MaxIdleWorkerDuration <= 0 {
		return 10 * time.Second
	}
	return wp.MaxIdleWorkerDuration
}
func (wp *workerPool) clean() {
	maxIdleWorkerDuration := wp.getMaxIdleWorkerDuration()
	criticalTime := time.Now().Add(-maxIdleWorkerDuration).UnixNano()

	for {
		current := wp.ready.head.Load()
		if current == nil || current.lastUseTime >= criticalTime {
			break
		}

		next := current.next
		if wp.ready.head.CompareAndSwap(current, next) {
			current.ch.Write(nil)
			wp.workerChanPool.Put(current)
		}
	}

	if wp.ready.head.Load() == nil {
		wp.ready.tail.Store(nil)
	}
}
func (wp *workerPool) Serve(c net.Conn) bool {
	ch := wp.getCh()
	if ch == nil {
		return false
	}
	ch.ch.Write(c)

	return true
}

var workerChanCap = func() uint32 {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}
	return 1
}()

func (wp *workerPool) getCh() *workerChan {
	var ch *workerChan
	var createWorker atomic.Bool

	ch = wp.ready.pop()
	if ch == nil {
		for {
			currentworkers := atomic.LoadInt32(&wp.workersCount)
			if currentworkers >= wp.MaxWorkersCount {
				break
			}
			if atomic.CompareAndSwapInt32(&wp.workersCount, currentworkers, currentworkers+1) {
				createWorker.Store(true)
				break
			}
		}
	}
	if ch == nil {
		vch := wp.workerChanPool.Get()
		ch = vch.(*workerChan)
		go func() {
			wp.workerFunc(ch)
			wp.workerChanPool.Put(vch)
		}()
	}
	return ch
}

func (wp *workerPool) release(ch *workerChan) bool {
	ch.lastUseTime = time.Now().UnixNano()
	if wp.mustStop.Load() {
		return false
	}

	wp.ready.push(ch)
	return true
}

func (wp *workerPool) workerFunc(ch *workerChan) {
	for {

		conn, _ := ch.ch.Read()

		if conn == nil {
			panic("workerFunc: data == nil")
		}

		var err error
		if err = wp.WorkerFunc(conn); err != nil && err != errHijacked {
			errStr := err.Error()
			if wp.LogAllErrors || !(strings.Contains(errStr, "broken pipe") ||
				strings.Contains(errStr, "reset by peer") ||
				strings.Contains(errStr, "request headers: small read buffer") ||
				strings.Contains(errStr, "unexpected EOF") ||
				strings.Contains(errStr, "i/o timeout") ||
				errors.Is(err, ErrBadTrailer)) {
				wp.Logger.Printf("error when serving connection %q<->%q: %v", conn.LocalAddr(), conn.RemoteAddr(), err)
			}
		}
		if err == errHijacked {
			wp.connState(conn, StateHijacked)
		} else {
			_ = conn.Close()
			wp.connState(conn, StateClosed)
		}

		if !wp.release(ch) {
			break
		}
	}

	atomic.AddInt32(&wp.workersCount, -1)

}
