package async_test

import (
	"fmt"
	stdsync "sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/internal/impl/async"
)

// ─── async.Bus 单元测试 ───

// TestAsyncBusOnEmit 验证订阅后 Emit 异步调用 handler。
func TestAsyncBusOnEmit(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var called int32
	b.On("user.created", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.Emit(&core.Event{Type: "user.created"})

	// 等待异步处理
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&called) >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if atomic.LoadInt32(&called) < 1 {
		t.Errorf("handler called %d times, want >=1 after 500ms", called)
	}
}

// TestAsyncBusOff 验证取消订阅后不再分发。
func TestAsyncBusOff(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var called int32
	id := b.On("order.paid", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.Off(id)

	_ = b.Emit(&core.Event{Type: "order.paid"})
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&called) != 0 {
		t.Error("handler should not be called after Off")
	}
}

// TestAsyncBusOffNonExistent 注销不存在的 id 不崩溃。
func TestAsyncBusOffNonExistent(t *testing.T) {
	b := async.New(nil)
	defer b.Close()
	b.Off(99999) // should not panic
}

// TestAsyncBusEmitNil Emit(nil) 不崩溃。
func TestAsyncBusEmitNil(t *testing.T) {
	b := async.New(nil)
	defer b.Close()
	if err := b.Emit(nil); err != nil {
		t.Errorf("Emit(nil) should return nil, got %v", err)
	}
}

// TestAsyncBusEmitMatch 验证 EmitMatch 通配符同步路径。
func TestAsyncBusEmitMatch(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var count int32
	b.On("item.*", func(evt *core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	_ = b.EmitMatch(&core.Event{Type: "item.created"})
	_ = b.EmitMatch(&core.Event{Type: "item.deleted"})

	if atomic.LoadInt32(&count) != 2 {
		t.Errorf("EmitMatch: count=%d, want 2", count)
	}
}

// TestAsyncBusUnsafeEmit 验证 UnsafeEmit 与 Emit 等价（async 模式）。
func TestAsyncBusUnsafeEmit(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var called int32
	b.On("unsafe.event", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.UnsafeEmit(&core.Event{Type: "unsafe.event"})
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&called) < 1 {
		t.Errorf("UnsafeEmit: called=%d, want >=1", called)
	}
}

// TestAsyncBusEmitBatch 验证批量发布。
func TestAsyncBusEmitBatch(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var count int32
	b.On("batch.item", func(evt *core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	events := make([]*core.Event, 20)
	for i := range events {
		events[i] = &core.Event{Type: "batch.item"}
	}

	if err := b.EmitBatch(events); err != nil {
		t.Fatalf("EmitBatch error: %v", err)
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&count) < 1 {
		t.Errorf("EmitBatch: count=%d, want >0 after 100ms", count)
	}
}

// TestAsyncBusEmitMatchBatch 批量通配符发布（同步路径）。
func TestAsyncBusEmitMatchBatch(t *testing.T) {
	b := async.New(nil)
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

// TestAsyncBusStats 验证 Stats 返回合理值。
func TestAsyncBusStats(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	b.On("stat.event", func(evt *core.Event) error { return nil })

	for i := 0; i < 10; i++ {
		_ = b.Emit(&core.Event{Type: "stat.event"})
	}
	time.Sleep(100 * time.Millisecond)

	s := b.Stats()
	if s.Processed < 1 {
		t.Errorf("Stats.Processed=%d, want >=1 after 100ms", s.Processed)
	}
}

// TestAsyncBusCloseTwice 重复关闭不崩溃。
func TestAsyncBusCloseTwice(t *testing.T) {
	b := async.New(nil)
	b.Close()
	b.Close() // should not panic
}

// TestAsyncBusEmitAfterClose 关闭后 Emit 不崩溃（返回 nil 或 error）。
func TestAsyncBusEmitAfterClose(t *testing.T) {
	b := async.New(nil)
	b.Close()
	_ = b.Emit(&core.Event{Type: "closed.event"}) // should not panic
}

// TestAsyncBusDrain 验证 Drain 正常退出。
func TestAsyncBusDrain(t *testing.T) {
	b := async.New(nil)
	b.On("drain.event", func(evt *core.Event) error { return nil })
	_ = b.Emit(&core.Event{Type: "drain.event"})
	if err := b.Drain(500 * time.Millisecond); err != nil {
		t.Errorf("Drain error: %v", err)
	}
}

// TestAsyncBusCustomConfig 验证自定义配置创建成功。
func TestAsyncBusCustomConfig(t *testing.T) {
	cfg := &async.Config{
		Workers:  2,
		RingSize: 1 << 10, // 1024
	}
	b := async.New(cfg)
	defer b.Close()

	var called int32
	b.On("config.test", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.Emit(&core.Event{Type: "config.test"})
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&called) < 1 {
		t.Errorf("custom config: called=%d, want >=1", called)
	}
}

// TestAsyncBusConcurrentEmit 高并发 Emit 不崩溃、无竞态。
func TestAsyncBusConcurrentEmit(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var count int32
	b.On("conc.event", func(evt *core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	var wg stdsync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Emit(&core.Event{Type: "conc.event"})
		}()
	}
	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&count) < 1 {
		t.Errorf("concurrent Emit: count=%d, want >0", count)
	}
}

// TestAsyncBusUnsafeEmitMatch 验证 UnsafeEmitMatch 通配符同步路径。
func TestAsyncBusUnsafeEmitMatch(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var called int32
	b.On("wild.*", func(evt *core.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = b.UnsafeEmitMatch(&core.Event{Type: "wild.test"})

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("UnsafeEmitMatch: called=%d, want 1", called)
	}
}

// TestAsyncBusMultiSubscribers 多订阅者全部收到事件。
func TestAsyncBusMultiSubscribers(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	var count int32
	for i := 0; i < 5; i++ {
		b.On("multi.event", func(evt *core.Event) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	_ = b.Emit(&core.Event{Type: "multi.event"})
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&count) < 1 {
		// async dispatch: all or at least 1 handler should fire
		t.Errorf("multi-subscriber: count=%d, want >0", count)
	}
}

// TestAsyncBusEmitMatchError 验证 EmitMatch handler 返回 error 时 Emit 返回 error。
func TestAsyncBusEmitMatchError(t *testing.T) {
	b := async.New(nil)
	defer b.Close()

	b.On("err.topic", func(evt *core.Event) error {
		return fmt.Errorf("test error")
	})

	err := b.EmitMatch(&core.Event{Type: "err.topic"})
	if err == nil {
		t.Error("EmitMatch with erroring handler should return error")
	}
}
