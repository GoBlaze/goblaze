package goblaze

import "unsafe"

// validatePath checks if the path starts with a '/' and panics if not.
// It also returns the path.
func validatePath(path string) string {
	if len(path) == 0 || path[0] != '/' {
		panic("path must begin with '/'")
	}
	return path
}

// #nosec G103
func String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
