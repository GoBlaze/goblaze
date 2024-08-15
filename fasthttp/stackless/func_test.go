package stackless

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewFuncMulti(t *testing.T) {
	t.Parallel()

	var n1, n2 uint64

	f1 := NewFunc(func(ctx any) {

		if ctx == nil {
			panic("ctx is nil")
		}

		v, ok := ctx.(int)
		if !ok {
			panic(fmt.Sprintf("ctx is not of type int: %T", ctx))
		}

		atomic.AddUint64(&n1, uint64(v))
	})

	f2 := NewFunc(func(ctx any) {

		if ctx == nil {
			panic("ctx is nil")
		}

		v, ok := ctx.(int)
		if !ok {
			panic(fmt.Sprintf("ctx is not of type int: %T", ctx))
		}

		atomic.AddUint64(&n2, uint64(v))
	})

	iterations := 4 * 1024

	f1Done := make(chan error, 1)
	go func() {
		var err error
		for i := 0; i < iterations; i++ {
			if !f1(3) {
				err = errors.New("f1 mustn't return false")
				break
			}
		}
		f1Done <- err
	}()

	f2Done := make(chan error, 1)
	go func() {
		var err error
		for i := 0; i < iterations; i++ {
			if !f2(5) {
				err = errors.New("f2 mustn't return false")
				break
			}
		}
		f2Done <- err
	}()

	select {
	case err := <-f1Done:
		if err != nil {
			t.Fatalf("unexpected error in f1: %v", err)
		}
	case <-time.After(10 * time.Second): // Increase timeout to ensure the test has enough time to complete
		t.Fatalf("timeout waiting for f1 to complete")
	}

	select {
	case err := <-f2Done:
		if err != nil {
			t.Fatalf("unexpected error in f2: %v", err)
		}
	case <-time.After(2 * time.Second): // Increase timeout to ensure the test has enough time to complete
		t.Fatalf("timeout waiting for f2 to complete")
	}

	// Verify the results
	expectedN1 := uint64(3 * iterations)
	if n1 != expectedN1 {
		t.Fatalf("unexpected n1: %d. Expecting %d", n1, expectedN1)
	}

	expectedN2 := uint64(5 * iterations)
	if n2 != expectedN2 {
		t.Fatalf("unexpected n2: %d. Expecting %d", n2, expectedN2)
	}
}
