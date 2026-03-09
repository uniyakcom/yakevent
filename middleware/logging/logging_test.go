package logging_test

import (
	"errors"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/middleware/logging"
)

// captureLogger 记录最近一次 Debug / Error 调用。
type captureLogger struct {
	debugMsg  string
	debugArgs []any
	errorMsg  string
	errorErr  error
	errorArgs []any
}

func (l *captureLogger) Debug(msg string, args ...any) {
	l.debugMsg = msg
	l.debugArgs = args
}
func (l *captureLogger) Error(msg string, err error, args ...any) {
	l.errorMsg = msg
	l.errorErr = err
	l.errorArgs = args
}

func makeEvent(typ string) *core.Event {
	return &core.Event{Type: typ, Timestamp: time.Now()}
}

func TestLoggingMiddlewareSuccess(t *testing.T) {
	log := &captureLogger{}
	mw := logging.New(log)
	h := mw(func(e *core.Event) error { return nil })
	if err := h(makeEvent("user.created")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.debugMsg == "" {
		t.Error("expected Debug call, got none")
	}
	if log.errorMsg != "" {
		t.Error("unexpected Error call on success path")
	}
}

func TestLoggingMiddlewareError(t *testing.T) {
	log := &captureLogger{}
	mw := logging.New(log)
	want := errors.New("boom")
	h := mw(func(e *core.Event) error { return want })
	if err := h(makeEvent("order.failed")); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if log.errorMsg == "" {
		t.Error("expected Error call, got none")
	}
	if !errors.Is(log.errorErr, want) {
		t.Errorf("expected error arg %v, got %v", want, log.errorErr)
	}
	if log.debugMsg != "" {
		t.Error("unexpected Debug call on error path")
	}
}

func TestLoggingMiddlewareEventTypeInArgs(t *testing.T) {
	log := &captureLogger{}
	mw := logging.New(log)
	h := mw(func(e *core.Event) error { return nil })
	_ = h(makeEvent("payment.settled"))
	foundType := false
	for _, a := range log.debugArgs {
		if s, ok := a.(string); ok && s == "payment.settled" {
			foundType = true
		}
	}
	if !foundType {
		t.Errorf("event type not present in debug args: %v", log.debugArgs)
	}
}

func TestLoggingMiddlewareErrorEventTypeInArgs(t *testing.T) {
	log := &captureLogger{}
	mw := logging.New(log)
	h := mw(func(e *core.Event) error { return errors.New("x") })
	_ = h(makeEvent("payment.failed"))
	foundType := false
	for _, a := range log.errorArgs {
		if s, ok := a.(string); ok && s == "payment.failed" {
			foundType = true
		}
	}
	if !foundType {
		t.Errorf("event type not present in error args: %v", log.errorArgs)
	}
}

func TestLoggingMiddlewareNilLoggerUsesDefault(t *testing.T) {
	// nil logger → DefaultLogger() should not panic
	mw := logging.New(nil)
	h := mw(func(e *core.Event) error { return nil })
	if err := h(makeEvent("test")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoggingMiddlewareNilLoggerErrorPath(t *testing.T) {
	mw := logging.New(nil)
	h := mw(func(e *core.Event) error { return errors.New("fail") })
	if err := h(makeEvent("test")); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoggingMiddlewareDurationRecorded(t *testing.T) {
	log := &captureLogger{}
	mw := logging.New(log)
	h := mw(func(e *core.Event) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	_ = h(makeEvent("slow"))
	foundDur := false
	for _, a := range log.debugArgs {
		if d, ok := a.(time.Duration); ok && d >= time.Millisecond {
			foundDur = true
		}
	}
	if !foundDur {
		t.Errorf("duration not present in debug args: %v", log.debugArgs)
	}
}

func TestDefaultLoggerNotNil(t *testing.T) {
	l := logging.DefaultLogger()
	if l == nil {
		t.Fatal("DefaultLogger() returned nil")
	}
}
