package fasthttp

import (
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	zenq "github.com/GoBlaze/goblaze/chan"
)

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

	ch          *zenq.ZenQ[net.Conn]
	_           [cacheLinePadSize - 8]byte
	lastUseTime int64
	_           [cacheLinePadSize - unsafe.Sizeof(int64(0))]byte
}

type workerChanStack struct {
	head, tail *workerChan
	_          [cacheLinePadSize - (2 * unsafe.Sizeof(uintptr(0)))]byte //nolint:unused
}

// push adds a workerChan to the top of the stack
func (s *workerChanStack) push(ch *workerChan) {
	ch.next = s.head
	s.head = ch
	if s.tail == nil {
		s.tail = ch
	}
}

func (s *workerChanStack) pop() *workerChan {
	head := s.head
	if head == nil {
		return nil
	}
	s.head = head.next
	if s.head == nil {
		s.tail = nil
	}
	return head
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
		ch.ch.Close()
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

	current := wp.ready.head
	for current != nil {
		next := current.next
		if current.lastUseTime < criticalTime {
			current.ch.Close()
			wp.workerChanPool.Put(current)
		} else {
			wp.ready.head = current
			break
		}
		current = next
	}
	wp.ready.tail = wp.ready.head
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
	if ch == nil && atomic.LoadInt32(&wp.workersCount) < wp.MaxWorkersCount {
		atomic.AddInt32(&wp.workersCount, 1)
		createWorker.Store(true)
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
			if wp.LogAllErrors || !(Contains(StringToBytes(errStr), StringToBytes("broken pipe")) ||
				Contains(StringToBytes(errStr), StringToBytes("reset by peer")) ||
				Contains(StringToBytes(errStr), StringToBytes("request headers: small read buffer")) ||
				Contains(StringToBytes(errStr), StringToBytes("unexpected EOF")) ||
				Contains(StringToBytes(errStr), StringToBytes("i/o timeout")) ||
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
