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
type funcs struct {
	WorkerFunc ServeHandler
	Logger     Logger
	connState  func(net.Conn, ConnState)
}

type workerPool struct {
	_              noCopy
	workerChanPool sync.Pool
	_              [cacheLinePadSize - unsafe.Sizeof(sync.Pool{})]byte //nolint:unused

	funcs *funcs

	_ [cacheLinePadSize - unsafe.Sizeof(funcs{})]byte //nolint:unused

	stopCh chan struct{}

	ready []*workerChan
	_     [cacheLinePadSize - unsafe.Sizeof([]*workerChan{})]byte //nolint:unused

	MaxWorkersCount int32
	_               [cacheLinePadSize - unsafe.Sizeof(int32(0))]byte //nolint:unused

	MaxIdleWorkerDuration time.Duration

	workersCount int32
	_            [cacheLinePadSize - unsafe.Sizeof(int32(0))]byte //nolint:unused

	LogAllErrors bool
	_            [cacheLinePadSize - unsafe.Sizeof(bool(false))]byte //nolint:unused
	mustStop     atomic.Bool
	_            [cacheLinePadSize - unsafe.Sizeof(atomic.Bool{})]byte //nolint:unused
}

type workerChan struct {
	lastUseTime time.Time
	ch          *zenq.ZenQ[net.Conn]
}

func (wp *workerPool) Start() {
	if wp.stopCh != nil {
		return
	}
	wp.stopCh = make(chan struct{})
	stopCh := wp.stopCh
	wp.workerChanPool.New = func() any {
		return &workerChan{
			ch: zenq.New[net.Conn](uint32(workerChanCap)),
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

	ready := wp.ready
	for i := range ready {
		// Close the ZenQ channel
		ready[i].ch.Close()
		ready[i] = nil
	}
	wp.ready = ready[:0]
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

	ready := wp.ready
	n := len(ready)

	l, r := 0, n-1
	for l <= r {
		mid := (l + r) / 2
		if ready[mid].lastUseTime.Before(criticalTime) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	if r < 0 {

		return
	}

	*scratch = append((*scratch)[:0], ready[:r+1]...)
	wp.ready = ready[r+1:]

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

var workerChanCap = func() int {
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

	ready := wp.ready
	n := len(ready) - 1
	if n < 0 {
		if atomic.LoadInt32(&wp.workersCount) < wp.MaxWorkersCount {
			createWorker.Store(true)
			atomic.AddInt32(&wp.workersCount, 1)
		}
	} else {
		ch = ready[n]
		ready[n] = nil
		wp.ready = ready[:n]
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
	ch.lastUseTime = time.Now()

	if wp.mustStop.Load() {

		return false
	}
	wp.ready = append(wp.ready, ch)

	return true
}

func (wp *workerPool) workerFunc(ch *workerChan) {
	for {

		conn, queueOpen := ch.ch.Read()
		if !queueOpen {
			// If the queue is closed, break the loop
			break
		}

		var err error
		if err = wp.funcs.WorkerFunc(conn); err != nil && err != errHijacked {
			errStr := err.Error()
			if wp.LogAllErrors || !(strings.Contains(errStr, "broken pipe") ||
				strings.Contains(errStr, "reset by peer") ||
				strings.Contains(errStr, "request headers: small read buffer") ||
				strings.Contains(errStr, "unexpected EOF") ||
				strings.Contains(errStr, "i/o timeout") ||
				errors.Is(err, ErrBadTrailer)) {
				wp.funcs.Logger.Printf("error when serving connection %q<->%q: %v", conn.LocalAddr(), conn.RemoteAddr(), err)
			}
		}
		if err == errHijacked {
			wp.funcs.connState(conn, StateHijacked)
		} else {
			_ = conn.Close()
			wp.funcs.connState(conn, StateClosed)
		}

		if !wp.release(ch) {
			break
		}
	}

	atomic.AddInt32(&wp.workersCount, -1)
}
