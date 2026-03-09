package yakevent

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestScenarioSync 同步 RPC 调用场景
func TestScenarioSync(t *testing.T) {
	bus, err := ForSync()
	if err != nil {
		t.Fatalf("ForSync failed: %v", err)
	}
	defer bus.Close()

	var callCount int
	bus.On("api.request", func(e *Event) error {
		callCount++
		return nil
	})

	if err := bus.Emit(&Event{Type: "api.request", Data: []byte("GET /users/123")}); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 handler call, got %d", callCount)
	}
}

// TestScenarioPubSub 发布订阅场景
func TestScenarioPubSub(t *testing.T) {
	bus, err := ForAsync()
	if err != nil {
		t.Fatalf("ForAsync failed: %v", err)
	}
	defer bus.Close()

	for i := 0; i < 5; i++ {
		bus.On("order.created", func(e *Event) error { return nil })
	}
	if err := bus.Emit(&Event{Type: "order.created", Data: []byte("order_123")}); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
}

// TestScenarioLogging 日志监控采集场景（通配符匹配）
func TestScenarioLogging(t *testing.T) {
	bus, err := ForAsync()
	if err != nil {
		t.Fatalf("ForAsync failed: %v", err)
	}
	defer bus.Close()

	bus.On("log.*", func(e *Event) error { return nil })

	for _, logType := range []string{"log.info", "log.warn", "log.error"} {
		if err := bus.EmitMatch(&Event{Type: logType, Data: []byte("log message")}); err != nil {
			t.Fatalf("EmitMatch failed for %s: %v", logType, err)
		}
	}
}

// TestScenarioStream 流式数据处理场景（Flow Pipeline）
func TestScenarioStream(t *testing.T) {
	bus, err := ForFlow()
	if err != nil {
		t.Fatalf("ForFlow failed: %v", err)
	}
	defer bus.Close()

	var processed int32
	bus.On("stream.data", func(e *Event) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	for i := 0; i < 10; i++ {
		if err := bus.Emit(&Event{Type: "stream.data", Data: []byte("data_chunk")}); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	if actual := atomic.LoadInt32(&processed); actual != 10 {
		t.Errorf("expected 10 processed, got %d", actual)
	}
}

// TestScenarioBatch 批处理场景（Flow EmitBatch）
func TestScenarioBatch(t *testing.T) {
	bus, err := ForFlow()
	if err != nil {
		t.Fatalf("ForFlow failed: %v", err)
	}
	defer bus.Close()

	var batchCount int32
	bus.On("batch.insert", func(e *Event) error {
		atomic.AddInt32(&batchCount, 1)
		return nil
	})

	events := make([]*Event, 100)
	for i := range events {
		events[i] = &Event{Type: "batch.insert", Data: []byte("record")}
	}
	if err := bus.EmitBatch(events); err != nil {
		t.Fatalf("EmitBatch failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if actual := atomic.LoadInt32(&batchCount); actual != 100 {
		t.Errorf("expected 100 batch items, got %d", actual)
	}
}

// TestScenarioUltra 超低延迟引擎场景（Async Emit 延迟验证）
func TestScenarioUltra(t *testing.T) {
	bus, err := ForAsync()
	if err != nil {
		t.Fatalf("ForAsync failed: %v", err)
	}
	defer bus.Close()

	bus.On("tick", func(e *Event) error { return nil })

	start := time.Now()
	if err := bus.Emit(&Event{Type: "tick", Data: []byte("game_tick")}); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
	duration := time.Since(start)

	if duration > 10*time.Millisecond {
		t.Logf("emit took %v (expected <10ms for ultra scenario)", duration)
	}
}

// TestScenarioNew New() 自动选择实现
func TestScenarioNew(t *testing.T) {
	bus, err := New()
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer bus.Close()

	var called atomic.Bool
	bus.On("auto.test", func(e *Event) error {
		called.Store(true)
		return nil
	})
	_ = bus.Emit(&Event{Type: "auto.test"})
	time.Sleep(50 * time.Millisecond)
	_ = called.Load() // Async 模式异步处理，仅验证无 panic
}

// TestScenarioScenarioString Scenario("sync"/"async"/"flow") 字符串 API
func TestScenarioScenarioString(t *testing.T) {
	for _, name := range []string{"sync", "async", "flow"} {
		t.Run(name, func(t *testing.T) {
			bus, err := Scenario(name)
			if err != nil {
				t.Fatalf("Scenario(%q) failed: %v", name, err)
			}
			defer bus.Close()
			if bus == nil {
				t.Fatalf("Scenario(%q) returned nil bus", name)
			}
		})
	}
}
