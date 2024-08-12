//go:build !race

package pool

import (
	"unsafe"
	_ "unsafe"
)

const (
	uint64SubtractionConstant = ^uint64(0)
)

func GetG() unsafe.Pointer

//go:linkname goReady runtime.goready
func goReady(goroutinePtr unsafe.Pointer, traceskip int)

//go:linkname mCall runtime.mcall
func mCall(fn func(unsafe.Pointer))

//go:linkname readGStatus runtime.readgstatus
func readGStatus(gp unsafe.Pointer) uint32

//go:linkname casGStatus runtime.casgstatus
func casGStatus(gp unsafe.Pointer, oldval, newval uint32)

//go:linkname dropG runtime.dropg
func dropG()

//go:linkname schedule runtime.schedule
func schedule()

//go:linkname goSchedM runtime.gosched_m
func goSchedM(gp unsafe.Pointer)

func fastPark(gp unsafe.Pointer) {
	dropG()
	casGStatus(gp, gRunning, gWaiting)
	schedule()
}

func safeReady(gp unsafe.Pointer) {
	for readGStatus(gp)&^gScan != gWaiting {
		mCall(goSchedM)
	}
	goReady(gp, 1)
}

//nolint:all
const (
	gIdle = iota
	gRunnable
	gRunning
	gSyscall
	gWaiting
	gMoribund
	gDead
	gEnqueue
	gCopyStack
	gPreempted

	// This G(goroutine)'s status.
	//
	// 	- gIdle: just allocated and has not yet been initialized.
	// 	- gRunnable: in run queue. User code isn't currently executing. The stack isn't owned.
	// 	- gRunning: goroutine may execute user code. The stack is owned by this.
	// 			It isn't on run queue. It is assigned an M and a P (g.m and g.m.p are valid).
	// 	- gSyscall: executing system call. It isn't executing user code. The stack is owned by this.
	// 			It isn't on run queue. It's assigned an M.
	// 	- gWaiting: goroutine is blocked in the runtime. It isn't executing user code.
	// 			It isn't on run queue, but should be recorded somewhere (e.g., a channel wait queue)
	// 			so it can be ready()d when necessary. The stack is not owned *except* that a channel
	// 			operation may read or write parts of the stack under the appropriate channel lock.
	// 			Otherwise, it's not safe to access the stack after a goroutine enters gWaiting
	// 			(e.g., it may get moved).
	// 	- gMoribund: currently unused, but hardcoded in gdb scripts.
	// 	- gDead: currently unused. It may be just exited, on free list, or just being initialized.
	// 			It isn't executing user code. It may or may not have a stack allocated. The G and
	// 			its stack (if any) are owned by the M that is exiting the G or that obtained the
	// 			G from the free list.
	// 	- gEnqueue: currently unused
	// 	- gCopyStack: Its stack is being moved. It isn't executing user code and isn't on run queue.
	// 			The stack is owned by the goroutine that put it in gCopyStack.
	// 	- gPreempted: goroutine stopped itself for a suspendG preemption. It is like gWaiting, but
	// 			nothing is yet responsible for ready()ing it. Some suspendG must CAS the status
	// 			to gWaiting to take responsibility for ready()ing this G.
)

const gScan = 0x1000
