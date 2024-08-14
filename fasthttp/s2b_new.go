//go:build go1.20

package fasthttp

import "unsafe"

func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
