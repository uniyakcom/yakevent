// Package logging 提供结构化日志中间件。
//
// 通过 Logger 接口解耦日志实现：
//   - 默认使用 log/slog（标准库，零额外依赖）
//   - 可注入 zerolog、zap 等任意实现
package logging

import (
	"log/slog"
	"time"

	"github.com/uniyakcom/yakevent/core"
)

// Logger 可插拔日志接口，允许调用方注入任意日志库。
//
// Error 将 err 独立于 args 传递，使适配器实现更直观（无需从 args 中扫描 error 值）。
type Logger interface {
	// Debug 记录调试日志（高吞吐场景建议使用，避免淹没日志）
	Debug(msg string, args ...any)
	// Error 记录错误日志，err 为触发错误，args 为附加键值对
	Error(msg string, err error, args ...any)
}

// slogAdapter 将 *slog.Logger 适配为 Logger 接口（默认实现）
type slogAdapter struct{ l *slog.Logger }

func (a *slogAdapter) Debug(msg string, args ...any) { a.l.Debug(msg, args...) }
func (a *slogAdapter) Error(msg string, err error, args ...any) {
	a.l.Error(msg, append([]any{"error", err}, args...)...)
}

// DefaultLogger 返回基于 slog.Default() 的默认 Logger 实现。
func DefaultLogger() Logger {
	return &slogAdapter{l: slog.Default()}
}

// New 返回结构化日志中间件。
//
// logger 为 nil 时使用 DefaultLogger()（slog.Default()）。
// 调用方可注入任何实现了 Logger 接口的日志库：
//
//	// 使用 zerolog（示例，zerolog 由调用方引入）
//	type zerologAdapter struct{ zl zerolog.Logger }
//	func (a *zerologAdapter) Debug(msg string, args ...any)        { a.zl.Debug().Fields(args).Msg(msg) }
//	func (a *zerologAdapter) Error(msg string, err error, args ...any) {
//		a.zl.Error().Err(err).Fields(args).Msg(msg)
//	}
//
//	bus.Use(logging.New(&zerologAdapter{zl: zerolog.New(os.Stdout)}))
func New(logger Logger) core.Middleware {
	if logger == nil {
		logger = DefaultLogger()
	}
	return func(h core.Handler) core.Handler {
		return func(evt *core.Event) error {
			start := time.Now()
			err := h(evt)
			dur := time.Since(start)
			if err != nil {
				logger.Error("event handler failed", err,
					"type", evt.Type,
					"duration", dur,
				)
			} else {
				logger.Debug("event processed",
					"type", evt.Type,
					"duration", dur,
				)
			}
			return err
		}
	}
}
