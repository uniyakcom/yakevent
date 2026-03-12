// Package timeout 提供 handler 执行超时中间件
package timeout

import (
	"errors"
	"fmt"
	"time"

	"github.com/uniyakcom/yakevent/core"
)

// ErrPoolFull 由 NewWithPool 在 worker pool 已满时返回。
var ErrPoolFull = errors.New("yakevent: timeout: worker pool saturated")

// Pool 定义 NewWithPool 所需的 worker pool 接口。
// *wpool.Pool 实现此接口，用户也可注入自定义实现。
type Pool interface {
	SubmitFunc(fn func()) bool
}

// New 返回超时中间件。
// handler 执行超过 d 时返回超时错误；handler 自身的 error 正常透传。
// 注意: 超时后 handler 所在 goroutine 仍在后台运行直至完成，无法强制取消。
// 对于需要上下文取消的场景，请在 handler 内部使用 context.Context。
// 若需限制并发 goroutine 数量，请改用 NewWithPool。
func New(d time.Duration) core.Middleware {
	return func(h core.Handler) core.Handler {
		return func(evt *core.Event) error {
			done := make(chan error, 1)
			go func() {
				done <- h(evt)
			}()
			select {
			case err := <-done:
				return err
			case <-time.After(d):
				return fmt.Errorf("yakevent: handler timeout after %v (event type: %q)", d, evt.Type)
			}
		}
	}
}

// NewWithPool 与 New 语义相同，但在提供的 pool 中执行 handler，
// 以限制并发 goroutine 的最大数量。
// 若 pool 已满（Submit 返回 false），立即返回 ErrPoolFull 而非死等。
//
// 示例（使用内置 wpool）:
//
//	import "github.com/uniyakcom/yakevent/internal/support/wpool"
//	pool := wpool.New(runtime.NumCPU(), 1000)
//	defer pool.Stop()
//	handler = yakevent.Chain(handler, timeout.NewWithPool(5*time.Second, pool))
func NewWithPool(d time.Duration, pool Pool) core.Middleware {
	return func(h core.Handler) core.Handler {
		return func(evt *core.Event) error {
			done := make(chan error, 1)
			if !pool.SubmitFunc(func() { done <- h(evt) }) {
				return ErrPoolFull
			}
			select {
			case err := <-done:
				return err
			case <-time.After(d):
				return fmt.Errorf("yakevent: handler timeout after %v (event type: %q)", d, evt.Type)
			}
		}
	}
}
