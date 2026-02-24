// Package recoverer 提供 panic 恢复中间件
package recoverer

import (
	"fmt"

	"github.com/uniyakcom/yakevent/core"
)

// PanicError 封装 handler panic 的错误类型
type PanicError struct {
	Value interface{}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("yakevent: handler panic: %v", e.Value)
}

// New 返回 panic 恢复中间件。
// handler 发生 panic 时，panic 值被捕获并以 *PanicError 形式返回，
// 防止 panic 传播到调用方 goroutine。
func New() core.Middleware {
	return func(h core.Handler) core.Handler {
		return func(evt *core.Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = &PanicError{Value: r}
				}
			}()
			return h(evt)
		}
	}
}
