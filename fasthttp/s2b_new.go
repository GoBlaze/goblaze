//go:build go1.20

package fasthttp

import "unsafe"

func s2b(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		Cap int
	}{s, len(s)},
	))
}
