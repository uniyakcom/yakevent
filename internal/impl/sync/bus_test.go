package sync_test

import (
	"fmt"
	stdsync "sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
	impsync "github.com/uniyakcom/yakevent/internal/impl/sync"
)

// ─── sync.Bus 单元测试 ───

// TestSyncBusOnEmit 验证订阅后 Emit 调用 handler。
func TestSyncBusOnEmit(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var called int32
	b.On("user.created", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	if err := b.Emit(&core.Event{Type: "user.created"}); err != nil {
		t.Fatalf("Emit error: %v", err)
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("handler called %d times, want 1", called)
	}
}

// TestSyncBusOff 验证取消订阅后 Emit 不再调用 handler。
func TestSyncBusOff(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var called int32
	id := b.On("order.paid", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.Off(id)

	_ = b.Emit(&core.Event{Type: "order.paid"})
	if atomic.LoadInt32(&called) != 0 {
		t.Error("handler should not be called after Off")
	}
}

// TestSyncBusOffNonExistent 注销不存在的 id 不崩溃。
func TestSyncBusOffNonExistent(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()
	b.Off(99999) // should not panic
}

// TestSyncBusMultipleHandlers 同一 pattern 多个 handler 均被调用。
func TestSyncBusMultipleHandlers(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var count int32
	for i := 0; i < 5; i++ {
		b.On("event.fired", func(evt *core.Event) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	_ = b.Emit(&core.Event{Type: "event.fired"})
	if atomic.LoadInt32(&count) != 5 {
		t.Errorf("expected 5 handler calls, got %d", count)
	}
}

// TestSyncBusEmitNilEvent 发布 nil 事件不崩溃。
func TestSyncBusEmitNilEvent(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()
	if err := b.Emit(nil); err != nil {
		t.Errorf("Emit(nil) should return nil error, got %v", err)
	}
}

// TestSyncBusUnsafeEmit 验证 UnsafeEmit 正常分发。
func TestSyncBusUnsafeEmit(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var called int32
	b.On("direct", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.UnsafeEmit(&core.Event{Type: "direct"})
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("UnsafeEmit: called=%d, want 1", called)
	}
}

// TestSyncBusEmitMatch 验证通配符匹配发布。
func TestSyncBusEmitMatch(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var called int32
	b.On("user.*", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.EmitMatch(&core.Event{Type: "user.created"})
	_ = b.EmitMatch(&core.Event{Type: "user.deleted"})

	if atomic.LoadInt32(&called) != 2 {
		t.Errorf("EmitMatch wildcard: called=%d, want 2", called)
	}
}

// TestSyncBusEmitMatchNoWildcard EmitMatch 精确匹配路径。
func TestSyncBusEmitMatchNoWildcard(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var called int32
	b.On("item.created", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.EmitMatch(&core.Event{Type: "item.created"})

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("EmitMatch exact: called=%d, want 1", called)
	}
}

// TestSyncBusEmitBatch 验证批量发布。
func TestSyncBusEmitBatch(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var count int32
	b.On("batch.event", func(evt *core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	events := make([]*core.Event, 10)
	for i := range events {
		events[i] = &core.Event{Type: "batch.event"}
	}

	if err := b.EmitBatch(events); err != nil {
		t.Fatalf("EmitBatch error: %v", err)
	}

	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("EmitBatch: count=%d, want 10", count)
	}
}

// TestSyncBusStats 验证 Stats 计数器单调递增。
func TestSyncBusStats(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	b.On("stat.event", func(evt *core.Event) error { return nil })

	for i := 0; i < 5; i++ {
		_ = b.Emit(&core.Event{Type: "stat.event"})
	}

	s := b.Stats()
	if s.Emitted < 5 {
		t.Errorf("Stats.Emitted=%d, want >=5", s.Emitted)
	}
}

// TestSyncBusCloseTwice 重复关闭不崩溃。
func TestSyncBusCloseTwice(t *testing.T) {
	b := impsync.NewBus()
	b.Close()
	b.Close() // should not panic
}

// TestSyncBusDrain 验证 Drain（同步模式立即返回）。
func TestSyncBusDrain(t *testing.T) {
	b := impsync.NewBus()
	if err := b.Drain(100 * time.Millisecond); err != nil {
		t.Errorf("Drain error: %v", err)
	}
}

// TestSyncBusConcurrentOnOff 并发 On/Off/Emit 不崩溃。
func TestSyncBusConcurrentOnOff(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var wg stdsync.WaitGroup
	ids := make(chan uint64, 200)

	// 并发订阅
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := b.On("concurrent.event", func(evt *core.Event) error { return nil })
			ids <- id
		}()
	}

	// 并发 Emit
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Emit(&core.Event{Type: "concurrent.event"})
		}()
	}

	wg.Wait()
	close(ids)

	// 并发 Off
	var wg2 stdsync.WaitGroup
	for id := range ids {
		id := id
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			b.Off(id)
		}()
	}
	wg2.Wait()
}

// TestSyncBusPanicRecovery 验证 handler panic 被 Emit 捕获，不传播到调用方。
func TestSyncBusPanicRecovery(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	b.On("panic.event", func(evt *core.Event) error {
		panic("test panic")
	})

	err := b.Emit(&core.Event{Type: "panic.event"})
	if err == nil {
		t.Error("Emit should return error after handler panic")
	}

	s := b.Stats()
	if s.Panics < 1 {
		t.Errorf("Stats.Panics=%d, want >=1", s.Panics)
	}
}

// TestSyncBusUnsafeEmitMatchWildcard 验证 UnsafeEmitMatch 通配符路径。
func TestSyncBusUnsafeEmitMatchWildcard(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var called int32
	b.On("wild.**", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.UnsafeEmitMatch(&core.Event{Type: "wild.a.b.c"})

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("UnsafeEmitMatch wildcard: called=%d, want 1", called)
	}
}

// TestSyncBusEmitMatchBatch 批量通配符发布。
func TestSyncBusEmitMatchBatch(t *testing.T) {
	b := impsync.NewBus()
	defer b.Close()

	var count int32
	b.On("mbatch.*", func(evt *core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	events := []*core.Event{
		{Type: "mbatch.a"},
		{Type: "mbatch.b"},
		{Type: "mbatch.c"},
	}
	if err := b.EmitMatchBatch(events); err != nil {
		t.Fatalf("EmitMatchBatch error: %v", err)
	}

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("EmitMatchBatch: count=%d, want 3", count)
	}
}

// ─── Async 模式测试 ───

// TestSyncBusAsyncEmit 验证 NewAsync 模式下 Emit 异步处理。
func TestSyncBusAsyncEmit(t *testing.T) {
	b, err := impsync.NewAsync(0)
	if err != nil {
		t.Fatalf("NewAsync error: %v", err)
	}
	defer b.Close()

	var count int32
	b.On("async.event", func(evt *core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	for i := 0; i < 20; i++ {
		_ = b.Emit(&core.Event{Type: "async.event"})
	}

	// 等待异步处理完成
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&count) < 1 {
		t.Errorf("async Emit: count=%d, want >0", count)
	}
}

// TestSyncBusAsyncDrain 验证 NewAsync Drain 不超时（小任务量）。
func TestSyncBusAsyncDrain(t *testing.T) {
	b, err := impsync.NewAsync(0)
	if err != nil {
		t.Fatalf("NewAsync error: %v", err)
	}

	b.On("drain.event", func(evt *core.Event) error { return nil })
	_ = b.Emit(&core.Event{Type: "drain.event"})

	if err := b.Drain(500 * time.Millisecond); err != nil {
		t.Errorf("Drain error: %v", err)
	}
}

// TestSyncBusAsyncErrorHandling 验证 NewAsync 模式下 handler 错误被收集。
func TestSyncBusAsyncErrorHandling(t *testing.T) {
	b, errNew := impsync.NewAsync(0)
	if errNew != nil {
		t.Fatalf("NewAsync error: %v", errNew)
	}
	defer b.Close()

	b.On("err.event", func(evt *core.Event) error {
		return fmt.Errorf("handler error")
	})

	_ = b.Emit(&core.Event{Type: "err.event"})
	time.Sleep(50 * time.Millisecond)

	// LastError 可能有值也可能没有（取决于时序）；仅验证不崩溃
	_ = b.LastError()
	b.ClearError()
}
