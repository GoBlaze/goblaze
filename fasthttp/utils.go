package fasthttp

import (
	"fmt"
	"reflect"
	"unicode/utf8"
	"unsafe"
)

type ErrorSizeUnmatch struct {
	fromLength int
	fromSize   int64

	toSize int64
}

func (err *ErrorSizeUnmatch) Error() string {
	return fmt.Sprintf(
		"size mismatch: source length = '%d',"+
			"source size = '%d', destination size = '%d'",
		err.fromLength, err.fromSize, err.toSize)
}

// validatePath checks if the path starts with a '/' and panics if not.
// It also returns the path.
func validatePath(path string) string {
	if len(path) == 0 || path[0] != '/' {
		panic("path must begin with '/'")
	}
	return path
}

//go:noinline
func String(b []byte) string {

	return unsafe.String(unsafe.SliceData(b), len(b))
}

func fastPark(gp unsafe.Pointer) {
	dropG()
	casGStatus(gp, gRunning, gWaiting)
	schedule()
}

//go:linkname schedule runtime.schedule
func schedule()

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

func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func CopyBytes(b []byte) []byte {
	return unsafe.Slice(unsafe.StringData(String(b)), len(b))
}

func Copy(b []byte, b1 []byte) ([]byte, []byte) {
	return []byte(String(b)), []byte(String(b1))
}

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

// go:nosplit is a compiler directive that tells the Go compiler to
// prevent the function from being split into multiple machine
// instructions. This is useful for code that needs to be fast and
// efficient.
//
// In this case, the function is marked with go:nosplit so that it
// won't be split into multiple machine instructions, which can
// improve performance.
//
// Note: go:nosplit should be used with caution, as it can sometimes
// lead to code that is slower than code that is split into multiple
// machine instructions.
//
// For more information, see: https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives
//
//go:noinline
func CopyString(s string) string {
	c := make([]byte, len(s))
	copy(c, StringToBytes(s))
	return String(c)
}

//go:noinline
func ConvertSlice[TFrom, TTo any](from []TFrom) ([]TTo, error) {
	var (
		zeroValFrom TFrom
		zeroValTo   TTo
	)

	maxSize := unsafe.Sizeof(zeroValFrom)
	minSize := unsafe.Sizeof(zeroValTo)

	if minSize > maxSize {
		swap(&minSize, &maxSize)
	}

	if unsafe.Sizeof(zeroValFrom) == minSize {
		if len(from)*int(minSize)%int(maxSize) != 0 {
			return nil, &ErrorSizeUnmatch{
				fromLength: len(from),
				fromSize:   int64(unsafe.Sizeof(zeroValFrom)),
				toSize:     int64(unsafe.Sizeof(zeroValTo)),
			}
		}

		header := *(*reflect.SliceHeader)(unsafe.Pointer(&from))
		header.Len = header.Len * int(minSize) / int(maxSize)
		header.Cap = header.Cap * int(minSize) / int(maxSize)
		result := *(*[]TTo)(unsafe.Pointer(&header))

		return result, nil
	} else {
		if len(from)*int(maxSize)%int(minSize) != 0 {
			return nil, &ErrorSizeUnmatch{
				fromLength: len(from),
				fromSize:   int64(unsafe.Sizeof(zeroValFrom)),
				toSize:     int64(unsafe.Sizeof(zeroValTo)),
			}
		}

		header := *(*reflect.SliceHeader)(unsafe.Pointer(&from))
		header.Len = header.Len * int(maxSize) / int(minSize)
		header.Cap = header.Cap * int(maxSize) / int(minSize)
		result := *(*[]TTo)(unsafe.Pointer(&header))

		return result, nil
	}
}

//go:noinline
func swap[T any](a, b *T) {
	tmp := *a
	*a = *b
	*b = tmp
}

//go:nosplit
//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ unsafe.Pointer, needzero bool) unsafe.Pointer

//go:linkname sysFree runtime.sysFree
func sysFree(v unsafe.Pointer, n uintptr, sysStat unsafe.Pointer)

//go:linkname sysFreeOS runtime.sysFreeOS
func sysFreeOS(v unsafe.Pointer, n uintptr)

// inline is a compiler hint that tells the compiler to inline the function.
// This can result in faster execution, but it can also increase the size of the executable.
// The compiler is free to ignore this hint, so it should not be relied upon.
//

