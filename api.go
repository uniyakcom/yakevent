// Package yakevent 提供高性能进程内事件总线
//
// 三种工作模式:
//   - Sync  (~15ns/op): 同步直调，RPC/中间件链路，error 立即返回
//   - Async (~33ns/op): Per-P SPSC 异步高吞吐，发布订阅/日志聚合
//   - Flow  (~70ns/op): 多阶段 Pipeline 批处理，ETL/窗口聚合
//
// 快速开始:
//
//	// 使用包级默认 Bus（Sync 模式）
//	yakevent.On("user.created", func(evt *yakevent.Event) error {
//	    fmt.Println("new user:", evt.Type)
//	    return nil
//	})
//	yakevent.Emit(&yakevent.Event{Type: "user.created"})
//
//	// 或自定义模式
//	bus, _ := yakevent.ForAsync()
//	defer bus.Drain(5 * time.Second)
package yakevent

import (
	"time"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/optimize"
)

// ═══════════════════════════════════════════════════════════════════
// 类型别名（方便外部直接使用 yakevent.XXX，无需二级引用）
// ═══════════════════════════════════════════════════════════════════

type (
	// Bus 事件总线接口
	Bus = core.Bus
	// Event 事件
	Event = core.Event
	// Handler 事件处理函数
	Handler = core.Handler
	// Middleware 中间件
	Middleware = core.Middleware
	// PanicHandler panic 回调
	PanicHandler = core.PanicHandler
	// Stats 运行时统计
	Stats = core.Stats
	// Profile 优化场景配置
	Profile = optimize.Profile
)

// ═══════════════════════════════════════════════════════════════════
// Bus 工厂函数
// ═══════════════════════════════════════════════════════════════════

// New 零配置创建 Bus — 自动根据 CPU 核心数选择最优实现
//   - >= 4 cores → Async（高并发最优）
//   - <  4 cores → Sync（低延迟最优）
func New() (Bus, error) { return Option(optimize.AutoDetect()) }

// ForSync 创建同步 Bus（~15ns/op，低延迟，适合 RPC/中间件链路）
func ForSync() (Bus, error) { return Option(optimize.Sync()) }

// ForAsync 创建 Per-P SPSC 异步 Bus（~33ns/op，高吞吐，适合发布订阅）
func ForAsync() (Bus, error) { return Option(optimize.Async()) }

// ForFlow 创建批处理 Pipeline Bus（~70ns/op，批次处理，适合 ETL）
func ForFlow() (Bus, error) { return Option(optimize.Flow()) }

// Scenario 按预设名称创建 Bus（"sync"/"async"/"flow"）
func Scenario(name string) (Bus, error) { return Option(optimize.Preset(name)) }

// Option 通过 Profile 创建 Bus（最灵活的配置路径）
func Option(p *Profile) (Bus, error) {
	if p == nil {
		p = optimize.Sync()
	}
	return optimize.Build(optimize.NewAdvisor().Advise(p))
}

// ═══════════════════════════════════════════════════════════════════
// 包级默认 Bus（惰性初始化，Sync 模式）
// ═══════════════════════════════════════════════════════════════════

var defaultBus Bus

func init() {
	b, err := ForSync()
	if err != nil {
		panic("yakevent: failed to init default bus: " + err.Error())
	}
	defaultBus = b
}

// Default 返回包级默认 Bus（Sync 模式，进程生命周期内有效）
func Default() Bus { return defaultBus }

// ═══════════════════════════════════════════════════════════════════
// 包级便捷函数（代理默认 Bus）
// ═══════════════════════════════════════════════════════════════════

// On 订阅事件，返回订阅 ID（用于 Off 取消订阅）
func On(pattern string, handler Handler) uint64 { return defaultBus.On(pattern, handler) }

// Off 取消订阅
func Off(id uint64) { defaultBus.Off(id) }

// Emit 发布事件（带 panic 保护）
func Emit(evt *Event) error { return defaultBus.Emit(evt) }

// UnsafeEmit 发布事件（零保护，极致性能）
func UnsafeEmit(evt *Event) error { return defaultBus.UnsafeEmit(evt) }

// EmitMatch 发布事件（支持通配符匹配）
func EmitMatch(evt *Event) error { return defaultBus.EmitMatch(evt) }

// UnsafeEmitMatch 通配符匹配发布（零保护，极致性能）
func UnsafeEmitMatch(evt *Event) error { return defaultBus.UnsafeEmitMatch(evt) }

// EmitBatch 批量发布事件
func EmitBatch(events []*Event) error { return defaultBus.EmitBatch(events) }

// EmitMatchBatch 批量发布事件（通配符匹配）
func EmitMatchBatch(events []*Event) error { return defaultBus.EmitMatchBatch(events) }

// GetStats 返回默认 Bus 运行时统计
func GetStats() Stats { return defaultBus.Stats() }

// Drain 优雅关闭默认 Bus（等待队列排空或超时）
func Drain(timeout time.Duration) error { return defaultBus.Drain(timeout) }

// Chain 将多个中间件组合应用到 Handler
// 用法: wrapped := yakevent.Chain(myHandler, logging.New(nil), recoverer.New())
func Chain(h Handler, mws ...Middleware) Handler { return core.Chain(h, mws...) }
