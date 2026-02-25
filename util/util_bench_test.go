package util

import (
	"sync/atomic"
	"testing"
)

// ── 单 goroutine（基础开销）──────────────────────────────────────────────────

func BenchmarkPerCPUCounterAdd(b *testing.B) {
	c := NewPerCPUCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Add(1)
	}
}

func BenchmarkPerCPUCounterRead(b *testing.B) {
	c := NewPerCPUCounter()
	for i := 0; i < 1000; i++ {
		c.Add(1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Read()
	}
}

// ── 多 goroutine 并发写（设计目标场景）────────────────────────────────────────
// 与 atomic.Int64 对比，展示 PerCPUCounter 在多核下的竞争优势

func BenchmarkPerCPUCounterAddParallel(b *testing.B) {
	c := NewPerCPUCounter()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Add(1)
		}
	})
}

func BenchmarkAtomicInt64AddParallel(b *testing.B) {
	var v atomic.Int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v.Add(1)
		}
	})
}
