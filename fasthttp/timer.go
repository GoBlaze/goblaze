package fasthttp

import (
	"sync"
	"time"

	zenq "github.com/GoBlaze/goblaze/chan"
	"github.com/GoBlaze/goblaze/tick"
)

func initTimer(t *tick.Timer, timeout time.Duration) *tick.Timer {
	if t == nil {
		return tick.NewTimer(timeout)
	}
	if t.Reset(timeout) {
		// developer sanity-check
		panic("BUG: active timer trapped into initTimer()")
	}
	return t
}

func stopTimer(t *tick.Timer) {
	if !t.Stop() {
		// Collect possibly added time from the channel
		// if timer has been stopped and nobody collected its value.
		if data := zenq.Select(t.C); data != nil {
			switch data.(type) {
			case time.Time:
				t.C.Read()

			}

		}
	}
}

// AcquireTimer returns a time.Timer from the pool and updates it to
// send the current time on its channel after at least timeout.
//
// The returned Timer may be returned to the pool with ReleaseTimer
// when no longer needed. This allows reducing GC load.
func AcquireTimer(timeout time.Duration) *tick.Timer {
	v := timerPool.Get()
	if v == nil {
		return tick.NewTimer(timeout)
	}
	t := v.(*tick.Timer)
	initTimer(t, timeout)
	return t
}

// ReleaseTimer returns the time.Timer acquired via AcquireTimer to the pool
// and prevents the Timer from firing.
//
// Do not access the released time.Timer or read from its channel otherwise
// data races may occur.
func ReleaseTimer(t *tick.Timer) {
	stopTimer(t)
	timerPool.Put(t)
}

var timerPool sync.Pool
