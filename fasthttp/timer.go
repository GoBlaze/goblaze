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
		data := zenq.Select(t.C)
		if data == nil {
			panic("BUG: timer must be stopped")
		}

		switch data.(type) {
		case time.Time:

		default:

		}

	}
}

func AcquireTimer(timeout time.Duration) *tick.Timer {
	v := timerPool.Get()
	if v == nil {
		return tick.NewTimer(timeout)
	}
	t := v.(*tick.Timer)
	initTimer(t, timeout)
	return t
}

func ReleaseTimer(t *tick.Timer) {
	stopTimer(t)
	timerPool.Put(t)
}

var timerPool sync.Pool
