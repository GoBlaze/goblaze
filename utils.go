package goblaze

import (
	"fmt"
	"reflect"
	"unicode/utf8"
	"unsafe"
)

type ErrorSizeUnmatch struct {
	fromLength int
	fromSize   int64
	toSize     int64
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

//go:inline
//go:nosplit
func String(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

//go:inline
//go:nosplit
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

//go:inline
//go:nosplit
func CopyBytes(b []byte) []byte {
	return unsafe.Slice(unsafe.StringData(String(b)), len(b))
}

//go:inline
//go:nosplit
func Copy(b []byte, b1 []byte) ([]byte, []byte) {
	return []byte(String(b)), []byte(String(b1))
}

//go:inline
//go:nosplit
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

func swap[T any](a, b *T) {
	tmp := *a
	*a = *b
	*b = tmp
}

//go:nosplit
//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ unsafe.Pointer, needzero bool) unsafe.Pointer

//go:nosplit
//go:inline
func MakeNoZero(l int) []byte {
	return unsafe.Slice((*byte)(mallocgc(uintptr(l), nil, false)), l)
}

//go:nosplit
//go:inline
func MakeNoZeroCap(l int, c int) []byte {
	return MakeNoZero(c)[:l]
}

type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

//go:nosplit
//go:inline
func SliceUnsafePointer[T any](slice []T) unsafe.Pointer {
	header := *(*sliceHeader)(unsafe.Pointer(&slice))
	return header.Data
}

type StringBuffer struct {
	noCopy No // nolint:structcheck
	buf    []byte
	addr   *StringBuffer
}

func NewStringBuffer(cap int) *StringBuffer {
	return &StringBuffer{
		buf: make([]byte, 0, cap),
	}
}

//go:nosplit
//go:inline
func (b *StringBuffer) String() string {
	return String(b.buf)
}

//go:nosplit
//go:inline
func (b *StringBuffer) Bytes() []byte {
	return b.buf
}

//go:nosplit
//go:inline
func (b *StringBuffer) Len() int {
	return len(b.buf)
}

//go:nosplit
//go:inline
func (b *StringBuffer) Cap() int {
	return cap(b.buf)
}

//go:nosplit
//go:inline
func (b *StringBuffer) Reset() {
	b.buf = b.buf[:0]
}

//go:nosplit
//go:inline
func (b *StringBuffer) grow(n int) {
	buf := MakeNoZero(2*cap(b.buf) + n)[:len(b.buf)]
	copy(buf, b.buf)
	b.buf = buf
}

func (b *StringBuffer) Grow(n int) {
	if n < 0 {
		panic("fast.StringBuffer.Grow: negative count")
	}
	if cap(b.buf)-len(b.buf) < n {
		b.grow(n)
	}
}

//go:nosplit
//go:inline
func (b *StringBuffer) Write(p []byte) (int, error) {
	b.copyCheck()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

//go:nosplit
//go:inline
func (b *StringBuffer) WriteByte(c byte) error {
	b.copyCheck()
	b.buf = append(b.buf, c)
	return nil
}

//go:nosplit
//go:inline
func (b *StringBuffer) WriteRune(r rune) (int, error) {
	b.copyCheck()
	n := len(b.buf)
	b.buf = utf8.AppendRune(b.buf, r)
	return len(b.buf) - n, nil
}

//go:nosplit
//go:inline
func (b *StringBuffer) WriteString(s string) (int, error) {
	b.copyCheck()
	b.buf = append(b.buf, s...)
	return len(s), nil
}

//go:nosplit
//go:nocheckptr
func noescape(p unsafe.Pointer) unsafe.Pointer {
	// This function is a no-op and should not be used. It is included for
	// compatibility with other code and should not be called directly.
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)

}

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