//go:noinline
func MakeNoZero(l int) []byte {
	return unsafe.Slice((*byte)(mallocgc(uintptr(l), nil, false)), l)
}

//go:noinline
func MakeNoZeroCap(l int, c int) []byte {
	return MakeNoZero(c)[:l]
}

type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

//go:inline
func SliceUnsafePointer[T any](slice []T) unsafe.Pointer {
	header := *(*sliceHeader)(unsafe.Pointer(&slice))
	return header.Data
}

type StringBuffer struct {
	_    noCopy // nolint:structcheck
	buf  []byte
	addr *StringBuffer
}

func NewStringBuffer(cap int) *StringBuffer {
	return &StringBuffer{
		buf: MakeNoZeroCap(0, cap),
	}
}

func (b *StringBuffer) String() string {
	return String(b.buf)
}

func (b *StringBuffer) Bytes() []byte {
	return b.buf
}

func (b *StringBuffer) Len() int {
	return len(b.buf)
}

func (b *StringBuffer) Cap() int {
	return cap(b.buf)
}

func (b *StringBuffer) Reset() {
	b.buf = b.buf[:0] // reuse the underlying storage
}

//go:inline
func (b *StringBuffer) grow(n int) {
	buf := MakeNoZero(2*cap(b.buf) + n)[:len(b.buf)]
	copy(buf, b.buf)
	b.buf = buf
}

func (b *StringBuffer) Grow(n int) {
	// Check if n is negative
	if n < 0 {
		// Panic with the message "fast.StringBuffer.Grow: negative count"
		panic("fast.StringBuffer.Grow: negative count")
	}

	// Check if the buffer's available capacity is less than n
	if cap(b.buf)-len(b.buf) < n {
		// Call the grow method to increase the capacity
		b.grow(n)
	}
}

func (b *StringBuffer) Write(p []byte) (int, error) {
	b.copyCheck()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *StringBuffer) WriteByte(c byte) error {
	b.copyCheck()
	b.buf = append(b.buf, c)
	return nil
}

func (b *StringBuffer) WriteRune(r rune) (int, error) {
	b.copyCheck()
	n := len(b.buf)
	b.buf = utf8.AppendRune(b.buf, r)
	return len(b.buf) - n, nil
}

func (b *StringBuffer) WriteString(s string) (int, error) {
	b.copyCheck()
	b.buf = append(b.buf, s...)
	return len(s), nil
}

//go:nosplit
//go:linkname noescape runtime.noescape
func noescape(p unsafe.Pointer) unsafe.Pointer

//go:nosplit
func (b *StringBuffer) copyCheck() {
	if b.addr == nil {

		b.addr = (*StringBuffer)(noescape(unsafe.Pointer(b)))
	} else if b.addr != b {
		panic("strings: illegal use of non-zero Builder copied by value")
	}
}

//go:noinline
func ConvertOne[TFrom, TTo any](from TFrom) (TTo, error) {
	var (
		zeroValFrom TFrom
		zeroValTo   TTo
	)

	if unsafe.Sizeof(zeroValFrom) != unsafe.Sizeof(zeroValTo) { // need same size to convert
		return zeroValTo, &ErrorSizeUnmatch{
			fromSize: int64(unsafe.Sizeof(zeroValFrom)),
			toSize:   int64(unsafe.Sizeof(zeroValTo)),
		}
	}

	value := *(*TTo)(unsafe.Pointer(&from))

	return value, nil
}

//go:nocheckptr
//go:noinline
func MustConvertOne[TFrom, TTo any](from TFrom) TTo {

	// #nosec G103
	return *(*TTo)(unsafe.Pointer(&from))

}

//go:noinline
func MakeNoZeroString(l int) []string {
	return unsafe.Slice((*string)(mallocgc(uintptr(l), nil, false)), l)
}

//go:inline
func MakeNoZeroCapString(l int, c int) []string {
	return MakeNoZeroString(c)[:l]
}

//go:noinline
func isEqual(v1, v2 any) bool {
	return reflect.ValueOf(v1).Pointer() == reflect.ValueOf(v2).Pointer()
}

//go:noinline
func isNil(v any) bool {

	return reflect.ValueOf(v).IsNil()
}
