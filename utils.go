package goblaze

import (
	"unsafe"
)

// validatePath checks if the path starts with a '/' and panics if not.
// It also returns the path.
func validatePath(path string) string {
	if len(path) == 0 || path[0] != '/' {
		panic("path must begin with '/'")
	}
	return path
}

//go:nosplit
func String(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

//go:nosplit
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

//go:nosplit
func CopyBytes(b []byte) []byte {
	return unsafe.Slice(unsafe.StringData(String(b)), len(b))
}

//go:nosplit
func Copy(b []byte, b1 []byte) ([]byte, []byte) {
	return []byte(String(b)), []byte(String(b1))
}

//go:nosplit
func CopyString(s string) string {
	c := make([]byte, len(s))
	copy(c, StringToBytes(s))
	return String(c)
}
