package retry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/middleware/retry"
)

func evt() *core.Event { return &core.Event{Type: "test"} }

// fastCfg returns a config with minimum wait intervals so tests run quickly.
func fastCfg() retry.Config {
	return retry.Config{
		MaxRetries:      3,
		InitialInterval: 1 * time.Microsecond,
		MaxInterval:     10 * time.Microsecond,
		Multiplier:      2.0,
	}
}

func TestRetrySuccessFirstTry(t *testing.T) {
	mw := retry.New(fastCfg())
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		return nil
	})
	if err := h(evt()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetrySuccessAfterRetries(t *testing.T) {
	cfg := fastCfg()
	mw := retry.New(cfg)
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err := h(evt()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryExhausted(t *testing.T) {
	cfg := fastCfg()
	mw := retry.New(cfg)
	want := errors.New("permanent")
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		return want
	})
	err := h(evt())
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}

func TestRetryShouldRetryFalseStopsEarly(t *testing.T) {
	cfg := fastCfg()
	cfg.ShouldRetry = func(err error) bool { return false }
	mw := retry.New(cfg)
	want := errors.New("not retryable")
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		return want
	})
	err := h(evt())
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
}

func TestRetryShouldRetryTrueRetriesAll(t *testing.T) {
	cfg := fastCfg()
	cfg.ShouldRetry = func(err error) bool { return true }
	mw := retry.New(cfg)
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		return errors.New("x")
	})
	h(evt()) //nolint:errcheck
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}

func TestRetryMaxRetriesZeroUsesDefault(t *testing.T) {
	mw := retry.New(retry.Config{
		InitialInterval: 1 * time.Microsecond,
		MaxInterval:     10 * time.Microsecond,
	})
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		return errors.New("x")
	})
	h(evt()) //nolint:errcheck
	if calls != 4 {
		t.Errorf("expected 4 calls with default MaxRetries=3, got %d", calls)
	}
}

func TestRetryMaxIntervalCapped(t *testing.T) {
	cfg := retry.Config{
		MaxRetries:      2,
		InitialInterval: 1 * time.Microsecond,
		MaxInterval:     1 * time.Microsecond,
		Multiplier:      100.0,
	}
	mw := retry.New(cfg)
	h := mw(func(e *core.Event) error { return errors.New("x") })
	start := time.Now()
	h(evt()) //nolint:errcheck
	if time.Since(start) > 100*time.Millisecond {
		t.Error("expected fast completion with capped interval")
	}
}

func TestRetryDefaultsNegativeMultiplier(t *testing.T) {
	cfg := retry.Config{
		MaxRetries:      1,
		InitialInterval: 1 * time.Microsecond,
		MaxInterval:     10 * time.Microsecond,
		Multiplier:      -1.0,
	}
	mw := retry.New(cfg)
	calls := 0
	h := mw(func(e *core.Event) error {
		calls++
		return errors.New("x")
	})
	h(evt()) //nolint:errcheck
	if calls != 2 {
		t.Errorf("expected 2 calls with MaxRetries=1, got %d", calls)
	}
}
