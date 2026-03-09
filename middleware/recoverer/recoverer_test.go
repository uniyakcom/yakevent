package recoverer_test

import (
	"errors"
	"testing"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/middleware/recoverer"
)

func evt() *core.Event { return &core.Event{Type: "test"} }

func TestRecovererNoPanic(t *testing.T) {
	mw := recoverer.New()
	h := mw(func(e *core.Event) error { return nil })
	if err := h(evt()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecovererPropagatesError(t *testing.T) {
	mw := recoverer.New()
	want := errors.New("normal error")
	h := mw(func(e *core.Event) error { return want })
	if err := h(evt()); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestRecovererCatchesPanic(t *testing.T) {
	mw := recoverer.New()
	h := mw(func(e *core.Event) error { panic("something bad") })
	err := h(evt())
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	var pe *recoverer.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PanicError, got %T: %v", err, err)
	}
}

func TestRecovererPanicValuePreserved(t *testing.T) {
	mw := recoverer.New()
	sentinel := errors.New("panic-error-value")
	h := mw(func(e *core.Event) error { panic(sentinel) })
	err := h(evt())
	var pe *recoverer.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PanicError, got %T", err)
	}
	if pe.Value != sentinel {
		t.Errorf("expected Value=%v, got %v", sentinel, pe.Value)
	}
}

func TestRecovererPanicString(t *testing.T) {
	mw := recoverer.New()
	h := mw(func(e *core.Event) error { panic("oops") })
	err := h(evt())
	var pe *recoverer.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PanicError, got %T", err)
	}
	if pe.Value != "oops" {
		t.Errorf("expected Value=oops, got %v", pe.Value)
	}
}

func TestPanicErrorMessageFormat(t *testing.T) {
	pe := &recoverer.PanicError{Value: "killer error"}
	got := pe.Error()
	want := "yakevent: handler panic: killer error"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestPanicErrorMessageInt(t *testing.T) {
	pe := &recoverer.PanicError{Value: 42}
	got := pe.Error()
	want := "yakevent: handler panic: 42"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestRecovererMultipleEvents(t *testing.T) {
	mw := recoverer.New()
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		if calls == 2 {
			panic("second call panics")
		}
		return nil
	})
	if err := h(evt()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := h(evt()); err == nil {
		t.Fatal("second call: expected error")
	}
	if err := h(evt()); err != nil {
		t.Fatalf("third call: %v", err)
	}
}
