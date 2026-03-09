package noop_test

import (
	"sync"
	"testing"

	"github.com/uniyakcom/yakevent/internal/support/noop"
)

func TestMutexSafeCanLockUnlock(t *testing.T) {
	mu := noop.NewMutex(true)
	var x int
	mu.Lock()
	x++
	mu.Unlock()
	_ = x
}

func TestMutexUnsafeNoPanic(t *testing.T) {
	mu := noop.NewMutex(false)
	var x int
	mu.Lock()
	x++
	mu.Unlock()
	_ = x
}

func TestMutexSafeActuallyLocks(t *testing.T) {
	mu := noop.NewMutex(true)
	var counter int
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if counter != goroutines {
		t.Errorf("expected %d, got %d (data race?)", goroutines, counter)
	}
}

func TestMutexUnsafeMultipleCalls(t *testing.T) {
	mu := noop.NewMutex(false)
	var x int
	for i := 0; i < 1000; i++ {
		mu.Lock()
		x++
		mu.Unlock()
	}
	_ = x
}

func TestRWMutexSafeCanLockUnlock(t *testing.T) {
	mu := noop.NewRWMutex(true)
	var x int
	mu.Lock()
	x++
	mu.Unlock()
	_ = x
}

func TestRWMutexSafeCanRLockRUnlock(t *testing.T) {
	mu := noop.NewRWMutex(true)
	var x int
	mu.RLock()
	x++
	mu.RUnlock()
	_ = x
}

func TestRWMutexUnsafeNoPanic(t *testing.T) {
	mu := noop.NewRWMutex(false)
	var x int
	mu.Lock()
	x++
	mu.Unlock()
	mu.RLock()
	x++
	mu.RUnlock()
	_ = x
}

func TestRWMutexUnsafeMultipleCalls(t *testing.T) {
	mu := noop.NewRWMutex(false)
	var x int
	for i := 0; i < 1000; i++ {
		mu.RLock()
		x++
		mu.RUnlock()
	}
	_ = x
}
