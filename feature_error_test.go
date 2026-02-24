package yakevent

import (
	"errors"
	"testing"
)

// TestErrorHandling handler 返回错误正常传递
func TestErrorHandling(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	expectedErr := errors.New("handler error")
	bus.On("error.test", func(e *Event) error { return expectedErr })

	err := bus.Emit(&Event{Type: "error.test", Data: []byte("data")})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

// TestMultipleHandlerErrors 多 handler 返回错误，返回第一个错误
func TestMultipleHandlerErrors(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	err1 := errors.New("handler 1 error")
	err2 := errors.New("handler 2 error")
	bus.On("multi.error", func(e *Event) error { return err1 })
	bus.On("multi.error", func(e *Event) error { return err2 })

	err := bus.Emit(&Event{Type: "multi.error", Data: []byte("data")})
	if err == nil {
		t.Fatal("expected error from multiple handlers, got nil")
	}
	if !errors.Is(err, err1) && !errors.Is(err, err2) {
		t.Errorf("expected error containing err1 or err2, got %v", err)
	}
}

// TestOffInvalidID 取消不存在的 ID 不应 panic
func TestOffInvalidID(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	bus.Off(99999)

	id := bus.On("test", func(e *Event) error { return nil })
	bus.Off(id)
	bus.Off(id) // 重复取消
}

// TestEmitBatchWithErrors 批量发布时错误正常传递
func TestEmitBatchWithErrors(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	expectedErr := errors.New("batch error")
	bus.On("batch.error", func(e *Event) error { return expectedErr })

	events := []*Event{
		{Type: "batch.error", Data: []byte("1")},
		{Type: "batch.error", Data: []byte("2")},
		{Type: "batch.error", Data: []byte("3")},
	}
	err := bus.EmitBatch(events)
	if err == nil {
		t.Error("expected error from batch emit, got nil")
	}
}

// TestEmitMatchNoMatch 通配符无匹配时不报错
func TestEmitMatchNoMatch(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	bus.On("user.*", func(e *Event) error {
		t.Error("should not be called - pattern doesn't match")
		return nil
	})

	err := bus.EmitMatch(&Event{Type: "order.created", Data: []byte("data")})
	if err != nil {
		t.Errorf("EmitMatch with no matches should not error, got %v", err)
	}
}

// TestEmitNilEvent Emit nil 不 panic
func TestEmitNilEvent(t *testing.T) {
	bus, _ := ForSync()
	defer bus.Close()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Emit(nil) should not panic, got: %v", r)
		}
	}()
	_ = bus.Emit(nil)
}
