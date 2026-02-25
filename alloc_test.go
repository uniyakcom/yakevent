package yakevent

import (
	"testing"

	implsync "github.com/uniyakcom/yakevent/internal/impl/sync"
	"github.com/uniyakcom/yakevent/internal/support/pool"
)

// TestArenaZeroAlloc 断言 Arena 路径确实零分配（替代 BenchmarkArena_Comparison/WithArena）。
// 使用 testing.AllocsPerRun 而非 benchmark，侧重正确性而非吞吐。
func TestArenaZeroAlloc(t *testing.T) {
	bus := implsync.NewBus()
	defer bus.Close()
	p := pool.Global()
	pool.SetEnableArena(true)
	bus.On("bench", func(e *Event) error { return nil })

	// 预热，排除第一次初始化干扰
	for i := 0; i < 10; i++ {
		evt := p.Acquire()
		evt.Type = "bench"
		evt.Data = p.AllocData(256)
		_ = bus.Emit(evt)
		p.Release(evt)
	}

	allocs := testing.AllocsPerRun(100, func() {
		evt := p.Acquire()
		evt.Type = "bench"
		evt.Data = p.AllocData(256)
		_ = bus.Emit(evt)
		p.Release(evt)
	})
	if allocs > 0 {
		t.Errorf("Arena path: expected 0 allocs/op, got %.0f", allocs)
	}
}

// TestEventPoolAcquireRelease 验证 EventPool Acquire/Release 基本正确性及零分配
// （替代 BenchmarkEventPool_AcquireRelease）。
func TestEventPoolAcquireRelease(t *testing.T) {
	p := pool.New()

	// 基本正确性
	evt := p.Acquire()
	if evt == nil {
		t.Fatal("Acquire returned nil")
	}
	evt.Type = "test"
	evt.Data = []byte("data")
	p.Release(evt)

	evt2 := p.Acquire()
	if evt2 == nil {
		t.Fatal("second Acquire returned nil")
	}
	p.Release(evt2)

	// 零分配断言
	data := []byte("data")
	allocs := testing.AllocsPerRun(100, func() {
		e := p.Acquire()
		e.Type = "test"
		e.Data = data
		p.Release(e)
	})
	if allocs > 0 {
		t.Errorf("EventPool Acquire/Release: expected 0 allocs/op, got %.0f", allocs)
	}
}

// TestEventPoolParallelSafe 验证 EventPool 并发使用无 panic / 数据竞争
// （替代 BenchmarkEventPool_Parallel，需配合 -race 运行）。
func TestEventPoolParallelSafe(t *testing.T) {
	p := pool.New()
	const goroutines = 8
	const iters = 1000
	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < iters; i++ {
				evt := p.Acquire()
				evt.Type = "test"
				p.Release(evt)
			}
		}()
	}
	for g := 0; g < goroutines; g++ {
		<-done
	}
}

// TestPatternMatchingZeroAlloc 断言 Sync Bus 通配符匹配路径零分配
// （替代 BenchmarkImplPatternMatching/Sync）。
func TestPatternMatchingZeroAlloc(t *testing.T) {
	bus := implsync.NewBus()
	defer bus.Close()
	bus.On("user.*", func(e *Event) error { return nil })
	evt := &Event{Type: "user.login"}

	allocs := testing.AllocsPerRun(100, func() {
		_ = bus.EmitMatch(evt)
	})
	if allocs > 0 {
		t.Errorf("pattern matching EmitMatch: expected 0 allocs/op, got %.0f", allocs)
	}
}
