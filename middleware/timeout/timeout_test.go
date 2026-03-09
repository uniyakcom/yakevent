package timeout_test

import (
	"strings"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/middleware/timeout"
)

func evt(typ string) *core.Event { return &core.Event{Type: typ} }

func TestTimeoutFastHandler(t *testing.T) {
	mw := timeout.New(100 * time.Millisecond)
	h := mw(func(e *core.Event) error { return nil })
	if err := h(evt("fast")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimeoutSlowHandlerTriggersTimeout(t *testing.T) {
	mw := timeout.New(5 * time.Millisecond)
	h := mw(func(e *core.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	err := h(evt("slow"))
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestTimeoutErrorMessageContainsDuration(t *testing.T) {
	d := 5 * time.Millisecond
	mw := timeout.New(d)
	h := mw(func(e *core.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	err := h(evt("slow"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), d.String()) {
		t.Errorf("expected duration %v in error %q", d, err.Error())
	}
}

func TestTimeoutErrorMessageContainsEventType(t *testing.T) {
	mw := timeout.New(5 * time.Millisecond)
	h := mw(func(e *core.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	err := h(evt("my.event"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "my.event") {
		t.Errorf("expected event type in error %q", err.Error())
	}
}

func TestTimeoutErrorMessagePrefix(t *testing.T) {
	mw := timeout.New(5 * time.Millisecond)
	h := mw(func(e *core.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	err := h(evt("x"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(err.Error(), "yakevent:") {
		t.Errorf("expected error to start with 'yakevent:', got %q", err.Error())
	}
}

func TestTimeoutHandlerErrorPropagated(t *testing.T) {
	mw := timeout.New(100 * time.Millisecond)
	h := mw(func(e *core.Event) error {
		return &customErr{"handler error"}
	})
	err := h(evt("e"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "handler error") {
		t.Errorf("expected handler error, got %q", err.Error())
	}
}

type customErr struct{ msg string }

func (e *customErr) Error() string { return e.msg }
