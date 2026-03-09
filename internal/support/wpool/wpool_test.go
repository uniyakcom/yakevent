package wpool_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/internal/support/wpool"
)

func TestWPoolRunsTask(t *testing.T) {
	p := wpool.New(2, 10)
	var ran atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	p.Submit(wpool.Func(func() {
		ran.Store(true)
		wg.Done()
	}))
	wg.Wait()
	p.Stop()
	if !ran.Load() {
		t.Fatal("task was not run")
	}
}

func TestWPoolRunsMultipleTasks(t *testing.T) {
	p := wpool.New(4, 32)
	const n = 50
	var counter atomic.Int32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		p.Submit(wpool.Func(func() {
			counter.Add(1)
			wg.Done()
		}))
	}
	wg.Wait()
	p.Stop()
	if got := int(counter.Load()); got != n {
		t.Errorf("expected %d task runs, got %d", n, got)
	}
}

func TestWPoolOnPanicCalled(t *testing.T) {
	p := wpool.New(1, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	var panicVal any
	var mu sync.Mutex
	p.OnPanic = func(v any) {
		mu.Lock()
		panicVal = v
		mu.Unlock()
		wg.Done()
	}
	p.Submit(wpool.Func(func() {
		panic("test panic")
	}))
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("OnPanic was not called within timeout")
	}
	mu.Lock()
	got := panicVal
	mu.Unlock()
	if got != "test panic" {
		t.Errorf("expected panic value %q, got %v", "test panic", got)
	}
	p.Stop()
}

func TestWPoolStopIdempotent(t *testing.T) {
	p := wpool.New(2, 16)
	p.Stop()
	p.Stop() // second Stop should not panic
}

func TestWPoolSubmitAfterStopReturnsFalse(t *testing.T) {
	p := wpool.New(2, 16)
	p.Stop()
	if ok := p.Submit(wpool.Func(func() {})); ok {
		t.Error("Submit after Stop should return false")
	}
}

func TestWPoolFuncWrapper(t *testing.T) {
	called := false
	task := wpool.Func(func() { called = true })
	task.Run()
	if !called {
		t.Fatal("Func task was not called by Run()")
	}
}

func TestWPoolDefaultSizes(t *testing.T) {
	p := wpool.New(0, 0)
	var wg sync.WaitGroup
	wg.Add(1)
	p.Submit(wpool.Func(func() { wg.Done() }))
	wg.Wait()
	p.Stop()
}

func TestWPoolSubmitFuncHelper(t *testing.T) {
	p := wpool.New(1, 16)
	var ran atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	p.SubmitFunc(func() {
		ran.Store(true)
		wg.Done()
	})
	wg.Wait()
	p.Stop()
	if !ran.Load() {
		t.Fatal("SubmitFunc task was not run")
	}
}
