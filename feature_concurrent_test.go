package yakevent

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentOnOff 并发订阅和取消订阅不死锁
func TestConcurrentOnOff(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := bus.On("concurrent.test", func(e *Event) error { return nil })
			time.Sleep(time.Millisecond)
			bus.Off(id)
		}()
	}
	wg.Wait()
}

// TestConcurrentEmit 并发发布事件，计数准确
func TestConcurrentEmit(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var counter int64
	bus.On("concurrent.emit", func(e *Event) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	const goroutines, eventsPerGoroutine = 100, 100
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				bus.Emit(&Event{Type: "concurrent.emit", Data: []byte("test")})
			}
		}()
	}
	wg.Wait()

	expected := int64(goroutines * eventsPerGoroutine)
	if actual := atomic.LoadInt64(&counter); actual != expected {
		t.Errorf("expected %d events, got %d", expected, actual)
	}
}

// TestConcurrentOnOffEmit 混合并发：订阅/取消/发布同时进行
func TestConcurrentOnOffEmit(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				id := bus.On("mixed.test", func(e *Event) error {
					atomic.AddInt64(&counter, 1)
					return nil
				})
				time.Sleep(time.Millisecond)
				bus.Off(id)
			}
		}()
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				bus.Emit(&Event{Type: "mixed.test", Data: []byte("data")})
				time.Sleep(time.Millisecond)
			}
		}()
	}
	wg.Wait()
	t.Logf("processed %d events in mixed concurrent scenario", counter)
}

// TestRaceConditions 竞态条件检测（需 -race）
func TestRaceConditions(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var wg sync.WaitGroup
	var ids []uint64
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := bus.On("race.test", func(e *Event) error { return nil })
			mu.Lock()
			ids = append(ids, id)
			mu.Unlock()
		}()
	}
	wg.Wait()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				bus.Emit(&Event{Type: "race.test", Data: []byte("test")})
			}
		}()
	}
	wg.Wait()

	for _, id := range ids {
		wg.Add(1)
		go func(handlerID uint64) {
			defer wg.Done()
			bus.Off(handlerID)
		}(id)
	}
	wg.Wait()
}

// TestConcurrentPatternMatching 并发通配符匹配
func TestConcurrentPatternMatching(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var counter int64
	for _, pattern := range []string{"user.*", "order.*", "system.*", "log.*"} {
		bus.On(pattern, func(e *Event) error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
	}

	eventTypes := []string{
		"user.login", "user.logout",
		"order.created", "order.paid",
		"system.startup", "system.shutdown",
		"log.info", "log.error",
	}

	var wg sync.WaitGroup
	for _, et := range eventTypes {
		wg.Add(1)
		go func(evtType string) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				bus.EmitMatch(&Event{Type: evtType, Data: []byte("data")})
			}
		}(et)
	}
	wg.Wait()

	expected := int64(len(eventTypes) * 50)
	if actual := atomic.LoadInt64(&counter); actual != expected {
		t.Errorf("expected %d pattern matches, got %d", expected, actual)
	}
}

// TestConcurrentBatchEmit 并发批量发布
func TestConcurrentBatchEmit(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var counter int64
	bus.On("batch.test", func(e *Event) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	const goroutines, batchSize = 10, 100
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			events := make([]*Event, batchSize)
			for j := range events {
				events[j] = &Event{Type: "batch.test", Data: []byte("data")}
			}
			bus.EmitBatch(events)
		}()
	}
	wg.Wait()

	expected := int64(goroutines * batchSize)
	if actual := atomic.LoadInt64(&counter); actual != expected {
		t.Errorf("expected %d events in batch, got %d", expected, actual)
	}
}

// TestNestedHandlerOn 在 handler 中订阅
func TestNestedHandlerOn(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var innerCalled bool
	bus.On("outer", func(e *Event) error {
		bus.On("inner", func(e *Event) error {
			innerCalled = true
			return nil
		})
		return nil
	})

	if err := bus.Emit(&Event{Type: "outer"}); err != nil {
		t.Fatalf("outer emit failed: %v", err)
	}
	if err := bus.Emit(&Event{Type: "inner"}); err != nil {
		t.Fatalf("inner emit failed: %v", err)
	}
	if !innerCalled {
		t.Error("nested handler not called")
	}
}

// TestNestedHandlerOff 在 handler 中取消自己
func TestNestedHandlerOff(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var callCount int
	var id uint64
	id = bus.On("self-cancel", func(e *Event) error {
		callCount++
		if callCount == 1 {
			bus.Off(id)
		}
		return nil
	})

	evt := &Event{Type: "self-cancel"}
	bus.Emit(evt)
	bus.Emit(evt)

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// TestConcurrentClose 并发关闭不 panic
func TestConcurrentClose(t *testing.T) {
	bus, _ := ForSync()

	bus.On("test", func(e *Event) error { return nil })

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				bus.Emit(&Event{Type: "test", Data: []byte("data")})
			}
		}()
	}

	time.Sleep(10 * time.Millisecond)
	bus.Close()
	wg.Wait()
}
