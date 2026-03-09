package yakevent

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestEdgeCaseZeroHandlers(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	if err := bus.Emit(&Event{Type: "no.handlers", Data: []byte("data")}); err != nil {
		t.Errorf("emit with no handlers should not error, got %v", err)
	}
}

func TestEdgeCaseEmptyData(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var called bool
	bus.On("empty.data", func(e *Event) error {
		called = true
		if len(e.Data) != 0 {
			t.Error("expected empty data")
		}
		return nil
	})

	_ = bus.Emit(&Event{Type: "empty.data", Data: nil})
	_ = bus.Emit(&Event{Type: "empty.data", Data: []byte{}})

	if !called {
		t.Error("handler not called for empty data events")
	}
}

func TestEdgeCaseLargeEventData(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var received int
	bus.On("large.data", func(e *Event) error {
		received = len(e.Data)
		return nil
	})

	largeData := make([]byte, 1024*1024) // 1 MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	if err := bus.Emit(&Event{Type: "large.data", Data: largeData}); err != nil {
		t.Errorf("large event failed: %v", err)
	}
	if received != len(largeData) {
		t.Errorf("expected %d bytes, got %d", len(largeData), received)
	}
}

func TestEdgeCaseDuplicateOff(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	id := bus.On("dup.off", func(e *Event) error { return nil })
	bus.Off(id)
	bus.Off(id)
	bus.Off(id)
}

func TestEdgeCaseOffZeroID(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()
	bus.Off(0)
}

func TestEdgeCaseManyHandlers(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	const handlerCount = 1000
	var callCount int
	for i := 0; i < handlerCount; i++ {
		bus.On("many.handlers", func(e *Event) error {
			callCount++
			return nil
		})
	}

	if err := bus.Emit(&Event{Type: "many.handlers"}); err != nil {
		t.Errorf("emit with many handlers failed: %v", err)
	}
	if callCount != handlerCount {
		t.Errorf("expected %d calls, got %d", handlerCount, callCount)
	}
}

func TestEdgeCaseVeryLongEventType(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	longType := string(make([]byte, 1024))
	var called bool
	bus.On(longType, func(e *Event) error {
		called = true
		return nil
	})

	if err := bus.Emit(&Event{Type: longType, Data: []byte("data")}); err != nil {
		t.Errorf("very long event type failed: %v", err)
	}
	if !called {
		t.Error("handler not called for long event type")
	}
}

func TestEdgeCaseSpecialCharsEventType(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	for _, eventType := range []string{
		"user:login", "order/created", "system@startup",
		"log#error", "event-name", "event_name",
	} {
		t.Run(eventType, func(t *testing.T) {
			var called bool
			id := bus.On(eventType, func(e *Event) error {
				called = true
				return nil
			})
			defer bus.Off(id)

			if err := bus.Emit(&Event{Type: eventType, Data: []byte("data")}); err != nil {
				t.Errorf("special char event type %q failed: %v", eventType, err)
			}
			if !called {
				t.Errorf("handler not called for %q", eventType)
			}
		})
	}
}

func TestEdgeCaseNestedWildcards(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var callCount int32
	bus.On("user.*.action.*", func(e *Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	_ = bus.EmitMatch(&Event{Type: "user.123.action.login"}) // 应匹配
	_ = bus.EmitMatch(&Event{Type: "user.123.login"})        // 不匹配

	if actual := atomic.LoadInt32(&callCount); actual != 1 {
		t.Errorf("nested wildcard should match exactly 1, got %d", actual)
	}
}

func TestEdgeCaseBatchEmpty(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	if err := bus.EmitBatch(nil); err != nil {
		t.Errorf("empty batch should not error, got %v", err)
	}
	if err := bus.EmitBatch([]*Event{}); err != nil {
		t.Errorf("empty slice batch should not error, got %v", err)
	}
}

func TestEdgeCaseBatchSingle(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var called bool
	bus.On("batch.single", func(e *Event) error {
		called = true
		return nil
	})

	if err := bus.EmitBatch([]*Event{{Type: "batch.single", Data: []byte("data")}}); err != nil {
		t.Errorf("single event batch failed: %v", err)
	}
	if !called {
		t.Error("handler not called for single event batch")
	}
}

func TestEdgeCaseRapidOnOff(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	for i := 0; i < 1000; i++ {
		id := bus.On("rapid.test", func(e *Event) error { return nil })
		bus.Off(id)
	}
}

func TestEdgeCaseSlowHandler(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var called bool
	bus.On("slow.handler", func(e *Event) error {
		time.Sleep(100 * time.Millisecond)
		called = true
		return nil
	})

	start := time.Now()
	if err := bus.Emit(&Event{Type: "slow.handler"}); err != nil {
		t.Errorf("slow handler failed: %v", err)
	}
	if !called {
		t.Error("slow handler not called")
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Error("slow handler completed too quickly")
	}
}

func TestEdgeCaseEventReuse(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	var callCount int
	bus.On("reuse.test", func(e *Event) error {
		callCount++
		return nil
	})

	evt := &Event{Type: "reuse.test", Data: []byte("data")}
	for i := 0; i < 10; i++ {
		_ = bus.Emit(evt)
	}
	if callCount != 10 {
		t.Errorf("expected 10 calls with reused event, got %d", callCount)
	}
}
