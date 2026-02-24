// Package timeout 提供 handler 执行超时中间件
package timeout

import (
	"fmt"
	"time"

	"github.com/uniyakcom/yakevent/core"
)

// New 返回超时中间件。
// handler 执行超过 d 时返回超时错误；handler 自身的 error 正常透传。
// 注意: 超时后 handler 所在 goroutine 仍在后台运行直至完成，无法强制取消。
// 对于需要上下文取消的场景，请在 handler 内部使用 context.Context。
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
