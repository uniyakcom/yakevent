package util

import (
	"testing"
)

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
