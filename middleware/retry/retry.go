// Package retry 提供指数退避重试中间件
package retry

import (
	"time"

	"github.com/uniyakcom/yakevent/core"
)

// Config 重试配置
type Config struct {
	// MaxRetries 最大重试次数（不含首次执行，默认 3）
	MaxRetries int

	// InitialInterval 首次重试等待时间（默认 100ms）
	InitialInterval time.Duration

	// MaxInterval 最大等待时间上限（默认 10s）
	MaxInterval time.Duration

	// Multiplier 每次重试的退避倍数（默认 2.0）
	Multiplier float64

	// ShouldRetry 可选：判断是否重试特定错误（nil 则对所有错误重试）
	ShouldRetry func(err error) bool
}

func (c *Config) defaults() {
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.InitialInterval <= 0 {
		c.InitialInterval = 100 * time.Millisecond
	}
	if c.MaxInterval <= 0 {
		c.MaxInterval = 10 * time.Second
	}
	if c.Multiplier <= 0 {
		c.Multiplier = 2.0
	}
}

// New 返回指数退避重试中间件。
// handler 返回 error 且未超出 MaxRetries 时，以 InitialInterval × Multiplier^n 退避重试。
func New(cfg Config) core.Middleware {
	cfg.defaults()
	return func(h core.Handler) core.Handler {
		return func(evt *core.Event) error {
			interval := cfg.InitialInterval
			for attempt := 0; ; attempt++ {
				err := h(evt)
				if err == nil {
					return nil
				}
				if attempt >= cfg.MaxRetries {
					return err
				}
				if cfg.ShouldRetry != nil && !cfg.ShouldRetry(err) {
					return err
				}
				time.Sleep(interval)
				next := time.Duration(float64(interval) * cfg.Multiplier)
				if next > cfg.MaxInterval {
					next = cfg.MaxInterval
				}
				interval = next
			}
		}
	}
}
