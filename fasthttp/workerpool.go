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
	_              noCopy
	workerChanPool sync.Pool

	WorkerFunc ServeHandler
	Logger     Logger
	connState  func(net.Conn, ConnState)

	stopCh chan struct{}

	ready workerChanStack

	MaxWorkersCount int32

	MaxIdleWorkerDuration time.Duration

	workersCount int32

	LogAllErrors bool

	mustStop atomic.Bool
}

type workerChan struct {
	lastUseTime time.Time
	ch          *zenq.ZenQ[net.Conn]
	next        *workerChan
}

type workerChanStack struct {
	top     unsafe.Pointer
	counter int32
}

// push adds a workerChan to the top of the stack
func (s *workerChanStack) push(ch *workerChan) {
	var (
		top  unsafe.Pointer
		item = ch
	)
	for {
		top = atomic.LoadPointer(&s.top)
		item.next = (*workerChan)(top) // Convert unsafe.Pointer to *workerChan
		if atomic.CompareAndSwapPointer(&s.top, top, unsafe.Pointer(item)) {
			atomic.AddInt32(&s.counter, 1)
			return
		}
	}
}

func (s *workerChanStack) pop() *workerChan {
	var top, next unsafe.Pointer
	for {
		top = atomic.LoadPointer(&s.top)
		if top == nil {
			return nil
		}
		next = unsafe.Pointer((*workerChan)(top).next)
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			atomic.AddInt32(&s.counter, -1)
			return (*workerChan)(top)
		}
	}
}

func (wp *workerPool) Start() {
	if wp.stopCh != nil {
		return
	}
	wp.stopCh = make(chan struct{})
	stopCh := wp.stopCh
	wp.workerChanPool.New = func() any {
		return &workerChan{
			ch: zenq.New[net.Conn](workerChanCap),
		}
	}
	go func() {
		var scratch []*workerChan
		for {
			wp.clean(&scratch)
			select {
			case <-stopCh:
				return
			default:
				time.Sleep(wp.getMaxIdleWorkerDuration())
			}
		}
	}()
}

func (wp *workerPool) Stop() {
	if wp.stopCh == nil {
		return
	}
	close(wp.stopCh)
	wp.stopCh = nil

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

func (wp *workerPool) clean(scratch *[]*workerChan) {
	maxIdleWorkerDuration := wp.getMaxIdleWorkerDuration()
	criticalTime := time.Now().Add(-maxIdleWorkerDuration)

	for {
		ch := wp.ready.pop()
		if ch == nil {
			break
		}

		if ch.lastUseTime.Before(criticalTime) {
			ch.ch.Close()
			wp.workerChanPool.Put(ch)
		} else {
			wp.ready.push(ch)
			break
		}
	}
	for _, ch := range *scratch {
		ch.ch.Close()
		wp.workerChanPool.Put(ch)
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
	// Use blocking workerChan if GOMAXPROCS=1.
	// This immediately switches Serve to WorkerFunc, which results
	// in higher performance (under go1.5 at least).
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}

	// Use non-blocking workerChan if GOMAXPROCS>1,
	// since otherwise the Serve caller (Acceptor) may lag accepting
	// new connections if WorkerFunc is CPU-bound.
	return 1
}()

func (wp *workerPool) getCh() *workerChan {
	var ch *workerChan
	var createWorker atomic.Bool

	ch = wp.ready.pop()
	if ch == nil {
		if atomic.LoadInt32(&wp.workersCount) < wp.MaxWorkersCount {
			createWorker.Store(true)
			atomic.AddInt32(&wp.workersCount, 1)
		}
	}

	if ch == nil {
		vch := wp.workerChanPool.Get()
		ch = vch.(*workerChan)
		go func() {
			_Ptr := GetG() // Capture the G pointer for the worker goroutine
			wp.workerFunc(ch, _Ptr)
			wp.workerChanPool.Put(vch)
		}()
	}
	return ch
}

//go:noescape
func GetG() unsafe.Pointer

func (wp *workerPool) release(ch *workerChan) bool {
	ch.lastUseTime = time.Now()

	if wp.mustStop.Load() {

		return false
	}
	wp.ready.push(ch)

	return true
}

func (wp *workerPool) workerFunc(ch *workerChan, _Ptr unsafe.Pointer) {
	for {
		conn, queueOpen := ch.ch.Read()
		if !queueOpen {
			break
		}

		var err error
		if err = wp.WorkerFunc(conn); err != nil && err != errHijacked {
			errStr := err.Error()
			if wp.LogAllErrors || !(strings.Contains(errStr, "broken pipe") ||
				strings.Contains(errStr, "reset by peer") ||
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

		// If no more tasks are available, park the goroutine
		if !wp.release(ch) {
			mCall(func(_ unsafe.Pointer) {
				fastPark(_Ptr)
			})
			break
		}
	}

	atomic.AddInt32(&wp.workersCount, -1)
}
