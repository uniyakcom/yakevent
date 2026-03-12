# yakevent

[![Go Version](https://img.shields.io/github/go-mod/go-version/uniyakcom/yakevent)](https://github.com/uniyakcom/yakevent/blob/main/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/uniyakcom/yakevent.svg)](https://pkg.go.dev/github.com/uniyakcom/yakevent)
[![Go Report Card](https://goreportcard.com/badge/github.com/uniyakcom/yakevent)](https://goreportcard.com/report/github.com/uniyakcom/yakevent)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Lint](https://github.com/uniyakcom/yakevent/actions/workflows/format.yml/badge.svg)](https://github.com/uniyakcom/yakevent/actions/workflows/format.yml)
[![Test](https://github.com/uniyakcom/yakevent/actions/workflows/test.yml/badge.svg)](https://github.com/uniyakcom/yakevent/actions/workflows/test.yml)


**English** | [中文](README.zh.md)

High-performance in-process event bus for Go — 3 implementations · 2 publish modes · 4-layer API, zero-alloc, zero-CAS, zero dependencies.

## Features

- **Zero external dependencies**: standard library only
- **3 Bus implementations**: Sync (direct call) / Async (Per-P SPSC) / Flow (pipeline processing)
- **2 publish modes**: `Emit` (safe path, defer/recover protection) / `UnsafeEmit` (zero protection, maximum performance)
- **4-layer API**: zero-config `New()` / preset `ForXxx()` / string `Scenario()` / full control `Option()`
- **Zero-alloc Emit**: all three implementations — 0 B/op, 0 allocs/op
- **Extreme performance**: UnsafeEmit ~3.8 ns (265 M/s), Sync Emit ~14 ns, Async ~23 ns (44 M/s)
- **Zero-CAS hot path**: Per-P SPSC ring, atomic Load/Store only (x86 ≈ plain MOV)
- **Pattern matching**: wildcards `*` (single level) and `**` (multi-level)
- **Pluggable middleware**: recoverer / retry / timeout / logging (Logger interface, supports zerolog/zap injection)

---

## Performance

### Benchmark comparison

```bash
./bench.sh
```

#### Linux — Intel Xeon E-2186G @ 3.80GHz (6C/12T)<sup>[1]</sup>

| Scenario | ns/op | M/s | allocs/op |
|----------|------:|----:|----------:|
| **UnsafeEmit single-thread** | **3.8 ns** | 265 | 0 |
| **Sync Emit single-thread** | **14 ns** | 72 | 0 |
| **Async single-thread** | **23 ns** | 44 | 0 |
| **Async high concurrency** | **16 ns** | 62 | 0 |
| **Flow single-thread** | **74 ns** | 13 | 0 |
| **Flow high concurrency** | **94 ns** | 10 | 0 |
| **Scenario batch insert** | **49 µs/op** | 20 | 0 |

<sup>[1]</sup> Source: [bench_linux_6c12t.txt](bench_linux_6c12t.txt), `go test -benchtime=1s -count=3`, Go 1.25.7, Linux 6.17, bare-metal local. CI Runner data: see [bench](#bench-branch-archive) branch.

---

## Quick Start

### Install

```bash
go get github.com/uniyakcom/yakevent
```

### Package-level API (simplest)

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

### Instantiation (choose from three presets)

```go
bus, _ := yakevent.ForAsync()   // Per-P SPSC high concurrency
defer bus.Close()

bus.On("order.**", func(e *yakevent.Event) error {
    fmt.Printf("Order event: %s\n", e.Type)
    return nil
})

bus.Emit(&yakevent.Event{Type: "order.created", Data: []byte(`{"id":123}`)})
```

### Middleware

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
    logging.New(nil),                                   // default slog
    timeout.New(5 * time.Second),                       // timeout protection
    retry.New(retry.Config{MaxRetries: 3}),             // exponential backoff retry
)

bus.On("order.created", handler)
```

---

## 3 Bus Implementations · 2 Publish Modes · 4-layer API

### 3 Bus Implementations

| Implementation | Core Technology | Use Case | Emit | UnsafeEmit | High Concurrency | error return |
|----------------|----------------|----------|:----:|:----------:|:----------------:|:------------:|
| **Sync** | Direct call + CoW atomic.Pointer | RPC hooks, auth checks, API middleware | **14 ns** | **3.8 ns** | 20 ns | ✅ |
| **Async** | Per-P SPSC ring + RCU + 3-tier spinning | Event bus, log aggregation, real-time push | 23 ns | = Emit | **16 ns** | ❌ |
| **Flow** | MPSC ring + Stage Pipeline | ETL streaming, window aggregation, batch loading | 74 ns | — | 94 ns | ❌ |

<sup>Source: [`bench_linux_6c12t.txt`](bench_linux_6c12t.txt), Intel Xeon E-2186G @ 3.80GHz 6C/12T</sup>

### 2 Publish Modes

| Mode | Description |
|------|-------------|
| **`Emit`** | Safe path — captures handler panics, updates Stats |
| **`UnsafeEmit`** | Zero protection — panic propagates to caller, no counting, maximum performance |

### 4-layer API

| Layer | API | Description |
|-------|-----|-------------|
| **L0 zero-config** | `yakevent.New()` | Auto-detect: ≥4 cores → Async, <4 cores → Sync |
| **L1 preset** | `yakevent.ForSync()` / `ForAsync()` / `ForFlow()` | Choose implementation |
| **L2 string** | `yakevent.Scenario("async")` | Config file / env-var driven |
| **L3 full control** | `yakevent.Option(profile)` | Custom Profile |
| **Package-level** | `yakevent.On()` / `yakevent.Emit()` | Global Sync singleton, zero init |

```go
// Package-level API (Sync semantics)
yakevent.On("event", handler)
yakevent.Emit(event)           // safe path, ~14 ns
yakevent.UnsafeEmit(event)     // zero protection, ~5.6 ns

// L1 three cores
bus, _ := yakevent.ForSync()     // synchronous direct call
bus, _ := yakevent.ForAsync()    // Per-P SPSC
bus, _ := yakevent.ForFlow()     // Pipeline

// L0 auto-detect
bus, _ := yakevent.New()         // ≥4 cores → Async, <4 cores → Sync

// L2 string config
bus, _ := yakevent.Scenario("async")

// L3 full control
bus, _ := yakevent.Option(&yakevent.Profile{Name: "async", Conc: 10000, TPS: 50000})
```

---

## Middleware

### Built-in middleware

```go
import (
    "github.com/uniyakcom/yakevent/middleware/recoverer"  // panic → error
    "github.com/uniyakcom/yakevent/middleware/retry"      // exponential backoff retry
    "github.com/uniyakcom/yakevent/middleware/timeout"    // handler timeout
    "github.com/uniyakcom/yakevent/middleware/logging"    // structured logging
)
```

### Pluggable logging (logging.Logger interface)

`logging.New(logger)` accepts any logger implementing the `Logger` interface:

```go
// Logger interface (2 methods only)
type Logger interface {
    Debug(msg string, args ...any)
    Error(msg string, err error, args ...any)
}
```

**Default** (`nil` → `slog.Default()`):

```go
logging.New(nil)
```

**Integrate [yaklog](https://github.com/uniyakcom/yaklog)** (yaklog is the yak\* ecosystem structured logger, also zero external deps):

```go
import (
    "github.com/uniyakcom/yaklog"
    "github.com/uniyakcom/yakevent/middleware/logging"
)

// yaklogAdapter adapts *yaklog.Logger to the logging.Logger interface.
type yaklogAdapter struct{ l *yaklog.Logger }

func (a *yaklogAdapter) Debug(msg string, args ...any) {
    e := a.l.Debug().Msg(msg)
    for i := 0; i+1 < len(args); i += 2 {
        k, _ := args[i].(string)
        e = e.Any(k, args[i+1])
    }
    e.Send()
}

func (a *yaklogAdapter) Error(msg string, err error, args ...any) {
    e := a.l.Error().Err(err).Msg(msg)
    for i := 0; i+1 < len(args); i += 2 {
        k, _ := args[i].(string)
        e = e.Any(k, args[i+1])
    }
    e.Send()
}

// Create a yaklog Logger (Console output, Debug level)
yl := yaklog.New(yaklog.Options{
    Level: yaklog.Debug,
    Out:   yaklog.Console(),
})

handler := yakevent.Chain(myHandler, logging.New(&yaklogAdapter{l: yl}))
```

---

## Architecture

### Shared SPSC architecture

Sync async mode and Async share the same `sched.ShardedScheduler`:

- **Producer**: `procPin → SPSC Enqueue (~3 ns) → procUnpin → wake`
- **Consumer**: `SPSC Dequeue → dispatch(snap, handlers) → processed++`
- **Worker**: 3-tier adaptive spinning (PAUSE spin → Gosched → channel park)

---

## Development & Testing

### Quick verify

```bash
go build ./...              # compile
go vet ./...                # static analysis
go test ./... -count=1      # functional tests
go test -race ./... -short  # race detection
```

### Scripts

#### `bench.sh` — Benchmark recorder

Runs the full benchmark suite and writes results to `bench_{os}_{cores}c{threads}t.txt` (e.g. `bench_linux_6c12t.txt`). The file header automatically includes timestamp, kernel version, Go version, and CPU model.

```bash
# Default: benchtime=1s, count=3
./bench.sh

# Custom duration and repeat count
./bench.sh 3s 5

# Usage: ./bench.sh [benchtime] [count]
#   benchtime  duration per benchmark run (default: 1s)
#   count      repetitions per benchmark for benchstat statistics (default: 3)
```

**Output file naming**: `bench_<os>_<cores>c<threads>t.txt`
- `<os>`: `linux` / `darwin` / `windows`
- `<cores>`: physical core count
- `<threads>`: logical thread count

---

#### `bench_guard.sh` — Performance regression guard

Compares the ns/op median of `BenchmarkImplSync` / `BenchmarkImplAsync` / `BenchmarkImplFlow` against a stored baseline. Exits with code 1 if any benchmark regresses beyond the threshold.

```bash
# Regression check (auto-creates baseline on first run)
./bench_guard.sh

# Force rebuild baseline (after hardware change or baseline expiry)
./bench_guard.sh force

# Custom threshold (default: 15%)
BENCH_THRESHOLD=20 ./bench_guard.sh

# Custom benchmark parameters
BENCH_TIME=3s BENCH_COUNT=7 ./bench_guard.sh
```

| Environment variable | Default | Description |
|----------------------|---------|-------------|
| `BENCH_THRESHOLD` | `15` | ns/op regression percentage threshold |
| `BENCH_TIME` | `2s` | Duration per benchmark run |
| `BENCH_COUNT` | `5` | Repetitions (median is used for comparison) |

Persistent files (platform-specific):

| File | Content | Updated when |
|------|---------|--------------|
| `bench_guard_baseline_{os}.txt` | Median ns/op per benchmark (used for comparison) | First run / `force` |
| `bench_guard_raw_{os}.txt` | Raw test data + comparison analysis | Overwritten each run |

> **Note**: This script is for same-environment regression detection only. Cross-Runner / different-hardware comparisons are not meaningful.

---

#### `lint.sh` — Code quality check

```bash
./lint.sh          # Full check: gofmt -s -w + go vet + golangci-lint
./lint.sh --vet    # gofmt -s -w + go vet only (skip golangci-lint)
./lint.sh --fix    # gofmt -s -w + vet + golangci-lint --fix (auto-fix)
./lint.sh --fmt    # Format only (gofmt -s -w), no vet/lint
./lint.sh --test   # Quick test: go test ./... -race -count=1
```

> `gofmt -s -w` runs automatically on all paths except `--test` and `--fmt`. The `-s` flag simplifies redundant composite literals, slice expressions, and similar constructs.

---

#### CI benchmarks (`bench.yml`)

4 platforms run in parallel. On PRs, HEAD is compared against base and `benchstat` diff percentages are reported. Results are archived to the `bench` branch:

| Platform | Runner | Arch |
|----------|--------|------|
| `ubuntu-latest` | Ubuntu 24.04 | x86_64 |
| `windows-latest` | Windows Server 2025 | x86_64 |
| `ubuntu-24.04-arm` | Ubuntu 24.04 | aarch64 |
| `macos-latest` | macOS 15 (Apple M) | aarch64 |

### bench branch archive

```
bench branch/
├── main/                          # auto-updated on push to main
│   ├── ubuntu-latest-amd64.txt
│   ├── windows-latest-amd64.txt
│   ├── ubuntu-24.04-arm-arm64.txt
│   └── macos-latest-arm64.txt
├── pr-{N}/                        # written when PR #{N} triggers
│   ├── {os}-{arch}.txt           # HEAD results
│   ├── {os}-{arch}-base.txt      # base commit results
│   └── {os}-{arch}-diff.txt      # benchstat diff
└── manual/YYYYMMDD/               # written on manual trigger
    └── {os}-{arch}.txt
```

Each file header includes environment metadata (os / arch / kernel / go version / commit sha / date) and can be compared directly with `benchstat`:

```bash
# View latest main branch CI data
git fetch origin bench
git show origin/bench:main/ubuntu-latest-amd64.txt

# Compare with local bare-metal data (cross-hardware: reference only)
benchstat bench_linux_6c12t.txt <(git show origin/bench:main/ubuntu-latest-amd64.txt)

# View regression comparison for a PR
git show origin/bench:pr-42/ubuntu-latest-amd64-diff.txt
```

> **Note**: CI Runners (shared VMs) have systematically different performance from bare-metal. Cross-environment ns/op values cannot be compared directly. Only same-Runner PR comparisons (`*-diff.txt`) are statistically meaningful.

---

## License

[MIT](LICENSE) © 2026 uniyak.com
