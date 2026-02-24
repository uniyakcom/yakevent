package middleware_test

import (
	"errors"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/middleware/logging"
	"github.com/uniyakcom/yakevent/middleware/recoverer"
	"github.com/uniyakcom/yakevent/middleware/retry"
	"github.com/uniyakcom/yakevent/middleware/timeout"
)

var nopEvent = &core.Event{Type: "test", Data: []byte("data")}

// ─── recoverer ────────────────────────────────────────────────────

func TestRecovererPanic(t *testing.T) {
	mw := recoverer.New()
	handler := mw(func(evt *core.Event) error {
		panic("test panic")
	})

	err := handler(nopEvent)
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}

	var pe *recoverer.PanicError
	if !errors.As(err, &pe) {
		t.Errorf("expected *PanicError, got %T: %v", err, err)
	}
}

func TestRecovererNoPanic(t *testing.T) {
	mw := recoverer.New()
	called := false
	handler := mw(func(evt *core.Event) error {
		called = true
		return nil
	})

	if err := handler(nopEvent); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestRecovererPassthrough(t *testing.T) {
	mw := recoverer.New()
	want := errors.New("business error")
	handler := mw(func(evt *core.Event) error { return want })

	err := handler(nopEvent)
	if !errors.Is(err, want) {
		t.Errorf("expected passthrough error %v, got %v", want, err)
	}
}

// ─── retry ────────────────────────────────────────────────────────

func TestRetryMiddleware(t *testing.T) {
	var attempts int
	mw := retry.New(retry.Config{
		MaxRetries:      2,
		InitialInterval: time.Millisecond,
	})
	handler := mw(func(evt *core.Event) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err := handler(nopEvent); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryExhausted(t *testing.T) {
	var attempts int
	mw := retry.New(retry.Config{
		MaxRetries:      2,
		InitialInterval: time.Millisecond,
	})
	handler := mw(func(evt *core.Event) error {
		attempts++
		return errors.New("persistent error")
	})

	if err := handler(nopEvent); err == nil {
		t.Error("expected error after retry exhaustion")
	}
	if attempts != 3 { // 1 initial + 2 retries
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryShouldRetry(t *testing.T) {
	errNoRetry := errors.New("no retry")
	var attempts int
	mw := retry.New(retry.Config{
		MaxRetries:      3,
		InitialInterval: time.Millisecond,
		ShouldRetry: func(err error) bool {
			return !errors.Is(err, errNoRetry)
		},
	})
	handler := mw(func(evt *core.Event) error {
		attempts++
		return errNoRetry
	})

	if err := handler(nopEvent); !errors.Is(err, errNoRetry) {
		t.Errorf("expected errNoRetry, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (should not retry)", attempts)
	}
}

// ─── timeout ──────────────────────────────────────────────────────

func TestTimeoutExceeded(t *testing.T) {
	mw := timeout.New(20 * time.Millisecond)
	handler := mw(func(evt *core.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	err := handler(nopEvent)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestTimeoutNotExceeded(t *testing.T) {
	mw := timeout.New(200 * time.Millisecond)
	handler := mw(func(evt *core.Event) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err := handler(nopEvent); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTimeoutPassthroughError(t *testing.T) {
	want := errors.New("handler error")
	mw := timeout.New(200 * time.Millisecond)
	handler := mw(func(evt *core.Event) error { return want })

	if err := handler(nopEvent); !errors.Is(err, want) {
		t.Errorf("expected %v, got %v", want, err)
	}
}

// ─── logging ──────────────────────────────────────────────────────

// captureLogger 捕获日志调用，用于测试验证
type captureLogger struct {
	debugCalls int
	errorCalls int
	lastErr    error
}

func (l *captureLogger) Debug(msg string, args ...any) { l.debugCalls++ }
func (l *captureLogger) Error(msg string, err error, args ...any) {
	l.errorCalls++
	l.lastErr = err
}

func TestLoggingSuccess(t *testing.T) {
	cl := &captureLogger{}
	mw := logging.New(cl)
	handler := mw(func(evt *core.Event) error { return nil })

	if err := handler(nopEvent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cl.debugCalls != 1 {
		t.Errorf("expected 1 debug call, got %d", cl.debugCalls)
	}
	if cl.errorCalls != 0 {
		t.Errorf("expected 0 error calls, got %d", cl.errorCalls)
	}
}

func TestLoggingError(t *testing.T) {
	cl := &captureLogger{}
	want := errors.New("handler error")
	mw := logging.New(cl)
	handler := mw(func(evt *core.Event) error { return want })

	if err := handler(nopEvent); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if cl.errorCalls != 1 {
		t.Errorf("expected 1 error call, got %d", cl.errorCalls)
	}
	if !errors.Is(cl.lastErr, want) {
		t.Errorf("logged error = %v, want %v", cl.lastErr, want)
	}
}

func TestLoggingDefaultNil(t *testing.T) {
	// nil logger 使用 DefaultLogger()，不 panic
	mw := logging.New(nil)
	handler := mw(func(evt *core.Event) error { return nil })
	if err := handler(nopEvent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── 中间件链组合 ──────────────────────────────────────────────────

func TestMiddlewareChain(t *testing.T) {
	var order []string

	mw1 := func(h core.Handler) core.Handler {
		return func(evt *core.Event) error {
			order = append(order, "mw1-before")
			err := h(evt)
			order = append(order, "mw1-after")
			return err
		}
	}
	mw2 := func(h core.Handler) core.Handler {
		return func(evt *core.Event) error {
			order = append(order, "mw2-before")
			err := h(evt)
			order = append(order, "mw2-after")
			return err
		}
	}
	base := func(evt *core.Event) error {
		order = append(order, "handler")
		return nil
	}

	chained := core.Chain(base, mw1, mw2)
	if err := chained(nopEvent); err != nil {
		t.Fatal(err)
	}

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestRecovererWithRetry(t *testing.T) {
	var attempts int
	mw := core.Chain(
		func(evt *core.Event) error {
			attempts++
			if attempts < 3 {
				panic("transient panic")
			}
			return nil
		},
		retry.New(retry.Config{MaxRetries: 5, InitialInterval: time.Millisecond}),
		recoverer.New(),
	)

	if err := mw(nopEvent); err != nil {
		t.Errorf("unexpected error after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}
