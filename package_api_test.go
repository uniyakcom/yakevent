package yakevent

import (
	"errors"
	"sync/atomic"
	"testing"
)

// TestPackageAPIDefault 验证 Default() 返回非 nil Bus
func TestPackageAPIDefault(t *testing.T) {
	bus := Default()
	if bus == nil {
		t.Fatal("Default() returned nil")
	}
}

// TestPackageAPIOnEmitOff 包级 API 基本流程：订阅→发布→取消
func TestPackageAPIOnEmitOff(t *testing.T) {
	var called int
	id := On("pkg.test.basic", func(e *Event) error {
		called++
		return nil
	})

	if err := Emit(&Event{Type: "pkg.test.basic", Data: []byte("hello")}); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}

	Off(id)

	called = 0
	_ = Emit(&Event{Type: "pkg.test.basic", Data: []byte("again")})
	if called != 0 {
		t.Errorf("expected 0 calls after Off, got %d", called)
	}
}

// TestPackageAPIEmitError 包级 Emit 传递 handler error
func TestPackageAPIEmitError(t *testing.T) {
	want := errors.New("handler failed")
	id := On("pkg.test.err", func(e *Event) error {
		return want
	})
	defer Off(id)

	err := Emit(&Event{Type: "pkg.test.err"})
	if err == nil {
		t.Fatal("expected error from Emit")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected %v, got %v", want, err)
	}
}

// TestPackageAPIEmitMatch 包级 EmitMatch 通配符匹配
func TestPackageAPIEmitMatch(t *testing.T) {
	var matched int
	id := On("pkg.test.*", func(e *Event) error {
		matched++
		return nil
	})
	defer Off(id)

	_ = EmitMatch(&Event{Type: "pkg.test.wildcard"})
	if matched != 1 {
		t.Errorf("expected 1 wildcard match, got %d", matched)
	}
}

// TestPackageAPIEmitBatch 包级 EmitBatch
func TestPackageAPIEmitBatch(t *testing.T) {
	var count int32
	id := On("pkg.test.batch", func(e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	defer Off(id)

	events := make([]*Event, 10)
	for i := range events {
		events[i] = &Event{Type: "pkg.test.batch", Data: []byte("item")}
	}
	if err := EmitBatch(events); err != nil {
		t.Fatalf("EmitBatch failed: %v", err)
	}
	if c := atomic.LoadInt32(&count); c != 10 {
		t.Errorf("expected 10, got %d", c)
	}
}

// TestPackageAPIGetStats 包级 GetStats
func TestPackageAPIGetStats(t *testing.T) {
	id := On("pkg.test.stats", func(e *Event) error { return nil })
	_ = Emit(&Event{Type: "pkg.test.stats"})
	Off(id)

	stats := GetStats()
	if stats.Emitted == 0 {
		t.Error("expected Emitted > 0")
	}
}

// TestPackageAPIMultipleHandlers 多 handler 按序执行
func TestPackageAPIMultipleHandlers(t *testing.T) {
	var order []int
	id1 := On("pkg.test.multi", func(e *Event) error {
		order = append(order, 1)
		return nil
	})
	id2 := On("pkg.test.multi", func(e *Event) error {
		order = append(order, 2)
		return nil
	})
	defer Off(id1)
	defer Off(id2)

	_ = Emit(&Event{Type: "pkg.test.multi"})
	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}
}

// TestPackageAPIZeroHandlers 无 handler 时 Emit 不报错
func TestPackageAPIZeroHandlers(t *testing.T) {
	err := Emit(&Event{Type: "pkg.test.nohandler"})
	if err != nil {
		t.Errorf("emit with no handlers should not error, got %v", err)
	}
}

// TestPackageAPIChain Chain 组合中间件
func TestPackageAPIChain(t *testing.T) {
	var order []string
	mw := func(h Handler) Handler {
		return func(e *Event) error {
			order = append(order, "mw")
			return h(e)
		}
	}
	base := func(e *Event) error {
		order = append(order, "handler")
		return nil
	}
	wrapped := Chain(base, mw)
	if err := wrapped(&Event{Type: "test"}); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "mw" || order[1] != "handler" {
		t.Errorf("unexpected order: %v", order)
	}
}
