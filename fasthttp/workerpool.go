package fasthttp

import (
	"errors"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	zenq "github.com/GoBlaze/goblaze/chan"
)

// workerPool serves incoming connections via a pool of workers
// in FILO order, i.e. the most recently stopped worker will serve the next
// incoming connection.
//
// Such a scheme keeps CPU caches hot (in theory).

type workerPool struct {
	_ noCopy

	workerChanPool sync.Pool

	WorkerFunc ServeHandler
	Logger     Logger
	connState  func(net.Conn, ConnState)

	stopQueue *zenq.ZenQ[struct{}]

	ready workerChanStack

	MaxWorkersCount int32

	MaxIdleWorkerDuration time.Duration

	workersCount int32

	LogAllErrors bool

	mustStop atomic.Bool
}

type workerChan struct {
	lastUseTime int64
	ch          *zenq.ZenQ[net.Conn]
	next        *workerChan
}

type workerChanStack struct {
	head, tail *workerChan
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
	if s.head == nil {
		return nil
	}

	ch := s.head
	s.head = ch.next
	if s.head == nil {
		s.tail = nil
	}
	return ch
}

func (wp *workerPool) Start() {
	if wp.stopQueue != nil {
		return
	}

	wp.stopQueue = zenq.New[struct{}](0) // Create a ZenQ for stop signals
	wp.workerChanPool.New = func() any {
		return &workerChan{
			ch: zenq.New[net.Conn](workerChanCap),
		}
	}

	go func() {
		for {
			wp.clean()
			if wp.stopQueue.IsClosed() {
				return
			}
			time.Sleep(wp.getMaxIdleWorkerDuration())
		}
	}()
}

func (wp *workerPool) Stop() {
	if wp.stopQueue == nil {
		return
	}

	wp.stopQueue.Close() // Notify all waiting workers to stop

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
	prev := wp.ready.head

	for current != nil {
		if current.lastUseTime < criticalTime {
			current.ch.Close()
			wp.workerChanPool.Put(current)
			prev.next = current.next
			if current == wp.ready.tail {
				wp.ready.tail = prev
			}
			current = prev.next
		} else {
			prev = current
			current = current.next
		}
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
		conn, queueOpen := ch.ch.Read()
		if !queueOpen {
			// If the queue is closed, break the loop
			break
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
