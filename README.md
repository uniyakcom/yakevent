# yakevent

[![Go Version](https://img.shields.io/github/go-mod/go-version/uniyakcom/yakevent)](https://github.com/uniyakcom/yakevent/blob/main/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/uniyakcom/yakevent.svg)](https://pkg.go.dev/github.com/uniyakcom/yakevent)
[![Go Report Card](https://goreportcard.com/badge/github.com/uniyakcom/yakevent)](https://goreportcard.com/report/github.com/uniyakcom/yakevent)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Lint](https://github.com/uniyakcom/yakevent/actions/workflows/format.yml/badge.svg)](https://github.com/uniyakcom/yakevent/actions/workflows/format.yml)
[![Test](https://github.com/uniyakcom/yakevent/actions/workflows/test.yml/badge.svg)](https://github.com/uniyakcom/yakevent/actions/workflows/test.yml)

High-performance in-process event bus for Go — 3 implementations · 2 publish modes · 4-layer API, zero-alloc, zero-CAS, zero dependencies.

高性能 Go 进程内事件总线 — 3 种实现 + 2 种发布模式 + 4 层 API，零分配，零 CAS，零外部依赖。

## 特性

- **零外部依赖**: 仅依赖标准库
- **3 种 Bus 实现**: Sync（同步直调）/ Async（Per-P SPSC）/ Flow（Pipeline 流处理）
- **2 种发布模式**: `Emit`（安全路径，defer/recover 保护）/ `UnsafeEmit`（零保护，极致性能）
- **4 层 API**: 零配置 `New()` / 场景 `ForXxx()` / 字符串 `Scenario()` / 完全控制 `Option()`
- **零分配 Emit**: 全部三实现 0 B/op, 0 allocs/op
- **极致性能**: UnsafeEmit ~5.6 ns（178 M/s），Sync Emit ~14 ns，Async ~22 ns（44 M/s）
- **零 CAS 热路径**: Per-P SPSC ring，atomic Load/Store only（x86 ≈ 普通 MOV）
- **模式匹配**: 通配符 `*`（单层）和 `**`（多层）
- **可插拔中间件**: recoverer / retry / timeout / logging（Logger 接口，支持注入 zerolog/zap）

---

## 性能

### 基准对比

```bash
go test -bench="BenchmarkImpl" -benchmem -benchtime=1s -count=3 -run="^$"
```

#### Linux — Intel Xeon Platinum @ 2.50GHz (1C/2T, 2 vCPU)

| 场景 | ns/op | allocs/op |
|------|------:|----------:|
| **UnsafeEmit 单线程** | **5.6 ns** | 0 |
| **Sync Emit 单线程** | **14 ns** | 0 |
| **Async 单线程** | **22 ns** | 0 |
| **Async 高并发** | **21 ns** | 0 |
| **Flow 批量吞吐** | **43 M/s** | 0 |

---

## 快速开始

### 安装

```bash
go get github.com/uniyakcom/yakevent
```

### 包级 API（最简方式）

```go
import "github.com/uniyakcom/yakevent"

func main() {
    yakevent.On("user.created", func(e *yakevent.Event) error {
        fmt.Printf("User: %s\n", string(e.Data))
        return nil
    })

    yakevent.Emit(&yakevent.Event{
        Type: "user.created",
        Data: []byte("alice"),
    })
}
```

### 实例化（三预设任选）

```go
bus, _ := yakevent.ForAsync()   // Per-P SPSC 高并发
defer bus.Close()

bus.On("order.**", func(e *yakevent.Event) error {
    fmt.Printf("Order event: %s\n", e.Type)
    return nil
})

bus.Emit(&yakevent.Event{Type: "order.created", Data: []byte(`{"id":123}`)})
```

### 中间件

```go
import (
    "github.com/uniyakcom/yakevent"
    "github.com/uniyakcom/yakevent/middleware/recoverer"
    "github.com/uniyakcom/yakevent/middleware/logging"
    "github.com/uniyakcom/yakevent/middleware/timeout"
    "github.com/uniyakcom/yakevent/middleware/retry"
)

handler := yakevent.Chain(
    func(e *yakevent.Event) error {
        return processOrder(e)
    },
    recoverer.New(),                                    // panic → error
    logging.New(nil),                                   // 默认 slog
    timeout.New(5 * time.Second),                       // 超时保护
    retry.New(retry.Config{MaxRetries: 3}),             // 指数退避重试
)

bus.On("order.created", handler)
```

---

## 3 种 Bus 实现 + 2 种发布模式 + 4 层 API

### 3 种 Bus 实现

| 实现 | 核心技术 | 适用场景 | Emit | UnsafeEmit | 高并发 | error 返回 |
|------|---------|---------|:---:|:---:|:---:|:---:|
| **Sync** | 同步直调 + CoW atomic.Pointer | RPC 钩子、权限校验、API 中间件 | **14 ns** | **5.6 ns** | 72 ns | ✅ |
| **Async** | Per-P SPSC ring + RCU + 三级空转 | 事件总线、日志聚合、实时推送 | 22 ns | = Emit | **21 ns** | ❌ |
| **Flow** | MPSC ring + Stage Pipeline | ETL 流处理、窗口聚合、批量数据加载 | 66 ns | — | 108 ns | ❌ |

### 2 种发布模式

| 模式 | 说明 |
|------|------|
| **`Emit`** | 安全路径，捕获 handler panic，更新 Stats |
| **`UnsafeEmit`** | 零保护，panic 传播到调用方，不计数，极致性能 |

### 4 层 API

| 层级 | API | 说明 |
|------|-----|------|
| **L0 零配置** | `yakevent.New()` | 自动检测：≥4 核用 Async，<4 核用 Sync |
| **L1 场景** | `yakevent.ForSync()` / `ForAsync()` / `ForFlow()` | 选定实现 |
| **L2 字符串** | `yakevent.Scenario("async")` | 配置文件/环境变量驱动 |
| **L3 完全控制** | `yakevent.Option(profile)` | 自定义 Profile |
| **包级** | `yakevent.On()` / `yakevent.Emit()` | 全局 Sync 单例，零初始化 |

```go
// 包级 API（Sync 语义）
yakevent.On("event", handler)
yakevent.Emit(event)           // 安全路径，~14 ns
yakevent.UnsafeEmit(event)     // 零保护，~5.6 ns

// L1 三核心
bus, _ := yakevent.ForSync()     // 同步直调
bus, _ := yakevent.ForAsync()    // Per-P SPSC
bus, _ := yakevent.ForFlow()     // Pipeline

// L0 自动检测
bus, _ := yakevent.New()         // ≥4 核 → Async，<4 核 → Sync

// L2 字符串配置
bus, _ := yakevent.Scenario("async")

// L3 完全控制
bus, _ := yakevent.Option(&yakevent.Profile{Name: "async", Conc: 10000, TPS: 50000})
```

---

## 中间件

### 内置中间件

```go
import (
    "github.com/uniyakcom/yakevent/middleware/recoverer"  // panic → error
    "github.com/uniyakcom/yakevent/middleware/retry"      // 指数退避重试
    "github.com/uniyakcom/yakevent/middleware/timeout"    // handler 超时
    "github.com/uniyakcom/yakevent/middleware/logging"    // 结构化日志
)
```

### 可插拔日志（logging.Logger 接口）

`logging.New(logger)` 接受任何实现了 `Logger` 接口的日志库：

```go
// Logger 接口（仅 2 个方法）
type Logger interface {
    Debug(msg string, args ...any)
    Error(msg string, err error, args ...any)
}
```

**默认**（`nil` → `slog.Default()`）：

```go
logging.New(nil)
```

**接入 zerolog**（zerolog 由调用方引入，yakevent 本体零依赖）：

```go
type zerologAdapter struct{ zl zerolog.Logger }
func (a *zerologAdapter) Debug(msg string, args ...any)        { a.zl.Debug().Fields(args).Msg(msg) }
func (a *zerologAdapter) Error(msg string, err error, args ...any) {
    a.zl.Error().Err(err).Fields(args).Msg(msg)
}

handler := yakevent.Chain(myHandler, logging.New(&zerologAdapter{zl: zerolog.New(os.Stdout)}))
```

---

## 架构设计

### 目录结构

```
yakevent/
├── core/                     # 核心接口（Bus / Event / Handler / Middleware）
│   ├── interfaces.go
│   └── matcher.go           # TrieMatcher 通配符匹配
├── middleware/               # 内置中间件
│   ├── recoverer/           # panic → error 恢复
│   ├── retry/               # 指数退避重试
│   ├── timeout/             # handler 执行超时
│   └── logging/             # 可插拔结构化日志（Logger 接口）
├── optimize/                 # Profile → Advisor → Factory
├── internal/impl/           # 三实现
│   ├── sync/                # 同步直调 + SyncAsync 子模式
│   ├── async/               # Per-P SPSC Bus
│   └── flow/                # Pipeline 批处理 Bus
├── internal/support/        # 基础设施
│   ├── noop/                # 可切换锁（nil mutex = 零开销）
│   ├── pool/                # 事件对象池 + Arena 内存管理
│   ├── sched/               # SPSC 分片调度器（Sync 异步 + Async 共用）
│   ├── spsc/                # Per-P SPSC ring buffer
│   └── wpool/               # Worker pool（分片 channel + 安全关闭）
├── util/                    # PerCPUCounter 工具
└── api.go                   # 统一 API 入口
```

### SPSC 共享架构

Sync 异步模式与 Async 共用同一 `sched.ShardedScheduler`：

- **Producer**: `procPin → SPSC Enqueue (~3 ns) → procUnpin → wake`
- **Consumer**: `SPSC Dequeue → dispatch(snap, handlers) → processed++`
- **Worker**: 三级自适应空转（PAUSE spin → Gosched → channel park）

---

## 开发与测试

### 快速验证

```bash
go build ./...              # 编译
go vet ./...                # 静态分析
go test ./... -count=1      # 功能测试
go test -race ./... -short  # 竞态检测
```

### 测试套件

| 文件 | 类型 | 说明 |
|------|------|------|
| `alloc_test.go` | 正确性 | Arena 零分配、EventPool 正确性、模式匹配零分配（`testing.AllocsPerRun`）|
| `package_api_test.go` | 功能 | 包级 API（On/Off/Emit/EmitMatch/EmitBatch/Stats） |
| `scenario_test.go` | 功能 | ForSync/ForAsync/ForFlow 全场景 |
| `feature_error_test.go` | 功能 | 单/多 handler 错误、批量错误 |
| `feature_concurrent_test.go` | 功能 | On/Off/Emit 竞态、嵌套订阅、并发 Close |
| `edge_cases_test.go` | 边界 | 零 handler、大数据、特殊字符、重复取消订阅 |
| `middleware/middleware_test.go` | 功能 | 中间件组合（recoverer/retry/timeout/logging） |
| `internal/impl/flow/bus_test.go` | 功能 | Flow Bus 订阅/批次/超时触发 |
| `impl_bench_test.go` | 基准 | 三实现核心性能守卫（Sync/Async/Flow 单线程 + 高并发 + 批量） |
| `util/util_bench_test.go` | 基准 | PerCPUCounter Add/Read 性能 |

### 性能基准

```bash
# 核心回归（快速，~10s）
go test -bench="BenchmarkImpl" -benchtime=1s -count=3 -run=^$

# 完整基准
go test -bench=. -benchmem -benchtime=1s -count=3 -run=^$

# 运行基准测试并保存结果到 bench_{os}_{cores}c{threads}t.txt
./bench.sh

# 自定义参数: benchtime=3s, count=5
./bench.sh 3s 5

# 性能回归检查（首次无基线时自动创建，阈值默认 15%）
./bench_guard.sh

# 强制重建基线（硬件变更后）
./bench_guard.sh force
```

`bench_guard.sh` 守卫 `BenchmarkImplSync` / `BenchmarkImplAsync` / `BenchmarkImplFlow` 三个单线程稳定基准的 ns/op 中位数，超出阈值退出码为 1。每次运行生成两个持久文件（按平台区分）：

| 文件 | 内容 | 更新时机 |
|---|---|---|
| `bench_guard_baseline_{os}.txt` | 各基准中位数（用于对比） | 首次运行 / `force` |
| `bench_guard_raw_{os}.txt` | 原始测试数据 + 对比分析结果 | 每次运行覆盖 |

CI 基准测试由 `bench.yml` 驱动，4 平台并发：

| 平台 | Runner | 架构 |
|---|---|---|
| `ubuntu-latest` | Ubuntu 24.04 | x86_64 |
| `windows-latest` | Windows Server 2025 | x86_64 |
| `ubuntu-24.04-arm` | Ubuntu 24.04 | aarch64 |
| `macos-latest` | macOS 15（Apple M） | aarch64 |

PR 时同机器对比 HEAD 与 base，`benchstat` 输出差异百分比。结果归档到独立的 `bench` 分支：

```
bench 分支/
├── main/                          # push to main 后自动更新
│   ├── ubuntu-latest-amd64.txt
│   ├── windows-latest-amd64.txt
│   ├── ubuntu-24.04-arm-arm64.txt
│   └── macos-latest-arm64.txt
├── pr-{N}/                        # PR #{N} 触发时写入
│   ├── {os}-{arch}.txt           # HEAD 结果
│   ├── {os}-{arch}-base.txt      # base commit 结果
│   └── {os}-{arch}-diff.txt      # benchstat 差异对比
└── manual/YYYYMMDD/               # 手动触发时写入
    └── {os}-{arch}.txt
```

---

## 许可证

MIT License
