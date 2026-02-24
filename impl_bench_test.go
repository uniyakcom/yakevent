package yakevent

import (
	"runtime"
	"testing"

	implasync "github.com/uniyakcom/yakevent/internal/impl/async"
	"github.com/uniyakcom/yakevent/internal/impl/flow"
	implsync "github.com/uniyakcom/yakevent/internal/impl/sync"
)

func BenchmarkImplSync(b *testing.B) {
	bus := implsync.NewBus()
	defer bus.Close()
	id := bus.On("bench", func(e *Event) error { return nil })
	defer bus.Off(id)
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bus.Emit(evt)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkImplSyncUnsafe(b *testing.B) {
	bus := implsync.NewBus()
	defer bus.Close()
	id := bus.On("bench", func(e *Event) error { return nil })
	defer bus.Off(id)
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bus.UnsafeEmit(evt)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkImplSyncHighConcurrency(b *testing.B) {
	bus := implsync.NewBus()
	defer bus.Close()
	for i := 0; i < 100; i++ {
		bus.On("bench", func(e *Event) error { return nil })
	}
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bus.Emit(evt)
		}
	})
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkImplAsync(b *testing.B) {
	bus := implasync.New(implasync.DefaultConfig())
	defer bus.Close()
	bus.On("bench", func(e *Event) error { return nil })
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bus.Emit(evt)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkImplAsyncHighConcurrency(b *testing.B) {
	bus := implasync.New(&implasync.Config{
		Workers:  runtime.NumCPU() / 2,
		RingSize: 1 << 13,
	})
	defer bus.Close()
	bus.On("bench", func(e *Event) error { return nil })
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bus.Emit(evt)
		}
	})
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkImplFlow(b *testing.B) {
	bus := flow.New([]flow.Stage{func(events []*Event) error { return nil }}, 100, 100)
	defer bus.Close()
	id := bus.On("bench", func(e *Event) error { return nil })
	defer bus.Off(id)
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bus.Emit(evt)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkImplFlowHighConcurrency(b *testing.B) {
	bus := flow.New([]flow.Stage{func(events []*Event) error { return nil }}, 1000, 100)
	defer bus.Close()
	for i := 0; i < 100; i++ {
		bus.On("bench", func(e *Event) error { return nil })
	}
	evt := &Event{Type: "bench", Data: []byte("data")}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bus.Emit(evt)
		}
	})
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "M/s")
}

func BenchmarkScenarioBatchBulkInsert(b *testing.B) {
	bus, err := ForFlow()
	if err != nil {
		b.Fatal(err)
	}
	defer bus.Close()
	bus.On("batch.insert", func(e *Event) error { return nil })
	const batchSize = 1000
	events := make([]*Event, batchSize)
	for i := range events {
		events[i] = &Event{Type: "batch.insert", Data: []byte("record")}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bus.EmitBatch(events)
	}
	b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds()/1e6, "M/s")
}
