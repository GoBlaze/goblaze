package stackless

import (
	"runtime"
	"sync"

	zenq "github.com/GoBlaze/goblaze/chan"
)

// NewFunc returns stackless wrapper for the function f.
//
// Unlike f, the returned stackless wrapper doesn't use stack space
// on the goroutine that calls it.
// The wrapper may save a lot of stack space if the following conditions
// are met:
//
//   - f doesn't contain blocking calls on network, I/O or channels;
//   - f uses a lot of stack space;
//   - the wrapper is called from high number of concurrent goroutines.
//
// The stackless wrapper returns false if the call cannot be processed
// at the moment due to high load.

func NewFunc(f func(ctx any)) func(ctx any) bool {
	if f == nil {
		panic("BUG: f cannot be nil")
	}

	funcWorkCh := zenq.New[*funcWork](uint32(runtime.GOMAXPROCS(-1) * 2048))
	onceInit := func() {
		n := runtime.GOMAXPROCS(-1)
		for i := 0; i < n; i++ {
			go funcWorker(funcWorkCh, f)
		}
	}
	var once sync.Once

	return func(ctx any) bool {
		if ctx == nil {
			panic("BUG: ctx cannot be nil")
		}

		once.Do(onceInit)
		fw := getFuncWork()
		fw.ctx = ctx

		if funcWorkCh.Write(fw) {
			putFuncWork(fw)
			return false
		}

		fw.done.Read()

		putFuncWork(fw)
		return true
	}
}

func funcWorker(funcWorkCh *zenq.ZenQ[*funcWork], f func(ctx any)) {

	for {
		if fw, open := funcWorkCh.Read(); open {

			f(fw.ctx)
			fw.done.Write(struct{}{})
		}
	}
}

func getFuncWork() *funcWork {
	v := funcWorkPool.Get()
	if v == nil {
		v = &funcWork{
			done: zenq.New[struct{}](0),
		}
	}
	return v.(*funcWork)
}

func putFuncWork(fw *funcWork) {

	fw.ctx = nil
	funcWorkPool.Put(fw)
}

var funcWorkPool sync.Pool

type funcWork struct {
	ctx  any
	done *zenq.ZenQ[struct{}]
}
