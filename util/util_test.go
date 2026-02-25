package util

import (
	"runtime"
	"sync"
	"testing"
)

func TestPerCPUCounterBasic(t *testing.T) {
	c := NewPerCPUCounter()

	if got := c.Read(); got != 0 {
		t.Fatalf("initial Read() = %d, want 0", got)
	}

	c.Add(1)
	c.Add(1)
	c.Add(1)
	if got := c.Read(); got != 3 {
		t.Fatalf("Read() = %d, want 3", got)
	}

	c.Add(-1)
	if got := c.Read(); got != 2 {
		t.Fatalf("after Add(-1): Read() = %d, want 2", got)
	}
}

func TestPerCPUCounterConcurrent(t *testing.T) {
	const goroutines = 64
	const addsPerGoroutine = 10000

	c := NewPerCPUCounter()
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				c.Add(1)
			}
		}()
	}
	wg.Wait()

	want := int64(goroutines * addsPerGoroutine)
	if got := c.Read(); got != want {
		t.Fatalf("concurrent Read() = %d, want %d", got, want)
	}
}

func TestNewPerCPUCounterSlotSize(t *testing.T) {
	c := NewPerCPUCounter()
	n := runtime.GOMAXPROCS(0)

	// slot 数须是 2 的幂且 >= max(n 向上取2的幂, 8)
	slots := c.mask + 1

	if slots&(slots-1) != 0 {
		t.Fatalf("slots=%d is not a power of 2", slots)
	}
	if slots < 8 {
		t.Fatalf("slots=%d < 8 (minimum)", slots)
	}

	// 必须能覆盖所有逻辑 CPU
	if slots < n {
		t.Fatalf("slots=%d < GOMAXPROCS=%d", slots, n)
	}
}
