package main

// #cgo CFLAGS: -I../ccode
// #cgo LDFLAGS: -L../ccode/ -lother
// #include <stdlib.h>
// #include <other.h>
import (
	"C"
)
import "fmt"

func main() {
	result := C.add(1, 2)
	fmt.Printf("Result: %d\n", result)
}
