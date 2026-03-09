package spsc_test

import (
	"testing"

	"github.com/uniyakcom/yakevent/internal/support/spsc"
)

func TestNewSPSCRingPowerOfTwo(t *testing.T) {
	r := spsc.NewSPSCRing[int](8)
	if r == nil {
		t.Fatal("expected non-nil ring")
	}
}

func TestNewSPSCRingPanicsOnNonPowerOfTwo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-power-of-2 size")
		}
	}()
	spsc.NewSPSCRing[int](7)
}

func TestNewSPSCRingDefaultsZeroSize(t *testing.T) {
	// size=0 → defaults to 8192
	r := spsc.NewSPSCRing[int](0)
	if r == nil {
		t.Fatal("expected non-nil ring")
	}
}

func TestSPSCEnqueueDequeue(t *testing.T) {
	r := spsc.NewSPSCRing[int](8)
	ok := r.Enqueue(42)
	if !ok {
		t.Fatal("Enqueue should succeed on empty ring")
	}
	v, ok := r.Dequeue()
	if !ok {
		t.Fatal("Dequeue should succeed after Enqueue")
	}
	if v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
}

func TestSPSCDequeueEmptyReturnsFalse(t *testing.T) {
	r := spsc.NewSPSCRing[int](8)
	_, ok := r.Dequeue()
	if ok {
		t.Fatal("Dequeue on empty ring should return false")
	}
}

func TestSPSCEnqueueFullReturnsFalse(t *testing.T) {
	const size = 4
	r := spsc.NewSPSCRing[int](size)
	for i := 0; i < size; i++ {
		if !r.Enqueue(i) {
			t.Fatalf("Enqueue %d failed before full", i)
		}
	}
	if r.Enqueue(99) {
		t.Fatal("Enqueue on full ring should return false")
	}
}

func TestSPSCFIFOOrder(t *testing.T) {
	r := spsc.NewSPSCRing[int](16)
	for i := 0; i < 8; i++ {
		r.Enqueue(i) //nolint:errcheck
	}
	for i := 0; i < 8; i++ {
		v, ok := r.Dequeue()
		if !ok {
			t.Fatalf("Dequeue %d failed", i)
		}
		if v != i {
			t.Errorf("expected %d, got %d", i, v)
		}
	}
}

func TestSPSCWrapAround(t *testing.T) {
	r := spsc.NewSPSCRing[int](4)
	// Fill, drain, fill again to test wrap-around
	for round := 0; round < 3; round++ {
		for i := 0; i < 4; i++ {
			if !r.Enqueue(round*10 + i) {
				t.Fatalf("round=%d Enqueue %d failed", round, i)
			}
		}
		for i := 0; i < 4; i++ {
			v, ok := r.Dequeue()
			if !ok {
				t.Fatalf("round=%d Dequeue %d failed", round, i)
			}
			if v != round*10+i {
				t.Errorf("round=%d expected %d, got %d", round, round*10+i, v)
			}
		}
	}
}
