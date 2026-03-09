package flow

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
)

func TestFlowBasic(t *testing.T) {
	var stageProcessed atomic.Int32
	stage := func(events []*core.Event) error {
		stageProcessed.Add(int32(len(events)))
		return nil
	}

	proc := New([]Stage{stage}, 10, 50*time.Millisecond)
	defer proc.Close()

	for i := 0; i < 25; i++ {
		if err := proc.Emit(&core.Event{Type: "test.topic", Data: []byte{byte(i)}}); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	processed, batches := proc.BatchStats()
	if processed != 25 {
		t.Errorf("expected 25 processed events, got %d", processed)
	}
	if batches < 2 {
		t.Errorf("expected at least 2 batches (size 10), got %d", batches)
	}
	if stageProcessed.Load() != 25 {
		t.Errorf("expected stage to process 25 events, got %d", stageProcessed.Load())
	}
}

func TestFlowSubscription(t *testing.T) {
	proc := New([]Stage{}, 5, 50*time.Millisecond)
	defer proc.Close()

	var received atomic.Int32
	var mu sync.Mutex
	var topics []string

	id := proc.On("test.*", func(evt *core.Event) error {
		received.Add(1)
		mu.Lock()
		topics = append(topics, evt.Type)
		mu.Unlock()
		return nil
	})
	if id == 0 {
		t.Fatal("expected non-zero subscription ID")
	}

	_ = proc.Emit(&core.Event{Type: "test.foo"})
	_ = proc.Emit(&core.Event{Type: "test.bar"})
	_ = proc.Emit(&core.Event{Type: "other.baz"}) // 不匹配

	time.Sleep(150 * time.Millisecond)

	if received.Load() != 2 {
		t.Errorf("expected 2 received events, got %d", received.Load())
	}
}

func TestFlowUnsubscribe(t *testing.T) {
	proc := New([]Stage{}, 5, 50*time.Millisecond)
	defer proc.Close()

	var count atomic.Int32
	id := proc.On("test.*", func(evt *core.Event) error {
		count.Add(1)
		return nil
	})

	_ = proc.Emit(&core.Event{Type: "test.1"})
	time.Sleep(100 * time.Millisecond)
	if count.Load() != 1 {
		t.Errorf("expected 1 event before unsubscribe, got %d", count.Load())
	}

	proc.Off(id)
	_ = proc.Emit(&core.Event{Type: "test.2"})
	time.Sleep(100 * time.Millisecond)
	if count.Load() != 1 {
		t.Errorf("expected still 1 event after unsubscribe, got %d", count.Load())
	}
}

func TestFlowBatchSizeTrigger(t *testing.T) {
	proc := New([]Stage{}, 5, 10*time.Second)
	defer proc.Close()

	var count atomic.Int32
	proc.On("batch.*", func(evt *core.Event) error {
		count.Add(1)
		return nil
	})

	for i := 0; i < 10; i++ {
		_ = proc.Emit(&core.Event{Type: "batch.item"})
	}

	time.Sleep(200 * time.Millisecond)
	if count.Load() != 10 {
		t.Errorf("expected 10 events processed by batch-size trigger, got %d", count.Load())
	}
}

func TestFlowTimeoutTrigger(t *testing.T) {
	proc := New([]Stage{}, 100, 50*time.Millisecond)
	defer proc.Close()

	var count atomic.Int32
	proc.On("timeout.*", func(evt *core.Event) error {
		count.Add(1)
		return nil
	})

	_ = proc.Emit(&core.Event{Type: "timeout.item"})

	time.Sleep(200 * time.Millisecond)
	if count.Load() != 1 {
		t.Errorf("expected 1 event processed by timeout trigger, got %d", count.Load())
	}
}

func TestFlowClose(t *testing.T) {
	proc := New([]Stage{}, 10, 50*time.Millisecond)

	var count atomic.Int32
	proc.On("close.test", func(evt *core.Event) error {
		count.Add(1)
		return nil
	})

	for i := 0; i < 5; i++ {
		_ = proc.Emit(&core.Event{Type: "close.test"})
	}
	time.Sleep(100 * time.Millisecond)
	proc.Close()

	// Close 后 Emit 不 panic
	_ = proc.Emit(&core.Event{Type: "close.test"})
}

func TestFlowDrain(t *testing.T) {
	proc := New([]Stage{}, 10, 50*time.Millisecond)
	defer proc.Close()

	proc.On("drain.test", func(evt *core.Event) error { return nil })
	_ = proc.Emit(&core.Event{Type: "drain.test"})

	err := proc.Drain(500 * time.Millisecond)
	if err != nil {
		t.Errorf("Drain failed: %v", err)
	}
}

func TestFlowStats(t *testing.T) {
	proc := New([]Stage{}, 5, 50*time.Millisecond)
	defer proc.Close()

	proc.On("stats.test", func(evt *core.Event) error { return nil })
	for i := 0; i < 10; i++ {
		_ = proc.Emit(&core.Event{Type: "stats.test"})
	}
	time.Sleep(200 * time.Millisecond)

	stats := proc.Stats()
	if stats.Emitted == 0 {
		t.Error("expected Emitted > 0")
	}
}
