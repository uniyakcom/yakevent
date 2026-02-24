// Package core 提供事件总线核心接口定义
package core

import (
	"time"
)

// Event 事件（字段按大小降序排列 → 减少编译器 padding → 最小化结构体体积）
type Event struct {
	Data      []byte            // 24 bytes (hot: 事件数据，读频率最高)
	Type      string            // 16 bytes (hot: 事件类型，用于路由)
	ID        string            // 16 bytes (warm)
	Source    string            // 16 bytes (cold)
	Metadata  map[string]string // 8 bytes  (cold: map 指针，含 GC 扫描开销)
	Timestamp time.Time         // 24 bytes (cold: 含 wall+ext+loc 指针)
}

// Handler 事件处理器
type Handler func(*Event) error

// Middleware 中间件函数类型，用于包装 Handler 形成处理链。
type Middleware func(Handler) Handler

// Chain 将多个中间件应用到 Handler（从左到右顺序执行）。
// mws[0] 最外层，mws[last] 最靠近 handler。
func Chain(h Handler, mws ...Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// PanicHandler panic 回调（可选，用户注册后接收 panic 通知）
type PanicHandler func(recovered interface{}, evt *Event)

// Stats 事件总线运行时统计
type Stats struct {
	Emitted   int64 // 已发布事件总数
	Processed int64 // 已处理事件总数（handler 执行完成）
	Panics    int64 // handler panic 次数
	Depth     int64 // 当前队列积压深度（仅 Ring Buffer 实现有值）
}

// Bus 事件总线接口
type Bus interface {
	On(pattern string, handler Handler) uint64
	Off(id uint64)
	Emit(evt *Event) error
	UnsafeEmit(evt *Event) error
	EmitMatch(evt *Event) error
	UnsafeEmitMatch(evt *Event) error
	EmitBatch(events []*Event) error
	EmitMatchBatch(events []*Event) error
	Stats() Stats
	Close()
	Drain(timeout time.Duration) error
}

// Flusher 支持手动刷新的 Bus（如 Flow 批处理模式）
type Flusher interface {
	Flush() error
}

// ErrorReporter 支持异步错误查询的 Bus（如 Sync 异步模式）
type ErrorReporter interface {
	LastError() error
	ClearError()
}

// Prewarmer 支持预热的 Bus（如 Sync 模式）
type Prewarmer interface {
	Prewarm(eventTypes []string)
}

// BatchStatter 支持批处理统计的 Bus（如 Flow 模式）
type BatchStatter interface {
	BatchStats() (processed, batches uint64)
}
